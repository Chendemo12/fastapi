package fastapi

import (
	"github.com/Chendemo12/fastapi/openapi"
	"io"
	"net/http"
)

// RouteErrorFormatter 路由函数返回错误时的处理函数，可用于格式化错误信息后返回给客户端
//
//	程序启动时会主动调用此方法用于生成openApi文档，所以此函数不应返回 map等类型，否则将无法生成openApi文档
//
//	当路由函数返回错误时，会调用此函数，返回值会作为响应码和响应内容, 返回值仅限于可以JSON序列化的消息体
//	默认情况下，错误码为500，错误信息会作为字符串直接返回给客户端
type RouteErrorFormatter func(c *Context, err error) (statusCode int, resp any)

// DependenceHandle 依赖函数 Depends/Hook
type DependenceHandle func(c *Context) error

// RouteErrorOpt 错误处理函数选项, 用于在 SetRouteErrorFormatter 方法里同时设置错误码和响应内容等内容
type RouteErrorOpt struct {
	StatusCode   int    `json:"statusCode" validate:"required" description:"请求错误时的状态码"`
	ResponseMode any    `json:"responseMode" validate:"required" description:"请求错误时的响应体，空则为字符串"`
	Description  string `json:"description,omitempty" description:"错误文档"`
}

// 默认的错误处理函数
var defaultRouteErrorFormatter RouteErrorFormatter = func(c *Context, err error) (statusCode int, resp any) {
	statusCode = DefaultErrorStatusCode
	resp = err.Error()
	c.response.Type = StringResponseType
	c.response.ContentType = openapi.MIMETextPlainCharsetUTF8

	return
}

// Handler 路由函数，实现逻辑类似于装饰器
//
// 路由处理方法(装饰器实现)，用于请求体校验和返回体序列化，同时注入全局服务依赖,
// 此方法接收一个业务层面的路由钩子方法 RouteIface.Call
//
// 方法首先会查找路由元信息，如果找不到则直接跳过验证环节，由路由器返回404
// 反之：
//
//  1. 申请一个 Context, 并初始化请求体、路由参数等
//  2. 之后会校验并绑定路由参数（包含路径参数和查询参数）是否正确，如果错误则直接返回422错误，反之会继续序列化并绑定请求体（如果存在）序列化成功之后会校验请求参数的正确性，
//  3. 校验通过后会调用 RouteIface.Call 并将返回值绑定在 Context 内的 Response 上
//  4. 校验返回值，并返回422或将返回值写入到实际的 response
func (f *Wrapper) Handler(ctx MuxContext) error {
	route, exist := f.finder.Get(openapi.CreateRouteIdentify(ctx.Method(), ctx.Path()))
	if !exist {
		// 正常来说，通过 Wrapper 注册的路由，不会走到这个分支
		return nil
	}

	// 找到定义的路由信息
	wrapperCtx := f.acquireCtx(ctx)
	defer f.releaseCtx(wrapperCtx)

	// 校验前依赖函数
	var err error
	for _, dep := range f.previousDeps {
		err = dep(wrapperCtx)
		if err != nil {
			// 依赖函数中断执行
			wrapperCtx.response.StatusCode, wrapperCtx.response.Content = f.routeErrorFormatter(wrapperCtx, err)
			return f.write(wrapperCtx)
		}
	}

	// 路由前的校验,此校验会就地修改 Context.response
	hasError := wrapperCtx.beforeWorkflow(route, f.conf.StopImmediatelyWhenErrorOccurs)
	if hasError {
		// 校验工作流不通过, 中断执行
		return f.write(wrapperCtx)
	}

	// 执行校验后依赖函数
	for _, dep := range f.afterDeps {
		err = dep(wrapperCtx)
		if err != nil {
			// 依赖函数中断执行
			wrapperCtx.response.StatusCode, wrapperCtx.response.Content = f.routeErrorFormatter(wrapperCtx, err)
			return f.write(wrapperCtx)
		}
	}

	//
	// 全部校验完成，执行处理函数并获取返回值, 此处已经完成全部请求参数的校验，调用失败也存在返回值
	route.Call(wrapperCtx)
	//

	// 路由后的校验，校验失败就地修改 Response
	hasError = wrapperCtx.afterWorkflow(route, f.conf.StopImmediatelyWhenErrorOccurs)

	return f.write(wrapperCtx) // 返回消息流
}

// ----------------------------------------	路由前的各种校验工作 ----------------------------------------

