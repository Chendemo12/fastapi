package openapi

import "reflect"

const ApiVersion = "3.1.0"

var (
	ValidateTagName    = "validate"
	QueryTagName       = "query"
	JsonTagName        = "json"
	DescriptionTagName = "description"
	DefaultValueTagNam = "default"
	ParamRequiredLabel = requiredTag
)

type DataType string

func (m DataType) IsBaseType() bool {
	switch m {
	case IntegerType, NumberType, BoolType, StringType:
		return true
	default:
		return false
	}
}

const (
	IntegerType DataType = "integer"
	NumberType  DataType = "number"
	StringType  DataType = "string"
	BoolType    DataType = "boolean"
	ObjectType  DataType = "object"
	ArrayType   DataType = "array"
)

const (
	IntMaximum   int64 = Int64Maximum
	IntMinimum   int64 = Int64Minimum
	Int8Maximum  int64 = 127
	Int8Minimum  int64 = -128
	Int16Maximum int64 = 32767
	Int16Minimum int64 = -32768
	Int32Maximum int64 = 2147483647
	Int32Minimum int64 = -2147483648
	Int64Maximum int64 = 9223372036854775807
	Int64Minimum int64 = -9223372036854775808

	UintMaximum   uint64 = Uint64Maximum
	UintMinimum   uint64 = Uint64Minimum
	Uint8Maximum  uint64 = 255
	Uint8Minimum  uint64 = 0
	Uint16Maximum uint64 = 65535
	Uint16Minimum uint64 = 0
	Uint32Maximum uint64 = 4294967295
	Uint32Minimum uint64 = 0
	Uint64Maximum uint64 = 9223372036854775809
	Uint64Minimum uint64 = 0
)

// RouteMethodSeparator 路由分隔符，用于分割路由方法和路径
const RouteMethodSeparator = "=|_0#0_|="

// 用于swagger的一些静态文件，来自FastApi
const (
	SwaggerCssName    = "swagger-ui.css"
	FaviconName       = "favicon.png"
	FaviconIcoName    = "favicon.ico"
	SwaggerJsName     = "swagger-ui-bundle.js"
	RedocJsName       = "redoc.standalone.js"
	JsonUrl           = "openapi.json"
	DocumentUrl       = "/docs"
	ReDocumentUrl     = "/redoc"
	SwaggerFaviconUrl = "https://fastapi.tiangolo.com/img/" + FaviconName
	SwaggerCssUrl     = "https://cdn.jsdelivr.net/npm/swagger-ui-dist@5/" + SwaggerCssName
	SwaggerJsUrl      = "https://cdn.jsdelivr.net/npm/swagger-ui-dist@5/" + SwaggerJsName
	RedocJsUrl        = "https://cdn.jsdelivr.net/npm/redoc@next/bundles/" + RedocJsName
)

const (
	RefName              = "$ref"
	RefPrefix            = "#/components/schemas/"
	ArrayTypePrefix      = "ArrayOf" // 对于数组类型，关联到一个新模型
	InnerModelNamePrefix = "fastapi."
)

// AnonymousModelNameConnector 为匿名结构体生成一个名称, 连接符
const AnonymousModelNameConnector = "_"

const ReminderWhenResponseModelIsNil = " `| 路由未明确定义返回值，文档处缺省为map类型，实际可以是任意类型`"

const TimePkg = "time.Time"
const ( // 针对时间类型的查询参数格式化选项
	TimeParamSchemaFormat     = "time"
	DateParamSchemaFormat     = "date"
	DateTimeParamSchemaFormat = "date-time"
)

// InnerModelsName 特殊的内部模型名称
var InnerModelsName = []string{
	"BaseModel",
	"BaseModelField",
	"BaseRouter",
}

var InnerModelsPkg = []string{
	"fastapi.BaseModel",
	"fastapi.BaseModelField",
	"fastapi.BaseRouter",
	"time.Location",
	"time.zone",
	"time.zoneTrans",
}

// IllegalRouteParamType 不支持的参数类型
var IllegalRouteParamType = []reflect.Kind{
	reflect.Invalid,
	reflect.Interface,
	reflect.Func,
	reflect.Chan,
	reflect.UnsafePointer,
	reflect.Map, // TODO Future-231126.7: 查询参数值不允许为map
}

const HeaderContentType = "Content-Type"

const (
	MIMETextXML                    string = "text/xml"
	MIMETextHTML                   string = "text/html"
	MIMETextPlain                  string = "text/plain"
	MIMETextCSS                    string = "text/css"
	MIMETextJavaScript             string = "text/javascript"
	MIMEApplicationXML             string = "application/xml"
	MIMEApplicationJSON            string = "application/json"
	MIMEApplicationForm            string = "application/x-www-form-urlencoded"
	MIMEOctetStream                string = "application/octet-stream"
	MIMEMultipartForm              string = "multipart/form-data"
	MIMETextXMLCharsetUTF8         string = "text/xml; charset=utf-8"
	MIMETextHTMLCharsetUTF8        string = "text/html; charset=utf-8"
	MIMETextPlainCharsetUTF8       string = "text/plain; charset=utf-8"
	MIMETextCSSCharsetUTF8         string = "text/css; charset=utf-8"
	MIMETextJavaScriptCharsetUTF8  string = "text/javascript; charset=utf-8"
	MIMEApplicationXMLCharsetUTF8  string = "application/xml; charset=utf-8"
	MIMEApplicationJSONCharsetUTF8 string = "application/json; charset=utf-8"
)

const ( // see validator.validator_instance.go
	validatorEnumLabel    = "oneof" // see validator.baked_in.go
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

//goland:noinspection GoUnusedGlobalVariable
var validatorOperators = map[string]string{
	",": ",", // 多操作符分割
	"|": "|", // 或操作
	"-": "-", // 跳过字段验证
}

var numberTypeValidatorLabels = [...]string{"lt", "gt", "lte", "gte", "eq", "ne", validatorEnumLabel}
var arrayTypeValidatorLabels = [...]string{"min", "max", "len"}
var stringTypeValidatorLabels = [...]string{"min", "max", validatorEnumLabel}

// ValidatorLabelToOpenapiLabel validator 标签和 Openapi 标签的对应关系
var ValidatorLabelToOpenapiLabel = map[string]string{
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
