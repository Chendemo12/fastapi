package fastapi

import (
	"fmt"
	"github.com/Chendemo12/fastapi/openapi"
	"io"
	"net/http"
)

// MiddlewareHandle 中间件函数
//
// 由于 Wrapper 的核心实现类似于装饰器,而非常规的中间件,因此应当通过 Wrapper 来定义中间件, 以避免通过 MuxWrapper 注册的中间件对 Wrapper 产生副作用;
// 此处中间件有校验前中间件和校验后中间件,分别通过 Wrapper.UsePrevious 和 Wrapper.UseAfter 注册;
// 当请求参数校验失败时不会执行 Wrapper.UseAfter 中间件, 请求参数会在 Wrapper.UsePrevious 执行完成之后被触发;
// 如果中间件要终止后续的流程,应返回 error, 错误消息会作为消息体返回给客户端, 响应状态码默认为400,可通过 Context.Status 进行修改;
type MiddlewareHandle func(c *Context) error

// Handler 路由函数，实现逻辑类似于中间件
//
// 路由处理方法(装饰器实现)，用于请求体校验和返回体序列化，同时注入全局服务依赖,
// 此方法接收一个业务层面的路由钩子方法 RouteIface.Call
//
// 方法首先会查找路由元信息，如果找不到则直接跳过验证环节，由路由器返回404
// 反之：
//
//  1. 申请一个 Context, 并初始化请求体、路由参数等
//  2. 之后会校验并绑定路由参数（包含路径参数和查询参数）是否正确，如果错误则直接返回422错误，反之会继续序列化并绑定请求体（如果存在）序列化成功之后会校验请求参数正确性，
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

	// 校验前中间件
	var err error
	for _, fc := range f.previousDeps {
		err = fc(wrapperCtx)
		if err != nil {
			// 中间件中断执行
			if wrapperCtx.response.StatusCode == 0 {
				wrapperCtx.response.StatusCode = http.StatusBadRequest
				wrapperCtx.response.Content = err.Error()
			} else {
				wrapperCtx.response.Content = err.Error()
			}
			return wrapperCtx.write()
		}
	}

	// 路由前的校验,此校验会就地修改 Context.response
	wrapperCtx.beforeWorkflow(route, f.conf.StopImmediatelyWhenErrorOccurs)
	if wrapperCtx.response.Content != nil {
		// 校验工作流不通过, 中断执行
		return wrapperCtx.write()
	}

	// 执行校验后中间件
	for _, fc := range f.afterDeps {
		err = fc(wrapperCtx)
		if err != nil {
			// 中间件中断执行
			if wrapperCtx.response.StatusCode == 0 {
				wrapperCtx.response.StatusCode = http.StatusBadRequest
				wrapperCtx.response.Content = err.Error()
			} else {
				wrapperCtx.response.Content = err.Error()
			}
			return wrapperCtx.write()
		}
	}
	//
	// 全部校验完成，执行处理函数并获取返回值, 此处已经完成全部请求参数的校验，调用失败也存在返回值
	route.Call(wrapperCtx)

	// 路由后的校验，校验失败就地修改 Response
	wrapperCtx.afterWorkflow(route, f.conf.StopImmediatelyWhenErrorOccurs)

	return wrapperCtx.write() // 返回消息流
}

// ----------------------------------------	路由前的各种校验工作 ----------------------------------------

// 执行用户自定义钩子函数前的工作流
func (c *Context) beforeWorkflow(route RouteIface, stopImmediately bool) {
	var ves []*openapi.ValidationError

	for _, link := range requestValidateLinks {
		ves = link(c, route, stopImmediately)
		if len(ves) > 0 { // 当任意环节校验失败时,即终止下文环节
			c.response.Type = JsonResponseType
			c.response.StatusCode = http.StatusUnprocessableEntity
			c.response.ContentType = openapi.MIMEApplicationJSONCharsetUTF8
			c.response.Content = &openapi.HTTPValidationError{Detail: ves}
			break
		}
	}
}

// ----------------------------------------	路由后的响应体校验工作 ----------------------------------------

// 主要是对响应体是否符合tag约束的校验，
func (c *Context) afterWorkflow(route RouteIface, stopImmediately bool) {
	var ves []*openapi.ValidationError

	for _, link := range responseValidateLinks {
		ves = link(c, route, stopImmediately)
		if len(ves) > 0 { // 当任意环节校验失败时,即终止下文环节
			// 校验不通过, 修改 Response.StatusCode 和 Response.Content
			c.response.StatusCode = http.StatusUnprocessableEntity
			c.response.ContentType = openapi.MIMEApplicationJSONCharsetUTF8
			c.response.Content = &openapi.HTTPValidationError{Detail: ves}
			break
		}
	}
}

// 写入响应体到响应字节流
func (c *Context) write() error {
	defer func() {
		if c.routeCancel != nil {
			c.routeCancel() // 当路由执行完毕时立刻关闭
		}
	}()

	if c.response == nil {
		// 自定义函数无任何返回值
		c.muxCtx.Status(http.StatusOK)
		return c.muxCtx.SendString("OK")
	}

	if c.response.StatusCode == http.StatusUnprocessableEntity {
		// 校验不通过，直接返回错误信息
		return c.muxCtx.JSON(http.StatusUnprocessableEntity, c.response.Content)
	}

	// 自定义函数存在返回值, 首先设置一下响应头
	if c.response.StatusCode == 0 {
		c.muxCtx.Status(http.StatusOK)
	} else {
		c.muxCtx.Status(c.response.StatusCode)
	}

	switch c.response.Type {

	case JsonResponseType, ErrResponseType: // Json类型
		return c.muxCtx.JSON(c.response.StatusCode, c.response.Content)

	case StringResponseType:
		return c.muxCtx.SendString(c.response.Content.(string))

	case HtmlResponseType: // 返回HTML页面
		// 设置返回类型
		c.muxCtx.Header(openapi.HeaderContentType, openapi.MIMETextHTMLCharsetUTF8)
		//return c.muxCtx.Render(c.response.StatusCode, bytes.NewReader(c.response.Content.(string)))
		return nil

	case FileResponseType: // 返回一个文件
		return c.muxCtx.File(c.response.Content.(string))

	case StreamResponseType: // 返回字节流
		return c.muxCtx.SendStream(c.response.Content.(io.Reader))

	case CustomResponseType:
		c.muxCtx.Header(openapi.HeaderContentType, c.response.ContentType)
		_, err := c.muxCtx.Write(c.response.Content.([]byte))
		return err

	default:
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
			ves = append(ves, &openapi.ValidationError{
				Loc:  []string{"query", binder.QModel.JsonName()},
				Msg:  fmt.Sprintf("value: '%s' is not a number", sv),
				Type: string(binder.QModel.SchemaType()),
				Ctx:  whereClientError,
			})
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
