package fastapi

import (
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/Chendemo12/fastapi-tool/helper"
	"github.com/Chendemo12/fastapi/openapi"
)

type MiddlewareHandle func() // 中间件函数

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

	// TODO Future: 校验前中间件
	// 路由前的校验,此校验会就地修改 Context.response
	wrapperCtx.beforeWorkflow(route, f.conf.StopImmediatelyWhenErrorOccurs)
	if wrapperCtx.response.Content != nil {
		// 校验工作流不通过, 中断执行
		return wrapperCtx.write()
	}

	// TODO Future: 执行校验后中间件

	//
	// 全部校验完成，执行处理函数并获取返回值, 此处已经完成全部请求参数的校验，调用失败也存在返回值
	route.Call(wrapperCtx)

	// 路由后的校验，校验失败就地修改 Response
	wrapperCtx.afterWorkflow(route)

	return wrapperCtx.write() // 返回消息流
}

// ----------------------------------------	路由前的各种校验工作 ----------------------------------------

// 将jsoniter 的反序列化错误转换成 接口错误类型
func jsoniterUnmarshalErrorToValidationError(err error) *openapi.ValidationError {
	// jsoniter 的反序列化错误格式：
	//
	// jsoniter.iter.ReportError():224
	//
	// 	iter.Error = fmt.Errorf("%s: %s, error found in #%v byte of ...|%s|..., bigger context ...|%s|...",
	//		operation, msg, iter.head-peekStart, parsing, context)
	//
	// 	err.Error():
	//
	// 	main.SimpleForm.name: ReadString: expmuxCtxts " or n, but found 2, error found in #10 byte of ...| "name": 23,
	//		"a|..., bigger context ...|{
	//		"name": 23,
	//		"age": "23",
	//		"sex": "F"
	// 	}|...
	msg := err.Error()
	ve := &openapi.ValidationError{Loc: []string{"body"}, Ctx: whereClientError}
	for i := 0; i < len(msg); i++ {
		if msg[i:i+1] == ":" {
			ve.Loc = append(ve.Loc, msg[:i])
			break
		}
	}
	if msgs := strings.Split(msg, jsoniterUnmarshalErrorSeparator); len(msgs) > 0 {
		_ = helper.JsonUnmarshal([]byte(msgs[jsonErrorFormIndex]), &ve.Ctx)
		ve.Msg = msgs[jsonErrorFieldMsgIndex][len(ve.Loc[1])+2:]
		if s := strings.Split(ve.Msg, ":"); len(s) > 0 {
			ve.Type = s[0]
		}
	}

	return ve
}

// 路径参数校验
//
//	@return	*Response 校验结果, 若为nil则校验通过
func (c *Context) pathParamsValidate(route RouteIface, stopImmediately bool) {
	// 路径参数校验
	//for _, p := range c.route.pathFields {
	//	value := c.muxCtx.Params(p.SchemaTitle())
	//	if p.IsRequired() && value == "" {
	//		// 不存在此路径参数, 但是此路径参数设置为必选
	//		c.response = validationErrorResponse(&openapi.ValidationError{
	//			Loc:  []string{"path", p.SchemaPkg()},
	//			Msg:  PathPsIsEmpty,
	//			Type: "string",
	//			Ctx:  whereClientError,
	//		})
	//	}
	//
	//	c.pathFields[p.SchemaPkg()] = value
	//}
}

// 查询参数校验，校验全部参数，如果存在多个错误的字段则全部校验完成后再统一返回
//
//	@return	*Response 校验结果, 若为nil则校验通过
func (c *Context) queryParamsValidate(route RouteIface, stopImmediately bool) {
	var ves []*openapi.ValidationError

	// 验证是否缺少必选参数
	for _, q := range route.Swagger().QueryFields {
		value := c.muxCtx.Query(q.SchemaTitle(), "")
		// 记录传入参数值，即便是空字符串
		c.queryFields[q.SchemaTitle()] = value
		if q.IsRequired() && value == "" {
			// 但是此查询参数设置为必选
			ves = append(ves, &openapi.ValidationError{
				Loc:  []string{"query", q.SchemaPkg()},
				Msg:  QueryPsIsEmpty,
				Type: string(q.Type),
				Ctx:  whereClientError,
			})
			if stopImmediately {
				break
			}
		}
	}

	if len(ves) > 0 { // 存在缺失参数
		c.response.StatusCode = http.StatusUnprocessableEntity
		c.response.Content = &openapi.HTTPValidationError{Detail: ves}
		c.response.Type = ErrResponseType
	}

	// 根据数据类型转换并校验参数值，比如: 定义为int类型，但是参数值为“abc”，虽然是存在的但是不合法
	// 转换规则按照 Context.queryFields 进行，只有转换成功后才进行校验
	for _, qmodel := range route.Swagger().QueryFields {
		v := c.queryFields[qmodel.SchemaTitle()]
		sv := v.(string)
		switch qmodel.SchemaType() {
		case openapi.IntegerType: // 如何区分有符号和无符号类型
			atoi, err := strconv.Atoi(sv)
			if err != nil {
				ves = append(ves, &openapi.ValidationError{
					Loc:  []string{"query", qmodel.SchemaTitle()},
					Msg:  fmt.Sprintf("value: '%s' is not an integer", sv),
					Type: string(openapi.IntegerType),
					Ctx:  whereClientError,
				})
				if stopImmediately {
					break
				}
			} else {
				// 符合规则，替换为转换后的值
				c.queryFields[qmodel.SchemaTitle()] = int64(atoi)
			}

		case openapi.BoolType:
			if sv == "false" || sv == "true" {
				atob, _ := strconv.ParseBool(sv)
				c.queryFields[qmodel.SchemaTitle()] = atob
			} else {
				ves = append(ves, &openapi.ValidationError{
					Loc:  []string{"query", qmodel.SchemaTitle()},
					Msg:  fmt.Sprintf("value: '%s' is not a bool", sv),
					Type: string(openapi.BoolType),
					Ctx:  whereClientError,
				})
				if stopImmediately {
					break
				}
			}

		case openapi.NumberType:
			atof, err := strconv.ParseFloat(sv, 64)
			if err != nil {
				ves = append(ves, &openapi.ValidationError{
					Loc:  []string{"query", qmodel.SchemaTitle()},
					Msg:  fmt.Sprintf("value: '%s' is not a number", sv),
					Type: string(openapi.NumberType),
					Ctx:  whereClientError,
				})
				if stopImmediately {
					break
				}
			} else {
				c.queryFields[qmodel.SchemaTitle()] = atof
			}
		}
	}
	if len(ves) > 0 { // 存在类型不匹配参数
		c.response.StatusCode = http.StatusUnprocessableEntity
		c.response.Content = &openapi.HTTPValidationError{Detail: ves}
		c.response.Type = ErrResponseType
	}

	// 验证数值是否符合范围约束
	for _, qmodel := range route.Swagger().QueryFields {
		v := c.queryFields[qmodel.SchemaTitle()]
		//err := route.QueryBinders()[qmodel.SchemaTitle()].Validate(v)
		//if len(err) > 0 {
		//	ves = append(ves, err...)
		//}
		_ = v
	}
	if len(ves) > 0 { // 存在范围越界参数
		c.response.StatusCode = http.StatusUnprocessableEntity
		c.response.Content = &openapi.HTTPValidationError{Detail: ves}
		c.response.Type = ErrResponseType
	}
}

