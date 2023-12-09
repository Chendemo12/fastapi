package openapi

import (
	"github.com/Chendemo12/fastapi/utils"
	"reflect"
	"unicode"
)

// QModel 查询参数或路径参数元数据, 此类型会进一步转换为 openapi.Parameter
type QModel struct {
	Name     string            `json:"name,omitempty" description:"字段名称"` // 如果是通过结构体生成,则为json标签名称
	Tag      reflect.StructTag `json:"tag,omitempty" description:"TAG"`   // 仅在结构体作为查询参数时有效
	Type     DataType          `json:"otype,omitempty" description:"openapi 数据类型"`
	Kind     reflect.Kind      `json:"Kind,omitempty" description:"反射类型"`
	InPath   bool              `json:"in_path,omitempty" description:"是否是路径参数"`
	Required bool              `json:"required,omitempty" description:"是否必须"`
}

func (q *QModel) Init() (err error) {
	if q.InPath {
		q.Required = true // 路径参数都是必须的
	} else {
		q.Required = IsFieldRequired(q.Tag)
	}
	return
}

func (q *QModel) SchemaTitle() string { return q.Name }

func (q *QModel) SchemaPkg() string { return q.Name }

// JsonName 对于查询参数结构体，其文档名称 tag 为 query
func (q *QModel) JsonName() string {
	return utils.QueryFieldTag(q.Tag, DefaultQueryTagName, q.Name)
}

// SchemaDesc 结构体文档注释
func (q *QModel) SchemaDesc() string {
	return utils.QueryFieldTag(q.Tag, DescriptionTagName, q.Name)
}

// SchemaType 模型类型
func (q *QModel) SchemaType() DataType { return q.Type }

// IsRequired 是否必须
func (q *QModel) IsRequired() bool { return q.Required }

// Schema 输出为OpenAPI文档模型,字典格式
//
//	{
//		"required": true,
//		"schema": {
//			"title": "names",
//			"type": "string",
//			"default": "jack"
//		},
//		"name": "names",
//		"in": "query"/"path"
//	}
func (q *QModel) Schema() (m map[string]any) {
	m["required"] = q.IsRequired()
	m["schema"] = dict{
		"title":   q.SchemaTitle(),
		"type":    q.SchemaType(),
		"default": GetDefaultV(q.Tag, q.SchemaType()),
	}
	m["name"] = q.SchemaPkg()
	m["in"] = "query"

	return
}

// InnerSchema 内部字段模型文档
func (q *QModel) InnerSchema() []SchemaIface {
	m := make([]SchemaIface, 0)
	return m
}

// StructToQModels 将一个结构体的每一个导出字段都转换为一个查询参数
func StructToQModels(rt reflect.Type) []*QModel {
	if rt.Kind() == reflect.Ptr {
		rt = rt.Elem() // 上浮指针
	}

	if rt.Kind() != reflect.Struct { // 仅 struct 有效
		return []*QModel{}
	}

	// 当此model作为查询参数时，此struct的每一个字段都将作为一个查询参数
	m := make([]*QModel, 0, rt.NumField())
	for i := 0; i < rt.NumField(); i++ {
		field := rt.Field(i)

		// 仅导出字段可用
		// TODO: Future-231203.8: 模型不支持嵌入
		if field.Anonymous || unicode.IsLower(rune(field.Name[0])) {
			continue
		}
		// 此结构体的任意字段有且仅支持 基本数据类型
		dataType := ReflectKindToType(field.Type.Kind())
		if !dataType.IsBaseType() {
			continue
		}

		m = append(m, &QModel{
			Name: field.Name, Tag: field.Tag, Type: dataType, Kind: field.Type.Kind(), InPath: false,
		})
	}
	return m
}
