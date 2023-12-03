package fastapi

import (
	"github.com/Chendemo12/fastapi/openapi"
	"reflect"
)

const ReminderWhenResponseModelIsNil = " `| 路由未明确定义返回值，文档处缺省为map类型，实际可以是任意类型`"

type RouteType string

const (
	GroupRouteType   RouteType = "GroupRoute"
	GenericRouteType RouteType = "GenericRoute"
)

// RouteIface 路由定义
// 路由组接口定义或泛型接口定义都需实现此接口
type RouteIface interface {
	Scanner
	RouteType() RouteType
	Swagger() *openapi.RouteSwagger           // 路由文档
	ResponseBinder() ModelBindMethod          // 响应体的处理接口,响应体只有一个
	RequestBinders() ModelBindMethod          // 请求体的处理接口,请求体也只有一个
	QueryBinders() map[string]ModelBindMethod // 查询参数的处理接口(查询参数名:处理接口)，查询参数可有多个
	NewRequestModel() reflect.Value           // TODO: 创建一个新的参数实例
	Call()                                    // 调用API
	Id() string
}

// BaseModel 基本数据模型, 对于上层的路由定义其请求体和响应体都应为继承此结构体的结构体
// 在 OpenApi 文档模型中,此模型的类型始终为 "object"
// 对于 BaseModel 其字段仍然可能会是 BaseModel
type BaseModel struct{}

// SchemaDesc 结构体文档注释
func (b *BaseModel) SchemaDesc() string { return openapi.InnerModelsName[0] }

// SchemaType 模型类型
func (b *BaseModel) SchemaType() openapi.DataType { return openapi.ObjectType }

func (b *BaseModel) IsRequired() bool { return true }