// 请求体校验
//
//	@return	*Response 校验结果, 若为nil则校验通过
func (c *Context) requestBodyValidate(route RouteIface, stopImmediately bool) {
	//resp = requestBodyMarshal(userSVC, route) // 请求体序列化
	//if resp != nil {
	//	return c.Status(resp.StatusCode).JSON(resp.Content)
	//}

	//resp = c.Validate(c.RequestBody)
	//if resp != nil {
	//	return c.Status(resp.StatusCode).JSON(resp.Content)
	//}
}

// 执行用户自定义钩子函数前的工作流
func (c *Context) beforeWorkflow(route RouteIface, stopImmediately bool) {
	links := []func(route RouteIface, stopImmediately bool){
		c.pathParamsValidate,  // 路径参数校验
		c.queryParamsValidate, // 查询参数校验
		c.requestBodyValidate, // 请求体自动校验
	}

	for _, link := range links {
		link(route, stopImmediately)
		if c.response.Content != nil {
			return // 当任意环节校验失败时,即终止下文环节
		}
	}
}

// 主要是对响应体是否符合tag约束的校验，
func (c *Context) afterWorkflow(route RouteIface) {
	links := []func(route RouteIface){
		c.responseValidate, // 路由返回值校验
	}

	for _, link := range links {
		link(route)
		if c.response != nil {
			return // 当任意环节校验失败时,即终止下文环节
		}
	}
}

// ----------------------------------------	路由后的响应体校验工作 ----------------------------------------

// 返回值校验root入口
//
//	@return	*Response 校验结果, 若校验不通过则修改 Response.StatusCode 和 Response.Content
func (c *Context) responseValidate(route RouteIface) {
	// 仅校验“非422的JSONResponse”
	if c.response.Type == JsonResponseType {
		// 内部返回的 422 也不再校验
		if c.response.StatusCode != http.StatusUnprocessableEntity {
			evs := route.ResponseBinder().Validate(nil)
			if len(evs) > 0 {
				//校验不通过, 修改 Response.StatusCode 和 Response.Content
				c.response.StatusCode = http.StatusUnprocessableEntity
				c.response.ContentType = openapi.MIMEApplicationJSONCharsetUTF8
				c.response.Content = &openapi.HTTPValidationError{Detail: evs}
			}
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

	// 自定义函数存在返回值
	c.muxCtx.Status(c.response.StatusCode) // 设置一下响应头

	if c.response.StatusCode == http.StatusUnprocessableEntity {
		// 校验不通过，直接返回错误信息
		return c.muxCtx.JSON(http.StatusUnprocessableEntity, c.response.Content)
	}

	switch c.response.Type {

	case JsonResponseType: // Json类型
		return c.muxCtx.JSON(c.response.StatusCode, c.response.Content)

	case StringResponseType:
		return c.muxCtx.SendString(c.response.Content.(string))

	case HtmlResponseType: // 返回HTML页面
		// 设置返回类型
		c.muxCtx.Header(openapi.HeaderContentType, openapi.MIMETextHTMLCharsetUTF8)
		//return c.muxCtx.Render(c.response.StatusCode, bytes.NewReader(c.response.Content.(string)))
		return nil

	case ErrResponseType:
		return c.muxCtx.JSON(c.response.StatusCode, c.response.Content)

	case StreamResponseType: // 返回字节流
		return c.muxCtx.SendStream(c.response.Content.(io.Reader))

	case FileResponseType: // 返回一个文件
		return c.muxCtx.File(c.response.Content.(string))

	case AdvancedResponseType:
		//return c.response.Content.(openapi.MuxHandler)(c.muxCtx)
		return nil

	case CustomResponseType:
		c.muxCtx.Header(openapi.HeaderContentType, c.response.ContentType)
		_, err := c.muxCtx.Write(c.response.Content.([]byte))
		return err

	default:
		return c.muxCtx.JSON(c.response.StatusCode, c.response.Content)
	}
}
