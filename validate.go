package fastapi

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/Chendemo12/fastapi/openapi"
	"github.com/Chendemo12/fastapi/utils"
	"github.com/go-playground/validator/v10"
	"golang.org/x/exp/constraints"
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

const ( // json序列化错误, 关键信息的序号
	jsoniterUnmarshalErrorSeparator = "|" // 序列化错误信息分割符, 定义于 validator/validator_instance.orSeparator
	jsonErrorFieldMsgIndex          = 0   // 错误原因
	jsonErrorFieldNameFormIndex     = 1   // 序列化错误的字段和值
	jsonErrorFormIndex              = 3   // 接收到的数据
)

var defaultValidator *validator.Validate
var structQueryBind *StructQueryBind

var emptyLocList = []string{"response"}
var modelDescLabel = "param description"
var whereErrorLabel = "where error"
var validateErrorTagLabel = "tag"
var whereServerError = map[string]any{whereErrorLabel: "server"}
var whereClientError = map[string]any{whereErrorLabel: "client"}

// ModelBinder 参数模型校验
type ModelBinder interface {
	Name() string                           // 名称，用来区分不同实现
	ModelName() string                      // 需要校验的模型名称，用于在校验未通过时生成错误信息
	RouteParamType() openapi.RouteParamType // 参数类型
	// Validate 校验方法
	// 对于响应体首先校验，然后再 Marshal；对于请求，首先 Unmarshal 然后再校验
	// 对于不需要 Context 参数的校验方法可默认设置为nil
	// requestParam 为需要验证的数据模型，如果验证通过，则第一个返回值为做了类型转换的 requestParam
	Validate(c *Context, requestParam any) (any, []*openapi.ValidationError)
}

// NothingModelBinder 空实现，用于什么也不做
type NothingModelBinder struct {
	modelName string
	paramType openapi.RouteParamType
}

func NewNothingModelBinder(model openapi.SchemaIface, paramType openapi.RouteParamType) *NothingModelBinder {
	return &NothingModelBinder{
		modelName: model.JsonName(),
		paramType: paramType,
	}
}

func (m *NothingModelBinder) Name() string { return "NothingModelBinder" }

// ModelName 该字段可以为空字符串
func (m *NothingModelBinder) ModelName() string {
	return m.modelName
}

func (m *NothingModelBinder) RouteParamType() openapi.RouteParamType {
	return m.paramType
}

func (m *NothingModelBinder) Validate(c *Context, requestParam any) (any, []*openapi.ValidationError) {
	return requestParam, nil
}

// IntModelBinder 有符号数字验证
type IntModelBinder[T constraints.Signed | ~string] struct {
	modelName string
	paramType openapi.RouteParamType
	Maximum   int64 `json:"maximum,omitempty"`
	Minimum   int64 `json:"minimum,omitempty"`
}

func (m *IntModelBinder[T]) Name() string { return "IntModelBinder" }

func (m *IntModelBinder[T]) ModelName() string {
	return m.modelName
}

func (m *IntModelBinder[T]) RouteParamType() openapi.RouteParamType {
	return m.paramType
}

func (m *IntModelBinder[T]) valid(v int64) []*openapi.ValidationError {
	if v > m.Maximum || v < m.Minimum {
		return []*openapi.ValidationError{{
			Loc:  []string{string(m.paramType), m.modelName},
			Ctx:  map[string]any{"where error": "client"},
			Msg:  fmt.Sprintf("value: %d not <= %d and >= %d", v, m.Maximum, m.Minimum),
			Type: string(openapi.IntegerType),
		}}
	}
	return nil
}

