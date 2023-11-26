package openapi

import (
	"github.com/Chendemo12/fastapi-tool/helper"
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

var Resp422 = &Response{
	StatusCode:  http.StatusUnprocessableEntity,
	Description: http.StatusText(http.StatusUnprocessableEntity),
	Content: &PathModelContent{
		MIMEType: MIMEApplicationJSON,
		Schema:   &ValidationError{},
	},
}

type dict map[string]any

// ValidationError 参数校验错误
type ValidationError struct {
	ModelSchema
	Ctx  map[string]any `json:"service" description:"Service"`
	Msg  string         `json:"msg" description:"Message" binding:"required"`
	Type string         `json:"type" description:"Error Type" binding:"required"`
	Loc  []string       `json:"loc" description:"Location" binding:"required"`
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
	bytes, err := helper.JsonMarshal(v)
	if err != nil {
		return v.SchemaDesc()
	}
	return string(bytes)
}

type HTTPValidationError struct {
	ModelSchema
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
