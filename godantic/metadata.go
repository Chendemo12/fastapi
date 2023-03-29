package godantic

import (
	"reflect"
	"strings"
	"unicode"
)

// 缓存全部的结构体元信息，以减少上层反射次数, 仅用于通过 BaseModel 获取 Metadata
var metaCache = &MetaCache{data: make([]*Metadata, 0)}

// BaseModelToMetadata 提取基本数据模型的元信息
//
//	@param	model	SchemaIface	基本数据模型
//	@return	*Metadata 基本数据模型的元信息
func BaseModelToMetadata(model SchemaIface) *Metadata {
	if model == nil { // 冗余校验
		return nil
	}
	if md, ok := model.(*Metadata); ok { // 接口处定义了基本数据类型和List
		return md
	}

	rt := reflect.TypeOf(model) // 由于接口定义，此处全部为结构体指针
	meta := StructReflect(rt)
	meta.description = model.SchemaDesc()
	meta.model = model // 关联一下自定义类型
	meta.rType = rt

	return meta
}

// GetMetadata 获取结构体的元信息
func GetMetadata(pkg string) *Metadata { return metaCache.Get(pkg) }

func StructReflect(rt reflect.Type) *Metadata {
	if rt.Kind() == reflect.Ptr {
		rt = rt.Elem()
	}

	if rt.Kind() != reflect.Struct {
		return nil
	}

	meta := &Metadata{ // 构造根模型元信息
		names:       []string{rt.Name(), rt.String()}, // 获取包名
		fields:      make([]*MetaField, 0),
		innerFields: make([]*MetaField, 0),
		oType:       ObjectType,
	}

	ref := ModelReflect{metadata: meta}
	for i := 0; i < rt.NumField(); i++ { // 此时肯定是个结构体
		field := rt.Field(i)
		ref.extractField(field, 0) // 0 根起点
	}

	return meta
}

type MetaField struct {
	RType reflect.Type `description:"反射字段类型"`
	Field
	Index     int  `json:"index" description:"当前字段所处的序号"`
	Exported  bool `json:"exported" description:"是否是导出字段"`
	Anonymous bool `json:"anonymous" description:"是否是嵌入字段"`
}

// IsInnerModel 是否是内部模型，如果是则需要重新生成其文档
func (m *MetaField) IsInnerModel() bool {
	if m.RType != nil {
		return m.RType.Kind() == reflect.Struct
	}
	return false
}

// InnerModel 获取内部模型
func (m *MetaField) InnerModel() *Metadata {
	if !m.IsInnerModel() {
		return nil
	}
	meta := StructReflect(m.RType)
	metaCache.Set(meta)
	return meta
}

// Metadata 数据模型 BaseModel 的元信息
type Metadata struct {
	model       SchemaIface     `description:"数据模型,对于预定义类型,此字段无意义"`
	description string          `description:"模型描述"`
	oType       OpenApiDataType `description:"OpenApi 数据类型"`
	names       []string        `description:"结构体名称,包名.结构体名称"`
	fields      []*MetaField    `description:"结构体字段"`
	innerFields []*MetaField    `description:"内部字段"`
	rType       reflect.Type    `description:"结构体元数据"`
}

func (m *Metadata) ReflectType() reflect.Type { return m.rType }

func (m *Metadata) Metadata() (*Metadata, error) { return m, nil }

// Name 获取结构体名称
func (m *Metadata) Name() string { return m.names[0] }

// String 结构体唯一标识：包名+结构体名称
func (m *Metadata) String() string { return m.names[1] }

// Fields 结构体字段
func (m *Metadata) Fields() []*MetaField { return m.fields }

// InnerFields 内部字段
func (m *Metadata) InnerFields() []*MetaField { return m.innerFields }

// Id 获取结构体的唯一标识
func (m *Metadata) Id() string { return m.String() }

// SchemaName 获取结构体的名称,默认包含包名
func (m *Metadata) SchemaName(exclude ...bool) string {
	if len(exclude) > 0 {
		return m.names[0]
	}
	return m.names[1]
}

// 结构体对象的schema文档
func (m *Metadata) objectSchema() map[string]any {
	// 模型标题排除包名
	schema := dict{
		"title":       m.SchemaName(true),
		"type":        m.SchemaType(),
		"description": m.SchemaDesc(),
	}

	required := make([]string, 0, len(m.fields))
	properties := make(map[string]any, len(m.fields))

	for _, field := range m.fields {
		if !field.Exported || field.Anonymous { // 非导出字段
			continue
		}

		properties[field.SchemaName(true)] = field.Schema()
		if field.IsRequired() {
			required = append(required, field.SchemaName(true))
		}
	}

	schema["required"], schema["properties"] = required, properties

	return schema
}

