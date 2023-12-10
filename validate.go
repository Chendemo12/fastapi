package fastapi

import (
	"context"
	"errors"
	"fmt"
	"github.com/Chendemo12/fastapi-tool/helper"
	"github.com/Chendemo12/fastapi/openapi"
	"github.com/go-playground/validator/v10"
	"reflect"
	"strconv"
	"strings"
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

var defaultValidator = validator.New()

var emptyLocList = []string{"response"}
var whereServerError = map[string]any{"where error": "server"}
var whereClientError = map[string]any{"where error": "client"}

type ValidateMethod interface {
	V(obj any) *openapi.ValidationError
}

type ModelBindMethod interface {
	Name() string // 名称
	// Validate TODO: 考虑将data转换为 []byte类型
	// 校验方法，对于响应首先校验，然后在 Marshal；对于请求，首先 Unmarshal 然后再校验
	// 对于不需要ctx参数的校验方法可默认设置为nil
	// data 为需要验证的数据模型，如果验证通过，则第一个返回值为做了类型转换的data
	Validate(ctx context.Context, data any) (any, []*openapi.ValidationError)
	Marshal(obj any) ([]byte, error)                                   // 序列化方法，通过 ContentType 确定响应体类型
	Unmarshal(stream []byte, obj any) (ves []*openapi.ValidationError) // 反序列化方法，通过 "http:header:Content-Type" 推断内容类型
	New() any                                                          // 创建一个新实例
}

// UnsignedInteger 无符号数字约束
type UnsignedInteger interface {
	~uint8 | ~uint16 | ~uint32 | ~uint64 | ~uint
}

// SignedInteger 有符号数字约束
type SignedInteger interface {
	~int8 | ~int16 | ~int32 | ~int64 | ~int
}

// NothingBindMethod 空实现，用于什么也不做
type NothingBindMethod struct{}

func (m *NothingBindMethod) Name() string { return "NothingBindMethod" }

func (m *NothingBindMethod) Validate(ctx context.Context, data any) (any, []*openapi.ValidationError) {
	return data, nil
}

func (m *NothingBindMethod) Marshal(obj any) ([]byte, error) {
	return []byte{}, nil
}

func (m *NothingBindMethod) Unmarshal(stream []byte, obj any) (ves []*openapi.ValidationError) {
	return
}
func (m *NothingBindMethod) New() any {
	return nil
}

// IntBindMethod 有符号数字验证
type IntBindMethod struct {
	Title   string       `json:"title,omitempty"`
	Kind    reflect.Kind `json:"kind,omitempty"`
	Maximum int64        `json:"maximum,omitempty"`
	Minimum int64        `json:"minimum,omitempty"`
}

func (m *IntBindMethod) Name() string { return "IntBindMethod" }

func (m *IntBindMethod) Validate(ctx context.Context, data any) (any, []*openapi.ValidationError) {
	var ves []*openapi.ValidationError
	// 首先 data 必须是字符串类型
	sv, ok := data.(string)
	if !ok {
		ves = append(ves, &openapi.ValidationError{
			Loc:  []string{"query", m.Title},
			Msg:  fmt.Sprintf("value: '%s' is not an integer", sv),
			Type: string(openapi.IntegerType),
			Ctx:  whereClientError,
		})

		return nil, ves
	}

	atoi, err := strconv.ParseInt(sv, 10, 0)
	if err != nil { // 无法转换为数字
		ves = append(ves, &openapi.ValidationError{
			Loc:  []string{"query", m.Title},
			Msg:  fmt.Sprintf("value: '%s' is not a signed integer", sv),
			Type: string(openapi.IntegerType),
			Ctx:  whereClientError,
		})
		return nil, ves
	}

	if atoi > m.Maximum || atoi < m.Minimum {
		ves = append(ves, &openapi.ValidationError{
			Ctx:  map[string]any{"where error": "client"},
			Msg:  fmt.Sprintf("value: %s not <= %d and >= %d", sv, m.Maximum, m.Minimum),
			Type: string(openapi.IntegerType),
			Loc:  []string{"param"},
		})
		return nil, ves
	}

	return atoi, ves
}

func (m *IntBindMethod) Marshal(obj any) ([]byte, error) {
	// 目前无实际作用，不实现
	return []byte{}, nil
}

func (m *IntBindMethod) Unmarshal(stream []byte, obj any) (ves []*openapi.ValidationError) {
	// 可以通过 binary.BigEndian.Int64 实现，目前不实现
	return
}

// New 返回int的零值
func (m *IntBindMethod) New() any {
	return 0
}

// UintBindMethod 无符号数字验证
type UintBindMethod struct {
	Title   string       `json:"title,omitempty"`
	Kind    reflect.Kind `json:"kind,omitempty"`
	Maximum uint64       `json:"maximum,omitempty"`
	Minimum uint64       `json:"minimum,omitempty"`
}

func (m *UintBindMethod) Name() string { return "UintBindMethod" }

func (m *UintBindMethod) Validate(ctx context.Context, data any) (any, []*openapi.ValidationError) {
	var ves []*openapi.ValidationError
	// 首先 data 必须是字符串类型
	sv, ok := data.(string)
	if !ok {
		ves = append(ves, &openapi.ValidationError{
			Loc:  []string{"query", m.Title},
			Msg:  fmt.Sprintf("value: '%s' is not an integer", sv),
			Type: string(openapi.IntegerType),
			Ctx:  whereClientError,
		})

		return nil, ves
	}

	atoi, err := strconv.ParseUint(sv, 10, 0)
	if err != nil { // 无法转换为数字
		ves = append(ves, &openapi.ValidationError{
			Loc:  []string{"query", m.Title},
			Msg:  fmt.Sprintf("value: '%s' is not an unsigned integer", sv),
			Type: string(openapi.IntegerType),
			Ctx:  whereClientError,
		})
		return nil, ves
	}

	if atoi > m.Maximum || atoi < m.Minimum {
		ves = append(ves, &openapi.ValidationError{
			Ctx:  map[string]any{"where error": "client"},
			Msg:  fmt.Sprintf("value: %s not <= %d and >= %d", sv, m.Maximum, m.Minimum),
			Type: string(openapi.IntegerType),
			Loc:  []string{"param"},
		})
		return nil, ves
	}

	return atoi, ves
}

func (m *UintBindMethod) Marshal(obj any) ([]byte, error) {
	// 目前无实际作用，不实现
	return []byte{}, nil
}

func (m *UintBindMethod) Unmarshal(stream []byte, obj any) (ves []*openapi.ValidationError) {
	// 可以通过 binary.BigEndian.Uint64 实现，目前不实现
	return
}

// New 返回uint的零值
func (m *UintBindMethod) New() any {
	return uint(0)
}

type FloatBindMethod struct {
	Title string       `json:"title,omitempty"`
	Kind  reflect.Kind `json:"kind,omitempty"`
}

func (m *FloatBindMethod) Name() string { return "FloatBindMethod" }

// Validate 验证字符串data是否是一个float类型，data 应为string类型
func (m *FloatBindMethod) Validate(ctx context.Context, data any) (any, []*openapi.ValidationError) {
	var ves []*openapi.ValidationError
	sv := data.(string)

	atof, err := strconv.ParseFloat(sv, 64)
	if err != nil {
		ves = append(ves, &openapi.ValidationError{
			Loc:  []string{"query", m.Title},
			Msg:  fmt.Sprintf("value: '%s' is not a number", sv),
			Type: string(openapi.NumberType),
			Ctx:  whereClientError,
		})

		return nil, ves
	}
	return atof, nil
}

func (m *FloatBindMethod) Marshal(obj any) ([]byte, error) {
	return []byte{}, nil
}

func (m *FloatBindMethod) Unmarshal(stream []byte, obj any) (ves []*openapi.ValidationError) {
	return
}

// New 返回float64的零值
func (m *FloatBindMethod) New() any {
	return float64(0)
}

type BoolBindMethod struct {
	Title string `json:"title,omitempty"`
}

func (m *BoolBindMethod) Name() string { return "BoolBindMethod" }

// Validate data 为字符串类型
func (m *BoolBindMethod) Validate(ctx context.Context, data any) (any, []*openapi.ValidationError) {
	var ves []*openapi.ValidationError
	sv := data.(string)

	atob, err := strconv.ParseBool(sv)
	if err != nil {
		ves = append(ves, &openapi.ValidationError{
			Loc:  []string{"query", m.Title},
			Msg:  fmt.Sprintf("value: '%s' is not a bool", sv),
			Type: string(openapi.BoolType),
			Ctx:  whereClientError,
		})
		return nil, ves
	}

	return atob, nil
}

func (m *BoolBindMethod) Marshal(obj any) ([]byte, error) {
	return []byte{}, nil
}

func (m *BoolBindMethod) Unmarshal(stream []byte, obj any) (ves []*openapi.ValidationError) {
	return ves
}

// New 返回 bool类型而零值false
func (m *BoolBindMethod) New() any {
	return false
}

type JsonBindMethod[T any] struct {
	Title string         `json:"title,omitempty"`
	Where map[string]any `json:"-"` // whereClientError / whereServerError
}

func (m *JsonBindMethod[T]) Name() string { return "JsonBindMethod" }

func (m *JsonBindMethod[T]) Validate(ctx context.Context, data T) (T, []*openapi.ValidationError) {
	var vErr validator.ValidationErrors // validator的校验错误信息
	err := defaultValidator.StructCtx(ctx, data)

	if ok := errors.As(err, &vErr); ok { // 模型验证错误
		ves := make([]*openapi.ValidationError, 0)
		for _, verr := range vErr {
			ves = append(ves, &openapi.ValidationError{
				Ctx:  m.Where,
				Msg:  verr.Error(),
				Type: verr.Type().String(),
				Loc:  []string{"body", verr.Field()},
			})
		}
		var n T
		return n, ves
	}
	return data, nil
}

func (m *JsonBindMethod[T]) Marshal(obj T) ([]byte, error) {
	return helper.JsonMarshal(obj)
}

func (m *JsonBindMethod[T]) Unmarshal(stream []byte, obj T) (ves []*openapi.ValidationError) {
	err := helper.JsonUnmarshal(stream, obj)
	if err != nil {
		ves = append(ves, jsoniterUnmarshalErrorToValidationError(err))
	}

	return
}

func (m *JsonBindMethod[T]) New() any {
	var value = new(T)
	return value
}

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
		err = helper.JsonUnmarshal([]byte(msgs[jsonErrorFormIndex]), &ve.Ctx)
		if err == nil {
			ve.Msg = msgs[jsonErrorFieldMsgIndex][len(ve.Loc[1])+2:]
			if s := strings.Split(ve.Msg, ":"); len(s) > 0 {
				ve.Type = s[0]
			}
		}
	}

	return ve
}

