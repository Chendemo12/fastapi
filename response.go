package fastapi

import (
	"fmt"
	"github.com/Chendemo12/fastapi/godantic"
	"net/http"
)

type ResponseType int

const (
	CustomResponseType ResponseType = iota + 1
	JsonResponseType
	StringResponseType
	StreamResponseType
	FileResponseType
	ErrResponseType
	HtmlResponseType
	AdvancedResponseType
)

const ( // error message
	ModelNotDefine     = "Data model is undefined"
	ModelNotMatch      = "Value type mismatch"
	ModelCannotString  = "The return value cannot be a string"
	ModelCannotNumber  = "The return value cannot be a number"
	ModelCannotInteger = "The return value cannot be a integer"
	ModelCannotBool    = "The return value cannot be a boolean"
	ModelCannotArray   = "The return value cannot be a array"
	PathPsIsEmpty      = "Path must not be empty"
	QueryPsIsEmpty     = "Query must not be empty"
)

var responseHeaders []*ResponseHeader
var emptyLocList = []string{"response"}
var whereServerError = map[string]any{"where error": "server"}
var whereClientError = map[string]any{"where error": "client"}

// ResponseHeader 自定义响应头
type ResponseHeader struct {
	Key   string `json:"key" Description:"Key" binding:"required"`
	Value string `json:"value" Description:"Value" binding:"required"`
}

// Response 路由返回值
type Response struct {
	Content     any          `json:"content"`     // 响应体
	ContentType string       `json:"contentType"` // 响应类型,默认为 application/json
	Type        ResponseType `json:"type"`        // 返回体类型
	StatusCode  int          `json:"status_code"` // 响应状态码
}

// validationErrorResponse 参数校验错误返回值
func validationErrorResponse(ves ...*godantic.ValidationError) *Response {
	return &Response{
		StatusCode: http.StatusUnprocessableEntity,
		Content:    &godantic.HTTPValidationError{Detail: ves},
		Type:       ErrResponseType,
	}
}

func modelCannotBeStringResponse(name ...string) *Response {
	vv := &godantic.ValidationError{
		Ctx:  map[string]any{"where error": "server", "msg": ModelCannotString},
		Msg:  ModelNotMatch,
		Type: string(godantic.StringType),
		Loc:  emptyLocList,
	}

	if len(name) > 0 {
		vv.Ctx["msg"] = fmt.Sprintf(
			"response model should be '%s', but 'string' returned", name[0],
		)
	}

	return &Response{
		StatusCode: http.StatusUnprocessableEntity,
		Content:    &godantic.HTTPValidationError{Detail: []*godantic.ValidationError{vv}},
	}
}

func modelCannotBeNumberResponse(name ...string) *Response {
	vv := &godantic.ValidationError{
		Ctx:  map[string]any{"where error": "server", "msg": ModelCannotNumber},
		Msg:  ModelNotMatch,
		Type: string(godantic.NumberType),
		Loc:  emptyLocList,
	}
	if len(name) > 0 {
		vv.Ctx["msg"] = fmt.Sprintf(
			"response model should be '%s', but 'string' returned", name[0],
		)
	}

	return &Response{
		StatusCode: http.StatusUnprocessableEntity,
		Content:    &godantic.HTTPValidationError{Detail: []*godantic.ValidationError{vv}},
	}
}

func modelCannotBeBoolResponse(name ...string) *Response {
	vv := &godantic.ValidationError{
		Ctx:  map[string]any{"where error": "server", "msg": ModelCannotBool},
		Msg:  ModelNotMatch,
		Type: string(godantic.BoolType),
		Loc:  emptyLocList,
	}
	if len(name) > 0 {
		vv.Ctx["msg"] = fmt.Sprintf(
			"response model should be '%s', but 'string' returned", name[0],
		)
	}
	return &Response{
		StatusCode: http.StatusUnprocessableEntity,
		Content:    &godantic.HTTPValidationError{Detail: []*godantic.ValidationError{vv}},
	}
}

func modelCannotBeIntegerResponse(name ...string) *Response {
	vv := &godantic.ValidationError{
		Ctx:  map[string]any{"where error": "server", "msg": ModelCannotInteger},
		Msg:  ModelNotMatch,
		Type: string(godantic.IntegerType),
		Loc:  emptyLocList,
	}
	if len(name) > 0 {
		vv.Ctx["msg"] = fmt.Sprintf(
			"response model should be '%s', but 'string' returned", name[0],
		)
	}
	return &Response{
		StatusCode: http.StatusUnprocessableEntity,
		Content:    &godantic.HTTPValidationError{Detail: []*godantic.ValidationError{vv}},
	}
}

func modelCannotBeArrayResponse(name ...string) *Response {
	vv := &godantic.ValidationError{
		Ctx:  map[string]any{"where error": "server", "msg": ModelCannotArray},
		Msg:  ModelNotMatch,
		Type: string(godantic.ArrayType),
		Loc:  emptyLocList,
	}
	if len(name) > 0 {
		vv.Ctx["msg"] = fmt.Sprintf(
			"response model should be '%s', but 'string' returned", name[0],
		)
	}

	return &Response{
		StatusCode: http.StatusUnprocessableEntity,
		Content:    &godantic.HTTPValidationError{Detail: []*godantic.ValidationError{vv}},
	}
}

// objectModelNotMatchResponse 结构体不匹配的错误返回体
//
//	@param	name	...string	注册的返回体,实际的返回体
func objectModelNotMatchResponse(name ...string) *Response {
	vv := &godantic.ValidationError{
		Ctx: map[string]any{
			"where error": "server",
			"msg": fmt.Sprintf(
				"response model should be '%s', but '%s' returned", name[0], name[1],
			),
		},
		Msg:  ModelNotMatch,
		Type: string(godantic.ObjectType),
		Loc:  []string{"response", name[0]},
	}
	if len(name) > 0 {
		vv.Ctx["msg"] = fmt.Sprintf(
			"response model should be '%s', but 'string' returned", name[0],
		)
	}

	return &Response{
		StatusCode: http.StatusUnprocessableEntity,
		Content:    &godantic.HTTPValidationError{Detail: []*godantic.ValidationError{vv}},
	}
}
