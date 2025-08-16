package openapi

import "strings"

// MultipartForm 包含文件上传的接口表单
type MultipartForm struct {
	scopePath    string         // 路由，辅助生成模型名称
	requestModel *BaseModelMeta `description:"同时存在请求体表单"`
}

func (f *MultipartForm) SchemaDesc() string { return "upload file" }

func (f *MultipartForm) SchemaType() DataType { return ObjectType }

func (f *MultipartForm) IsRequired() bool { return true }

func (f *MultipartForm) SchemaPkg() string {
	return "fastapi." + f.SchemaTitle()
}

func (f *MultipartForm) SchemaTitle() string {
	if f.requestModel != nil {
		return f.requestModel.SchemaTitle() + "_MultipartForm"
	}

	return strings.ReplaceAll(f.scopePath, "/", "_") + "_MultipartForm"
}

func (f *MultipartForm) JsonName() string { return f.SchemaTitle() }

func (f *MultipartForm) Schema() (m map[string]any) {
	m = dict{}
	m["title"] = f.SchemaTitle()
	m["type"] = ObjectType
	m["required"] = []string{"file"}
	properties := dict{
		"file": map[string]string{
			"title":  "File",
			"type":   string(StringType),
			"format": FileParamSchemaFormat,
		},
	}

	if f.requestModel != nil {
		m["required"] = []string{"file", f.requestModel.JsonName()}
		properties[f.requestModel.JsonName()] = RefPrefix + f.requestModel.SchemaPkg()
	}

	m["properties"] = properties
	return m
}

func (f *MultipartForm) InnerSchema() []SchemaIface {
	return nil
}
