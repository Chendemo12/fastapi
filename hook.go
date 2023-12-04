package fastapi

import (
	"bytes"
	"github.com/Chendemo12/fastapi/utils"
	"io"
	"net/http"
	"reflect"
	"strings"

	"github.com/Chendemo12/fastapi-tool/helper"
	"github.com/Chendemo12/fastapi/openapi"
	"github.com/gofiber/fiber/v2"
	fiberu "github.com/gofiber/fiber/v2/utils"
)

// HandlerFunc 路由处理函数
type HandlerFunc = func(c *Context) *Response

// Deprecated:RouteModelValidateHandlerFunc 返回值校验方法
//
//	@param	resp	any					响应体
//	@param	meta	*openapi.BaseModelMeta	模型元数据
//	@return	*Response 响应体
type RouteModelValidateHandlerFunc func(resp any, meta *openapi.BaseModelMeta) *Response

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
	// 	main.SimpleForm.name: ReadString: expects " or n, but found 2, error found in #10 byte of ...| "name": 23,
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

// 查找注册的路由，校验的基础
func (c *Context) findRoute() RouteIface {
	// TODO Future:
	// Route().Path 获取注册的路径，
	id := openapi.CreateRouteIdentify(c.ec.Method(), c.ec.Route().Path)
	item, ok := wrapper.finder.Get(id)
	if !ok {
		return nil
	}
	route, ok := item.(*GroupRoute)
	if !ok {
		return nil
	}
	return route
}

// 路径参数校验
//
//	@return	*Response 校验结果, 若为nil则校验通过
func (c *Context) pathParamsValidate() {
	// 路径参数校验
	//for _, p := range c.route.PathFields {
	//	value := c.ec.Params(p.SchemaTitle())
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
	//	c.PathFields[p.SchemaPkg()] = value
	//}
}

// 查询参数校验
//
//	@return	*Response 校验结果, 若为nil则校验通过
func (c *Context) queryParamsValidate() {
	for _, q := range c.route.Swagger().QueryFields {
		value := c.ec.Query(q.SchemaPkg())
		if q.IsRequired() && value == "" {
			// 但是此查询参数设置为必选
			c.response = validationErrorResponse(&openapi.ValidationError{
				Loc:  []string{"query", q.SchemaPkg()},
				Msg:  QueryPsIsEmpty,
				Type: "string",
				Ctx:  whereClientError,
			})
		}
		c.QueryFields[q.SchemaPkg()] = value
	}
}

// 请求体校验
//
//	@return	*Response 校验结果, 若为nil则校验通过
func (c *Context) requestBodyValidate() {
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
}

// ----------------------------------------	路由前的各种校验工作 ----------------------------------------

// 返回值校验root入口
//
//	@return	*Response 校验结果, 若校验不通过则修改 Response.StatusCode 和 Response.Content
func (c *Context) responseBodyValidate() {
	// 仅校验 JSONResponse 和 StringResponse
	if c.response.Type != JsonResponseType && c.response.Type != StringResponseType {
		return
	}

	// 返回值校验，若响应体为nil或关闭了参数校验，则返回原内容
	//resp := c.route.responseValidate(c.response.Content, c.route.Swagger().ResponseModel)

	// 校验不通过, 修改 Response.StatusCode 和 Response.Content
	//if resp != nil {
	//	c.response.StatusCode = resp.StatusCode
	//	c.response.Content = resp.Content
	//	c.Logger().Warn(c.response.Content)
	//}
}

// ----------------------------------------	路由后的响应体校验工作 ----------------------------------------

// 写入响应体到响应字节流
func (c *Context) write() *Response {
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
func routeModelDoNothing(content any, meta *openapi.BaseModelMeta) *Response {
	return nil
}

func boolResponseValidation(content any, meta *openapi.BaseModelMeta) *Response {
	rt := utils.ReflectObjectType(content)
	if rt.Kind() != reflect.Bool {
		// 校验不通过, 修改 Response.StatusCode 和 Response.Content
		return modelCannotBeBoolResponse(meta.Name())
	}

	return nil
}

func stringResponseValidation(content any, meta *openapi.BaseModelMeta) *Response {
	// TODO: 对于字符串类型，减少内存复制
	if meta.SchemaType() != openapi.StringType {
		return modelCannotBeStringResponse(meta.Name())
	}

	return nil
}

func integerResponseValidation(content any, meta *openapi.BaseModelMeta) *Response {
	rt := utils.ReflectObjectType(content)
	if openapi.ReflectKindToType(rt.Kind()) != openapi.IntegerType {
		return modelCannotBeIntegerResponse(meta.Name())
	}

	return nil
}

func numberResponseValidation(content any, meta *openapi.BaseModelMeta) *Response {
	rt := utils.ReflectObjectType(content)
	if openapi.ReflectKindToType(rt.Kind()) != openapi.NumberType {
		return modelCannotBeNumberResponse(meta.Name())
	}

	return nil
}

func arrayResponseValidation(content any, meta *openapi.BaseModelMeta) *Response {
	rt := utils.ReflectObjectType(content)
	if openapi.ReflectKindToType(rt.Kind()) != openapi.ArrayType {
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

func structResponseValidation(content any, meta *openapi.BaseModelMeta) *Response {
	// 类型校验
	//rt := openapi.ReflectObjectType(content)
	//if rt.Kind() != reflect.Struct && meta.String() != rt.String() {
	//	return objectModelNotMatchResponse(meta.String(), rt.String())
	//}
	// 字段类型校验, 字段的值需符合tag要求
	resp := wrapper.service.Validate(content, whereServerError)
	if resp != nil {
		return resp
	}

	return nil
}