// Validate 验证并转换数据类型
// 如果是string类型，则按照定义进行转换，反之则直接返回
func (m *IntModelBinder[T]) Validate(c *Context, requestParam any) (any, []*openapi.ValidationError) {
	switch v := requestParam.(type) {
	case string:
		result, err := strconv.ParseInt(v, 10, 64)
		if err != nil {
			return requestParam, []*openapi.ValidationError{{
				Loc:  []string{string(m.paramType), m.modelName},
				Msg:  fmt.Sprintf("value: '%s' is not an integer", requestParam),
				Type: string(openapi.IntegerType),
				Ctx:  whereClientError,
			}}
		}
		return result, m.valid(result)
	case int:
		return v, m.valid(int64(v))
	case int8:
		return v, m.valid(int64(v))
	case int16:
		return v, m.valid(int64(v))
	case int32:
		return v, m.valid(int64(v))
	case int64:
		return v, m.valid(v)
	default:
		return requestParam, []*openapi.ValidationError{{
			Loc:  []string{string(m.paramType), m.modelName},
			Msg:  fmt.Sprintf("value: '%s' is not an integer", requestParam),
			Type: string(openapi.IntegerType),
			Ctx:  whereClientError,
		}}
	}
}

// UintModelBinder 无符号数字验证
type UintModelBinder[T constraints.Unsigned | ~string] struct {
	modelName string
	paramType openapi.RouteParamType
	Maximum   uint64 `json:"maximum,omitempty"`
	Minimum   uint64 `json:"minimum,omitempty"`
}

func (m *UintModelBinder[T]) Name() string { return "UintModelBinder" }

func (m *UintModelBinder[T]) ModelName() string {
	return m.modelName
}

func (m *UintModelBinder[T]) RouteParamType() openapi.RouteParamType {
	return m.paramType
}

func (m *UintModelBinder[T]) valid(v uint64) []*openapi.ValidationError {
	if v > m.Maximum || v < m.Minimum {
		return []*openapi.ValidationError{{
			Ctx:  map[string]any{"where error": "client"},
			Msg:  fmt.Sprintf("value: %d not <= %d and >= %d", v, m.Maximum, m.Minimum),
			Type: string(openapi.IntegerType),
			Loc:  []string{string(m.paramType), m.modelName},
		}}
	}
	return nil
}

// Validate 验证并转换数据类型
// 如果是string类型，则按照定义进行转换，反之则直接返回
func (m *UintModelBinder[T]) Validate(c *Context, requestParam any) (any, []*openapi.ValidationError) {
	switch v := requestParam.(type) {
	case string:
		result, err := strconv.ParseUint(v, 10, 64)
		if err != nil {
			return requestParam, []*openapi.ValidationError{{
				Loc:  []string{string(m.paramType), m.modelName},
				Msg:  fmt.Sprintf("value: '%s' is not an integer", requestParam),
				Type: string(openapi.IntegerType),
				Ctx:  whereClientError,
			}}
		}
		return result, m.valid(result)
	case uint:
		return v, m.valid(uint64(v))
	case uint8:
		return v, m.valid(uint64(v))
	case uint16:
		return v, m.valid(uint64(v))
	case uint32:
		return v, m.valid(uint64(v))
	case uint64:
		return v, m.valid(v)
	default:
		return requestParam, []*openapi.ValidationError{{
			Loc:  []string{string(m.paramType), m.modelName},
			Msg:  fmt.Sprintf("value: '%s' is not an integer", requestParam),
			Type: string(openapi.IntegerType),
			Ctx:  whereClientError,
		}}
	}
}

type FloatModelBinder[T constraints.Float | ~string] struct {
	modelName string
	paramType openapi.RouteParamType
}

func (m *FloatModelBinder[T]) Name() string { return "FloatModelBinder" }

func (m *FloatModelBinder[T]) ModelName() string {
	return m.modelName
}

func (m *FloatModelBinder[T]) RouteParamType() openapi.RouteParamType {
	return m.paramType
}

// Validate 验证并转换数据类型
// 如果是string类型，则按照定义进行转换，反之则直接返回
func (m *FloatModelBinder[T]) Validate(c *Context, requestParam any) (any, []*openapi.ValidationError) {
	var ves []*openapi.ValidationError
	var result float64
	var err error

	switch v := requestParam.(type) {
	case string:
		result, err = strconv.ParseFloat(v, 64)
	case float32:
		result = float64(v)
	case float64:
		result = v
	default:
		err = fmt.Errorf("cannot convert %s to float", v)
	}

	if err != nil {
		ves = append(ves, &openapi.ValidationError{
			Loc:  []string{string(m.paramType), m.modelName},
			Msg:  fmt.Sprintf("value: '%s' is not an number", requestParam),
			Type: string(openapi.NumberType),
			Ctx:  whereClientError,
		})
		//} else {
		// 暂不验证范围
		//if result > m.Maximum || result < m.Minimum {
		//	ves = append(ves, &openapi.ValidationError{
		//		Ctx:  map[string]any{"where error": "client"},
		//		Msg:  fmt.Sprintf("value: %d not <= %d and >= %d", result, m.Maximum, m.Minimum),
		//		Type: string(openapi.NumberType),
		//		Loc:  []string{string(m.paramType), m.modelName},
		//	})
		//}
	}

	return result, ves
}

