package fastapi

import "github.com/Chendemo12/fastapi/openapi"

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
	Type() RouteType
	Swagger() *openapi.RouteSwagger           // 路由文档
	ResponseBinder() ModelBindMethod          // 响应体的处理接口,响应体只有一个
	RequestBinders() ModelBindMethod          // 请求体的处理接口,请求体也只有一个
	QueryBinders() map[string]ModelBindMethod // 查询参数的处理接口(查询参数名:处理接口)，查询参数可有多个
	NewRequestModel() any                     // TODO: 创建一个新的参数实例
	Call()                                    // 调用API
}