// 执行用户自定义钩子函数前的工作流
func (c *Context) beforeWorkflow(route RouteIface, stopImmediately bool) (hasError bool) {
	var ves []*openapi.ValidationError

	for _, link := range requestValidateLinks {
		ves = link(c, route, stopImmediately)
		if len(ves) > 0 { // 当任意环节校验失败时,即终止下文环节
			c.response.Type = JsonResponseType
			c.response.StatusCode = http.StatusUnprocessableEntity
			c.response.ContentType = openapi.MIMEApplicationJSONCharsetUTF8
			c.response.Content = &openapi.HTTPValidationError{Detail: ves}
			return true
		}
	}

	return
}

// ----------------------------------------	路由后的响应体校验工作 ----------------------------------------

// 主要是对响应体是否符合tag约束的校验，
func (c *Context) afterWorkflow(route RouteIface, stopImmediately bool) (hasError bool) {
	var ves []*openapi.ValidationError

	for _, link := range responseValidateLinks {
		ves = link(c, route, stopImmediately)
		if len(ves) > 0 { // 当任意环节校验失败时,即终止下文环节
			// 校验不通过, 修改 Response.StatusCode 和 Response.Content
			c.response.StatusCode = http.StatusUnprocessableEntity
			c.response.ContentType = openapi.MIMEApplicationJSONCharsetUTF8
			c.response.Content = &openapi.HTTPValidationError{Detail: ves}
			return true
		}
	}
	return
}

// 写入响应体到响应字节流
func (f *Wrapper) write(c *Context) error {
	defer func() {
		if c.routeCancel != nil {
			c.routeCancel() // 当路由执行完毕时立刻关闭
		}
	}()

	// 首先设置一下默认响应状态码
	if c.response.StatusCode == 0 {
		c.response.StatusCode = http.StatusOK
	}

	f.beforeWrite(c) // 执行钩子

	c.muxCtx.Status(c.response.StatusCode)

	switch c.response.Type {
	case StringResponseType:
		// 设置返回类型
		if c.response.ContentType != "" {
			c.muxCtx.Header(openapi.HeaderContentType, c.response.ContentType)
		} else {
			c.muxCtx.Header(openapi.HeaderContentType, openapi.MIMETextPlainCharsetUTF8)
		}
		return c.muxCtx.SendString(c.response.Content.(string))

	case HtmlResponseType: // 返回HTML页面
		c.muxCtx.Header(openapi.HeaderContentType, openapi.MIMETextHTMLCharsetUTF8)
		//return c.muxCtx.Render(c.response.StatusCode, bytes.NewReader(c.response.Content.(string)))
		return c.muxCtx.SendString(c.response.Content.(string))

	case FileResponseType: // 返回一个文件
		if c.response.ContentType != "" {
			c.muxCtx.Header(openapi.HeaderContentType, c.response.ContentType)
		} else {
			c.muxCtx.Header(openapi.HeaderContentType, openapi.MIMETextPlain)
		}
		return c.muxCtx.File(c.response.Content.(string))

	case StreamResponseType: // 返回字节流
		if c.response.ContentType != "" {
			c.muxCtx.Header(openapi.HeaderContentType, c.response.ContentType)
		}
		return c.muxCtx.SendStream(c.response.Content.(io.Reader))

	default: // Json类型, any类型
		return c.muxCtx.JSON(c.response.StatusCode, c.response.Content)
	}
}

var requestValidateLinks = []func(c *Context, route RouteIface, stopImmediately bool) []*openapi.ValidationError{
	pathParamsValidate,  // 路径参数校验
	queryParamsValidate, // 查询参数校验
	structQueryValidate, // 结构体查询参数校验
	requestBodyValidate, // 请求体自动校验
}

var responseValidateLinks = []func(c *Context, route RouteIface, stopImmediately bool) []*openapi.ValidationError{
	responseValidate, // 路由返回值校验
}

// 路径参数校验, 路径参数均为字符串类型，主要验证是否存在，不校验范围、值是否合理
// 对于路径参数，如果缺少理论上不会匹配到相应的路由
func pathParamsValidate(c *Context, route RouteIface, stopImmediately bool) []*openapi.ValidationError {
	var ves []*openapi.ValidationError
	// 路径参数校验
	for _, p := range route.Swagger().PathFields {
		// 对于路径参数，JsonName 和 SchemaTitle 一致
		value := c.muxCtx.Params(p.JsonName(), "")
		c.pathFields[p.JsonName()] = value // 存储路径参数，即便是空字符串

		if value == "" { // 路径参数都是必须的
			ves = append(ves, &openapi.ValidationError{
				Loc:  []string{"path", p.SchemaTitle()},
				Msg:  PathPsIsEmpty,
				Type: "string", // 路径参数都是字符串类型
				Ctx:  whereClientError,
			})
			if stopImmediately {
				break
			}
		}
	}

	return ves
}

