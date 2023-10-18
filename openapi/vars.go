package openapi

import (
	"github.com/Chendemo12/fastapi-tool/helper"
	"net/http"
	"reflect"
)

//goland:noinspection GoUnusedGlobalVariable
var validatorOperators = map[string]string{
	",": ",", // 多操作符分割
	"|": "|", // 或操作
	"-": "-", // 跳过字段验证
}

// Validator 标签和 Openapi 标签的对应关系
var validatorLabelToOpenapiLabel = map[string]string{
	"required":      "required",         // 必填
	"omitempty":     "omitempty",        // 空时忽略
	"len":           "len",              // 长度
	"eq":            "eq",               // 等于
	"gt":            "minimum",          // 大于
	"gte":           "exclusiveMinimum", // >= 大于等于
	"lt":            "maximum",          // < 小于
	"lte":           "exclusiveMaximum", // <= 小于等于
	"eqfield":       "eqfield",          // 同一结构体字段相等
	"nefield":       "nefield",          // 同一结构体字段不相等
	"gtfield":       "gtfield",          // 大于同一结构体字段
	"gtefield":      "gtefield",         // 大于等于同一结构体字段
	"ltfield":       "ltfield",          // 小于同一结构体字段
	"ltefield":      "ltefield",         // 小于等于同一结构体字段
	"eqcsfield":     "eqcsfield",        // 跨不同结构体字段相等
	"necsfield":     "necsfield",        // 跨不同结构体字段不相等
	"gtcsfield":     "gtcsfield",        // 大于跨不同结构体字段
	"gtecsfield":    "gtecsfield",       // 大于等于跨不同结构体字段
	"ltcsfield":     "ltcsfield",        // 小于跨不同结构体字段
	"ltecsfield":    "ltecsfield",       // 小于等于跨不同结构体字段
	"min":           "minLength",        // 最小值
	"max":           "maxLength",        // 最大值
	"structonly":    "structonly",       // 仅验证结构体，不验证任何结构体字段
	"nostructlevel": "nostructlevel",    // 不运行任何结构级别的验证
	// 向下延伸验证，多层向下需要多个dive标记,
	// [][]string validate:"gt=0,dive,len=1,dive,required"
	"dive": "dive",
	// 与dive同时使用，用于对map对象的键的和值的验证，keys为键，endkeys为值,
	// map[string]string validate:"gt=0,dive,keys,eq=1|eq=2,endkeys,required"
	"dive Keys & EndKeys": "dive Keys & EndKeys",
	"required_with":       "required_with",     // 其他字段其中一个不为空且当前字段不为空Field validate:"required_with=Field1 Field2"
	"required_with_all":   "required_with_all", // 其他所有字段不为空且当前字段不为空Field validate:"required_with_all=Field1 Field2"required_without其他字段其中一个为空且当前字段不为空Field `validate:"required_without=Field1 Field2"required_without_all其他所有字段为空且当前字段不为空Field validate:"required_without_all=Field1 Field2"
	"isdefault":           "default",           // 是默认值Field validate:"isdefault=0"
	"oneof":               "enum",              // 枚举, 其中之一Field validate:"oneof=5 7 9"
	"containsfield":       "containsfield",     // 字段包含另一个字段Field validate:"containsfield=Field2"
	"excludesfield":       "excludesfield",     // 字段不包含另一个字段Field validate:"excludesfield=Field2"
	"unique":              "unique",            // 是否唯一，通常用于切片或结构体Field validate:"unique"
	"alphanum":            "alphanum",          // 字符串值是否只包含 ASCII 字母数字字符
	"alphaunicode":        "alphaunicode",      // 字符串值是否只包含 unicode 字符
	"numeric":             "numeric",           // 字符串值是否包含基本的数值
	"hexadecimal":         "hexadecimal",       // 字符串值是否包含有效的十六进制
	"hexcolor":            "hexcolor",          // 字符串值是否包含有效的十六进制颜色
	"lowercase":           "lowercase",         // 符串值是否只包含小写字符
	"uppercase":           "uppercase",         // 符串值是否只包含大写字符
	"email":               "email",             // 字符串值包含一个有效的电子邮件
	"json":                "json",              // 字符串值是否为有效的 JSON
	"file":                "file",              // 符串值是否包含有效的文件路径，以及该文件是否存在于计算机上
	"url":                 "url",               // 符串值是否包含有效的 url
	"uri":                 "uri",               // 符串值是否包含有效的 uri
	"base64":              "base64",            // 字符串值是否包含有效的 base64值
	"contains":            "contains",          // 字符串值包含子字符串值Field validate:"contains=@"
	"containsany":         "containsany",       // 字符串值包含子字符串值中的任何字符Field validate:"containsany=abc"
	"containsrune":        "containsrune",      // 字符串值包含提供的特殊符号值Field validate:"containsrune=☢"
	"excludes":            "excludes",          // 字符串值不包含子字符串值Field validate:"excludes=@"
	"excludesall":         "excludesall",       // 字符串值不包含任何子字符串值Field validate:"excludesall=abc"
	"excludesrune":        "excludesrune",      // 字符串值不包含提供的特殊符号值Field validate:"containsrune=☢"
	"startswith":          "startswith",        // 字符串以提供的字符串值开始Field validate:"startswith=abc"
	"endswith":            "endswith",          // 字符串以提供的字符串值结束Field validate:"endswith=abc"
	"ip":                  "ip",                // 字符串值是否包含有效的 IP 地址Field validate:"ip"
	//
	"datetime":  "datetime:2006-01-02 15:04:05", // 日期时间
	"timestamp": "timestamp",                    // 时间戳
	"ipv4":      "ipv4",                         // IPv4地址
	"ipv6":      "ipv6",                         // IPv6地址
	"cidr":      "cidr",                         // CIDR地址
	"cidrv4":    "cidrv4",                       // CIDR IPv4地址
	"cidrv6":    "cidrv6",
	"rgb":       "rgb",  // RGB颜色值
	"rgba":      "rgba", // RGBA颜色值
}

