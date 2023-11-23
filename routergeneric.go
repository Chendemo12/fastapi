package fastapi

import (
	"github.com/Chendemo12/fastapi/internal/constant"
	"github.com/Chendemo12/fastapi/openapi"
	"github.com/gofiber/fiber/v2"
	"net/http"
	"reflect"
	"runtime"
	"strings"
)

// APIRouter 创建一个路由组
func APIRouter(prefix string, tags []string) *Router {
	fgr := &Router{
		Prefix:     prefix,
		Tags:       tags,
		deprecated: false,
	}
	fgr.routes = make(map[string]*Route, 0) // 初始化map,并保证为空
	return fgr
}

type Option struct {
	Summary       string                 `json:"summary" description:"摘要描述"`
	ResponseModel openapi.SchemaIface    `json:"response_model" description:"响应体模型"`
	RequestModel  openapi.SchemaIface    `json:"request_model" description:"请求体模型"`
	Params        openapi.QueryParameter `json:"params" description:"查询参数,结构体"`
	Description   string                 `json:"description" description:"路由描述"`
	Tags          []string               `json:"tags" description:"路由标签"`
	Dependencies  []DependencyFunc       `json:"-" description:"依赖"`
	Handlers      []HandlerFunc          `json:"-" description:"处理函数"`
	Deprecated    bool                   `json:"deprecated" description:"是否禁用"`
}

// Deprecated: Route 一个完整的路由对象，此对象会在程序启动时生成swagger文档
// 其中相对路径Path不能重复，否则后者会覆盖前者
// TODO: 重写为泛型实现
type Route struct {
	ResponseModel    *openapi.Metadata             `description:"响应体元数据"`
	RequestModel     *openapi.Metadata             `description:"请求体元数据"`
	requestValidate  RouteModelValidateHandlerFunc `description:"请求体校验函数"`
	responseValidate RouteModelValidateHandlerFunc `description:"返回值校验函数"`
	Description      string                        `json:"description" description:"详细描述"`
	Summary          string                        `json:"summary" description:"摘要描述"`
	Method           string                        `json:"method" description:"请求方法"`
	RelativePath     string                        `json:"relative_path" description:"相对路由"`
	Tags             []string                      `json:"tags" description:"路由标签"`
	QueryFields      []*openapi.QModel             `json:"-" description:"查询参数"`
	Handlers         []fiber.Handler               `json:"-" description:"处理函数"`
	Dependencies     []DependencyFunc              `json:"-" description:"依赖"`
	PathFields       []*openapi.QModel             `json:"-" description:"路径参数"`
	deprecated       bool                          `description:"是否禁用"`
	handleInNum      int
	handleOutNum     int
}

// Path 获得路由
//
//	@param	prefix	string	路由组前缀
func (f *Route) Path(prefix string) string { return CombinePath(prefix, f.RelativePath) }

// NewRequestModel 创建一个新的请求体模型
func (f *Route) NewRequestModel() any {
	if f.ResponseModel == nil {
		return nil
	}

	switch f.RequestModel.SchemaType() {
	case openapi.StringType:
		return ""
	case openapi.BoolType:
		return false
	case openapi.IntegerType:
		return 0
	case openapi.NumberType:
		return 0.0
	case openapi.ArrayType:
		// TODO: support array types
		return make([]string, 0)
	default:
		return reflect.New(f.RequestModel.ReflectType())
	}
}

// Deprecated:Router 一个独立的路由组，Prefix路由组前缀，其内部的子路由均包含此前缀
// TODO: 内部区分泛型接口和组有组接口 形如 Container
type Router struct {
	routes     map[string]*Route
	Prefix     string
	Tags       []string
	deprecated bool
}

// Routes 获取路由组内部定义的全部子路由信息
func (f *Router) Routes() map[string]*Route { return f.routes }

// Deprecate 禁用整个路由组路由
func (f *Router) Deprecate() *Router {
	f.deprecated = true
	return f
}

// Activate 激活整个路由组路由
func (f *Router) Activate() *Router {
	f.deprecated = false
	return f
}

