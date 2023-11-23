package fastapi

import (
	"github.com/Chendemo12/fastapi/openapi"
	"reflect"
)

// RouteSeparator 路由分隔符，用于分割路由方法和路径
const RouteSeparator = "|_0#0_|"

const ReminderWhenResponseModelIsNil = " `| 路由未明确定义返回值，文档处缺省为string类型，实际可以是任意类型`"

// RouteIface 路由定义
type RouteIface interface {
	Swagger() *RouteSwagger                   // 路由文档
	ResponseBinder() ModelBindMethod          // 响应体的处理接口,响应体只有一个
	RequestBinders() ModelBindMethod          // 请求体的处理接口,请求体也只有一个
	QueryBinders() map[string]ModelBindMethod // 查询参数的处理接口(查询参数名:处理接口)，查询参数可有多个
	NewRequestModel() any                     //
	Call()                                    // 调用API
}

// RouteSwagger 路由文档定义，所有类型的路由均包含此部分
type RouteSwagger struct {
	Url           string            `json:"url" description:"完整请求路由"`
	RelativePath  string            `json:"relative_path" description:"相对路由"`
	Method        string            `json:"method" description:"请求方法"`
	Summary       string            `json:"summary" description:"摘要描述"`
	Description   string            `json:"description" description:"详细描述"`
	Tags          []string          `json:"tags" description:"路由标签"`
	RequestModel  *openapi.Metadata `description:"请求体元数据"`
	ResponseModel *openapi.Metadata `description:"响应体元数据"`
	QueryFields   []*openapi.QModel `json:"-" description:"查询参数"`
	PathFields    []*openapi.QModel `json:"-" description:"路径参数"`
	Deprecated    bool              `json:"deprecated" description:"是否禁用"`
}

func (r *RouteSwagger) HandlerOutNum() int { return 2 }

// GenericRoute 泛型路由定义
type GenericRoute[T openapi.SchemaIface] struct {
	swagger   *RouteSwagger
	prototype T
}

// GroupRoute 路由组路由定义
type GroupRoute struct {
	swagger       *RouteSwagger
	handlerInNum  int            // 路由函数入参数量
	handlerOutNum int            // 路由函数出参数量
	inParams      []reflect.Type // 不包含第一个 Context
	outParams     reflect.Type   // 不包含最后一个 error
}

func (r *GroupRoute) Swagger() *RouteSwagger {
	return r.swagger
}

func (r *GroupRoute) ResponseBinder() ModelBindMethod {
	//TODO implement me
	panic("implement me")
}

func (r *GroupRoute) RequestBinders() ModelBindMethod {
	//TODO implement me
	panic("implement me")
}

func (r *GroupRoute) QueryBinders() map[string]ModelBindMethod {
	//TODO implement me
	panic("implement me")
}

func (r *GroupRoute) NewRequestModel() any {
	//TODO implement me
	panic("implement me")
}

func (r *GroupRoute) Call() {
	//TODO implement me
	panic("implement me")
}

func (r *GroupRoute) initSwagger() *GroupRoute {
	// 反射构建数据模型文档
	return r
}