type BoolModelBinder struct {
	modelName string
	paramType openapi.RouteParamType
}

func (m *BoolModelBinder) Name() string { return "BoolModelBinder" }

func (m *BoolModelBinder) ModelName() string {
	return m.modelName
}

func (m *BoolModelBinder) RouteParamType() openapi.RouteParamType {
	return m.paramType
}

// Validate data 为字符串类型
func (m *BoolModelBinder) Validate(c *Context, requestParam any) (any, []*openapi.ValidationError) {
	sv := requestParam.(string)

	atob, err := strconv.ParseBool(sv)
	if err != nil {
		var ves []*openapi.ValidationError
		ves = append(ves, &openapi.ValidationError{
			Loc:  []string{"query", m.modelName},
			Msg:  fmt.Sprintf("value: '%s' is not a bool", sv),
			Type: string(openapi.BoolType),
			Ctx:  whereClientError,
		})
		return false, ves
	}

	return atob, nil
}

// JsonModelBinder json数据类型验证器,适用于泛型路由
type JsonModelBinder[T any] struct {
	modelName string
	paramType openapi.RouteParamType
}

func (m *JsonModelBinder[T]) Name() string { return "JsonModelBinder" }

func (m *JsonModelBinder[T]) ModelName() string {
	return m.modelName
}

func (m *JsonModelBinder[T]) RouteParamType() openapi.RouteParamType {
	return m.paramType
}

func (m *JsonModelBinder[T]) where(key, value string) map[string]any {
	var where = make(map[string]any)
	if m.paramType == openapi.RouteParamResponse {
		where[whereErrorLabel] = whereServerError[whereErrorLabel]
	} else {
		where[whereErrorLabel] = whereClientError[whereErrorLabel]
	}
	if key != "" {
		where[key] = value
	}

	return where
}

func (m *JsonModelBinder[T]) Validate(c *Context, requestParam any) (any, []*openapi.ValidationError) {
	var vErr validator.ValidationErrors // validator的校验错误信息
	err := defaultValidator.Struct(requestParam)

	if ok := errors.As(err, &vErr); ok { // 模型验证错误
		ves := make([]*openapi.ValidationError, 0)
		for _, verr := range vErr {
			ves = append(ves, &openapi.ValidationError{
				Ctx:  m.where(validateErrorTagLabel, verr.Tag()),
				Msg:  verr.Error(),
				Type: verr.Type().String(),
				Loc:  []string{"body", m.modelName, verr.Field()},
			})
		}
		var n T
		return n, ves
	}
	return requestParam, nil
}

// TimeModelBinder 时间校验方法
type TimeModelBinder struct {
	modelName string
	paramType openapi.RouteParamType
}

func (m *TimeModelBinder) Name() string { return "TimeModelBinder" }

func (m *TimeModelBinder) ModelName() string {
	return m.modelName
}

func (m *TimeModelBinder) RouteParamType() openapi.RouteParamType {
	return m.paramType
}

var timeLayouts = []string{time.TimeOnly, time.Kitchen}
var dateLayouts = []string{time.DateOnly}
var dateTimeLayouts = []string{time.DateTime, time.RFC3339, time.DateOnly, time.TimeOnly, time.Kitchen, time.RFC3339Nano,
	time.RFC822, time.ANSIC, time.UnixDate, time.RubyDate, time.RFC822Z, time.RFC850,
	time.RFC1123, time.RFC1123Z, time.Stamp, time.StampMilli, time.StampMicro, time.StampNano,
}

