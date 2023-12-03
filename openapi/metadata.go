package openapi

import (
	"github.com/Chendemo12/fastapi/utils"
	"reflect"
	"strings"
	"unicode"
)

// BaseModelMeta 所有数据模型 ModelSchema 的元信息
type BaseModelMeta struct {
	Param       *RouteParam
	Description string                    `description:"模型描述"`
	fields      []*BaseModelField         `description:"结构体字段"`
	doc         map[string]any            `description:"模型文档"`
	innerDocs   map[string]map[string]any `description:"内部模型文档,名称:文档"`
	innerModels []*BaseModelField         `description:"子模型, 对于未命名结构体，给其指定一个结构体名称"`
	itemModel   *BaseModelMeta            `description:"当此模型为数组时, 记录内部元素的模型,同样可能是个数组"`
}

func NewBaseModelMeta(param *RouteParam) *BaseModelMeta {
	meta := &BaseModelMeta{}
	meta.Param = param
	return meta
}

func (m *BaseModelMeta) Init() (err error) {
	err = m.Scan()

	return
}

func (m *BaseModelMeta) Scan() (err error) {
	err = m.scanModel()
	if err != nil {
		return err
	}

	// 构建模型文档
	err = m.scanSwagger()
	return
}

// ScanInner 无内部模型，无需向下递归解析 BaseModelField
func (m *BaseModelMeta) ScanInner() (err error) {
	return
}

// 解析结构体, 提取字段, Model 不允许为nil
// 此方法的最终产物就是解析为一个个的 BaseModelField
func (m *BaseModelMeta) scanModel() (err error) {
	m.fields = make([]*BaseModelField, 0)
	m.innerModels = make([]*BaseModelField, 0)

	if m.Param.Type.IsBaseType() {
		// 接口方法处返回了基本类型,或请求体参数为基本类型, 无需进一步解析
		return
	}

	// 检测到数组或结构体, 解析模型信息
	if m.Param.Type == ArrayType {
		// 方法处直接返回数组, 递归处理子元素
		param := NewRouteParam(m.Param.CopyPrototype().Elem(), 0)
		m.itemModel = NewBaseModelMeta(param)
		err = m.itemModel.Init()

	} else if m.Param.Type == ObjectType {
		// 方法处返回结构体或结构体指针
		err = m.scanObject()
	}

	return
}

// 解析结构体
func (m *BaseModelMeta) scanObject() (err error) {
	rt := m.Param.CopyPrototype()
	if rt.Kind() == reflect.Ptr {
		rt = rt.Elem()
	}

	if rt.Kind() == reflect.Map {
		// 对于map, 无法获得其字段信息,因为在生成文档时,没有任何字段,会直接显示成 {}
		return
	}

	// 此时肯定是结构体了
	for i := 0; i < rt.NumField(); i++ {
		field := rt.Field(i)
		// 此处无需过滤字段，文档生成时会过滤
		m.scanStructField(rt, field, 0) // field0 根起点
	}
	return
}

// 提取结构体字段信息并添加到元信息中
func (m *BaseModelMeta) scanStructField(fatherStruct reflect.Type, field reflect.StructField, depth int) {
	// 过滤模型基类
	if utils.Has[string](InnerModelsName, field.Name) {
		return
	}

	// ---------------------------------- 获取字段信息 ----------------------------------
	fieldMeta := &BaseModelField{
		Exported:  unicode.IsUpper(rune(field.Name[0])),
		Anonymous: field.Anonymous,
		rType:     field.Type,
	}
	fieldMeta.Tag = field.Tag
	fieldMeta.Type = ReflectKindToType(field.Type.Kind())
	fieldMeta.Description = utils.QueryFieldTag(field.Tag, DescriptionTagName, field.Name)

	m.addField(fieldMeta, depth)

	switch fieldMeta.SchemaType() {
	case IntegerType, NumberType, BoolType, StringType:
		// 基本类型,无需继续递归处理
		fieldMeta.Pkg, fieldMeta.Name = field.PkgPath, field.Name
		return

	case ObjectType:
		// 字段为结构体，指针，接口，map等
		if utils.Has[reflect.Kind](IllegalRouteParamType, field.Type.Kind()) {
			// 接口或map无需继续向下递归
			return
		}

		elemType := utils.GetElementType(field.Type)
		fieldMeta.Pkg, fieldMeta.Name = confirmStructFieldName(fatherStruct, field, elemType)
		m.scanFieldWhichIsStruct(field, fieldMeta, elemType, depth+1)

	case ArrayType: // 字段为数组
		elemType := utils.GetElementType(field.Type) // 子元素类型
		fieldMeta.Pkg, fieldMeta.Name = confirmStructFieldName(fatherStruct, field, elemType)
		m.scanFieldWhichIsArray(field, fieldMeta, elemType, depth+1)
	}
}

