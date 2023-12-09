package fastapi

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/Chendemo12/fastapi/openapi"
	"github.com/Chendemo12/fastapi/utils"
	"net/http"
	"reflect"
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

type ValidateMethod interface {
	V(obj any) *openapi.ValidationError
}

type ModelBindMethod interface {
	Name() string                                 // 名称
	Validate(data any) []*openapi.ValidationError // 校验方法，对于响应首先校验，然后在 Marshal；对于请求，首先 Unmarshal 然后再校验
	Marshal(obj any) ([]byte, error)              // 序列化方法，通过 ContentType 确定响应体类型
	Unmarshal(stream []byte, obj any) (err error) // 反序列化方法，通过 "http:header:Content-Type" 推断内容类型
	New() any                                     // 创建一个新实例
}

type JsonBindMethod struct {
	validates []ValidateMethod
	rType     reflect.Type
}

func (m *JsonBindMethod) Name() string {
	return "JsonBindMethod"
}

func (m *JsonBindMethod) Validate(data any) []*openapi.ValidationError {
	var ves []*openapi.ValidationError

	for _, validate := range m.validates {
		err := validate.V(data)
		if err != nil {
			ves = append(ves, err)
		}
	}

	return ves
}

func (m *JsonBindMethod) Marshal(obj any) ([]byte, error) {
	return json.Marshal(obj)
}

func (m *JsonBindMethod) Unmarshal(stream []byte, obj any) (err error) {
	err = json.Unmarshal(stream, obj)
	return
}

func (m *JsonBindMethod) New() any {
	obj := reflect.New(m.rType)
	return obj.Interface()
}

type IntegerBindMethod struct {
	unsigned        bool // 无符号类型
	UnsignedMaximum uint64
	UnsignedMinimum uint64
	SignedMaximum   int64
	SignedMinimum   int64
}

func (m *IntegerBindMethod) Name() string {
	return "IntegerBindMethod"
}

func (m *IntegerBindMethod) Validate(data any) []*openapi.ValidationError {
	var ves []*openapi.ValidationError
	var err *openapi.ValidationError

	links := []func(data any) *openapi.ValidationError{
		UnsignedIntegerMaximumV[uint64](m.UnsignedMaximum, false),
		SignedIntegerMaximumV[int64](m.SignedMaximum, false),
	}

	for _, link := range links {
		err = link(data)
		if err != nil {
			ves = append(ves, err)
		}
	}

	return ves
}

func (m *IntegerBindMethod) Marshal(obj any) ([]byte, error) {
	//TODO implement me
	panic("implement me")
}

func (m *IntegerBindMethod) Unmarshal(stream []byte, obj any) (err error) {
	//TODO implement me
	panic("implement me")
}

func (m *IntegerBindMethod) New() any {
	//TODO implement me
	panic("implement me")
}

type UnsignedInteger interface {
	~uint8 | ~uint16 | ~uint32 | ~uint64 | ~uint
}

type SignedInteger interface {
	~int8 | ~int16 | ~int32 | ~int64 | ~int
}

// UnsignedIntegerMaximumV 无符号最大值校验
func UnsignedIntegerMaximumV[T UnsignedInteger](maximum T, eq bool) func(obj any) *openapi.ValidationError {
	return func(obj any) *openapi.ValidationError {
		if eq && obj.(T) > maximum {
			return &openapi.ValidationError{
				Ctx:  map[string]any{"where error": "client"},
				Msg:  fmt.Sprintf("value: %d not <= %d", obj, maximum),
				Type: string(openapi.IntegerType),
				Loc:  []string{"param"},
			}
		}

		if !eq && obj.(T) >= maximum {
			return &openapi.ValidationError{
				Ctx:  map[string]any{"where error": "client"},
				Msg:  fmt.Sprintf("value: %d not < %d", obj, maximum),
				Type: string(openapi.IntegerType),
				Loc:  []string{"param"},
			}
		}

		return nil
	}
}

// SignedIntegerMaximumV 有符号最大值校验
func SignedIntegerMaximumV[T SignedInteger](minimum T, eq bool) func(obj any) *openapi.ValidationError {
	return func(obj any) *openapi.ValidationError {
		if eq && obj.(T) < minimum {
			return &openapi.ValidationError{
				Ctx:  map[string]any{"where error": "client"},
				Msg:  fmt.Sprintf("value: %d not <= %d", obj, minimum),
				Type: string(openapi.IntegerType),
				Loc:  []string{"param"},
			}
		}

		if !eq && obj.(T) <= minimum {
			return &openapi.ValidationError{
				Ctx:  map[string]any{"where error": "client"},
				Msg:  fmt.Sprintf("value: %d not < %d", obj, minimum),
				Type: string(openapi.IntegerType),
				Loc:  []string{"param"},
			}
		}

		return nil
	}
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

// NothingBindMethod 空实现，用于什么也不做
type NothingBindMethod struct{}

func (m *NothingBindMethod) Name() string {
	return "NothingBindMethod"
}

func (m *NothingBindMethod) Validate(data any) []*openapi.ValidationError {
	return nil
}

func (m *NothingBindMethod) Marshal(obj any) ([]byte, error) {
	return []byte{}, nil
}

func (m *NothingBindMethod) Unmarshal(stream []byte, obj any) (err error) {
	return
}

func (m *NothingBindMethod) New() any {
	return nil
}

// 一下不能直接返回 Response
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
