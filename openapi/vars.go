package openapi

import (
	"github.com/Chendemo12/fastapi/utils"
	"net/http"
)

const (
	ValidationErrorName     string = "ValidationError"
	HttpValidationErrorName string = "HTTPValidationError"
)

// ValidationErrorDefinition 422 表单验证错误模型
var ValidationErrorDefinition = &ValidationError{}

// ValidationErrorResponseDefinition 请求体相应体错误消息
var ValidationErrorResponseDefinition = &HTTPValidationError{}

var RouteErrorOption = &RouteErrorOpt{
	StatusCode:   http.StatusInternalServerError,
	ResponseMode: nil,
	Description:  http.StatusText(http.StatusInternalServerError),
}

type dict map[string]any

// RouteErrorOpt 错误处理函数选项
type RouteErrorOpt struct {
	StatusCode   int            `json:"statusCode" validate:"required" description:"请求错误时的状态码"`
	ResponseMode *BaseModelMeta `json:"responseMode" validate:"required" description:"请求错误时的响应体，空则为字符串"`
	Description  string         `json:"description,omitempty" description:"错误文档"`
}

// ValidationError 参数校验错误
type ValidationError struct {
	Ctx  map[string]any `json:"service,omitempty" description:"附加消息"`
	Type string         `json:"type" binding:"required" description:"参数类型"`
	Msg  string         `json:"msg" binding:"required" description:"错误消息"`
	Loc  []string       `json:"loc" binding:"required" description:"参数定位"`
}

func (v *ValidationError) SchemaDesc() string { return "参数校验错误" }

func (v *ValidationError) SchemaType() DataType { return ObjectType }

func (v *ValidationError) SchemaTitle() string { return ValidationErrorName }

func (v *ValidationError) SchemaPkg() string { return InnerModelNamePrefix + ValidationErrorName }

func (v *ValidationError) JsonName() string { return v.SchemaTitle() }

func (v *ValidationError) Schema() (m map[string]any) {
	return dict{
		"title": ValidationErrorName,
		"type":  ObjectType,
		"properties": dict{
			"loc": dict{
				"title": "Location",
				"type":  "array",
				"items": dict{"anyOf": []map[string]string{{"type": "string"}, {"type": "integer"}}},
			},
			"msg":  dict{"title": "Message", "type": "string"},
			"type": dict{"title": "Error Type", "type": "string"},
		},
		requiredTag: []string{"loc", "msg", "type"},
	}
}

// InnerSchema 内部字段模型文档
func (v *ValidationError) InnerSchema() []SchemaIface {
	m := make([]SchemaIface, 0)
	return m
}

func (v *ValidationError) Error() string {
	bytes, err := utils.JsonMarshal(v)
	if err != nil {
		return v.SchemaDesc()
	}
	return string(bytes)
}

type HTTPValidationError struct {
	Detail []*ValidationError `json:"detail" description:"Detail" binding:"required"`
}

func (v *HTTPValidationError) SchemaPkg() string {
	return InnerModelNamePrefix + HttpValidationErrorName
}

func (v *HTTPValidationError) SchemaTitle() string { return HttpValidationErrorName }

func (v *HTTPValidationError) JsonName() string { return v.SchemaTitle() }

func (v *HTTPValidationError) SchemaType() DataType { return ObjectType }

func (v *HTTPValidationError) SchemaDesc() string { return "路由参数校验错误" }

func (v *HTTPValidationError) Schema() map[string]any {
	ve := ValidationError{}
	return dict{
		"title":    HttpValidationErrorName,
		"type":     ObjectType,
		"required": []string{"detail"},
		"properties": dict{
			"detail": dict{
				"title": "Detail",
				"type":  "array",
				"items": dict{"$ref": RefPrefix + ve.SchemaPkg()},
			},
		},
	}
}

// InnerSchema 内部字段模型文档
func (v *HTTPValidationError) InnerSchema() []SchemaIface {
	m := make([]SchemaIface, 0)
	return m
}

func (v *HTTPValidationError) Error() string {
	if len(v.Detail) > 0 {
		return v.Detail[0].Error()
	}
	return "HttpValidationErrorName: "
}

func (v *HTTPValidationError) String() string { return v.Error() }
