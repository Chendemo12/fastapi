package godantic

import (
	"errors"
	"reflect"
	"strconv"
	"strings"
)

const ( // see validator.baked_in.go
	validatorEnumLabel = "oneof"
)

const ( // see validator.validator_instance.go
	defaultTagName        = "validate"
	utf8HexComma          = "0x2C"
	utf8Pipe              = "0x7C"
	tagSeparator          = ","
	orSeparator           = "|"
	tagKeySeparator       = "="
	structOnlyTag         = "structonly"
	noStructLevelTag      = "nostructlevel"
	omitempty             = "omitempty"
	isdefault             = "isdefault"
	requiredWithoutAllTag = "required_without_all"
	requiredWithoutTag    = "required_without"
	requiredWithTag       = "required_with"
	requiredWithAllTag    = "required_with_all"
	requiredIfTag         = "required_if"
	requiredUnlessTag     = "required_unless"
	excludedWithoutAllTag = "excluded_without_all"
	excludedWithoutTag    = "excluded_without"
	excludedWithTag       = "excluded_with"
	excludedWithAllTag    = "excluded_with_all"
	excludedIfTag         = "excluded_if"
	excludedUnlessTag     = "excluded_unless"
	skipValidationTag     = "-"
	diveTag               = "dive"
	keysTag               = "keys"
	endKeysTag            = "endkeys"
	requiredTag           = "required"
	namespaceSeparator    = "."
	leftBracket           = "["
	rightBracket          = "]"
	restrictedTagChars    = ".[],|=+()`~!@#$%^&*\\\"/?<>{}"
	restrictedAliasErr    = "Alias '%s' either contains restricted characters or is the same as a restricted tag needed for normal operation"
	restrictedTagErr      = "Tag '%s' either contains restricted characters or is the same as a restricted tag needed for normal operation"
)

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

type dict map[string]any

// Field 基本数据模型, 此模型不可再分, 同时也是 BaseModel 的字段类型
// 但此类型不再递归记录,仅记录一个关联模型为基本
type Field struct {
	_pkg        string            `description:"包名.结构体名"`
	Title       string            `json:"title,omitempty" description:"字段名称"`
	Type        OpenApiDataType   `json:"type,omitempty" description:"openapi 数据类型"`
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
			m["items"] = map[string]OpenApiDataType{"type": StringType}
		case string(BoolType):
			m["items"] = map[string]OpenApiDataType{"type": BoolType}
		case string(NumberType):
			m["items"] = map[string]OpenApiDataType{"type": NumberType}
		case string(IntegerType):
			m["items"] = map[string]OpenApiDataType{"type": IntegerType}
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
func (f *Field) SchemaType() OpenApiDataType { return f.Type }

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
func (b *BaseModel) SchemaType() OpenApiDataType { return ObjectType }

func (b *BaseModel) IsRequired() bool { return true }

// Metadata 获取反射后的字段元信息, 此字段应慎重使用
func (b *BaseModel) Metadata() (*Metadata, error) {
	meta := GetMetadata(b.ID())
	if meta != nil {
		return meta, nil
	}

	return nil, errors.New("struct is not a BaseModel")
}

// StringsToInts 将字符串数组转换成int数组, 简单实现
//
//	@param	strs	[]string	输入字符串数组
//	@return	[]int 输出int数组
func StringsToInts(strs []string) []int {
	ints := make([]int, 0)

	for _, s := range strs {
		i, err := strconv.Atoi(s)
		if err != nil {
			continue
		}
		ints = append(ints, i)
	}

	return ints
}

// StringsToFloats 将字符串数组转换成float64数组, 简单实现
//
//	@param	strs	[]string	输入字符串数组
//	@return	[]float64 输出float64数组
func StringsToFloats(strs []string) []float64 {
	floats := make([]float64, len(strs))

	for _, s := range strs {
		i, err := strconv.ParseFloat(s, 10)
		if err != nil {
			continue
		}
		floats = append(floats, i)
	}

	return floats
}
