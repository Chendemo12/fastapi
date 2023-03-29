package godantic

type SchemaIface interface {
	// Schema 输出为OpenAPI文档模型,字典格式
	Schema() (m map[string]any)
	// SchemaName 获取结构体的名称,默认包含包名
	SchemaName(exclude ...bool) string
	// SchemaDesc 结构体文档注释
	SchemaDesc() string
	// SchemaType 模型类型
	SchemaType() OpenApiDataType
	// IsRequired 字段是否必须
	IsRequired() bool
	// Metadata 获取反射后的字段元信息, 允许上层处理
	Metadata() (*Metadata, error)
}

type DictIface interface {
	// Map 将结构体转换为字典视图
	Map() (m map[string]any)
	// Dict 将结构体转换为字典视图，并允许过滤一些字段或添加一些键值对到字典中
	Dict(exclude []string, include map[string]any) (m map[string]any)
	// Exclude 将结构体转换为字典视图，并过滤一些字段
	Exclude(exclude ...string) (m map[string]any)
	// Include 将结构体转换为字典视图，并允许添加一些键值对到字典中
	Include(include map[string]any) (m map[string]any)
}

type Iface interface {
	SchemaIface
	DictIface
	// Validate 检验实例是否符合tag要求
	Validate(stc any) []*ValidationError
	// ParseRaw 从原始字节流中解析结构体对象
	ParseRaw(stc []byte) []*ValidationError
	// Copy 拷贝一个新的空实例对象
	Copy() any
}

type QueryParameter interface {
	//Fields() []*QModel
}