func confirmStructFieldName(fatherStruct reflect.Type, field reflect.StructField, fieldType reflect.Type) (string, string) {
	var pkg, name string
	if utils.IsAnonymousStruct(fieldType) {
		// 未命名的结构体类型, 没有名称, 分配包名和名称 == 结构体名.字段名Model
		name = fatherStruct.Name() + field.Name + AnonymousModelNameConnector + "Model"
		pkg = fatherStruct.String() + field.Name + AnonymousModelNameConnector + "Model"
	} else {
		pkg, name = fieldType.String(), fieldType.Name()
	}

	return pkg, name
}

// 处理字段是数组的元素
//
//	@param	elemType	reflect.Type		子元素类型
//	@param	metadata	*ModelMeta			根模型元信息
//	@param	metaField	*BaseModelField		字段元信息
func (m *BaseModelMeta) scanFieldWhichIsArray(fatherModel reflect.StructField, fieldMeta *BaseModelField, elemType reflect.Type, depth int) {
	if elemType.Kind() == reflect.Pointer { // 数组元素为指针结构体
		elemType = elemType.Elem()
	}

	// 处理数组的子元素
	pkg, name := assignModelNames(fatherModel, elemType)

	kind := elemType.Kind()
	switch kind {
	case reflect.String:
		fieldMeta.ItemRef = string(StringType)

	case reflect.Bool:
		fieldMeta.ItemRef = string(BoolType)

	case reflect.Array, reflect.Slice, reflect.Chan: // [][]*Student
		// TODO: maybe not work
		fieldMeta.ItemRef = pkg
		mf := &BaseModelField{
			Pkg:         pkg,
			Name:        name,
			Tag:         "",
			Description: fieldMeta.Description,
			ItemRef:     "",
			Type:        ArrayType,
			Exported:    true,
			Anonymous:   false,
			rType:       elemType,
		}

		m.addField(mf, depth)
		m.scanFieldWhichIsArray(fatherModel, mf, elemType.Elem(), depth+1)

	case reflect.Struct:
		fieldMeta.ItemRef = pkg
		rt := utils.GetElementType(elemType)
		m.scanFieldWhichIsStruct(fatherModel, fieldMeta, rt, depth+1)

	default:
		if reflect.Bool < kind && kind <= reflect.Uint64 {
			fieldMeta.ItemRef = string(IntegerType)
		}
		if reflect.Float32 <= kind && kind <= reflect.Complex128 {
			fieldMeta.ItemRef = string(NumberType)
		}
	}
}

// 处理字段是结构体的元素
//
//	@param	elemType	reflect.Type		子元素类型
//	@param	metadata	*ModelMeta			根模型元信息
//	@param	metaField	*BaseModelField		字段元信息
func (m *BaseModelMeta) scanFieldWhichIsStruct(fieldStruct reflect.StructField, fieldMeta *BaseModelField, fieldType reflect.Type, depth int) {
	// 首先记录一下结构体自身, 不设置为 BaseModelMeta 原因在于，避免递归处理，将模型展平
	mf := &BaseModelField{Exported: true, Anonymous: false, rType: fieldType, Type: ObjectType}
	mf.Description = fieldMeta.Description
	// 计算子结构体的包信息
	if fieldType.Name() == "" { // 匿名结构体, 将上层分配的自定义名称作为此结构体的标识
		mf.Pkg = fieldMeta.Pkg
		mf.Name = fieldMeta.Name
	} else {
		mf.Pkg = fieldType.String() // 具名结构体，获得真实名称
		mf.Name = fieldType.Name()
	}

	// 将上一个字段关联此模型
	fieldMeta.ItemRef = mf.Pkg

	m.addField(mf, depth)

	for i := 0; i < fieldType.NumField(); i++ {
		field := fieldType.Field(i)
		m.scanStructField(fieldType, field, depth+1)
	}
}

