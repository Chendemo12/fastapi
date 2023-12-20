package fastapi

import (
	"context"
	"errors"
	"fmt"
	"github.com/Chendemo12/fastapi-tool/helper"
	"github.com/Chendemo12/fastapi/openapi"
	"github.com/go-playground/validator/v10"
	jsoniter "github.com/json-iterator/go"
	"reflect"
	"strconv"
	"strings"
	"time"
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

type ModelBindMethod interface {
	Name() string // 名称
	// Validate
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

	// 对于float64类型暂不验证范围是否合理
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
	sv := data.(string)

	atob, err := strconv.ParseBool(sv)
	if err != nil {
		var ves []*openapi.ValidationError
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

// JsonBindMethod json数据类型验证器,适用于泛型路由
type JsonBindMethod[T any] struct {
	Title          string `json:"title,omitempty"`
	RouteParamType openapi.RouteParamType
}

func (m *JsonBindMethod[T]) where(key, value string) map[string]any {
	var where = make(map[string]any)
	if m.RouteParamType == openapi.RouteParamResponse {
		where[whereErrorLabel] = whereServerError[whereErrorLabel]
	} else {
		where[whereErrorLabel] = whereClientError[whereErrorLabel]
	}
	if key != "" {
		where[key] = value
	}

	return where
}

func (m *JsonBindMethod[T]) Name() string { return "JsonBindMethod" }

func (m *JsonBindMethod[T]) Validate(ctx context.Context, data T) (T, []*openapi.ValidationError) {
	var vErr validator.ValidationErrors // validator的校验错误信息
	err := defaultValidator.StructCtx(ctx, data)

	if ok := errors.As(err, &vErr); ok { // 模型验证错误
		ves := make([]*openapi.ValidationError, 0)
		for _, verr := range vErr {
			ves = append(ves, &openapi.ValidationError{
				Ctx:  m.where(validateErrorTagLabel, verr.Tag()),
				Msg:  verr.Error(),
				Type: verr.Type().String(),
				Loc:  []string{"body", m.Title, verr.Field()},
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
		ve := ParseJsoniterError(err, m.RouteParamType, m.Title)
		ves = append(ves, ve)
	}

	return
}

func (m *JsonBindMethod[T]) New() any {
	var value = new(T)
	return value
}

// TimeBindMethod 时间校验方法
type TimeBindMethod struct {
	Title string `json:"title,omitempty" description:"查询参数名"`
}

func (m *TimeBindMethod) Name() string { return "TimeBindMethod" }

// Validate 验证一个字符串是否是一个有效的时间字符串
// @return time.Time
func (m *TimeBindMethod) Validate(ctx context.Context, data any) (any, []*openapi.ValidationError) {
	sv := data.(string) // 肯定是string类型

	var err error
	var t time.Time
	layouts := []string{time.TimeOnly, time.Kitchen}
	for _, layout := range layouts {
		t, err = time.Parse(layout, sv)
		if err == nil {
			return t, nil
		}
	}

	var ves []*openapi.ValidationError
	ves = append(ves, &openapi.ValidationError{
		Loc:  []string{"query", m.Title},
		Msg:  fmt.Sprintf("value: '%s' is not a time, err:%v", sv, err),
		Type: string(openapi.StringType),
		Ctx:  whereClientError,
	})
	return nil, ves
}

func (m *TimeBindMethod) Marshal(obj any) ([]byte, error) {
	t, ok := obj.(time.Time)
	if ok {
		return []byte(t.Format(time.TimeOnly)), nil
	}
	return nil, errors.New("obj is not a time")
}

// Unmarshal time 类型不支持反序列化
func (m *TimeBindMethod) Unmarshal(stream []byte, obj any) (ves []*openapi.ValidationError) {
	return nil
}

func (m *TimeBindMethod) New() any { return nil }

// DateBindMethod 日期校验
type DateBindMethod struct {
	Title string `json:"title,omitempty" description:"查询参数名"`
}

func (m *DateBindMethod) Name() string { return "DateBindMethod" }

func (m *DateBindMethod) Validate(ctx context.Context, data any) (any, []*openapi.ValidationError) {
	sv := data.(string) // 肯定是string类型

	var err error
	var t time.Time
	layouts := []string{time.DateOnly}
	for _, layout := range layouts {
		t, err = time.Parse(layout, sv)
		if err == nil {
			return t, nil
		}
	}

	var ves []*openapi.ValidationError
	ves = append(ves, &openapi.ValidationError{
		Loc:  []string{"query", m.Title},
		Msg:  fmt.Sprintf("value: '%s' is not a date, err:%v", sv, err),
		Type: string(openapi.StringType),
		Ctx:  whereClientError,
	})
	return nil, ves
}

func (m *DateBindMethod) Marshal(obj any) ([]byte, error) {
	t, ok := obj.(time.Time)
	if ok {
		return []byte(t.Format(time.DateOnly)), nil
	}
	return nil, errors.New("obj is not a date")
}

func (m *DateBindMethod) Unmarshal(stream []byte, obj any) (ves []*openapi.ValidationError) {
	//TODO implement me
	panic("implement me")
}

func (m *DateBindMethod) New() any { return nil }

// DateTimeBindMethod 日期时间校验
type DateTimeBindMethod struct {
	Title string `json:"title,omitempty" description:"查询参数名"`
}

func (m *DateTimeBindMethod) Name() string { return "DateTimeBindMethod" }

func (m *DateTimeBindMethod) Validate(ctx context.Context, data any) (any, []*openapi.ValidationError) {
	sv := data.(string) // 肯定是string类型

	var err error
	var t time.Time
	// 按照常用频率排序
	layouts := []string{time.DateTime, time.RFC3339, time.DateOnly, time.TimeOnly, time.Kitchen, time.RFC3339Nano,
		time.RFC822, time.ANSIC, time.UnixDate, time.RubyDate, time.RFC822Z, time.RFC850,
		time.RFC1123, time.RFC1123Z, time.Stamp, time.StampMilli, time.StampMicro, time.StampNano,
	}
	for _, layout := range layouts {
		t, err = time.Parse(layout, sv)
		if err == nil {
			return t, nil
		}
	}

	var ves []*openapi.ValidationError
	ves = append(ves, &openapi.ValidationError{
		Loc:  []string{"query", m.Title},
		Msg:  fmt.Sprintf("value: '%s' is not a datetime, err:%v", sv, err),
		Type: string(openapi.StringType),
		Ctx:  whereClientError,
	})
	return nil, ves
}

func (m *DateTimeBindMethod) Marshal(obj any) ([]byte, error) {
	t, ok := obj.(time.Time)
	if ok {
		return []byte(t.Format(time.DateTime)), nil
	}
	return nil, errors.New("obj is not a datetime")
}

func (m *DateTimeBindMethod) Unmarshal(stream []byte, obj any) (ves []*openapi.ValidationError) {
	return nil
}

func (m *DateTimeBindMethod) New() any { return nil }

// StructQueryBind 结构体查询参数验证器
type StructQueryBind struct {
	json jsoniter.API
}

func (m *StructQueryBind) Unmarshal(params map[string]any, obj any) *openapi.ValidationError {
	s, err := m.json.Marshal(params)
	if err != nil {
		return ParseJsoniterError(err, openapi.RouteParamQuery, "")
	}
	err = m.json.Unmarshal(s, obj)
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
	var where = make(map[string]any)
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

// ParseValidatorError 解析Validator的错误消息
// 如果不存在错误,则返回nil; 如果 err 是 validator.ValidationErrors 的错误, 则解析并返回详细的错误原因,反之则返回模糊的错误原因
func ParseValidatorError(err error, loc openapi.RouteParamType, objName string) []*openapi.ValidationError {
	if err == nil {
		return nil
	}

	var vErr validator.ValidationErrors // validator的校验错误信息
	var ves []*openapi.ValidationError
	var where = make(map[string]any)

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

	// 初始化结构体查询参数方法
	var queryStructJsonConf = jsoniter.Config{
		IndentionStep:                 0,                    // 指定格式化序列化输出时的空格缩进数量
		EscapeHTML:                    false,                // 转义输出HTML
		MarshalFloatWith6Digits:       true,                 // 指定浮点数序列化输出时最多保留6位小数
		ObjectFieldMustBeSimpleString: true,                 // 开启该选项后，反序列化过程中不会对你的json串中对象的字段字符串可能包含的转义进行处理，因此你应该保证你的待解析json串中对象的字段应该是简单的字符串(不包含转义)
		SortMapKeys:                   false,                // 指定map类型序列化输出时按照其key排序
		UseNumber:                     false,                // 指定反序列化时将数字(整数、浮点数)解析成json.Number类型
		DisallowUnknownFields:         false,                // 当开启该选项时，反序列化过程如果解析到未知字段，即在结构体的schema定义中找不到的字段时，不会跳过然后继续解析，而会返回错误
		TagKey:                        openapi.QueryTagName, // 指定tag字符串，默认情况为"json"
		OnlyTaggedField:               false,                // 当开启该选项时，只有带上tag的结构体字段才会被序列化输出
		ValidateJsonRawMessage:        false,                // json.RawMessage类型的字段在序列化时会原封不动地进行输出。开启这个选项后，json-iterator会校验这种类型的字段包含的是否一个合法的json串，如果合法，原样输出；否则会输出"null"
		CaseSensitive:                 false,                // 开启该选项后，你的待解析json串中的对象的字段必须与你的schema定义的字段大小写严格一致
	}
	structQueryBind = &StructQueryBind{json: queryStructJsonConf.Froze()}
}
