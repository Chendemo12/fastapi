package openapi

import (
	"errors"
	"reflect"
	"strings"
)

// Field 基本数据模型, 此模型不可再分, 同时也是 BaseModel 的字段类型
// 但此类型不再递归记录,仅记录一个关联模型为基本
type Field struct {
	_pkg        string            `description:"包名.结构体名"`
	Title       string            `json:"title,omitempty" description:"字段名称"`
	Type        DataType          `json:"type,omitempty" description:"openapi 数据类型"`
	Tag         reflect.StructTag `json:"tag" description:"字段标签"`
	Description string            `json:"description,omitempty" description:"说明"`
	ItemRef     string            `description:"子元素类型, 仅Type=array/object时有效"`
}

// Schema 生成字段的详细描述信息
//
//	// 字段为结构体类型
//
//	"position_sat": {
//		"title": "position_sat",
//		"type": "object"
//		"description": "position_sat",
//		"required": false,
//		"$ref": "#/comonents/schemas/example.PositionGeo",
//	}
//
//	// 字段为数组类型, 数组元素为基本类型
//
//	"traffic_timeslot": {
//		"title": "traffic_timeslot",
//		"type": "array"
//		"description": "业务时隙编号数组",
//		"required": false,
//		"items": {
//			"type": "integer"
//		},
//	}
//
//	// 字段为数组类型, 数组元素为自定义结构体类型
//
//	"Detail": {
//		"title": "Detail",
//		"type": "array"
//		"description": "Detail",
//		"required": true,
//		"items": {
//			"$ref": "#/comonents/schemas/ValidationError"
//		},
//	}
func (f *Field) Schema() (m map[string]any) {
	// 最基础的属性，必须
	m = dict{
		"name":        f.SchemaName(true),
		"title":       f.Title,
		"type":        f.Type,
		"required":    f.IsRequired(),
		"description": f.SchemaDesc(),
	}
	// 以validate标签为准
	validatorLabelsMap := make(map[string]string, 0)
	validateTag := QueryFieldTag(f.Tag, defaultTagName, "")

	// 解析Tag
	labels := strings.Split(validateTag, ",")
	for _, label := range labels {
		if label == requiredTag {
			continue
		}
		// 剔除空格
		label = strings.TrimSpace(label)
		vars := strings.Split(label, "=")
		if len(vars) < 2 {
			continue
		}
		validatorLabelsMap[vars[0]] = vars[1]
	}

	// 生成默认值
	if v, ok := validatorLabelsMap[isdefault]; ok {
		m[validatorLabelToOpenapiLabel[isdefault]] = v
	}

	// 为不同的字段类型生成相应的描述
	switch f.Type {
	case IntegerType: // 生成数字类型的最大最小值
		for _, label := range numberTypeValidatorLabels {
			if v, ok := validatorLabelsMap[label]; ok {
				if label == validatorEnumLabel { // 生成字段的枚举值
					m[validatorLabelToOpenapiLabel[label]] = StringsToInts(strings.Split(v, " "))
				} else {
					m[validatorLabelToOpenapiLabel[label]] = v
				}
			}
		}
	case NumberType: // 生成数字类型的最大最小值
		for _, label := range numberTypeValidatorLabels {
			if v, ok := validatorLabelsMap[label]; ok {
				if label == validatorEnumLabel { // 生成字段的枚举值
					m[validatorLabelToOpenapiLabel[label]] = StringsToFloats(strings.Split(v, " "))
				} else {
					m[validatorLabelToOpenapiLabel[label]] = v
				}
			}
		}

	case StringType: // 生成字符串类型的最大最小长度
		for _, label := range stringTypeValidatorLabels {
			if v, ok := validatorLabelsMap[label]; ok {
				if label == validatorEnumLabel {
					m[validatorLabelToOpenapiLabel[label]] = strings.Split(v, " ")
				} else {
					m[validatorLabelToOpenapiLabel[label]] = v
				}
			}
		}

	case ArrayType:
		// 为数组类型生成子类型描述
		switch f.ItemRef {
		case "", string(StringType): // 缺省为string
			m["items"] = map[string]DataType{"type": StringType}
		case string(BoolType):
			m["items"] = map[string]DataType{"type": BoolType}
		case string(NumberType):
			m["items"] = map[string]DataType{"type": NumberType}
		case string(IntegerType):
			m["items"] = map[string]DataType{"type": IntegerType}
		default: // 数组子元素为关联类型
			m["items"] = map[string]string{"$ref": RefPrefix + f.ItemRef}
		}

		// 限制数组的长度
		for _, label := range arrayTypeValidatorLabels {
			if v, ok := validatorLabelsMap[label]; ok {
				m[validatorLabelToOpenapiLabel[label]] = v
			}
		}

	case ObjectType:
		if f.ItemRef != "" { // 字段类型为自定义结构体，生成关联类型，此内部结构体已注册
			m["$ref"] = RefPrefix + f.ItemRef
		}

	default:
	}

	return
}

