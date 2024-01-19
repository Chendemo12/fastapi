package openapi

// ModelSchema 数据模型定义
// 作为路由请求体或响应体的数据模型都必须实现此方法
type ModelSchema interface {
	SchemaDesc() string   // 模型文档注释
	SchemaType() DataType // 模型类型
	IsRequired() bool     // 模型是否必须
}

// SchemaIface 定义数据模型
// 对于泛型接口响应体模型必须实现此接口
// 对于接口体方法接口则通过反射判断是否实现了此接口
type SchemaIface interface {
	ModelSchema
	SchemaPkg() string          // 包名.模型名
	SchemaTitle() string        // 获取模型名不包含包名, 如果是结构体字段，则为字段原始名称
	JsonName() string           // 通常情况下与 SchemaTitle 相同，如果是结构体字段，则为字段json标签的值
	Schema() (m map[string]any) // 输出为OpenAPI文档模型,字典格式
	InnerSchema() []SchemaIface // 适用于数组类型，以及结构体字段仍为结构体的类型
}