// 查询参数校验
//
//	对于查询参数，仅自定义校验非结构体查询参数，对于结构体查询参数通过反序列化结构体后利用validate实现
//
//	验证顺序：
//		验证是否缺少必选参数, 缺少则break
//		根据数据类型转换参数值, 不符合数值类型则break
//		验证数值是否符合范围约束, 不符合则break
//
//	对于存在多个错误的字段:
//		如果 stopImmediately=false, 则全部校验完成后再统一返回
//		反之则在遇到第一个错误后就立刻返回错误消息
//
//	@return []*openapi.ValidationError 校验结果, 若为nil则校验通过
func queryParamsValidate(c *Context, route RouteIface, stopImmediately bool) []*openapi.ValidationError {
	var ves []*openapi.ValidationError

	// 验证是否缺少必选参数
	for _, q := range route.Swagger().QueryFields {
		value := c.muxCtx.Query(q.JsonName(), "")
		if value != "" {
			// 记录传入参数值，如果是空字符串则不记录，否则会影响 Query 方法的使用
			c.queryFields[q.JsonName()] = value
		} else {
			if q.IsRequired() {
				// 但是此查询参数设置为必选
				ves = append(ves, &openapi.ValidationError{
					Loc:  []string{"query", q.JsonName()},
					Msg:  QueryPsIsEmpty,
					Type: string(q.DataType),
					Ctx:  whereClientError,
				})
				if stopImmediately {
					break
				}
			}
		}
	}

	if len(ves) > 0 { // 存在缺失参数
		return ves
	}

	// 根据数据类型转换并校验参数值，比如: 定义为int类型，但是参数值为“abc”，虽然是存在的但是不合法
	// 转换规则按照 QModel 定义进行，只有转换成功后才进行校验
	for _, binder := range route.QueryBinders() {
		v, ok := c.queryFields[binder.QModel.JsonName()]
		if !ok { // 此参数值不存在
			continue
		}

		sv := v.(string)
		value, err := binder.Method.Validate(c.routeCtx, sv)
		if err != nil {
			ves = append(ves, err...)
			if stopImmediately {
				break
			}
		} else {
			c.queryFields[binder.QModel.JsonName()] = value
		}
	}

	return ves
}

// 验证结构体查询参数(如果存在)
func structQueryValidate(c *Context, route RouteIface, stopImmediately bool) []*openapi.ValidationError {
	if !route.HasStructQuery() { // 不存在
		return nil
	}

	var ves []*openapi.ValidationError
	var instance = route.NewStructQuery()

	if c.muxCtx.BindQueryNotImplemented() {
		// 采用自定义实现
		values := map[string]any{}
		for _, q := range route.Swagger().QueryFields {
			if q.InStruct {
				v, ok := c.queryFields[q.JsonName()]
				if ok {
					values[q.JsonName()] = v
				}
			}
		}

		ves = structQueryBind.Bind(values, instance)
	} else {
		err := c.muxCtx.BindQuery(instance)
		ves = ParseValidatorError(err, openapi.RouteParamQuery, "")
	}

	c.queryStruct = instance // 关联参数

	return ves
}

// 请求体校验
func requestBodyValidate(c *Context, route RouteIface, stopImmediately bool) []*openapi.ValidationError {
	if route.Swagger().RequestContentType != openapi.MIMEApplicationJSON {
		return nil
	}

	var instance any
	var ves []*openapi.ValidationError
	var err error

	switch route.Swagger().Method {
	case http.MethodPost, http.MethodPut, http.MethodPatch:
		instance = route.NewRequestModel()
	default:
		instance = nil
	}

	if instance != nil {
		// 存在请求体,首先进行反序列化,之后校验参数是否合法,校验通过后绑定到 Context
		if c.muxCtx.ShouldBindNotImplemented() {
			err = c.muxCtx.BodyParser(instance)
			if err != nil {
				ve := ParseJsoniterError(err, openapi.RouteParamRequest, route.Swagger().RequestModel.SchemaTitle())
				ve.Ctx[modelDescLabel] = route.Swagger().RequestModel.SchemaDesc()
				ves = append(ves, ve)
			} else {
				// 反序列化成功,校验模型
				c.requestModel, ves = route.RequestBinders().Method.Validate(c.routeCtx, instance)
			}
		} else {
			err = c.muxCtx.ShouldBind(instance)
			ves = ParseValidatorError(err, openapi.RouteParamRequest, route.Swagger().RequestModel.SchemaTitle())
			if len(ves) > 0 {
				ves[0].Ctx[modelDescLabel] = route.Swagger().RequestModel.SchemaDesc()
			}
		}
	}

	return ves
}

// 返回值校验入口
func responseValidate(c *Context, route RouteIface, stopImmediately bool) []*openapi.ValidationError {
	return route.ResponseValidate(c, stopImmediately)
}
