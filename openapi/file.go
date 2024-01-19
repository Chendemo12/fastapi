package openapi

// UploadFileMeta 上传文件
type UploadFileMeta struct {
	requestModel *BaseModelMeta `description:"同时存在请求体表单"`
}

func (f *UploadFileMeta) ContentType() string { return MIMEMultipartForm }

func (f *UploadFileMeta) SchemaDesc() string { return "upload file" }

func (f *UploadFileMeta) SchemaType() DataType { return ObjectType }

func (f *UploadFileMeta) IsRequired() bool { return true }

func (f *UploadFileMeta) SchemaPkg() string { return "fastapi.UploadFileMeta" }

func (f *UploadFileMeta) SchemaTitle() string { return "UploadFileMeta" }

func (f *UploadFileMeta) JsonName() string { return "UploadFileMeta" }

func (f *UploadFileMeta) Schema() (m map[string]any) {
	m["title"] = f.SchemaTitle()
	m["type"] = f.ContentType()
	properties := dict{
		"file": map[string]string{
			"title":  "File",
			"type":   "string",
			"format": FileParamSchemaFormat,
		},
	}

	if f.requestModel != nil {
		m["required"] = []string{"file", f.requestModel.JsonName()}
		properties[f.requestModel.JsonName()] = RefPrefix + f.requestModel.SchemaPkg()
	} else {
		m["required"] = []string{"file"}
	}

	m["properties"] = properties
	return m
}

func (f *UploadFileMeta) InnerSchema() []SchemaIface {
	if f.requestModel != nil {
		return []SchemaIface{f.requestModel}
	}
	return nil
}