// 数组类型的schema文档
func (m *Metadata) arraySchema() map[string]any {
	switch m.innerFields[0].SchemaType() {

	// 依据规范,基本类型仅需注释type即可
	case IntegerType, NumberType, BoolType, StringType:
		return dict{
			"title":       ArrayTypePrefix + m.SchemaName(true),
			"type":        ArrayType,
			"description": m.SchemaDesc(),
			"items":       dict{"type": m.innerFields[0].SchemaType()},
		}
	default: // 数组或结构体类型, 关联模型
		return dict{
			"title":       m.SchemaName(true),
			"name":        m.SchemaName(),
			"type":        ArrayType,
			"description": m.innerFields[0].SchemaDesc(),
			"items": map[string]string{
				RefName: RefPrefix + m.innerFields[0].SchemaName(),
			},
		}
	}
}

// Schema 输出为OpenAPI文档模型,字典格式
// 数组类型: 需要单独处理, 取其 fields 的第一个元素作为子资源素的实际类型
// 基本数据类型：取其 fields 的第一个元素, description同样取fields 的第一个元素
// 结构体类型: 需处理全部的 fields 和 innerFields
func (m *Metadata) Schema() map[string]any {
	switch m.oType {

	case ArrayType:
		return m.arraySchema()

	case IntegerType, NumberType, BoolType, StringType:
		return m.innerFields[0].Schema()

	default:
		return m.objectSchema()
	}
}

// SchemaDesc 结构体文档注释
func (m *Metadata) SchemaDesc() string {
	switch m.oType {
	case ObjectType:
		return m.description
	default: // 预定义类型,数组类型,基本数据类型
		return m.innerFields[0].Description
	}
}

// SchemaType 模型类型
func (m *Metadata) SchemaType() OpenApiDataType {
	switch m.oType {
	case ObjectType, ArrayType:
		return m.oType
	default: // 预定义类型,基本数据类型
		return m.innerFields[0].OType
	}
}

// IsRequired 字段是否必须
func (m *Metadata) IsRequired() bool { return true }

// AddField 添加字段记录
func (m *Metadata) AddField(field *MetaField)      { m.fields = append(m.fields, field) }
func (m *Metadata) AddInnerField(field *MetaField) { m.innerFields = append(m.innerFields, field) }

type ModelReflect struct {
	metadata *Metadata
}

func (m *ModelReflect) Metadata() *Metadata { return m.metadata }

// 提取结构体字段信息并添加到元信息中
func (m *ModelReflect) extractField(structField reflect.StructField, no int) {
	// 过滤模型基类
	if structField.Anonymous && (structField.Name == "BaseModel" || structField.Name == "Field") {
		return
	}
	// 过滤约定的匿名字段
	if strings.HasPrefix(structField.Name, "_") {
		return
	}

	// ---------------------------------- 获取字段信息 ----------------------------------
	fieldMeta := m.structFieldToMetaField(structField)
	if no < 1 { // 根模型字段
		m.metadata.AddField(fieldMeta)
	} else {
		m.metadata.AddInnerField(fieldMeta)
	}

	switch fieldMeta.OType {
	case IntegerType, NumberType, BoolType, StringType:
		return // 基本类型,无需继续递归处理

	case ObjectType: // 字段为结构体，指针，接口，map等
		if structField.Type.Kind() == reflect.Interface || structField.Type.Kind() == reflect.Map {
			return // 接口或map无需继续向下递归
		}

		no += 1
		// 结构体或结构体指针
		fieldElemType := structField.Type
		if fieldElemType.Kind() == reflect.Ptr {
			fieldElemType = fieldElemType.Elem()
		}

		m.parseFieldWhichIsStruct(fieldElemType, fieldMeta, no)

	case ArrayType: // 字段为数组
		no += 1
		fieldElemType := structField.Type.Elem() // 子元素类型
		m.parseFieldWhichIsArray(fieldElemType, fieldMeta, no)
	}
}