// IncludeRouter 挂载一个子路由组,目前仅支持在子路由组初始化后添加
//
//	@param	router	*Router	子路由组
func (f *Router) IncludeRouter(router *Router) *Router {
	for _, route := range router.Routes() {
		route.RelativePath = CombinePath(router.Prefix, route.RelativePath)
		f.routes[route.RelativePath+RouteSeparator+route.Method] = route // 允许地址相同,方法不同的路由

	}

	return f
}

func (f *Router) method(
	method string, // 路由方法
	relativePath string, // 相对路由
	summary string, // 路由摘要
	queryModel openapi.QueryParameter, // 查询参数, POST/PATCH/PUT
	requestModel openapi.SchemaIface, // 请求体, POST/PATCH/PUT
	responseModel openapi.SchemaIface, // 响应体, All
	handler HandlerFunc, // handler
) *Route {
	route := &Route{
		Method:        method,
		RelativePath:  relativePath,
		PathFields:    make([]*openapi.QModel, 0), // 路径参数
		QueryFields:   make([]*openapi.QModel, 0), // 查询参数
		RequestModel:  nil,                        // 请求体
		ResponseModel: nil,                        // 响应体
		Summary:       summary,
		Handlers:      nil,
		Dependencies:  make([]DependencyFunc, 0),
		Tags:          f.Tags,
		Description:   method + " " + summary,
		deprecated:    false,
	}

	if route.Summary == "" {
		funcName := runtime.FuncForPC(reflect.ValueOf(handler).Pointer()).Name()
		parts := strings.Split(funcName, ".")
		funcName = parts[len(parts)-1]
		route.Summary = funcName
	}

	if requestModel != nil {
		route.RequestModel = openapi.BaseModelToMetadata(requestModel)
		// TODO: 请求体校验方法
	} else {
		// 缺省以屏蔽请求体校验
		route.requestValidate = routeModelDoNothing
	}

	if responseModel != nil {
		route.ResponseModel = openapi.BaseModelToMetadata(responseModel)

		switch route.ResponseModel.SchemaType() {

		case openapi.StringType:
			route.responseValidate = stringResponseValidation
		case openapi.BoolType:
			route.responseValidate = boolResponseValidation
		case openapi.NumberType:
			route.responseValidate = numberResponseValidation
		case openapi.IntegerType:
			route.responseValidate = integerResponseValidation
		case openapi.ArrayType:
			route.responseValidate = arrayResponseValidation
		case openapi.ObjectType:
			route.responseValidate = structResponseValidation
		}
	} else {
		// 对于返回值类型，允许缺省返回值以屏蔽返回值校验
		route.responseValidate = routeModelDoNothing
	}

	// 路由处理函数，默认仅一个
	handlers := []fiber.Handler{routeHandler(handler)}
	deprecated := false // 是否禁用此路由
	if f.deprecated {   // 若路由组被禁用，则此路由必禁用
		deprecated = true
	}

	route.deprecated = deprecated
	route.Handlers = handlers

	// 确保路径以/开头，若路由为空，则以路由组前缀为路由路径
	if len(relativePath) > 0 && !strings.HasPrefix(relativePath, constant.PathSeparator) {
		relativePath = constant.PathSeparator + relativePath
	}

	if queryModel != nil {
		route.QueryFields = append(route.QueryFields, openapi.ParseToQueryModels(queryModel)...)
	}
	// 若缺省返回值则在接口处追加描述
	if responseModel == nil {
		route.Description = route.Description + ReminderWhenResponseModelIsNil
	}

	// 生成路径参数
	if pp, found := DoesPathParamsFound(route.RelativePath); found {
		for name, required := range pp {
			qm := &openapi.QModel{
				Title:  name,
				Name:   name,
				Tag:    reflect.StructTag(`json:"` + name + `,omitempty"`),
				Type:   openapi.StringType,
				InPath: true,
			}
			if required {
				qm.Tag = reflect.StructTag(`json:"` + name + `" validate:"required" binding:"required"`)
			}
			route.PathFields = append(route.PathFields, qm)
		}
	}

	f.routes[relativePath+RouteSeparator+method] = route // 允许地址相同,方法不同的路由

	return route
}

