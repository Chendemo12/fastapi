package openapi

type SchemaIface interface {
	// Schema 输出为OpenAPI文档模型,字典格式
	Schema() (m map[string]any)
	// SchemaName 获取结构体的名称,默认包含包名
	SchemaName(exclude ...bool) string
	// SchemaDesc 结构体文档注释
	SchemaDesc() string
	// SchemaType 模型类型
	SchemaType() DataType
	// IsRequired 字段是否必须
	IsRequired() bool
	// Metadata 获取反射后的字段元信息, 允许上层处理
	Metadata() (*Metadata, error)
}

type QueryParameter interface {
	//Fields() []*QModel
}
