package fastapi

import (
	"bytes"
	"github.com/Chendemo12/fastapi/godantic"
	"github.com/Chendemo12/fastapi/internal/core"
	"github.com/Chendemo12/fastapi/openapi"
	"github.com/Chendemo12/fastapi/tool"
	"github.com/gofiber/fiber/v2"
	fiberu "github.com/gofiber/fiber/v2/utils"
	"io"
	"net/http"
	"reflect"
	"strings"
)

var recoverHandler StackTraceHandlerFunc = nil
var fiberErrorHandler fiber.ErrorHandler = nil // 设置fiber自定义错误处理函数

// HandlerFunc 路由处理函数
type HandlerFunc = func(s *Context) *Response

// StackTraceHandlerFunc 错误堆栈处理函数, 即 recover 方法
type StackTraceHandlerFunc = func(c *fiber.Ctx, e any)

// RouteModelValidateHandlerFunc 返回值校验方法
//
//	@param	resp	any					响应体
//	@param	meta	*godantic.Metadata	模型元数据
//	@return	*Response 响应体
type RouteModelValidateHandlerFunc func(resp any, meta *godantic.Metadata) *Response

// routeHandler 路由处理方法(装饰器实现)，用于请求体校验和返回体序列化，同时注入全局服务依赖,
// 此方法接收一个业务层面的路由钩子方法 HandlerFunc，
// 该方法有且仅有1个参数：&Context, 有且必须有一个返回值 *Response
//
// routeHandler 方法首先会申请一个 Context, 并初始化请求体、路由参数、fiber.Ctx
// 之后会校验并绑定路由参数（包含路径参数和查询参数）是否正确，如果错误则直接返回422错误，反之会继续序列化并绑定请求体（如果存在）
// 序列化成功之后会校验请求参数正确性（开关控制），校验通过后会接着将ctx传入handler
// 执行handler之后将校验返回值（开关控制），并返回422或写入响应体。
//
//	@return	fiber.Handler fiber路由处理方法
func routeHandler(handler HandlerFunc) fiber.Handler {
	return func(c *fiber.Ctx) error {
		// Acquire Ctx with fiber.Ctx request from pool
		ctx := appEngine.AcquireCtx(c)
		// Release Ctx to pool
		defer appEngine.ReleaseCtx(ctx)

		if ctx.findRoute() != nil { // 获取请求路由 -- 存在路由信息
			ctx.workflow() // 路由前的校验
		}
		if ctx.response != nil {
			// 校验工作流不通过, 中断执行
			return ctx.write()
		}

		//
		// 执行处理函数并获取返回值
		ctx.response = handler(ctx)
		// 路由返回值校验
		ctx.responseBodyValidate()
		return ctx.write() // 返回消息流
	}
}

// 将jsoniter 的反序列化错误转换成 接口错误类型
func jsoniterUnmarshalErrorToValidationError(err error) *godantic.ValidationError {
	// jsoniter 的反序列化错误格式：
	//
	// jsoniter.iter.ReportError():224
	//
	// 	iter.Error = fmt.Errorf("%s: %s, error found in #%v byte of ...|%s|..., bigger context ...|%s|...",
	//		operation, msg, iter.head-peekStart, parsing, context)
	//
	// 	err.Error():
	//
	// 	main.SimpleForm.name: ReadString: expects " or n, but found 2, error found in #10 byte of ...| "name": 23,
	//		"a|..., bigger context ...|{
	//		"name": 23,
	//		"age": "23",
	//		"sex": "F"
	// 	}|...
	msg := err.Error()
	ve := &godantic.ValidationError{Loc: []string{"body"}, Ctx: whereClientError}
	for i := 0; i < len(msg); i++ {
		if msg[i:i+1] == ":" {
			ve.Loc = append(ve.Loc, msg[:i])
			break
		}
	}
	if msgs := strings.Split(msg, jsoniterUnmarshalErrorSeparator); len(msgs) > 0 {
		_ = tool.Unmarshal([]byte(msgs[jsonErrorFormIndex]), &ve.Ctx)
		ve.Msg = msgs[jsonErrorFieldMsgIndex][len(ve.Loc[1])+2:]
		if s := strings.Split(ve.Msg, ":"); len(s) > 0 {
			ve.Type = s[0]
		}
	}

	return ve
}

