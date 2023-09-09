package fastapi

import (
	"github.com/Chendemo12/fastapi/godantic"
	"github.com/Chendemo12/fastapi/internal/constant"
	"github.com/gofiber/fiber/v2"
	"net/http"
	"reflect"
	"runtime"
	"strings"
)

// RouteSeparator 路由分隔符，用于分割路由方法和路径
const RouteSeparator = "|_0#0_|"
const WebsocketMethod = "WS"
const ReminderWhenResponseModelIsNil = " `| 路由未明确定义返回值，文档处缺省为string类型，实际可以是任意类型`"

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
	Summary       string                  `json:"summary" description:"摘要描述"`
	ResponseModel godantic.SchemaIface    `json:"response_model" description:"响应体模型"`
	RequestModel  godantic.SchemaIface    `json:"request_model" description:"请求体模型"`
	Params        godantic.QueryParameter `json:"params" description:"查询参数,结构体"`
	Description   string                  `json:"description" description:"路由描述"`
	Tags          []string                `json:"tags" description:"路由标签"`
	Dependencies  []DependencyFunc        `json:"-" description:"依赖"`
	Handlers      []HandlerFunc           `json:"-" description:"处理函数"`
	Deprecated    bool                    `json:"deprecated" description:"是否禁用"`
}

// Route 一个完整的路由对象，此对象会在程序启动时生成swagger文档
// 其中相对路径Path不能重复，否则后者会覆盖前者
type Route struct {
	ResponseModel    *godantic.Metadata            `description:"响应体元数据"`
	RequestModel     *godantic.Metadata            `description:"请求体元数据"`
	requestValidate  RouteModelValidateHandlerFunc `description:"请求体校验函数"`
	responseValidate RouteModelValidateHandlerFunc `description:"返回值校验函数"`
	Description      string                        `json:"description" description:"详细描述"`
	Summary          string                        `json:"summary" description:"摘要描述"`
	Method           string                        `json:"method" description:"请求方法"`
	RelativePath     string                        `json:"relative_path" description:"相对路由"`
	Tags             []string                      `json:"tags" description:"路由标签"`
	QueryFields      []*godantic.QModel            `json:"-" description:"查询参数"`
	Handlers         []fiber.Handler               `json:"-" description:"处理函数"`
	Dependencies     []DependencyFunc              `json:"-" description:"依赖"`
	PathFields       []*godantic.QModel            `json:"-" description:"路径参数"`
	deprecated       bool                          `description:"是否禁用"`
}

func (f *Route) LowerMethod() string { return strings.ToLower(f.Method) }

// Deprecate 禁用路由
func (f *Route) Deprecate() *Route {
	f.deprecated = true
	return f
}

// AddDependency 添加依赖项，用于在执行路由函数前执行一个自定义操作，此操作作用于参数校验通过之后
//
//	@param	fcs	DependencyFunc	依赖项
func (f *Route) AddDependency(fcs ...DependencyFunc) *Route {
	if len(fcs) > 0 {
		f.Dependencies = append(f.Dependencies, fcs...)
	}
	return f
}

// AddD 添加依赖项，用于在执行路由函数前执行一个自定义操作，此操作作用于参数校验通过之后
//
//	@param	fcs	DependencyFunc	依赖项
func (f *Route) AddD(fcs ...DependencyFunc) *Route { return f.AddDependency(fcs...) }

// SetDescription 设置一个路由的详细描述信息
//
//	@param	Description	string	详细描述信息
func (f *Route) SetDescription(description string) *Route {
	f.Description = description
	return f
}

// SetD 设置一个路由的详细描述信息
//
//	@param	Description	string	详细描述信息
func (f *Route) SetD(description string) *Route { return f.SetDescription(description) }

// SetQueryParams 设置查询参数,此空struct的每一个字段都将作为一个单独的查询参数
// 且此结构体的任意字段有且仅支持 string 类型
//
//	@param	m	godantic.QueryParameter	查询参数对象,
func (f *Route) SetQueryParams(m godantic.QueryParameter) *Route {
	if m != nil {
		f.QueryFields = godantic.ParseToQueryModels(m) // 转换为内部模型
	}
	return f
}

// SetQ 设置查询参数,此空struct的每一个字段都将作为一个单独的查询参数
// 且此结构体的任意字段有且仅支持 string 类型
//
//	@param	m	godantic.QueryParameter	查询参数对象,
func (f *Route) SetQ(m godantic.QueryParameter) *Route { return f.SetQueryParams(m) }

// SetRequestModel 设置请求体对象,此model应为一个空struct实例,而非指针类型,且仅"GET",http.MethodDelete有效
//
//	@param	m	any	请求体对象
func (f *Route) SetRequestModel(m godantic.SchemaIface) *Route {
	if f.Method != http.MethodGet && f.Method != http.MethodDelete {
		f.RequestModel = godantic.BaseModelToMetadata(m)
	}
	return f
}

// SetReq 设置请求体对象
//
//	@param	m	any	请求体对象
func (f *Route) SetReq(m godantic.SchemaIface) *Route { return f.SetRequestModel(m) }

// Path 合并路由
//
//	@param	prefix	string	路由组前缀
func (f *Route) Path(prefix string) string { return CombinePath(prefix, f.RelativePath) }

// NewRequestModel 创建一个新的请求体模型
func (f *Route) NewRequestModel() any {
	if f.ResponseModel == nil {
		return nil
	}

	switch f.RequestModel.SchemaType() {
	case godantic.StringType:
		return ""
	case godantic.BoolType:
		return false
	case godantic.IntegerType:
		return 0
	case godantic.NumberType:
		return 0.0
	case godantic.ArrayType:
		// TODO: support array types
		return make([]string, 0)
	default:
		return reflect.New(f.RequestModel.ReflectType())
	}
}

