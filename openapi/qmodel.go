package openapi

import (
	"errors"
	"reflect"
	"unicode"
)

// QModel 查询参数或路径参数模型, 此类型会进一步转换为 openapi.Parameter
type QModel struct {
	Title  string            `json:"title,omitempty" description:"字段标题"`
	Name   string            `json:"name,omitempty" description:"字段名称"`
	Tag    reflect.StructTag `json:"tag,omitempty" description:"TAG"`
	Type   DataType          `json:"otype,omitempty" description:"openapi 数据类型"`
	InPath bool              `json:"in_path,omitempty" description:"是否是路径参数"`
}

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
		"title":   q.Title,
		"type":    q.SchemaType(),
		"default": GetDefaultV(q.Tag, q.SchemaType()),
	}
	m["name"] = q.SchemaName()
	m["in"] = "query"
	return
}

// SchemaName 获取名称,以json字段为准
func (q *QModel) SchemaName(exclude ...bool) string { return q.Name }

// SchemaDesc 结构体文档注释
func (q *QModel) SchemaDesc() string { return QueryFieldTag(q.Tag, "description", q.Title) }

// SchemaType 模型类型
func (q *QModel) SchemaType() DataType { return q.Type }

// InnerSchema 内部字段模型文档, 全名:文档
func (q *QModel) InnerSchema() (m map[string]map[string]any) {
	m = make(map[string]map[string]any)
	return
}

// IsRequired 是否必须
func (q *QModel) IsRequired() bool             { return IsFieldRequired(q.Tag) }
func (q *QModel) Metadata() (*Metadata, error) { return nil, errors.New("QModel has no metadata") }
func (q *QModel) SetId(id string)              {}

// QueryModel 查询参数基类
type QueryModel struct{}

func (q *QueryModel) Fields() []*QModel {
	return ParseToQueryModels(q) // 此方法无意义
}

// ParseToQueryModels 将一个结构体转换为 QueryModel
func ParseToQueryModels(q QueryParameter) []*QModel {
	rt := ReflectObjectType(q) // 总是指针

	if rt.Kind() != reflect.Struct { // 仅 struct 有效
		return []*QModel{}
	}

	// 当此model作为查询参数时，此struct的每一个字段都将作为一个查询参数
	m := make([]*QModel, 0, rt.NumField())
	for i := 0; i < rt.NumField(); i++ {
		field := rt.Field(i)

		// 仅导出字段可用
		if field.Anonymous || unicode.IsLower(rune(field.Name[0])) {
			continue
		}
		// 此结构体的任意字段有且仅支持 string 类型
		if field.Type.Kind() != reflect.String {
			continue
		}

		m = append(m, &QModel{
			Title:  field.Name,
			Name:   QueryJsonName(field.Tag, field.Name),
			InPath: false,
			Tag:    field.Tag,
			Type:   StringType, // 全部转换为 string 类型
		})
	}
	return m
}