// 查找注册的路由，校验的基础
func (c *Context) findRoute() *Route {
	// Route().Path 获取注册的路径，
	c.route = c.svc.queryRoute(c.ec.Method(), c.ec.Route().Path)
	return c.route
}

// 路径参数校验
//
//	@return	*Response 校验结果, 若为nil则校验通过
func (c *Context) pathParamsValidate() {
	// 路径参数校验
	for _, p := range c.route.PathFields {
		value := c.ec.Params(p.SchemaName())
		if p.IsRequired() && value == "" {
			// 不存在此路径参数, 但是此路径参数设置为必选
			c.response = validationErrorResponse(&godantic.ValidationError{
				Loc:  []string{"path", p.SchemaName()},
				Msg:  PathPsIsEmpty,
				Type: "string",
				Ctx:  whereClientError,
			})
		}

		c.PathFields[p.SchemaName()] = value
	}
}

// 查询参数校验
//
//	@return	*Response 校验结果, 若为nil则校验通过
func (c *Context) queryParamsValidate() {
	for _, q := range c.route.QueryFields {
		value := c.ec.Query(q.SchemaName())
		if q.IsRequired() && value == "" {
			// 但是此查询参数设置为必选
			c.response = validationErrorResponse(&godantic.ValidationError{
				Loc:  []string{"query", q.SchemaName()},
				Msg:  QueryPsIsEmpty,
				Type: "string",
				Ctx:  whereClientError,
			})
		}
		c.QueryFields[q.SchemaName()] = value
	}
}

func (c *Context) dependencyDone() {
	for i := 0; i < len(c.route.Dependencies); i++ {
		if resp := c.route.Dependencies[i](c); resp != nil {
			c.response = resp
			break
		}
	}
}

