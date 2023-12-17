package fastapi

import (
	"github.com/Chendemo12/fastapi/openapi"
	"github.com/Chendemo12/fastapi/pathschema"
	"reflect"
)

type RouteType string

const (
	RouteTypeGroup   RouteType = "GroupRoute"
	RouteTypeGeneric RouteType = "GenericRoute"
)

// RouteIface 路由定义
// 路由组接口定义或泛型接口定义都需实现此接口
type RouteIface interface {
	Scanner
	RouteType() RouteType
	Swagger() *openapi.RouteSwagger           // 路由文档
	QueryBinders() []*ParamBinder             // 查询参数的处理接口(查询参数名:处理接口)，每一个查询参数都必须绑定一个 ParamBinder
	RequestBinders() *ParamBinder             // 请求体的处理接口,请求体也只有一个
	ResponseBinder() *ParamBinder             // 响应体的处理接口,响应体只有一个
	NewInParams(ctx *Context) []reflect.Value // 创建一个完整的函数入参实例列表, 此方法会在完成请求参数校验 RequestBinders，QueryBinders 之后执行
	NewStructQuery() any                      // 创建一个结构体查询参数实例,对于POST/PATCH/PUT, 即为 NewInParams 的最后一个元素; 对于GET/DELETE则为nil
	NewRequestModel() any                     // 创建一个请求体实例,对于POST/PATCH/PUT, 即为 NewInParams 的最后一个元素; 对于GET/DELETE则为nil
	HasStructQuery() bool                     // 是否存在结构体查询参数，如果存在则会调用 NewStructQuery 获得结构体实例
	Call(ctx *Context)                        // 调用API, 需要将响应结果写入 Response 内
	Id() string
}

// ParamBinder 参数验证模型
type ParamBinder struct {
	Method         ModelBindMethod        `json:"-"`
	QModel         *openapi.QModel        `json:"-"`
	RequestModel   *openapi.BaseModelMeta `json:"-"`
	ResponseModel  *openapi.BaseModelMeta `json:"-"`
	Title          string                 `json:"title,omitempty"`
	RouteParamType openapi.RouteParamType
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

// ================================================================================

// InferBinderMethod 利用反射推断参数的校验器
func InferBinderMethod(param openapi.SchemaIface, prototypeKind reflect.Kind, modelType openapi.RouteParamType) ModelBindMethod {
	if param == nil {
		return &NothingBindMethod{}
	}

	var binder ModelBindMethod
	switch prototypeKind {

	case reflect.Int, reflect.Int64:
		binder = &IntBindMethod{
			Title:   param.SchemaTitle(),
			Kind:    prototypeKind,
			Maximum: openapi.IntMaximum,
			Minimum: openapi.IntMinimum,
		}
	case reflect.Int8:
		binder = &IntBindMethod{
			Title:   param.SchemaTitle(),
			Kind:    prototypeKind,
			Maximum: openapi.Int8Maximum,
			Minimum: openapi.Int8Minimum,
		}
	case reflect.Int16:
		binder = &IntBindMethod{
			Title:   param.SchemaTitle(),
			Kind:    prototypeKind,
			Maximum: openapi.Int16Maximum,
			Minimum: openapi.Int16Minimum,
		}
	case reflect.Int32:
		binder = &IntBindMethod{
			Title:   param.SchemaTitle(),
			Kind:    prototypeKind,
			Maximum: openapi.Int32Maximum,
			Minimum: openapi.Int32Minimum,
		}

	case reflect.Uint, reflect.Uint64:
		binder = &UintBindMethod{
			Title:   param.SchemaTitle(),
			Kind:    prototypeKind,
			Maximum: openapi.UintMaximum,
			Minimum: openapi.UintMinimum,
		}
	case reflect.Uint8:
		binder = &UintBindMethod{
			Title:   param.SchemaTitle(),
			Kind:    prototypeKind,
			Maximum: openapi.Uint8Maximum,
			Minimum: openapi.Uint8Minimum,
		}
	case reflect.Uint16:
		binder = &UintBindMethod{
			Title:   param.SchemaTitle(),
			Kind:    prototypeKind,
			Maximum: openapi.Uint16Maximum,
			Minimum: openapi.Uint16Minimum,
		}
	case reflect.Uint32:
		binder = &UintBindMethod{
			Title:   param.SchemaTitle(),
			Kind:    prototypeKind,
			Maximum: openapi.Uint32Maximum,
			Minimum: openapi.Uint32Minimum,
		}
	case reflect.Bool:
		binder = &BoolBindMethod{Title: param.SchemaTitle()}
	case reflect.Float32, reflect.Float64:
		binder = &FloatBindMethod{
			Title: param.SchemaTitle(),
			Kind:  prototypeKind,
		}
	case reflect.String:
		binder = &NothingBindMethod{}
	case reflect.Struct:
		if modelType == openapi.RouteParamResponse {
			binder = &JsonBindMethod[any]{Title: param.SchemaTitle(), ModelDesc: param.SchemaDesc(), RouteParamType: modelType}
		} else {
			binder = &JsonBindMethod[any]{Title: param.SchemaTitle(), ModelDesc: param.SchemaDesc(), RouteParamType: modelType}
		}
	default:
		binder = &NothingBindMethod{}
	}

	return binder
}

// ================================================================================

// NewBaseRouter 用于获取后端服务基本信息的路由组
//
//	# Usage
//
//	router := NewBaseRouter(Config{})
//	app.IncludeRouter(router)
func NewBaseRouter(conf Config) GroupRouter {
	return &BaseGroupRouter{
		Title:   conf.Title,
		Version: conf.Version,
		Desc:    conf.Description,
		Debug:   conf.Debug,
	}
}

type BaseGroupRouter struct {
	BaseRouter
	Title   string
	Version string
	Desc    string
	Debug   bool
}

func (r *BaseGroupRouter) Prefix() string {
	return "/api"
}

func (r *BaseGroupRouter) Tags() []string {
	return []string{"Base"}
}

func (r *BaseGroupRouter) PathSchema() pathschema.RoutePathSchema {
	return pathschema.Default()
}

func (r *BaseGroupRouter) Summary() map[string]string {
	return map[string]string{
		"GetTitle":       "获取软件名",
		"GetDescription": "获取软件描述信息",
		"GetVersion":     "获取软件版本号",
		"GetDebug":       "获取调试开关",
		"GetHeartbeat":   "心跳检测",
	}
}

func (r *BaseGroupRouter) Description() map[string]string {
	return map[string]string{}
}

func (r *BaseGroupRouter) GetTitle(c *Context) (string, error) {
	return r.Title, nil
}

func (r *BaseGroupRouter) GetDescription(c *Context) (string, error) {
	return r.Desc, nil
}

func (r *BaseGroupRouter) GetVersion(c *Context) (string, error) {
	return r.Version, nil
}

func (r *BaseGroupRouter) GetDebug(c *Context) (bool, error) {
	return r.Debug, nil
}

func (r *BaseGroupRouter) GetHeartbeat(c *Context) (string, error) {
	return "pong", nil
}