// =================================== 👇 以下用于泛型的返回值校验 ===================================

// objectModelNotMatchResponse 结构体不匹配的错误返回体
//
//	@param	name	...string	注册的返回体,实际的返回体名称
func objectModelNotMatchResponse(name ...string) *openapi.ValidationError {
	ve := &openapi.ValidationError{
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
		ve.Ctx["msg"] = fmt.Sprintf(
			"response model should be '%s', but 'string' returned", name[0],
		)
	}

	return ve
}

func modelCannotBeStringResponse(name ...string) *openapi.ValidationError {
	ve := &openapi.ValidationError{
		Ctx:  map[string]any{"where error": "server", "msg": ModelCannotString},
		Msg:  ModelNotMatch,
		Type: string(openapi.StringType),
		Loc:  emptyLocList,
	}

	if len(name) > 0 {
		ve.Ctx["msg"] = fmt.Sprintf(
			"response model should be '%s', but 'string' returned", name[0],
		)
	}

	return ve
}

func modelCannotBeNumberResponse(name ...string) *openapi.ValidationError {
	ve := &openapi.ValidationError{
		Ctx:  map[string]any{"where error": "server", "msg": ModelCannotNumber},
		Msg:  ModelNotMatch,
		Type: string(openapi.NumberType),
		Loc:  emptyLocList,
	}
	if len(name) > 0 {
		ve.Ctx["msg"] = fmt.Sprintf(
			"response model should be '%s', but 'string' returned", name[0],
		)
	}

	return ve
}