func (f *Router) methodWithOpt(
	method string,
	relativePath string,
	handler HandlerFunc,
	opt *Option,
) *Route {
	route := f.method(
		method,
		relativePath,
		opt.Summary,
		opt.Params,
		opt.RequestModel,
		opt.ResponseModel,
		handler,
	)

	if opt.Description != "" {
		route.Description = opt.Description
	}
	if opt.Deprecated {
		route.deprecated = true
	}
	if len(opt.Dependencies) > 0 {
		route.Dependencies = append(route.Dependencies, opt.Dependencies...)
	}
	for _, _handler := range opt.Handlers {
		route.Handlers = append(route.Handlers, routeHandler(_handler))
	}

	return route
}

func (f *Router) Get(path string, handler HandlerFunc, opts ...Option) *Route {
	opt := cleanOpts(opts...)

	return f.methodWithOpt(http.MethodGet, path, handler, opt)
}

func (f *Router) Post(path string, handler HandlerFunc, opts ...Option) *Route {
	opt := cleanOpts(opts...)

	return f.methodWithOpt(http.MethodPost, path, handler, opt)
}

func (f *Router) Delete(path string, handler HandlerFunc, opts ...Option) *Route {
	opt := cleanOpts(opts...)

	return f.methodWithOpt(http.MethodDelete, path, handler, opt)
}

func (f *Router) Patch(path string, handler HandlerFunc, opts ...Option) *Route {
	opt := cleanOpts(opts...)

	return f.methodWithOpt(http.MethodPatch, path, handler, opt)
}

func (f *Router) Put(path string, handler HandlerFunc, opts ...Option) *Route {
	opt := cleanOpts(opts...)

	return f.methodWithOpt(http.MethodPut, path, handler, opt)
}

// CombinePath 合并路由
//
//	@param	prefix	string	路由前缀
//	@param	path	string	路由
func CombinePath(prefix, path string) string {
	if path == "" {
		return prefix
	}
	if !strings.HasPrefix(prefix, constant.PathSeparator) {
		prefix = constant.PathSeparator + prefix
	}

	if strings.HasSuffix(prefix, constant.PathSeparator) && strings.HasPrefix(path, constant.PathSeparator) {
		return prefix[:len(prefix)-1] + path
	}
	return prefix + path
}

// DoesPathParamsFound 是否查找到路径参数
//
//	@param	path	string	路由
func DoesPathParamsFound(path string) (map[string]bool, bool) {
	pathParameters := make(map[string]bool, 0)
	// 查找路径中的参数
	for _, p := range strings.Split(path, constant.PathSeparator) {
		if strings.HasPrefix(p, constant.PathParamPrefix) {
			// 识别到路径参数
			if strings.HasSuffix(p, constant.OptionalPathParamSuffix) {
				// 可选路径参数
				pathParameters[p[1:len(p)-1]] = false
			} else {
				pathParameters[p[1:]] = true
			}
		}
	}
	return pathParameters, len(pathParameters) > 0
}

func cleanOpts(opts ...Option) *Option {
	opt := &Option{
		Summary:       "",
		Params:        nil,
		RequestModel:  nil,
		ResponseModel: nil,
		Description:   "",
		Tags:          make([]string, 0),
		Dependencies:  make([]DependencyFunc, 0),
		Handlers:      make([]HandlerFunc, 0),
		Deprecated:    false,
	}
	if len(opts) > 0 {
		opt.Summary = opts[0].Summary
		opt.Params = opts[0].Params
		opt.RequestModel = opts[0].RequestModel
		opt.ResponseModel = opts[0].ResponseModel
		opt.Description = opts[0].Description
		opt.Deprecated = opts[0].Deprecated

		if len(opts[0].Tags) > 0 {
			opt.Tags = opts[0].Tags
		}
		if len(opts[0].Dependencies) > 0 {
			opt.Dependencies = opts[0].Dependencies
		}
		if len(opts[0].Handlers) > 0 {
			opt.Handlers = opts[0].Handlers
		}
	}

	return opt
}