// 解析模型文档
// 此方法的最终产物就是构建出 doc 字典文档
func (m *BaseModelMeta) scanSwagger() (err error) {
	// 区分基本类型和自定义类型
	switch m.Param.Type {

	case IntegerType, NumberType, BoolType, StringType:
		// 匹配到基本类型
		err = m.scanBaseSwagger()
	case ArrayType:
		err = m.scanArraySwagger()
	default:
		err = m.scanObjectSwagger()
	}

	return
}

// 生成基本类型的文档
func (m *BaseModelMeta) scanBaseSwagger() (err error) {
	// 最基础的属性，必须
	m.doc = map[string]any{
		"name":  m.SchemaPkg(),
		"title": m.Name(),
		"type":  m.SchemaType(),
	}

	rt := m.Param.CopyPrototype()
	if rt.Kind() == reflect.Ptr {
		rt = rt.Elem()
	}

	// 为不同的字段类型生成相应的描述
	switch rt.Kind() {

	case reflect.Int, reflect.Int64:
		// 生成数字类型的最大最小值
		m.doc[ValidatorLabelToOpenapiLabel["lte"]] = 9223372036854775807
		m.doc[ValidatorLabelToOpenapiLabel["gte"]] = -9223372036854775808
		m.Description = "有符号的数字类型" // 重写注释
	case reflect.Int8:
		m.doc[ValidatorLabelToOpenapiLabel["lte"]] = 127
		m.doc[ValidatorLabelToOpenapiLabel["gte"]] = -128
		m.Description = "8位有符号的数字类型"
	case reflect.Int16:
		m.doc[ValidatorLabelToOpenapiLabel["lte"]] = 32767
		m.doc[ValidatorLabelToOpenapiLabel["gte"]] = -32768
		m.Description = "16位有符号的数字类型"
	case reflect.Int32:
		m.doc[ValidatorLabelToOpenapiLabel["lte"]] = 2147483647
		m.doc[ValidatorLabelToOpenapiLabel["gte"]] = -2147483648
		m.Description = "32位有符号的数字类型"

	case reflect.Uint, reflect.Uint64:
		m.doc[ValidatorLabelToOpenapiLabel["lte"]] = uint64(9223372036854775809)
		m.doc[ValidatorLabelToOpenapiLabel["gte"]] = 0
		m.Description = "无符号的数字类型"
	case reflect.Uint8:
		m.doc[ValidatorLabelToOpenapiLabel["lte"]] = uint8(255)
		m.doc[ValidatorLabelToOpenapiLabel["gte"]] = 0
		m.Description = "8位无符号的数字类型"
	case reflect.Uint16:
		m.doc[ValidatorLabelToOpenapiLabel["lte"]] = uint16(65535)
		m.doc[ValidatorLabelToOpenapiLabel["gte"]] = 0
		m.Description = "16位无符号的数字类型"
	case reflect.Uint32:
		m.doc[ValidatorLabelToOpenapiLabel["lte"]] = uint32(4294967295)
		m.doc[ValidatorLabelToOpenapiLabel["gte"]] = 0
		m.Description = "32位无符号的数字类型"

	case reflect.Float32:
		m.Description = "32位的浮点类型"
	case reflect.Float64:
		m.Description = "64位的浮点类型"

	case reflect.String:
		// 生成字符串类型的最大最小长度
		m.Description = "字符串类型"
	default:
	}

	m.doc[DefaultParamRequiredLabel] = m.IsRequired()
	m.doc[DescriptionTagName] = m.SchemaDesc()

	return
}

