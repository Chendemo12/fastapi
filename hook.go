package fastapi

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/Chendemo12/fastapi-tool/helper"
	"github.com/Chendemo12/fastapi/openapi"
)

type MiddlewareHandle func() // 中间件函数

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
func (c *Context) pathParamsValidate(route RouteIface) {
	// 路径参数校验
	//for _, p := range c.route.PathFields {
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
	//	c.PathFields[p.SchemaPkg()] = value
	//}
}

// 查询参数校验
//
//	@return	*Response 校验结果, 若为nil则校验通过
func (c *Context) queryParamsValidate(route RouteIface) {
	for _, q := range route.Swagger().QueryFields {
		value := c.muxCtx.Query(q.SchemaPkg())
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
func (c *Context) requestBodyValidate(route RouteIface) {
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
func (c *Context) beforeWorkflow(route RouteIface) {
	links := []func(route RouteIface){
		c.pathParamsValidate,  // 路径参数校验
		c.queryParamsValidate, // 查询参数校验
		c.requestBodyValidate, // 请求体自动校验
	}

	for _, link := range links {
		link(route)
		if c.response != nil {
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

// ----------------------------------------	路由前的各种校验工作 ----------------------------------------

// 返回值校验root入口
//
//	@return	*Response 校验结果, 若校验不通过则修改 Response.StatusCode 和 Response.Content
func (c *Context) responseValidate(route RouteIface) {
	// TODO: 修改验证方式，由 ModelBindMethod 实现
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
func (c *Context) write() error {
	defer func() {
		if c.routeCancel != nil {
			c.routeCancel() // 当路由执行完毕时立刻关闭
		}
	}()

	if c.response == nil {
		// 自定义函数无任何返回值
		return c.muxCtx.SendString("OK")
	}

	// 自定义函数存在返回值
	c.muxCtx.Status(c.response.StatusCode) // 设置一下响应头

	if c.response.StatusCode == http.StatusUnprocessableEntity {
		// 校验不通过，无需进一步解析
		return c.muxCtx.JSON(c.response.StatusCode, c.response.Content)
	}

	switch c.response.Type {

	case JsonResponseType: // Json类型
		return c.muxCtx.JSON(c.response.StatusCode, c.response.Content)

	case StringResponseType:
		return c.muxCtx.SendString(c.response.Content.(string))

	case HtmlResponseType: // 返回HTML页面
		// 设置返回类型
		c.muxCtx.SetHeader(openapi.HeaderContentType, c.response.ContentType)
		return c.muxCtx.JSON(c.response.StatusCode, c.response.Content.(string))

	case ErrResponseType:
		return c.muxCtx.JSON(c.response.StatusCode, c.response.Content)

	case StreamResponseType: // 返回字节流
		_, err := c.muxCtx.Write(c.response.Content.([]byte))
		return err

	case FileResponseType: // 返回一个文件
		//return c.muxCtx.Download(c.response.Content.(string))

	case AdvancedResponseType:
		//return c.response.Content.(openapi.Handler)(c.muxCtx)

	case CustomResponseType:
		c.muxCtx.SetHeader(openapi.HeaderContentType, c.response.ContentType)
		switch c.response.ContentType {

		case openapi.MIMETextHTML, openapi.MIMETextHTMLCharsetUTF8:
			return c.muxCtx.SendString(c.response.Content.(string))
		case openapi.MIMEApplicationJSON, openapi.MIMEApplicationJSONCharsetUTF8:
			return c.muxCtx.JSON(c.response.StatusCode, c.response.Content)
		case openapi.MIMETextXML, openapi.MIMEApplicationXML, openapi.MIMETextXMLCharsetUTF8, openapi.MIMEApplicationXMLCharsetUTF8:
			return c.muxCtx.XML(c.response.Content)
		case openapi.MIMETextPlain, openapi.MIMETextPlainCharsetUTF8:
			return c.muxCtx.SendString(c.response.Content.(string))
		//case openapi.MIMETextJavaScript, openapi.MIMETextJavaScriptCharsetUTF8:
		//case openapi.MIMEApplicationForm:
		//case openapi.MIMEOctetStream:
		//case openapi.MIMEMultipartForm:
		default:
			return c.muxCtx.JSON(c.response.StatusCode, c.response.Content)
		}
	default:
		return c.muxCtx.JSON(c.response.StatusCode, c.response.Content)
	}
	err := c.muxCtx.JSON(c.response.StatusCode, c.response.Content)
	if err != nil {
		c.Logger().Warn(fmt.Sprintf(
			"write response failed, method: '%s', url: '%s', statusCode: '%d', err: %v",
			c.muxCtx.Method(), c.muxCtx.Path(), c.response.StatusCode, err,
		))
	}
	return err
}
