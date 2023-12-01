package fastapi

import (
	"fmt"
	"github.com/Chendemo12/fastapi/openapi"
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

var emptyLocList = []string{"response"}
var whereServerError = map[string]any{"where error": "server"}
var whereClientError = map[string]any{"where error": "client"}

// Response 路由返回值
type Response struct {
	Content     any          `json:"content"`     // 响应体
	ContentType string       `json:"contentType"` // 响应类型,默认为 application/json
	Type        ResponseType `json:"type"`        // 返回体类型
	StatusCode  int          `json:"status_code"` // 响应状态码
}

// validationErrorResponse 参数校验错误返回值
func validationErrorResponse(ves ...*openapi.ValidationError) *Response {
	return &Response{
		StatusCode: http.StatusUnprocessableEntity,
		Content:    &openapi.HTTPValidationError{Detail: ves},
		Type:       ErrResponseType,
	}
}

func modelCannotBeStringResponse(name ...string) *Response {
	vv := &openapi.ValidationError{
		Ctx:  map[string]any{"where error": "server", "msg": ModelCannotString},
		Msg:  ModelNotMatch,
		Type: string(openapi.StringType),
		Loc:  emptyLocList,
	}

	if len(name) > 0 {
		vv.Ctx["msg"] = fmt.Sprintf(
			"response model should be '%s', but 'string' returned", name[0],
		)
	}

	return &Response{
		StatusCode: http.StatusUnprocessableEntity,
		Content:    &openapi.HTTPValidationError{Detail: []*openapi.ValidationError{vv}},
	}
}

func modelCannotBeNumberResponse(name ...string) *Response {
	vv := &openapi.ValidationError{
		Ctx:  map[string]any{"where error": "server", "msg": ModelCannotNumber},
		Msg:  ModelNotMatch,
		Type: string(openapi.NumberType),
		Loc:  emptyLocList,
	}
	if len(name) > 0 {
		vv.Ctx["msg"] = fmt.Sprintf(
			"response model should be '%s', but 'string' returned", name[0],
		)
	}

	return &Response{
		StatusCode: http.StatusUnprocessableEntity,
		Content:    &openapi.HTTPValidationError{Detail: []*openapi.ValidationError{vv}},
	}
}

func modelCannotBeBoolResponse(name ...string) *Response {
	vv := &openapi.ValidationError{
		Ctx:  map[string]any{"where error": "server", "msg": ModelCannotBool},
		Msg:  ModelNotMatch,
		Type: string(openapi.BoolType),
		Loc:  emptyLocList,
	}
	if len(name) > 0 {
		vv.Ctx["msg"] = fmt.Sprintf(
			"response model should be '%s', but 'string' returned", name[0],
		)
	}
	return &Response{
		StatusCode: http.StatusUnprocessableEntity,
		Content:    &openapi.HTTPValidationError{Detail: []*openapi.ValidationError{vv}},
	}
}

func modelCannotBeIntegerResponse(name ...string) *Response {
	vv := &openapi.ValidationError{
		Ctx:  map[string]any{"where error": "server", "msg": ModelCannotInteger},
		Msg:  ModelNotMatch,
		Type: string(openapi.IntegerType),
		Loc:  emptyLocList,
	}
	if len(name) > 0 {
		vv.Ctx["msg"] = fmt.Sprintf(
			"response model should be '%s', but 'string' returned", name[0],
		)
	}
	return &Response{
		StatusCode: http.StatusUnprocessableEntity,
		Content:    &openapi.HTTPValidationError{Detail: []*openapi.ValidationError{vv}},
	}
}

func modelCannotBeArrayResponse(name ...string) *Response {
	vv := &openapi.ValidationError{
		Ctx:  map[string]any{"where error": "server", "msg": ModelCannotArray},
		Msg:  ModelNotMatch,
		Type: string(openapi.ArrayType),
		Loc:  emptyLocList,
	}
	if len(name) > 0 {
		vv.Ctx["msg"] = fmt.Sprintf(
			"response model should be '%s', but 'string' returned", name[0],
		)
	}

	return &Response{
		StatusCode: http.StatusUnprocessableEntity,
		Content:    &openapi.HTTPValidationError{Detail: []*openapi.ValidationError{vv}},
	}
}

// objectModelNotMatchResponse 结构体不匹配的错误返回体
//
//	@param	name	...string	注册的返回体,实际的返回体
func objectModelNotMatchResponse(name ...string) *Response {
	vv := &openapi.ValidationError{
		Ctx: map[string]any{
			"where error": "server",
			"msg": fmt.Sprintf(
				"response model should be '%s', but '%s' returned", name[0], name[1],
			),
		},
		Msg:  ModelNotMatch,
		Type: string(openapi.ObjectType),
		Loc:  []string{"response", name[0]},
	}
	if len(name) > 0 {
		vv.Ctx["msg"] = fmt.Sprintf(
			"response model should be '%s', but 'string' returned", name[0],
		)
	}

	return &Response{
		StatusCode: http.StatusUnprocessableEntity,
		Content:    &openapi.HTTPValidationError{Detail: []*openapi.ValidationError{vv}},
	}
}
