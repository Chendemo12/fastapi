package fastapi

import (
	"github.com/Chendemo12/fastapi/openapi"
	"io"
)

// UploadFile 上传文件
// 通过声明此类型,以接收来自用户的上传文件
type UploadFile struct {
	requestModel *openapi.BaseModelMeta `description:"同时存在请求体表单"`
	Filename     string
	File         io.Reader
	Headers      map[string]string
}

func (f *UploadFile) ContentType() string { return openapi.MIMEMultipartForm }

func (f *UploadFile) SchemaDesc() string { return "upload file" }

func (f *UploadFile) SchemaType() openapi.DataType { return openapi.ObjectType }

func (f *UploadFile) IsRequired() bool { return true }

func (f *UploadFile) SchemaPkg() string { return "fastapi.UploadFile" }

func (f *UploadFile) SchemaTitle() string { return "UploadFile" }

func (f *UploadFile) JsonName() string { return "UploadFile" }

func (f *UploadFile) Schema() (m map[string]any) {
	m["title"] = f.SchemaTitle()
	m["type"] = f.ContentType()
	properties := Dict{
		"file": map[string]string{
			"title":  "File",
			"type":   "string",
			"format": openapi.FileParamSchemaFormat,
		},
	}

	if f.requestModel != nil {
		m["required"] = []string{"file", f.requestModel.JsonName()}
		properties[f.requestModel.JsonName()] = openapi.RefPrefix + f.requestModel.SchemaPkg()
	} else {
		m["required"] = []string{"file"}
	}

	m["properties"] = properties
	return m
}

func (f *UploadFile) InnerSchema() []openapi.SchemaIface {
	if f.requestModel != nil {
		return []openapi.SchemaIface{f.requestModel}
	}
	return nil
}