// Validate 验证一个字符串是否是一个有效的时间字符串
// @return time.Time
func (m *TimeModelBinder) Validate(c *Context, requestParam any) (any, []*openapi.ValidationError) {
	sv := requestParam.(string) // 肯定是string类型

	var err error
	var t time.Time
	for _, layout := range timeLayouts {
		t, err = time.Parse(layout, sv)
		if err == nil {
			return t, nil
		}
	}

	var ves []*openapi.ValidationError
	ves = append(ves, &openapi.ValidationError{
		Loc:  []string{"query", m.modelName},
		Msg:  fmt.Sprintf("value: '%s' is not a time, err:%v", sv, err),
		Type: string(openapi.StringType),
		Ctx:  whereClientError,
	})
	return nil, ves
}

// DateModelBinder 日期校验
type DateModelBinder struct {
	modelName string
	paramType openapi.RouteParamType
}

func (m *DateModelBinder) Name() string { return "DateModelBinder" }

func (m *DateModelBinder) ModelName() string {
	return m.modelName
}

func (m *DateModelBinder) RouteParamType() openapi.RouteParamType {
	return m.paramType
}

func (m *DateModelBinder) Validate(c *Context, requestParam any) (any, []*openapi.ValidationError) {
	sv := requestParam.(string) // 肯定是string类型

	var err error
	var t time.Time
	for _, layout := range dateLayouts {
		t, err = time.Parse(layout, sv)
		if err == nil {
			return t, nil
		}
	}

	var ves []*openapi.ValidationError
	ves = append(ves, &openapi.ValidationError{
		Loc:  []string{"query", m.modelName},
		Msg:  fmt.Sprintf("value: '%s' is not a date, err:%v", sv, err),
		Type: string(openapi.StringType),
		Ctx:  whereClientError,
	})
	return nil, ves
}

// DateTimeModelBinder 日期时间校验
type DateTimeModelBinder struct {
	modelName string
	paramType openapi.RouteParamType
}

func (m *DateTimeModelBinder) Name() string { return "DateTimeModelBinder" }

func (m *DateTimeModelBinder) ModelName() string {
	return m.modelName
}

func (m *DateTimeModelBinder) RouteParamType() openapi.RouteParamType {
	return m.paramType
}

func (m *DateTimeModelBinder) Validate(c *Context, requestParam any) (any, []*openapi.ValidationError) {
	sv := requestParam.(string) // 肯定是string类型

	var err error
	var t time.Time
	for _, layout := range dateTimeLayouts {
		t, err = time.Parse(layout, sv)
		if err == nil {
			return t, nil
		}
	}

	var ves []*openapi.ValidationError

	if timeErr, ok := errors.AsType[*time.ParseError](err); ok {
		ves = append(ves, &openapi.ValidationError{
			Loc:  []string{"query", m.modelName},
			Msg:  fmt.Sprintf("value: '%s' is not a datetime, err:%s", sv, err.Error()),
			Type: string(openapi.StringType),
			Ctx: map[string]any{
				whereErrorLabel: whereClientError[whereErrorLabel],
				"layout":        timeErr.Layout,
			},
		})
	} else {
		ves = append(ves, &openapi.ValidationError{
			Loc:  []string{"query", m.modelName},
			Msg:  fmt.Sprintf("value: '%s' is not a datetime, err:%s", sv, err.Error()),
			Type: string(openapi.StringType),
			Ctx:  whereClientError,
		})
	}

	return nil, ves
}

type RequestModelBinder struct {
	modelName string
	paramType openapi.RouteParamType
}

func (m *RequestModelBinder) Name() string {
	return "RequestModelBinder"
}

func (m *RequestModelBinder) ModelName() string {
	return m.modelName
}

func (m *RequestModelBinder) RouteParamType() openapi.RouteParamType {
	return m.paramType
}