// SchemaName swagger文档字段名
func (f *Field) SchemaName(exclude ...bool) string {
	if len(exclude) > 0 {
		return QueryJsonName(f.Tag, f.Title)
	}
	return f._pkg
}

// SchemaDesc 字段注释说明
func (f *Field) SchemaDesc() string { return f.Description }

// SchemaType 模型类型
func (f *Field) SchemaType() DataType { return f.Type }

// IsRequired 字段是否必须
func (f *Field) IsRequired() bool { return IsFieldRequired(f.Tag) }

// IsArray 字段是否是数组类型
func (f *Field) IsArray() bool { return f.Type == ArrayType }

// InnerSchema 内部字段模型文档, 全名:文档
func (f *Field) InnerSchema() (m map[string]map[string]any) {
	m = make(map[string]map[string]any)
	return
}

func (f *Field) Metadata() (*Metadata, error) {
	if f._pkg == "" {
		rt := reflect.TypeOf(f).Elem()
		f._pkg = rt.String()
	}

	meta := GetMetadata(f._pkg)
	if meta != nil {
		return meta, nil
	}

	return nil, errors.New("struct is not a BaseModel")
}

func (f *Field) SetId(id string) { f._pkg = id }

// BaseModel 基本数据模型, 对于上层的 app.Route 其请求和相应体都应为继承此结构体的结构体
// 在 OpenApi 文档模型中,此模型的类型始终为 "object";
// 对于 BaseModel 其字段仍然可能会 BaseModel
type BaseModel struct{}

func (b *BaseModel) ID() string {
	rt := reflect.TypeOf(b).Elem()
	return rt.String()
}

// Schema 输出为OpenAPI文档模型,字典格式
//
//	{
//		"title": "examle.MyTimeslot",
//		"type": "object"
//		"description": "examle.mytimeslot",
//		"required": [],
//		"properties": {
//			"control_timeslot": {
//				"title": "control_timeslot",
//				"type": "array"
//				"description": "控制时隙编号数组",
//				"required": false,
//				"items": {
//					"type": "integer"
//				},
//			},
//			"superframe_count": {
//				"title": "superframe_count",
//				"type": "integer"
//				"description": "超帧计数",
//				"required": false,
//			},
//		},
//	},
func (b *BaseModel) Schema() (m map[string]any) {
	// 模型标题排除包名
	m = dict{
		"title":       b.SchemaName(true),
		"type":        b.SchemaType(),
		"description": b.SchemaDesc(),
	}

	meta := GetMetadata(b.ID())
	required := make([]string, 0, len(meta.fields))
	properties := make(map[string]any, len(meta.fields))

	for _, field := range meta.fields {
		if !field.Exported || field.Anonymous { // 非导出字段
			continue
		}

		properties[field.SchemaName(true)] = field.Schema()
		if field.IsRequired() {
			required = append(required, field.SchemaName(true))
		}
	}

	m["required"], m["properties"] = required, properties

	return
}

// SchemaName 获取结构体的名称,默认包含包名
//
//	@param	exclude	[]bool	是否排除包名LL
func (b *BaseModel) SchemaName(exclude ...bool) string {
	meta := GetMetadata(b.ID())
	if len(exclude) > 0 { // 排除包名
		return meta.names[0]
	} else {
		return meta.names[1]
	}
}

// SchemaDesc 结构体文档注释
func (b *BaseModel) SchemaDesc() string {
	meta, err := b.Metadata()
	if err != nil {
		return ""
	}

	return meta.description
}

// SchemaType 模型类型
func (b *BaseModel) SchemaType() DataType { return ObjectType }

func (b *BaseModel) IsRequired() bool { return true }

// Metadata 获取反射后的字段元信息, 此字段应慎重使用
func (b *BaseModel) Metadata() (*Metadata, error) {
	meta := GetMetadata(b.ID())
	if meta != nil {
		return meta, nil
	}

	return nil, errors.New("struct is not a BaseModel")
}