func modelCannotBeBoolResponse(name ...string) *openapi.ValidationError {
	ve := &openapi.ValidationError{
		Ctx:  map[string]any{"where error": "server", "msg": ModelCannotBool},
		Msg:  ModelNotMatch,
		Type: string(openapi.BoolType),
		Loc:  emptyLocList,
	}
	if len(name) > 0 {
		ve.Ctx["msg"] = fmt.Sprintf(
			"response model should be '%s', but 'string' returned", name[0],
		)
	}
	return ve
}

func modelCannotBeIntegerResponse(name ...string) *openapi.ValidationError {
	ve := &openapi.ValidationError{
		Ctx:  map[string]any{"where error": "server", "msg": ModelCannotInteger},
		Msg:  ModelNotMatch,
		Type: string(openapi.IntegerType),
		Loc:  emptyLocList,
	}
	if len(name) > 0 {
		ve.Ctx["msg"] = fmt.Sprintf(
			"response model should be '%s', but 'string' returned", name[0],
		)
	}
	return ve
}

func modelCannotBeArrayResponse(name ...string) *openapi.ValidationError {
	ve := &openapi.ValidationError{
		Ctx:  map[string]any{"where error": "server", "msg": ModelCannotArray},
		Msg:  ModelNotMatch,
		Type: string(openapi.ArrayType),
		Loc:  emptyLocList,
	}
	if len(name) > 0 {
		ve.Ctx["msg"] = fmt.Sprintf(
			"response model should be '%s', but 'string' returned", name[0],
		)
	}

	return ve
}
