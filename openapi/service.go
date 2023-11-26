package openapi

import (
	"net/http"
	"reflect"
)

// MakeOperationRequestBody 将路由中的 *openapi.BaseModelMeta 转换成 openapi 的请求体 RequestBody
func MakeOperationRequestBody(model *BaseModelMeta) *RequestBody {
	if model == nil {
		return &RequestBody{}
	}

	return &RequestBody{
		Required: model.IsRequired(),
		Content: &PathModelContent{
			MIMEType: MIMEApplicationJSON,
			Schema:   model,
		},
	}
}

// Deprecated: MakeOperationResponses 将路由中的 *openapi.BaseModelMeta 转换成 openapi 的返回体 []*Response
func MakeOperationResponses(model *BaseModelMeta) []*Response {
	if model == nil { // 若返回值为空，则设置为字符串
		model = &BaseModelMeta{}
	}

	m := make([]*Response, 2) // 200 + 422
	// 200 接口处注册的返回值
	m[0] = &Response{
		StatusCode:  http.StatusOK,
		Description: http.StatusText(http.StatusOK),
		Content: &PathModelContent{
			MIMEType: MIMEApplicationJSON,
			Schema:   model,
		},
	}
	// 422 所有接口默认携带的请求体校验错误返回值
	m[1] = Resp422

	return m
}

// Deprecated:
func QModelToParameter(model *QModel) *Parameter {
	p := &Parameter{
		ParameterBase: ParameterBase{
			Name:        model.SchemaPkg(),
			Description: model.SchemaDesc(),
			In:          InQuery,
			Required:    model.IsRequired(),
			Deprecated:  false,
		},
		Schema: &ParameterSchema{
			Type:  model.SchemaType(),
			Title: model.SchemaTitle(),
		},
		Default: GetDefaultV(model.Tag, model.SchemaType()),
	}

	if model.InPath {
		p.In = InPath
	}

	return p
}

func getModelNames(fieldMeta *BaseModelField, fieldType reflect.Type) (string, string) {
	var pkg, name string
	if isAnonymousStruct(fieldType) {
		// 未命名的结构体类型, 没有名称, 分配包名和名称
		name = fieldMeta.Name + "Model"
		//pkg = fieldMeta.Pkg + AnonymousModelNameConnector + name
		pkg = fieldMeta.Pkg
	} else {
		pkg = fieldType.String() // 关联模型
		name = fieldType.Name()
	}

	return pkg, name
}