func (m *ModelReflect) structFieldToMetaField(field reflect.StructField) *MetaField {
	mf := &MetaField{
		Field: Field{
			_pkg:        field.PkgPath,
			Title:       field.Name,
			Tag:         field.Tag,
			Description: QueryFieldTag(field.Tag, "description", field.Name),
			ItemRef:     "",
			OType:       ReflectKindToOType(field.Type.Kind()),
		},
		Index:     field.Index[0],
		Exported:  unicode.IsUpper(rune(field.Name[0])),
		Anonymous: field.Anonymous,
		RType:     field.Type,
	}

	if field.PkgPath == "" { // 对于结构体字段，此值无意义
		mf._pkg = m.metadata.String() + "." + field.Name
	}

	return mf
}

// 处理字段是数组的元素
//
//	@param	elemType	reflect.Type	子元素类型
//	@param	metadata	*Metadata		根模型元信息
//	@param	metaField	*MetaField		字段元信息
func (m *ModelReflect) parseFieldWhichIsArray(elemType reflect.Type, fieldMeta *MetaField, no int) {
	if elemType.Kind() == reflect.Pointer { // 数组元素为指针结构体
		elemType = elemType.Elem()
	}

	// 处理数组的子元素
	kind := elemType.Kind()
	switch kind {
	case reflect.String:
		fieldMeta.ItemRef = String.SchemaName(true)
	case reflect.Bool:
		fieldMeta.ItemRef = Bool.SchemaName(true)

	case reflect.Array, reflect.Slice, reflect.Chan: // [][]*Student
		mf := &MetaField{
			Field: Field{
				_pkg:        elemType.String(),
				Title:       elemType.Name(),
				Tag:         "",
				Description: fieldMeta.Description,
				ItemRef:     "",
				OType:       ArrayType,
			},
			Index:     0,
			Exported:  true,
			Anonymous: false,
			RType:     elemType,
		}
		fieldMeta.ItemRef = elemType.String()
		m.metadata.AddInnerField(mf)
		no += 1
		m.parseFieldWhichIsArray(elemType.Elem(), mf, no)

	case reflect.Struct:
		mf := &MetaField{
			Field: Field{
				_pkg:        elemType.String(),
				Title:       elemType.Name(),
				Tag:         "",
				Description: fieldMeta.Description,
				ItemRef:     "",
				OType:       ObjectType,
			},
			Index:     0,
			Exported:  true,
			Anonymous: false,
			RType:     elemType,
		}
		m.metadata.AddInnerField(mf)
		fieldMeta.ItemRef = elemType.String()
		no += 1
		for i := 0; i < elemType.NumField(); i++ { // 此时必不是指针
			field := elemType.Field(i)
			m.extractField(field, no) // 递归
		}

	default:
		if reflect.Bool < kind && kind <= reflect.Uint64 {
			fieldMeta.ItemRef = Int.SchemaName(true)
		}
		if reflect.Float32 <= kind && kind <= reflect.Complex128 {
			fieldMeta.ItemRef = Float.SchemaName(true)
		}
	}
}

// 处理字段是结构体的元素
//
//	@param	elemType	reflect.Type	子元素类型
//	@param	metadata	*Metadata		根模型元信息
//	@param	metaField	*MetaField		字段元信息
func (m *ModelReflect) parseFieldWhichIsStruct(elemType reflect.Type, fieldMeta *MetaField, no int) {
	fieldMeta.ItemRef = elemType.String() // 关联模型

	mf := &MetaField{
		Field: Field{
			_pkg:        elemType.String(),
			Title:       elemType.Name(),
			Tag:         "",
			Description: "",
			ItemRef:     "",
			OType:       ObjectType,
		},
		Index:     0,
		Exported:  true,
		Anonymous: false,
		RType:     elemType,
	}
	//if method, ok := elemType.MethodByName("SchemaName"); ok {
	//}
	m.metadata.AddInnerField(mf) // 此时必然是内部结构体
}

// MetaCache Metadata 缓存
type MetaCache struct {
	data []*Metadata
}

func (m *MetaCache) Get(pkg string) *Metadata {
	for i := 0; i < len(m.data); i++ {
		if m.data[i].String() == pkg {
			return m.data[i]
		}
	}
	return nil
}

// Set 保存一个元信息，存在则更新
func (m *MetaCache) Set(meta *Metadata) {
	for i := 0; i < len(m.data); i++ {
		if m.data[i].String() == meta.String() {
			m.data[i] = meta
			return
		}
	}
	m.data = append(m.data, meta)
}
