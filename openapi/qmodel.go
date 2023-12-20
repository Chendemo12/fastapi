package openapi

import (
	"github.com/Chendemo12/fastapi/utils"
	"reflect"
	"unicode"
)

// QModel 查询参数或路径参数元数据, 此类型对应于swagger中的: openapi.Parameter
type QModel struct {
	Name     string            `json:"name,omitempty" description:"字段名称"`
	DataType DataType          `json:"data_type,omitempty" description:"openapi 数据类型"`
	Tag      reflect.StructTag `json:"tag,omitempty" description:"TAG"`
	JName    string            `json:"json_name,omitempty" description:"json标签名"`
	QName    string            `json:"query_name,omitempty" description:"query标签名"`
	Desc     string            `json:"description,omitempty" description:"参数描述"`
	Kind     reflect.Kind      `json:"Kind,omitempty" description:"反射类型"`
	Required bool              `json:"required,omitempty" description:"是否必须"`
	InPath   bool              `json:"in_path,omitempty" description:"是否是路径参数"`
	InStruct bool              `json:"in_struct,omitempty" description:"是否是结构体字段参数"`
	IsTime   bool              `json:"is_time,omitempty" description:"是否是时间类型"`
}

// Init 解析并缓存字段名
func (q *QModel) Init() (err error) {
	if q.InPath {
		q.Required = true // 路径参数都是必须的
	} else {
		q.Required = IsFieldRequired(q.Tag)
	}
	// 解析并缓存字段名
	q.Desc = utils.QueryFieldTag(q.Tag, DescriptionTagName, q.Name)
	q.QName = utils.QueryFieldTag(q.Tag, QueryTagName, utils.QueryFieldTag(q.Tag, JsonTagName, q.SchemaTitle()))

	if q.InStruct {
		q.JName = q.QName
	} else {
		q.JName = utils.QueryFieldTag(q.Tag, JsonTagName, q.SchemaTitle())
	}

	return
}

func (q *QModel) SchemaTitle() string { return q.Name }

func (q *QModel) SchemaPkg() string { return q.Name }

// JsonName 对于查询参数结构体，其文档名称 tag 默认为 query
// query -> json -> fieldName
func (q *QModel) JsonName() string { return q.JName }

// SchemaDesc 结构体文档注释
func (q *QModel) SchemaDesc() string { return q.Desc }

// SchemaType 模型类型
func (q *QModel) SchemaType() DataType { return q.DataType }

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
	if q.InPath {
		m["in"] = "path"
	} else {
		m["in"] = "query"
	}
	if q.IsTime {
		m["format"] = "date-time"
	}

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
	// 对于字段类型，仅支持基本类型和time类型，不能为结构体类型
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
		qm := &QModel{
			Name:     field.Name,
			Tag:      field.Tag,
			DataType: dataType,
			Kind:     field.Type.Kind(),
			InPath:   false,
			InStruct: true,
		}

		switch dataType {
		case ArrayType: // 不支持数组类型的查询参数
		case ObjectType:
			if field.Type.String() == TimePkg { // time.Time
				qm.DataType = StringType
				qm.IsTime = true
			}
		default:
		}

		m = append(m, qm)
	}
	return m
}
