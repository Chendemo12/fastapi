package fastapi

import (
	"fmt"
	"net/http"

	"github.com/Chendemo12/fastapi/openapi"
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
	StatusCode   int                 `json:"statusCode" validate:"required" description:"请求错误时的状态码"`
	Description  string              `json:"description,omitempty" description:"错误文档"`
	ContentType  openapi.ContentType `json:"contentType" validate:"required" description:"请求错误时的内容类型"`
	ResponseMode any                 `json:"responseMode" validate:"required" description:"请求错误时的响应体，空则为字符串"`
}

// 默认的错误处理函数
var defaultRouteErrorFormatter RouteErrorFormatter = func(c *Context, err error) (statusCode int, resp any) {
	if c.response.StatusCode != 0 {
		statusCode = c.response.StatusCode
	} else {
		statusCode = DefaultErrorStatusCode
	}
	resp = err.Error()

	return
}

// Handler 路由函数 MuxHandler，实现逻辑类似于装饰器
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
			return f.write(wrapperCtx, route, openapi.MIMEApplicationJSONCharsetUTF8)
		}
	}

	// 路由前的校验,此校验会就地修改 Context.response
	hasError := wrapperCtx.beforeWorkflow(route, f.conf.StopImmediatelyWhenErrorOccurs)
	if hasError {
		// 校验工作流不通过, 中断执行
		return f.write(wrapperCtx, route, openapi.MIMEApplicationJSONCharsetUTF8)
	}

	// 执行校验后依赖函数
	for _, dep := range f.afterDeps {
		err = dep(wrapperCtx)
		if err != nil {
			// 依赖函数中断执行
			wrapperCtx.response.StatusCode, wrapperCtx.response.Content = f.routeErrorFormatter(wrapperCtx, err)
			return f.write(wrapperCtx, route, openapi.MIMEApplicationJSONCharsetUTF8)
		}
	}

	//
	// 全部校验完成，执行处理函数并获取返回值, 此处已经完成全部请求参数的校验，调用失败也存在返回值
	params := route.NewInParams(wrapperCtx)
	result := route.Call(params)
	last := result[LastOutParamOffset]
	if last.IsNil() || !last.IsValid() {
		// err=nil, 不存在错误，则校验返回值，如果存在错误，则直接返回错误信息
		wrapperCtx.response.StatusCode = http.StatusOK
		wrapperCtx.response.Content = result[FirstOutParamOffset].Interface()

		// 路由后的校验，校验失败就地修改 Response
		hasError = wrapperCtx.afterWorkflow(route, f.conf.DisableResponseValidate, f.conf.StopImmediatelyWhenErrorOccurs)
		if hasError {
			// 校验工作流不通过, 中断执行
			return f.write(wrapperCtx, route, openapi.MIMEApplicationJSONCharsetUTF8)
		} else {
			// 路由正常响应
			return f.write(wrapperCtx, route, route.Swagger().ResponseContentType) // 返回消息流
		}
	} else {
		// 存在错误，则返回错误信息
		err := last.Interface().(error)
		wrapperCtx.response.StatusCode, wrapperCtx.response.Content = f.routeErrorFormatter(wrapperCtx, err)

		return f.write(wrapperCtx, route, openapi.MIMEApplicationJSONCharsetUTF8)
	}
}

// ----------------------------------------	路由前的各种校验工作 ----------------------------------------

// 执行用户自定义钩子函数前的工作流
func (c *Context) beforeWorkflow(route RouteIface, stopImmediately bool) (hasError bool) {
	var ves []*openapi.ValidationError

	for _, link := range requestValidateLinks {
		ves = link(c, route, stopImmediately)
		if len(ves) > 0 { // 当任意环节校验失败时,即终止下文环节
			c.response.StatusCode = http.StatusUnprocessableEntity
			c.response.Content = &openapi.HTTPValidationError{Detail: ves}
			return true
		}
	}

	return
}

// ----------------------------------------	路由后的响应体校验工作 ----------------------------------------

// 主要是对响应体是否符合tag约束的校验，
func (c *Context) afterWorkflow(route RouteIface, disableResponseValidate, stopImmediately bool) (hasError bool) {
	if disableResponseValidate {
		return
	}

	var ves []*openapi.ValidationError

	for _, link := range responseValidateLinks {
		ves = link(c, route, stopImmediately)
		if len(ves) > 0 { // 当任意环节校验失败时,即终止下文环节
			// 校验不通过, 修改 Response.StatusCode 和 Response.Content
			c.response.StatusCode = http.StatusUnprocessableEntity
			c.response.Content = &openapi.HTTPValidationError{Detail: ves}
			return true
		}
	}
	return
}

