package fastapi

import (
	"bytes"
	"fmt"
	"github.com/Chendemo12/fastapi/utils"
	"net/http"
	"reflect"
	"strings"

	"github.com/Chendemo12/fastapi-tool/helper"
	"github.com/Chendemo12/fastapi/openapi"
)

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

// 路径参数校验
//
//	@return	*Response 校验结果, 若为nil则校验通过
func (c *Context) pathParamsValidate(route RouteIface) {
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
func (c *Context) workflow(route RouteIface) {
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
	defer c.routeCancel() // 当路由执行完毕时立刻关闭

	err := c.muxCtx.Write(c.response)
	if err != nil {
		c.Logger().Warn(fmt.Sprintf(
			"write response failed, method: '%s', url: '%s', statusCode: '%d', err: %v",
			c.muxCtx.Method(), c.muxCtx.Path(), c.response.StatusCode, err,
		))
	}
	return err
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
