package godantic

import (
	"reflect"

	"github.com/Chendemo12/fastapi-tool/helper"
)

// ValidationError 参数校验错误
type ValidationError struct {
	BaseModel
	Ctx  map[string]any `json:"service" description:"Service"`
	Msg  string         `json:"msg" description:"Message" binding:"required"`
	Type string         `json:"type" description:"Error RType" binding:"required"`
	Loc  []string       `json:"loc" description:"Location" binding:"required"`
}

func (v *ValidationError) SchemaDesc() string { return "参数校验错误" }

func (v *ValidationError) SchemaType() OpenApiDataType { return ObjectType }

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

func (v *HTTPValidationError) SchemaType() OpenApiDataType { return ObjectType }

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
				RType: rt,
				Field: Field{
					_pkg:        rt.String(),
					Title:       rt.Name(),
					Tag:         "",
					Description: model.SchemaDesc(),
					ItemRef:     "",
					OType:       ObjectType,
				},
				Exported:  true,
				Anonymous: false,
			},
		}

		return meta
	}
}

func makeMetadata(title, desc, tag string, otype OpenApiDataType) *Metadata {
	field := Field{
		_pkg:        InnerModelNamePrefix + title,
		Title:       title,
		Tag:         reflect.StructTag(tag + ` description:"` + desc + `"`),
		Description: desc,
		ItemRef:     "",
		OType:       otype,
	}

	return &Metadata{
		description: desc,
		oType:       otype,
		names:       []string{title, InnerModelNamePrefix + title},
		fields:      []*MetaField{},
		model:       nil,
		innerFields: []*MetaField{
			{
				RType:     nil,
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
		"Int8",
		"8位有符号的数字类型",
		`json:"int8" gte:"-128" lte:"127"`,
		IntegerType,
	)

	Int16 = makeMetadata(
		"Int16",
		"16位有符号的数字类型",
		`json:"int16" gte:"-32768" lte:"32767"`,
		IntegerType,
	)

	Int32 = makeMetadata(
		"Int32",
		"32位有符号的数字类型",
		`json:"int32" gte:"-2147483648" lte:"2147483647"`,
		IntegerType,
	)

	Int64 = makeMetadata(
		"Int64",
		"64位有符号的数字类型",
		`json:"int64" gte:"-9223372036854775808" lte:"9223372036854775807"`,
		IntegerType,
	)

	Int = makeMetadata(
		"Int",
		"有符号的数字类型",
		`json:"int" gte:"-9223372036854775808" lte:"9223372036854775807"`,
		IntegerType,
	)

	// ------------------------------------- uint ---------------------------------------

	Uint8 = makeMetadata(
		"Uint8",
		"8位无符号的数字类型",
		`json:"uint8" gte:"0" lte:"255"`,
		IntegerType,
	)

	Uint16 = makeMetadata(
		"Uint16",
		"16位无符号的数字类型",
		`json:"uint16" gte:"0" lte:"65535"`,
		IntegerType,
	)

	Uint32 = makeMetadata(
		"Uint32",
		"32位无符号的数字类型",
		`json:"uint32" gte:"0" lte:"4294967295"`,
		IntegerType,
	)

	Uint64 = makeMetadata(
		"Uint64",
		"64位无符号的数字类型",
		`json:"uint64" gte:"0" lte:"18446744073709551615"`,
		IntegerType,
	)

	// ------------------------------------- Float ---------------------------------------

	Float32 = makeMetadata(
		"Float32",
		"32位的浮点类型",
		`json:"float32"`,
		NumberType,
	)

	Float64 = makeMetadata(
		"Float64",
		"64位的浮点类型",
		`json:"float64"`,
		NumberType,
	)

	Float = makeMetadata(
		"Float",
		"64位的浮点类型",
		`json:"float"`,
		NumberType,
	)

	// ------------------------------------- other ---------------------------------------

	String = makeMetadata(
		"String",
		"字符串类型",
		`json:"string" min:"0" max:"255"`,
		StringType,
	)

	Bool = makeMetadata(
		"Bool",
		"布尔类型",
		`json:"boolean" oneof:"true false"`,
		BoolType,
	)
)
