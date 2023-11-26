package openapi

import (
	"reflect"
)

// ModelSchema 数据模型定义
// 作为路由请求体或响应体的数据模型都必须实现此方法
type ModelSchema interface {
	SchemaDesc() string   // 模型文档注释
	SchemaType() DataType // 模型类型
	IsRequired() bool     // 模型是否必须
}

// BaseModel 基本数据模型, 对于上层的路由定义其请求体和响应体都应为继承此结构体的结构体
// 在 OpenApi 文档模型中,此模型的类型始终为 "object"
// 对于 BaseModel 其字段仍然可能会是 BaseModel
type BaseModel struct{}

// SchemaDesc 结构体文档注释
func (b *BaseModel) SchemaDesc() string { return InnerModelsName[0] }

// SchemaType 模型类型
func (b *BaseModel) SchemaType() DataType { return ObjectType }

func (b *BaseModel) IsRequired() bool { return true }

// SchemaIface 定义数据模型
// 对于泛型接口响应体模型必须实现此接口
// 对于接口体方法接口则通过反射判断是否实现了此接口
type SchemaIface interface {
	ModelSchema
	SchemaPkg() string          // 包名.模型名
	SchemaTitle() string        // 获取模型名不包含包名, 如果是结构体字段，则为字段原始名称
	JsonName() string           // 通常情况下与 SchemaTitle 相同，如果是结构体字段，则为字段json标签的值
	Schema() (m map[string]any) // 输出为OpenAPI文档模型,字典格式
	// InnerSchema() map[string]map[string]any TODO Future:
}

const (
	SchemaDescMethodName string = "SchemaDesc"
	SchemaTypeMethodName string = "SchemaType"
)

func ReflectCallSchemaDesc(re reflect.Type) string {
	method, found := re.MethodByName(SchemaDescMethodName)
	if found {
		// 创建一个的实例
		newValue := reflect.New(re).Elem()
		result := method.Func.Call([]reflect.Value{newValue})
		desc := result[0].String()
		return desc
	} else {
		return ""
	}
}
