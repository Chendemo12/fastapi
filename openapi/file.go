package openapi

import "strings"

var (
	MultipartFormFileName  = "file"  // 表单数据中文件的键名称
	MultipartFormParamName = "param" // 表单数据中json参数的键名称
)

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
	properties := dict{}

	m["title"] = f.SchemaTitle()
	m["type"] = ObjectType
	m["required"] = []string{MultipartFormFileName}

	// 文件部分
	properties[MultipartFormFileName] = map[string]string{
		"title":  strings.ToTitle(MultipartFormFileName),
		"type":   string(StringType),
		"format": FileParamSchemaFormat,
	}

	// 表单部分
	if f.requestModel != nil {
		//m["required"] = []string{"file", f.requestModel.JsonName()}
		//properties[f.requestModel.JsonName()] = dict{RefName: RefPrefix + f.requestModel.SchemaPkg()}
		// 默认给个参数名
		m["required"] = []string{MultipartFormFileName, MultipartFormParamName}
		properties[MultipartFormParamName] = dict{RefName: RefPrefix + f.requestModel.SchemaPkg()}
	}

	m["properties"] = properties
	return m
}

func (f *MultipartForm) InnerSchema() []SchemaIface {
	return nil
}