// 写入响应体, 依据 contentType 的不同，有不同的写入行为
func (f *Wrapper) write(c *Context, route RouteIface, contentType openapi.ContentType) error {
	defer func() {
		if c.routeCancel != nil {
			c.routeCancel() // 当路由执行完毕时立刻关闭
		}
	}()

	f.beforeWrite(c) // 执行钩子

	// 设置状态码
	c.muxCtx.Status(c.response.StatusCode)

	switch contentType {
	case openapi.MIMEApplicationJSON, openapi.MIMEApplicationJSONCharsetUTF8:
		return c.muxCtx.JSON(c.response.StatusCode, c.response.Content)

	case openapi.MIMETextPlainCharsetUTF8, openapi.MIMETextPlain:
		return c.muxCtx.SendString(c.response.Content.(string))

	case openapi.MIMEOctetStream: // 返回一个字节流或文件
		if file, ok := c.response.Content.(*FileResponse); !ok {
			c.muxCtx.Status(http.StatusInternalServerError)
			return c.muxCtx.JSON(http.StatusInternalServerError, fmt.Sprintf("'%s' the return value type is not *FileResponse", route.Swagger().RelativePath))
		} else { // 返回一个文件
			switch file.mode {
			case FileResponseModeSendFile:
				return c.muxCtx.File(file.filepath)
			case FileResponseModeFileAttachment: // 文件附件
				return c.muxCtx.FileAttachment(file.filepath, file.filename)
			case FileResponseModeReaderFile:
				c.muxCtx.Header(openapi.HeaderContentType, string(contentType))
				c.muxCtx.Header("Content-Disposition", fmt.Sprintf("attachment; filename=%s", file.filename))
				return c.muxCtx.SendStream(file.reader, -1)
			case FileResponseModeStream: // 任意字节流
				c.muxCtx.Header(openapi.HeaderContentType, string(contentType))
				return c.muxCtx.SendStream(file.reader, -1)
			default:
				return c.muxCtx.JSON(http.StatusInternalServerError, fmt.Sprintf("'%s' the return value has wrong field", route.Swagger().RelativePath))
			}
		}

	case openapi.MIMEEventStreamCharsetUTF8, openapi.MIMEEventStream:
		// sse 推流没有明确的结束信息
		return nil

	default: // Json类型, any类型
		c.muxCtx.Header(openapi.HeaderContentType, string(contentType))
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
				Type: string(openapi.StringType), // 路径参数都是字符串类型
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
		v, ok := c.queryFields[binder.ModelName()]
		if !ok { // 此参数值不存在
			continue
		}

		sv := v.(string)
		value, err := binder.Validate(c, sv)
		if err != nil {
			ves = append(ves, err...)
			if stopImmediately {
				break
			}
		} else {
			c.queryFields[binder.ModelName()] = value
		}
	}

	return ves
}

// 验证结构体查询参数(如果存在)
func structQueryValidate(c *Context, route RouteIface, stopImmediately bool) []*openapi.ValidationError {
	if !route.HasStructQuery() { // 不存在
		return nil
	}

	c.queryStruct = route.NewStructQuery()

	values := map[string]any{}
	for _, q := range route.Swagger().QueryFields {
		if q.InStruct {
			v, ok := c.queryFields[q.JsonName()]
			if ok {
				values[q.JsonName()] = v
			}
		}
	}

	return structQueryBind.Bind(values, c.queryStruct)
}

// 请求体校验, 支持识别文件
func requestBodyValidate(c *Context, route RouteIface, stopImmediately bool) []*openapi.ValidationError {
	var ves []*openapi.ValidationError
	if route.Swagger().RequestModel != nil {
		requestParam := route.NewRequestModel()
		c.requestModel, ves = route.RequestBinders().Validate(c, requestParam)
	}

	return ves
}

// 返回值校验入口
func responseValidate(c *Context, route RouteIface, stopImmediately bool) []*openapi.ValidationError {
	if c.response.StatusCode == http.StatusOK || c.response.StatusCode == 0 {
		// TODO: 对于JSON类型，此校验浪费性能, 尝试通过某种方式绕过
		_, ves := route.ResponseBinder().Validate(c, c.response.Content)
		if len(ves) > 0 {
			ves[0].Ctx[modelDescLabel] = route.Swagger().ResponseModel.SchemaDesc()
		}
		return ves
	}

	return nil
}