func (m *RequestModelBinder) Validate(c *Context, requestParam any) (any, []*openapi.ValidationError) {
	// 存在请求体,首先进行反序列化,之后校验参数是否合法,校验通过后绑定到 Context
	var ves []*openapi.ValidationError

	validated, err := c.mux.ShouldBind(requestParam)
	if err != nil {
		// 转换错误
		if validated {
			ves = ParseValidatorError(err, openapi.RouteParamRequest, m.modelName)
		} else {
			ve := ParseJsoniterError(err, openapi.RouteParamRequest, m.modelName)
			if ve != nil {
				ves = append(ves, ve)
			}
		}
		if len(ves) > 0 {
			ves[0].Ctx[modelDescLabel] = m.modelName
		}
		return requestParam, ves
	} else {
		// 没有错误，需要判断是否校验过了，如果没有，则进行校验
		if !validated {
			err := defaultValidator.Struct(requestParam)
			if err != nil {
				ves := ParseValidatorError(err, openapi.RouteParamRequest, m.modelName)
				if len(ves) > 0 {
					ves[0].Ctx[modelDescLabel] = m.modelName
				}
			}
		}
		return requestParam, ves
	}
}

// FileModelBinder 文件请求体验证
type FileModelBinder struct {
	modelName string
}

func (m *FileModelBinder) Name() string {
	return "FileModelBinder"
}

func (m *FileModelBinder) ModelName() string {
	return m.modelName
}

func (m *FileModelBinder) RouteParamType() openapi.RouteParamType {
	return openapi.RouteParamRequest
}

func (m *FileModelBinder) Validate(c *Context, requestParam any) (any, []*openapi.ValidationError) {
	// 存在上传文件定义，则从 multiform-data 中获取上传参数
	forms, err := c.mux.MultipartForm()
	if err != nil {
		return requestParam, []*openapi.ValidationError{{
			Loc:  []string{"requestBody", "multiform-data"},
			Msg:  "read multiform failed",
			Type: string(openapi.ObjectType),
			Ctx:  whereClientError,
		}}
	}

	files := forms.File[openapi.MultipartFormFileName]
	if len(files) == 0 {
		return requestParam, []*openapi.ValidationError{{
			Loc:  []string{"requestBody", "multiform-data", openapi.MultipartFormFileName},
			Msg:  fmt.Sprintf("'%s' value not found", openapi.MultipartFormFileName),
			Type: string(openapi.ObjectType),
			Ctx:  whereClientError,
		}}
	}
	c.file = &File{files}

	return requestParam, nil
}

// FileWithParamModelBinder 文件+json 混合请求体验证
type FileWithParamModelBinder struct {
	FileModelBinder
	paramType openapi.RouteParamType
}

func (m *FileWithParamModelBinder) Name() string {
	return "FileWithParamModelBinder"
}

func (m *FileWithParamModelBinder) Validate(c *Context, requestParam any) (any, []*openapi.ValidationError) {
	// 首先验证文件是否通过
	_, ves := m.FileModelBinder.Validate(c, requestParam)
	if len(ves) > 0 {
		return nil, ves
	}

	forms, _ := c.MuxContext().MultipartForm()
	// 文件验证通过，校验json参数
	params := forms.Value[openapi.MultipartFormParamName]
	if len(params) == 0 {
		return requestParam, []*openapi.ValidationError{{
			Loc:  []string{"requestBody", "multiform-data", openapi.MultipartFormParamName},
			Msg:  fmt.Sprintf("'%s' value not found", openapi.MultipartFormParamName),
			Type: string(openapi.ObjectType),
			Ctx:  whereClientError,
		}}
	}

	// json 参数绑定
	err := c.Unmarshal([]byte(params[0]), requestParam)
	if err != nil {
		ve := ParseJsoniterError(err, openapi.RouteParamRequest, m.modelName)
		ve.Ctx[modelDescLabel] = m.modelName
		return requestParam, []*openapi.ValidationError{ve}
	}

	// json 校验
	err = defaultValidator.Struct(requestParam)
	if err != nil {
		ves := ParseValidatorError(err, openapi.RouteParamRequest, m.modelName)
		if len(ves) > 0 {
			ves[0].Ctx[modelDescLabel] = m.modelName
		}
		return requestParam, ves
	}

	return requestParam, nil
}

// StructQueryBind 结构体查询参数验证器
type StructQueryBind struct{}

