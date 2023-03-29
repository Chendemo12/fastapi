package godantic

const (
	RefName              = "$ref"
	RefPrefix            = "#/components/schemas/"
	ArrayTypePrefix      = "ArrayOf" // 对于数组类型，关联到一个新模型
	InnerModelNamePrefix = "godantic."
)

const (
	ValidationErrorName     string = "ValidationError"
	HttpValidationErrorName string = "HTTPValidationError"
)

type OpenApiDataType string

const (
	IntegerType OpenApiDataType = "integer"
	NumberType  OpenApiDataType = "number"
	StringType  OpenApiDataType = "string"
	BoolType    OpenApiDataType = "boolean"
	ObjectType  OpenApiDataType = "object"
	ArrayType   OpenApiDataType = "array"
)