var numberTypeValidatorLabels = [...]string{"lt", "gt", "lte", "gte", "eq", "ne", validatorEnumLabel}
var arrayTypeValidatorLabels = [...]string{"min", "max", "len"}
var stringTypeValidatorLabels = [...]string{"min", "max", validatorEnumLabel}

// ValidationErrorDefinition 422 表单验证错误模型
var ValidationErrorDefinition = &ValidationError{}

// ValidationErrorResponseDefinition 请求体相应体错误消息
var ValidationErrorResponseDefinition = &HTTPValidationError{}

var Resp422 = &Response{
	StatusCode:  http.StatusUnprocessableEntity,
	Description: http.StatusText(http.StatusUnprocessableEntity),
	Content: &PathModelContent{
		MIMEType: MIMEApplicationJSON,
		Schema:   &ValidationError{},
	},
}

type dict map[string]any

// ValidationError 参数校验错误
type ValidationError struct {
	BaseModel
	Ctx  map[string]any `json:"service" description:"Service"`
	Msg  string         `json:"msg" description:"Message" binding:"required"`
	Type string         `json:"type" description:"Error Type" binding:"required"`
	Loc  []string       `json:"loc" description:"Location" binding:"required"`
}

func (v *ValidationError) SchemaDesc() string { return "参数校验错误" }

func (v *ValidationError) SchemaType() DataType { return ObjectType }

func (v *ValidationError) Schema() (m map[string]any) {
	return dict{
		"title": ValidationErrorName,
		"type":  ObjectType,
		"properties": dict{
			"loc": dict{
				"title": "Location",
				"type":  "array",
				"items": dict{"anyOf": []map[string]string{{"type": "string"}, {"type": "integer"}}},
			},
			"msg":  dict{"title": "Message", "type": "string"},
			"type": dict{"title": "Error Type", "type": "string"},
		},
		"required": []string{"loc", "msg", "type"},
	}
}

func (v *ValidationError) SchemaName(exclude ...bool) string {
	if len(exclude) > 0 {
		return ValidationErrorName
	} else {
		return InnerModelNamePrefix + ValidationErrorName
	}
}

func (v *ValidationError) Error() string {
	bytes, err := helper.JsonMarshal(v)
	if err != nil {
		return v.SchemaDesc()
	}
	return string(bytes)
}

type HTTPValidationError struct {
	BaseModel
	Detail []*ValidationError `json:"detail" description:"Detail" binding:"required"`
}

func (v *HTTPValidationError) SchemaType() DataType { return ObjectType }

func (v *HTTPValidationError) Schema() map[string]any {
	ve := ValidationError{}
	return dict{
		"title":    HttpValidationErrorName,
		"type":     ObjectType,
		"required": []string{"detail"},
		"properties": dict{
			"detail": dict{
				"title": "Detail",
				"type":  "array",
				"items": dict{"$ref": RefPrefix + ve.SchemaName()},
			},
		},
	}
}