// 请求体校验
//
//	@return	*Response 校验结果, 若为nil则校验通过
func (c *Context) requestBodyValidate() {
	if core.RequestValidateDisabled {
		return
	}
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
func (c *Context) workflow() {
	links := []func(){
		// 路径参数和查询参数校验
		c.pathParamsValidate,  // 路由参数校验
		c.queryParamsValidate, // 查询参数校验
		c.requestBodyValidate, // 请求体自动校验
	}

	for _, link := range links {
		link()
		if c.response != nil {
			return // 当任意环节校验失败时,即终止下文环节
		}
	}

	// ------------------------------- 校验通过或禁用自动校验 -------------------------------
	// 处理依赖项
	c.dependencyDone()
}

// ----------------------------------------	路由前的各种校验工作 ----------------------------------------

// 返回值校验root入口
//
//	@return	*Response 校验结果, 若校验不通过则修改 Response.StatusCode 和 Response.Content
func (c *Context) responseBodyValidate() {

	// 对于返回值类型，允许缺省返回值以屏蔽返回值校验
	if c.route.ResponseModel == nil || core.ResponseValidateDisabled {
		return
	}

	// 仅校验 JSONResponse 和 StringResponse
	if c.response.Type != JsonResponseType && c.response.Type != StringResponseType {
		return
	}

	// 返回值校验，若响应体为nil或关闭了参数校验，则返回原内容
	resp := c.route.responseValidate(c.response.Content, c.route.ResponseModel)

	// 校验不通过, 修改 Response.StatusCode 和 Response.Content
	if resp != nil {
		c.response.StatusCode = resp.StatusCode
		c.response.Content = resp.Content
		c.Logger().Warn(c.response.Content)
	}
}

// ----------------------------------------	路由后的响应体校验工作 ----------------------------------------

// 写入响应体到响应字节流
func (c *Context) write() error {
	defer c.routeCancel() // 当路由执行完毕时立刻关闭
	if c.response == nil {
		// 自定义函数无任何返回值
		return c.ec.Status(fiber.StatusOK).SendString(fiberu.StatusMessage(fiber.StatusOK))
	}

	// 自定义函数存在返回值
	c.ec.Status(c.response.StatusCode) // 设置一下响应头

	if c.response.StatusCode == http.StatusUnprocessableEntity {
		return c.ec.JSON(c.response.Content)
	}

	switch c.response.Type {

	case JsonResponseType: // Json类型
		return c.ec.JSON(c.response.Content)

	case StringResponseType:
		return c.ec.SendString(c.response.Content.(string))

	case HtmlResponseType: // 返回HTML页面
		// 设置返回类型
		c.ec.Set(fiber.HeaderContentType, c.response.ContentType)
		return c.ec.SendString(c.response.Content.(string))

	case ErrResponseType:
		return c.ec.JSON(c.response.Content)

	case StreamResponseType: // 返回字节流
		return c.ec.SendStream(c.response.Content.(io.Reader))

	case FileResponseType: // 返回一个文件
		return c.ec.Download(c.response.Content.(string))

	case AdvancedResponseType:
		return c.response.Content.(fiber.Handler)(c.ec)

	case CustomResponseType:
		c.ec.Set(fiber.HeaderContentType, c.response.ContentType)
		switch c.response.ContentType {

		case fiber.MIMETextHTML, fiber.MIMETextHTMLCharsetUTF8:
			return c.ec.SendString(c.response.Content.(string))
		case fiber.MIMEApplicationJSON, fiber.MIMEApplicationJSONCharsetUTF8:
			return c.ec.JSON(c.response.Content)
		case fiber.MIMETextXML, fiber.MIMEApplicationXML, fiber.MIMETextXMLCharsetUTF8, fiber.MIMEApplicationXMLCharsetUTF8:
			return c.ec.XML(c.response.Content)
		case fiber.MIMETextPlain, fiber.MIMETextPlainCharsetUTF8:
			return c.ec.SendString(c.response.Content.(string))
		//case fiber.MIMETextJavaScript, fiber.MIMETextJavaScriptCharsetUTF8:
		//case fiber.MIMEApplicationForm:
		//case fiber.MIMEOctetStream:
		//case fiber.MIMEMultipartForm:
		default:
			return c.ec.JSON(c.response.Content)
		}
	default:
		return c.ec.JSON(c.response.Content)
	}
}

// 未定义返回值或关闭了返回值校验
func routeModelDoNothing(content any, meta *godantic.Metadata) *Response {
	return nil
}

func boolResponseValidation(content any, meta *godantic.Metadata) *Response {
	rt := godantic.ReflectObjectType(content)
	if rt.Kind() != reflect.Bool {
		// 校验不通过, 修改 Response.StatusCode 和 Response.Content
		return modelCannotBeBoolResponse(meta.Name())
	}

	return nil
}

func stringResponseValidation(content any, meta *godantic.Metadata) *Response {
	// TODO: 对于字符串类型，减少内存复制
	if meta.SchemaType() != godantic.StringType {
		return modelCannotBeStringResponse(meta.Name())
	}

	return nil
}

func integerResponseValidation(content any, meta *godantic.Metadata) *Response {
	rt := godantic.ReflectObjectType(content)
	if godantic.ReflectKindToOType(rt.Kind()) != godantic.IntegerType {
		return modelCannotBeIntegerResponse(meta.Name())
	}

	return nil
}

func numberResponseValidation(content any, meta *godantic.Metadata) *Response {
	rt := godantic.ReflectObjectType(content)
	if godantic.ReflectKindToOType(rt.Kind()) != godantic.NumberType {
		return modelCannotBeNumberResponse(meta.Name())
	}

	return nil
}

func arrayResponseValidation(content any, meta *godantic.Metadata) *Response {
	rt := godantic.ReflectObjectType(content)
	if godantic.ReflectKindToOType(rt.Kind()) != godantic.ArrayType {
		// TODO: notImplemented 暂不校验子元素
		return modelCannotBeArrayResponse("Array")
	} else {
		if rt.Elem().Kind() == reflect.Uint8 { // 对于字节流对象, 覆盖以响应正确的数值
			return &Response{
				StatusCode:  http.StatusOK,
				Content:     bytes.NewReader(content.([]byte)),
				Type:        StreamResponseType,
				ContentType: openapi.MIMETextPlain,
			}
		}
	}

	return nil
}

func structResponseValidation(content any, meta *godantic.Metadata) *Response {
	rt := godantic.ReflectObjectType(content)
	// 类型校验
	if rt.Kind() != reflect.Struct && meta.String() != rt.String() {
		return objectModelNotMatchResponse(meta.String(), rt.String())
	}
	// 字段类型校验, 字段的值需符合tag要求
	resp := appEngine.service.Validate(content, whereServerError)
	if resp != nil {
		return resp
	}

	return nil
}