// Router 一个独立的路由组，Prefix路由组前缀，其内部的子路由均包含此前缀
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
	queryModel godantic.QueryParameter, // 查询参数, POST/PATCH/PUT
	requestModel godantic.SchemaIface, // 请求体, POST/PATCH/PUT
	responseModel godantic.SchemaIface, // 响应体, All
	handler HandlerFunc, // handler
) *Route {
	route := &Route{
		Method:        method,
		RelativePath:  relativePath,
		PathFields:    make([]*godantic.QModel, 0), // 路径参数
		QueryFields:   make([]*godantic.QModel, 0), // 查询参数
		RequestModel:  nil,                         // 请求体
		ResponseModel: nil,                         // 响应体
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
		route.RequestModel = godantic.BaseModelToMetadata(requestModel)
		// TODO: 请求体校验方法
	} else {
		// 缺省以屏蔽请求体校验
		route.requestValidate = routeModelDoNothing
	}

	if responseModel != nil {
		route.ResponseModel = godantic.BaseModelToMetadata(responseModel)

		switch route.ResponseModel.SchemaType() {

		case godantic.StringType:
			route.responseValidate = stringResponseValidation
		case godantic.BoolType:
			route.responseValidate = boolResponseValidation
		case godantic.NumberType:
			route.responseValidate = numberResponseValidation
		case godantic.IntegerType:
			route.responseValidate = integerResponseValidation
		case godantic.ArrayType:
			route.responseValidate = arrayResponseValidation
		case godantic.ObjectType:
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
		route.QueryFields = append(route.QueryFields, godantic.ParseToQueryModels(queryModel)...)
	}
	// 若缺省返回值则在接口处追加描述
	if responseModel == nil {
		route.Description = route.Description + ReminderWhenResponseModelIsNil
	}

	// 生成路径参数
	if pp, found := DoesPathParamsFound(route.RelativePath); found {
		for name, required := range pp {
			qm := &godantic.QModel{
				Title:  name,
				Name:   name,
				Tag:    reflect.StructTag(`json:"` + name + `,omitempty"`),
				Type:   godantic.StringType,
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
		route.SetDescription(opt.Description)
	}
	if opt.Deprecated {
		route.Deprecate()
	}
	if len(opt.Dependencies) > 0 {
		route.AddDependency(opt.Dependencies...)
	}
	for _, _handler := range opt.Handlers {
		route.Handlers = append(route.Handlers, routeHandler(_handler))
	}

	return route
}

// GET http get method
//
//	@param	path			string					相对路径,必须以"/"开头
//	@param	summary			string					路由摘要信息
//	@param	queryModel		godantic.QueryParameter	查询参数，仅支持struct类型
//	@param	responseModel	godantic.SchemaIface	响应体对象,	此model应为一个空struct实例,而非指针类型
//	@param	handler			[]HandlerFunc			路由处理方法
//	@param	addition		any						附加参数，如："deprecated"用于禁用此路由
func (f *Router) GET(
	path string,
	responseModel godantic.SchemaIface,
	summary string,
	handler HandlerFunc,
) *Route {
	// 对于查询参数仅允许struct类型
	return f.method(
		http.MethodGet,
		path,
		summary,
		nil,
		nil,
		responseModel,
		handler,
	)
}

// DELETE http delete method
//
//	@param	path			string					相对路径,必须以"/"开头
//	@param	summary			string					路由摘要信息
//	@param	responseModel	godantic.SchemaIface	响应体对象,	此model应为一个空struct实例,而非指针类型
//	@param	handler			[]HandlerFunc			路由处理方法
//	@param	addition		any						附加参数
func (f *Router) DELETE(
	path string, responseModel godantic.SchemaIface, summary string, handler HandlerFunc,
) *Route {
	// 对于查询参数仅允许struct类型
	return f.method(
		http.MethodDelete,
		path,
		summary,
		nil,
		nil,
		responseModel,
		handler,
	)
}

// POST http post method
//
//	@param	path			string					相对路径,必须以"/"开头
//	@param	summary			string					路由摘要信息
//	@param	requestModel	godantic.SchemaIface	请求体对象,	此model应为一个空struct实例,而非指针类型
//	@param	responseModel	godantic.SchemaIface	响应体对象,	此model应为一个空struct实例,而非指针类型
//	@param	handler			[]HandlerFunc			路由处理方法
//	@param	addition		any						附加参数，如："deprecated"用于禁用此路由
func (f *Router) POST(
	path string,
	requestModel, responseModel godantic.SchemaIface,
	summary string,
	handler HandlerFunc,
) *Route {
	return f.method(
		http.MethodPost,
		path,
		summary,
		nil,
		requestModel,
		responseModel,
		handler,
	)
}

// PATCH http patch method
func (f *Router) PATCH(
	path string,
	requestModel, responseModel godantic.SchemaIface,
	summary string,
	handler HandlerFunc,
) *Route {
	return f.method(
		http.MethodPatch,
		path,
		summary,
		nil,
		requestModel,
		responseModel,
		handler,
	)
}

// PUT http put method
func (f *Router) PUT(
	path string,
	requestModel, responseModel godantic.SchemaIface,
	summary string,
	handler HandlerFunc,
) *Route {
	return f.method(
		http.MethodPut,
		path,
		summary,
		nil,
		requestModel,
		responseModel,
		handler,
	)
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