// Unmarshal todo 性能损耗过大了
func (m *StructQueryBind) Unmarshal(params map[string]any, obj any) *openapi.ValidationError {
	s, err := utils.JsonMarshal(params)
	if err != nil {
		return ParseJsoniterError(err, openapi.RouteParamQuery, "")
	}
	err = utils.JsonUnmarshal(s, obj)
	if err != nil {
		return ParseJsoniterError(err, openapi.RouteParamQuery, "")
	}
	return nil
}

func (m *StructQueryBind) Validate(obj any) []*openapi.ValidationError {
	err := defaultValidator.StructCtx(context.Background(), obj)
	if err != nil {
		ves := ParseValidatorError(err, openapi.RouteParamQuery, "")
		return ves
	}
	return nil
}

func (m *StructQueryBind) Bind(params map[string]any, obj any) []*openapi.ValidationError {
	ve := m.Unmarshal(params, obj)
	if ve != nil {
		return []*openapi.ValidationError{ve}
	}
	return m.Validate(obj)
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

// =================================== 👇 methods ===================================

func newValidateErrorCtx(where map[string]any, key, value string) map[string]any {
	m := map[string]any{}
	m[whereErrorLabel] = where[whereErrorLabel]
	m[key] = value

	return m
}

// ParseJsoniterError 将jsoniter 的反序列化错误转换成 接口错误类型
func ParseJsoniterError(err error, loc openapi.RouteParamType, objName string) *openapi.ValidationError {
	if err == nil {
		return nil
	}
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
	var where map[string]any
	if loc == openapi.RouteParamResponse {
		where = whereServerError
	} else {
		where = whereClientError
	}
	ve := &openapi.ValidationError{Loc: []string{string(loc)}, Ctx: where}
	if objName != "" {
		ve.Loc = append(ve.Loc, objName)
	}

	for i := 0; i < len(msg); i++ {
		if msg[i:i+1] == ":" {
			ve.Loc = append(ve.Loc, msg[:i])
			break
		}
	}
	if msgs := strings.Split(msg, jsoniterUnmarshalErrorSeparator); len(msgs) >= jsonErrorFieldMsgIndex+1 {
		err = utils.JsonUnmarshal([]byte(msgs[jsonErrorFormIndex]), &ve.Ctx)
		if err == nil {
			if len(ve.Loc) >= 2 {
				ve.Msg = msgs[jsonErrorFieldMsgIndex][len(ve.Loc[1])+2:]
			}
			if s := strings.Split(ve.Msg, ":"); len(s) > 0 {
				ve.Type = s[0]
			}
		}
	}

	return ve
}

// ParseValidatorError 解析Validator的错误消息
// 如果不存在错误,则返回nil; 如果 err 是 validator.ValidationErrors 的错误, 则解析并返回详细的错误原因,反之则返回模糊的错误原因
func ParseValidatorError(err error, loc openapi.RouteParamType, objName string) []*openapi.ValidationError {
	if err == nil {
		return nil
	}

	var vErr validator.ValidationErrors // validator的校验错误信息
	var ves []*openapi.ValidationError
	var where map[string]any

	if loc == openapi.RouteParamResponse {
		where = whereServerError
	} else {
		where = whereClientError
	}

	if ok := errors.As(err, &vErr); ok { // Validator的模型验证错误
		for _, verr := range vErr {
			ve := &openapi.ValidationError{
				Ctx:  newValidateErrorCtx(where, validateErrorTagLabel, verr.Tag()),
				Msg:  verr.Error(),
				Type: verr.Type().String(),
			}
			if objName != "" {
				ve.Loc = []string{string(loc), objName, verr.Field()}
			} else {
				ve.Loc = []string{string(loc), verr.Field()}
			}
			ves = append(ves, ve)
		}
	} else {
		ves = append(ves, &openapi.ValidationError{
			Ctx:  where,
			Msg:  err.Error(),
			Type: string(openapi.ObjectType),
			Loc:  []string{string(loc)},
		})
	}

	return ves
}

func LazyInit() {
	// 初始化默认结构体验证器
	defaultValidator = validator.New()
	defaultValidator.SetTagName(openapi.ValidateTagName)

	structQueryBind = &StructQueryBind{}
}