func (m *BaseModelMeta) scanObjectSwagger() (err error) {
	// 判断类型是否实现了 SchemaIface 接口
	desc := ReflectCallSchemaDesc(m.Param.CopyPrototype())
	if desc != "" {
		m.Description = desc
	} else {
		m.Description = m.Param.Pkg
	}
	m.doc = map[string]any{}

	// 组合出模型文档
	schema := dict{
		"title":            m.SchemaTitle(), // 模型标题排除包名
		"type":             m.SchemaType(),
		DescriptionTagName: m.SchemaDesc(),
	}

	required := make([]string, 0, len(m.fields))
	properties := make(map[string]any, len(m.fields))

	for _, field := range m.fields {
		// TODO Future-231203.8: 模型不支持嵌入;
		if !field.Exported || field.Anonymous { // 非导出字段
			continue
		}

		// NOTICE: 显示为 json 标签名称，而非结构体名称
		properties[field.JsonName()] = field.Schema()
		if field.IsRequired() {
			required = append(required, field.JsonName())
		}
	}

	schema[DefaultParamRequiredLabel], schema["properties"] = required, properties

	m.doc = schema

	return
}

// 数组类型，递归解析子元素
func (m *BaseModelMeta) scanArraySwagger() (err error) {
	switch m.itemModel.SchemaType() {
	// 依据规范,基本类型仅需注释type即可
	case IntegerType, NumberType, BoolType, StringType:
		m.doc = dict{
			"title": ArrayTypePrefix + m.SchemaTitle(),
			"items": dict{"type": m.itemModel.SchemaType()},
		}
	default: // 数组或结构体类型, 关联模型
		m.doc = dict{
			"title": m.SchemaTitle(),
			"name":  m.SchemaPkg(),
			"items": map[string]string{
				RefName: RefPrefix + m.itemModel.SchemaPkg(),
			},
		}
	}

	// 将子元素的文档作为此模型的文档，如果子元素是结构体则反射获取其文档
	m.Description = ArrayTypePrefix + m.itemModel.SchemaDesc()
	if m.itemModel.SchemaType() == ObjectType {
		if desc := ReflectCallSchemaDesc(m.itemModel.Param.Prototype); desc != "" {
			m.Description = ArrayTypePrefix + desc
		}
	}

	m.doc[DescriptionTagName] = m.SchemaDesc()
	m.doc["type"] = ArrayType

	return
}

// 添加字段记录
//
//	@param	depth	int	节点层级数
func (m *BaseModelMeta) addField(field *BaseModelField, depth int) {
	if depth < 1 {
		m.fields = append(m.fields, field)
	} else {
		m.innerModels = append(m.innerModels, field)
	}
}

func (m *BaseModelMeta) Name() string { return m.Param.Name }

func (m *BaseModelMeta) SchemaPkg() string { return m.Param.Pkg }

// SchemaTitle 获取结构体的名称,默认包含包名
func (m *BaseModelMeta) SchemaTitle() string { return m.Param.Name }

func (m *BaseModelMeta) JsonName() string { return m.SchemaTitle() }

// SchemaDesc 模型文档注释
func (m *BaseModelMeta) SchemaDesc() string {
	return m.Description
}

// SchemaType 模型类型
func (m *BaseModelMeta) SchemaType() DataType {
	return m.Param.Type
}

// IsRequired 模型都是必须的
func (m *BaseModelMeta) IsRequired() bool {
	return true
}

// Schema 输出为OpenAPI文档模型,字典格式
// 数组类型: 需要单独处理, 取其 fields 的第一个元素作为子元素的实际类型
// 结构体类型: 需处理全部的 fields 和 innerFields
func (m *BaseModelMeta) Schema() (dict map[string]any) {
	dict = m.doc
	return
}

// InnerSchema 内部字段模型文档
func (m *BaseModelMeta) InnerSchema() []SchemaIface {
	ss := make([]SchemaIface, len(m.innerModels))
	for i := 0; i < len(m.innerModels); i++ {
		ss[i] = m.innerModels[i]
	}

	return ss
}

