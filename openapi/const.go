package openapi

const ApiVersion = "3.1.0"

type DataType string

const (
	IntegerType DataType = "integer"
	NumberType  DataType = "number"
	StringType  DataType = "string"
	BoolType    DataType = "boolean"
	ObjectType  DataType = "object"
	ArrayType   DataType = "array"
)

const (
	ValidationErrorName     string = "ValidationError"
	HttpValidationErrorName string = "HTTPValidationError"
)

// 用于swagger的一些静态文件，来自FastApi
const (
	SwaggerCssName    = "swagger-ui.css"
	FaviconName       = "favicon.png"
	FaviconIcoName    = "favicon.ico"
	SwaggerJsName     = "swagger-ui-bundle.js"
	RedocJsName       = "redoc.standalone.js"
	JsonUrl           = "openapi.json"
	SwaggerFaviconUrl = "https://fastapi.tiangolo.com/img/" + FaviconName
	SwaggerCssUrl     = "https://cdn.jsdelivr.net/npm/swagger-ui-dist@5/" + SwaggerCssName
	SwaggerJsUrl      = "https://cdn.jsdelivr.net/npm/swagger-ui-dist@5/" + SwaggerJsName
	RedocJsUrl        = "https://cdn.jsdelivr.net/npm/redoc@next/bundles/" + RedocJsName
)

const (
	PathParamPrefix         = ":" // 路径参数起始字符
	PathSeparator           = "/" // 路径分隔符
	OptionalPathParamSuffix = "?" // 可选路径参数结束字符
)

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

const (
	RefName              = "$ref"
	RefPrefix            = "#/components/schemas/"
	ArrayTypePrefix      = "ArrayOf" // 对于数组类型，关联到一个新模型
	InnerModelNamePrefix = "fastapi."
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