func (v *HTTPValidationError) SchemaName(exclude ...bool) string {
	if len(exclude) > 0 {
		return HttpValidationErrorName
	} else {
		return InnerModelNamePrefix + HttpValidationErrorName
	}
}

func (v *HTTPValidationError) SchemaDesc() string { return "路由参数校验错误" }

func (v *HTTPValidationError) Error() string {
	if len(v.Detail) > 0 {
		return v.Detail[0].Error()
	}
	return "HttpValidationErrorName: "
}

func (v *HTTPValidationError) String() string { return v.Error() }

func List(model SchemaIface) *Metadata {
	if field, ok := model.(*Metadata); ok { // 预定义模型
		meta := &Metadata{}
		*meta = *field
		meta.oType = ArrayType
		return meta
	} else {
		rt := ReflectObjectType(model)
		meta := makeMetadata(ArrayTypePrefix+rt.Name(), model.SchemaDesc(), "", ArrayType)
		meta.model = model

		meta.innerFields = []*MetaField{
			{
				rType: rt,
				Field: Field{
					_pkg:        rt.String(),
					Title:       rt.Name(),
					Tag:         "",
					Description: model.SchemaDesc(),
					ItemRef:     "",
					Type:        ObjectType,
				},
				Exported:  true,
				Anonymous: false,
			},
		}

		return meta
	}
}

func makeMetadata(title, desc, tag string, otype DataType) *Metadata {
	field := Field{
		_pkg:        InnerModelNamePrefix + title,
		Title:       title,
		Tag:         reflect.StructTag(tag + ` description:"` + desc + `"`),
		Description: desc,
		ItemRef:     "",
		Type:        otype,
	}

	return &Metadata{
		description: desc,
		oType:       otype,
		names:       []string{title, InnerModelNamePrefix + title},
		fields:      []*MetaField{},
		model:       nil,
		innerFields: []*MetaField{
			{
				rType:     nil,
				Field:     field,
				Exported:  true,
				Anonymous: false,
			},
		},
	}
}

var (
	// ------------------------------------- int ---------------------------------------

	Int8 = makeMetadata(
		"int8",
		"8位有符号的数字类型",
		`json:"int8" gte:"-128" lte:"127"`,
		IntegerType,
	)

	Int16 = makeMetadata(
		"int16",
		"16位有符号的数字类型",
		`json:"int16" gte:"-32768" lte:"32767"`,
		IntegerType,
	)

	Int32 = makeMetadata(
		"int32",
		"32位有符号的数字类型",
		`json:"int32" gte:"-2147483648" lte:"2147483647"`,
		IntegerType,
	)

	Int64 = makeMetadata(
		"int64",
		"64位有符号的数字类型",
		`json:"int64" gte:"-9223372036854775808" lte:"9223372036854775807"`,
		IntegerType,
	)

	Int = makeMetadata(
		"int",
		"有符号的数字类型",
		`json:"int" gte:"-9223372036854775808" lte:"9223372036854775807"`,
		IntegerType,
	)

	// ------------------------------------- uint ---------------------------------------

	Uint8 = makeMetadata(
		"uint8",
		"8位无符号的数字类型",
		`json:"uint8" gte:"0" lte:"255"`,
		IntegerType,
	)

	Uint16 = makeMetadata(
		"uint16",
		"16位无符号的数字类型",
		`json:"uint16" gte:"0" lte:"65535"`,
		IntegerType,
	)

	Uint32 = makeMetadata(
		"uint32",
		"32位无符号的数字类型",
		`json:"uint32" gte:"0" lte:"4294967295"`,
		IntegerType,
	)

	Uint64 = makeMetadata(
		"uint64",
		"64位无符号的数字类型",
		`json:"uint64" gte:"0" lte:"18446744073709551615"`,
		IntegerType,
	)

	// ------------------------------------- Float ---------------------------------------

	Float32 = makeMetadata(
		"float32",
		"32位的浮点类型",
		`json:"float32"`,
		NumberType,
	)

	Float64 = makeMetadata(
		"float64",
		"64位的浮点类型",
		`json:"float64"`,
		NumberType,
	)

	Float = makeMetadata(
		"float",
		"64位的浮点类型",
		`json:"float"`,
		NumberType,
	)

	// ------------------------------------- other ---------------------------------------

	String = makeMetadata(
		"string",
		"字符串类型",
		`json:"string" min:"0" max:"255"`,
		StringType,
	)

	Bool = makeMetadata(
		"bool",
		"布尔类型",
		`json:"boolean" oneof:"true false"`,
		BoolType,
	)
)
