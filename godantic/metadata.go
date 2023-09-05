package godantic

import (
	"reflect"
	"strings"
	"unicode"
)

// AnonymousModelNameConnector 为匿名结构体生成一个名称, 连接符
const AnonymousModelNameConnector = "_"

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

	meta := &Metadata{}
	meta.FromModel(model)

	return meta
}

type MetaField struct {
	RType reflect.Type `description:"反射字段类型"`
	Field
	Index     int  `json:"index" description:"当前字段所处的序号"`
	Exported  bool `json:"exported" description:"是否是导出字段"`
	Anonymous bool `json:"anonymous" description:"是否是嵌入字段"`
}

// IsBaseModel 是否仍然是个基本模型
func (m *MetaField) IsBaseModel() (bool, *Metadata) {
	if m.RType != nil && (m.RType.Kind() == reflect.Struct || m.RType.Kind() == reflect.Ptr) {
		meta := &Metadata{}
		meta.FromReflectType(m.RType)
		metaCache.Set(meta)
		return true, meta
	}
	return false, nil
}

// Metadata 数据模型 BaseModel 的元信息
type Metadata struct {
	model       SchemaIface     `description:"数据模型,对于预定义类型,此字段无意义"`
	rType       reflect.Type    `description:"结构体元数据"`
	description string          `description:"模型描述"`
	oType       OpenApiDataType `description:"OpenApi 数据类型"`
	names       []string        `description:"结构体名称,包名.结构体名称"`
	fields      []*MetaField    `description:"结构体字段"`
	innerFields []*MetaField    `description:"内部字段"`
	//innerModel []*Metadata `description:"内部模型"`
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

func (m *Metadata) IsRequired() bool { return true }

// AddField 添加字段记录
//
//	@param	depth	int	节点层级数
func (m *Metadata) AddField(field *MetaField, depth int) {
	if depth < 1 {
		m.fields = append(m.fields, field)
	} else {
		m.innerFields = append(m.innerFields, field)
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

// 提取结构体字段信息并添加到元信息中
func (m *Metadata) extractStructField(structField reflect.StructField, depth int) {
	// 过滤模型基类
	if structField.Anonymous && (structField.Name == "BaseModel" || structField.Name == "Field") {
		return
	}
	// 过滤约定的字段
	if strings.HasPrefix(structField.Name, "_") {
		return
	}
	// 未导出字段
	if !unicode.IsUpper(rune(structField.Name[0])) {
		return
	}
	// ---------------------------------- 获取字段信息 ----------------------------------
	fieldMeta := structFieldToMetaField(structField, m.String())
	m.AddField(fieldMeta, depth)

	switch fieldMeta.OType {
	case IntegerType, NumberType, BoolType, StringType:
		return // 基本类型,无需继续递归处理

	case ObjectType: // 字段为结构体，指针，接口，map等
		if structField.Type.Kind() == reflect.Interface || structField.Type.Kind() == reflect.Map {
			return // 接口或map无需继续向下递归
		}
		// Metadata TODO:
		m.parseFieldWhichIsStruct(fieldMeta, structField, depth+1)

	case ArrayType: // 字段为数组
		elemType := structField.Type.Elem() // 子元素类型
		m.parseFieldWhichIsArray(fieldMeta, elemType, depth+1)
	}
}

// 处理字段是数组的元素
//
//	@param	elemType	reflect.Type	子元素类型
//	@param	metadata	*Metadata		根模型元信息
//	@param	metaField	*MetaField		字段元信息
func (m *Metadata) parseFieldWhichIsArray(fieldMeta *MetaField, elemType reflect.Type, depth int) {
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
		depth += 1
		m.AddField(mf, depth)
		m.parseFieldWhichIsArray(mf, elemType.Elem(), depth)

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
		fieldMeta.ItemRef = elemType.String()
		depth += 1
		m.AddField(mf, depth)
		for i := 0; i < elemType.NumField(); i++ { // 此时必不是指针
			field := elemType.Field(i)
			m.extractStructField(field, depth) // 递归
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
func (m *Metadata) parseFieldWhichIsStruct(fieldMeta *MetaField, field reflect.StructField, depth int) {
	elemType := field.Type
	// 结构体或结构体指针
	if elemType.Kind() == reflect.Ptr {
		elemType = elemType.Elem()
	}

	// 计算子结构体的包信息
	var pkg, name string
	if elemType.Name() == "" { // 未命名的结构体类型，没有名称, 分配包名和名称
		name = fieldMeta.Title + "Model"
		pkg = fieldMeta._pkg + AnonymousModelNameConnector + name
	} else {
		pkg = elemType.String() // 关联模型
		name = elemType.Name()
	}
	// 关联子模型
	fieldMeta.ItemRef = pkg

	// 首先记录一下结构体自身
	mf := &MetaField{
		Index:     0,
		Exported:  true,
		Anonymous: false,
		RType:     elemType,
	}
	mf.Title = name
	mf.OType = ObjectType // 标记此为一个模型，后面会继续生成其文档
	mf._pkg = pkg

	m.AddField(mf, depth) // 此时必然是内部结构体

	//// TODO: 名称计算错误
	//for i := 0; i < elemType.NumField(); i++ {
	//	childField := elemType.Field(i)
	//	m.extractStructField(childField, depth) // 递归
	//}
}

// FromModel 从模型构造元数据，仅支持结构体
func (m *Metadata) FromModel(model SchemaIface) {
	m.description = model.SchemaDesc()
	m.model = model             // 关联一下自定义类型
	rt := reflect.TypeOf(model) // 由于接口定义，此处全部为结构体指针

	m.FromReflectType(rt)
}

// FromReflectType 从反射类型种构造元数据
func (m *Metadata) FromReflectType(rt reflect.Type) {
	if rt.Kind() == reflect.Ptr {
		rt = rt.Elem()
	}

	if rt.Kind() != reflect.Struct {
		return
	}

	// 构造模型元信息
	m.rType = rt
	m.names = []string{rt.Name(), rt.String()} // 获取包名
	m.fields = make([]*MetaField, 0)
	m.innerFields = make([]*MetaField, 0)
	m.oType = ObjectType

	for i := 0; i < rt.NumField(); i++ { // 此时肯定是个结构体
		field := rt.Field(i)
		m.extractStructField(field, 0) // 0 根起点
	}
}

// ----------------------------------------------------------------------------

// 缓存全部的结构体元信息，以减少上层反射次数, 仅用于通过 BaseModel 获取 Metadata
var metaCache = &MetaCache{data: make([]*Metadata, 0)}

// GetMetadata 获取结构体的元信息
func GetMetadata(pkg string) *Metadata { return metaCache.Get(pkg) }

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

func structFieldToMetaField(structField reflect.StructField, fatherPkg string) *MetaField {
	fieldMeta := &MetaField{
		Index:     structField.Index[0],
		Exported:  true,
		Anonymous: structField.Anonymous,
		RType:     structField.Type,
	}
	fieldMeta.Title = structField.Name
	fieldMeta.Tag = structField.Tag
	fieldMeta.Description = QueryFieldTag(structField.Tag, "description", structField.Name)
	fieldMeta.OType = ReflectKindToOType(structField.Type.Kind())
	fieldMeta.ItemRef = ""

	if structField.PkgPath != "" {
		fieldMeta._pkg = structField.PkgPath
	} else {
		// 遇到匿名结构体，分配一个名称
		fieldMeta._pkg = fatherPkg + AnonymousModelNameConnector + structField.Name
	}

	return fieldMeta
}