// BaseModelField 模型的字段元数据
// 基本数据模型, 此模型不可再分, 同时也是 ModelSchema 的字段类型
// 但此类型不再递归记录,仅记录一个关联模型为基本
type BaseModelField struct {
	Pkg         string            `description:"包名.结构体名.字段名"`
	Name        string            `json:"name" description:"字段名称"`
	Type        DataType          `json:"type" description:"openapi 数据类型"`
	Tag         reflect.StructTag `json:"tag" description:"字段标签"`
	Description string            `json:"description,omitempty" description:"说明"`
	ItemRef     string            `description:"子元素类型, 仅Type=array/object时有效"`
	Exported    bool              `description:"是否是导出字段"`
	Anonymous   bool              `description:"是否是嵌入字段"`
	rType       reflect.Type      `description:"反射字段类型"`
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
func (f *BaseModelField) Schema() (m map[string]any) {
	// 最基础的属性，必须
	m = dict{
		"name":        f.JsonName(), // NOTICE: 显示为 json 标签名称，而非结构体名称
		"title":       f.Name,
		"type":        f.Type,
		"required":    f.IsRequired(),
		"description": f.SchemaDesc(),
	}
	// 以validate标签为准
	validatorLabelsMap := map[string]string{}
	validateTag := utils.QueryFieldTag(f.Tag, DefaultValidateTagName, "")

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
		m[ValidatorLabelToOpenapiLabel[isdefault]] = v
	}

	// 为不同的字段类型生成相应的描述
	switch f.Type {
	case IntegerType: // 生成数字类型的最大最小值
		for _, label := range numberTypeValidatorLabels {
			if v, ok := validatorLabelsMap[label]; ok {
				if label == validatorEnumLabel { // 生成字段的枚举值
					m[ValidatorLabelToOpenapiLabel[label]] = utils.StringsToInts(strings.Split(v, " "))
				} else {
					m[ValidatorLabelToOpenapiLabel[label]] = v
				}
			}
		}
	case NumberType: // 生成数字类型的最大最小值
		for _, label := range numberTypeValidatorLabels {
			if v, ok := validatorLabelsMap[label]; ok {
				if label == validatorEnumLabel { // 生成字段的枚举值
					m[ValidatorLabelToOpenapiLabel[label]] = utils.StringsToFloats(strings.Split(v, " "))
				} else {
					m[ValidatorLabelToOpenapiLabel[label]] = v
				}
			}
		}

	case StringType: // 生成字符串类型的最大最小长度
		for _, label := range stringTypeValidatorLabels {
			if v, ok := validatorLabelsMap[label]; ok {
				if label == validatorEnumLabel {
					m[ValidatorLabelToOpenapiLabel[label]] = strings.Split(v, " ")
				} else {
					m[ValidatorLabelToOpenapiLabel[label]] = v
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
				m[ValidatorLabelToOpenapiLabel[label]] = v
			}
		}

	case ObjectType: // 简体
		if f.ItemRef != "" { // 字段类型为自定义结构体，生成关联类型，此内部结构体已注册
			m["$ref"] = RefPrefix + f.ItemRef
		}

	default:
	}

	return
}

func (f *BaseModelField) SchemaPkg() string { return f.Pkg }

// SchemaTitle swagger文档字段名
func (f *BaseModelField) SchemaTitle() string { return f.Name }

func (f *BaseModelField) JsonName() string { return utils.QueryJsonName(f.Tag, f.Name) }

// SchemaDesc 字段注释说明
func (f *BaseModelField) SchemaDesc() string { return f.Description }

// SchemaType 模型类型
func (f *BaseModelField) SchemaType() DataType { return f.Type }

// IsRequired 字段是否必须
func (f *BaseModelField) IsRequired() bool { return IsFieldRequired(f.Tag) }

// IsArray 字段是否是数组类型
func (f *BaseModelField) IsArray() bool { return f.Type == ArrayType }

// InnerSchema 内部字段模型文档
func (f *BaseModelField) InnerSchema() []SchemaIface {
	m := make([]SchemaIface, 0)
	return m
}
