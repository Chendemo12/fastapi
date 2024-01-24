package fastapi

import (
	"errors"
	"fmt"
	"github.com/Chendemo12/fastapi/openapi"
	"github.com/Chendemo12/fastapi/pathschema"
	"github.com/Chendemo12/fastapi/utils"
	"net/http"
	"path"
	"reflect"
	"strings"
	"time"
	"unicode"
)

// GroupRouteHandler 路由组路由函数签名，其中any可以是具体的类型，但不应该是 Response
type GroupRouteHandler func(c *Context, params ...any) (any, error)

// GroupRouter 结构体路由组定义
// 用法：首先实现此接口，然后通过调用 Wrapper.IncludeRoute 方法进行注册绑定
type GroupRouter interface {
	// Prefix 路由组前缀，无需考虑是否以/开头或结尾
	// 如果为空则通过 PathSchema 方案进行格式化
	Prefix() string
	// Tags 标签，如果为空则设为结构体名称的大驼峰形式，去掉可能存在的http方法名
	Tags() []string
	// PathSchema 路由解析规则，对路由前缀和路由地址都有效
	PathSchema() pathschema.RoutePathSchema
	// Summary 允许对单个方法路由的文档摘要信息进行定义
	// 方法名:摘要信息
	Summary() map[string]string
	// Description 方法名:描述信息
	Description() map[string]string
	// Path 允许对方法的路由进行重载, 方法名:相对路由
	// 由于以函数名确定方法路由的方式暂无法支持路径参数, 因此可通过此方式来定义路径参数
	// 但是此处定义的路由不应该包含查询参数
	// 路径参数以:开头, 查询参数以?开头
	Path() map[string]string

	// InParamsName 允许对函数入参名称进行修改，仅适用于基本类型和time.Time类型的参数
	// 由于go在编译后无法获得函数或方法的入参名称，只能获得入参的类型和偏移量，
	// 因此在openapi的文档生成中，作为查询参数的函数入参无法正确显示出查询参数名称，取而代之的是手动分配的一个虚假参数名，此名称会影响api的调用和查询参数的解析
	// 对于此情况，推荐使用结构体来定义查询参数，以获得更好的使用体验
	// 此外，对于入参较少的情况，允许通过手动的方式来分配一个名称。
	//
	//
	//	对于方法：ManyGet(c *Context, age int, name string, graduate bool, source float64)
	//
	//	在未手动指定名称的情况下, 查询参数解析为：
	//		age int => int_2
	//		name string => string_3
	//		graduate bool => bool_4
	//		source float64 => float64_5
	//
	//	通过一下方式来手动指定名称：
	//		{
	//			"ManyGet": {
	//				2: "age",
	//				3: "name",
	//				4: "graduate",
	//				5: "source",
	//			},
	//		}
	InParamsName() map[string]map[int]string
}

// BaseGroupRouter (面向对象式)路由组基类
// 需实现 GroupRouter 接口
//
// 其中以 Get,Post,Delete,Patch,Put 字符串(不区分大小写)开头或结尾并以 (XXX, error)形式为返回值的方法会被作为路由处理
// 其url的确定按照 RoutePathSchema 接口进行解析。
//
// 对于作为路由的方法签名有如下要求：
//
//	1：参数：
//
//		第一个参数必须为 *Context
//		如果有多个参数, 除第一个参数和最后一个参数允许为结构体外, 其他参数必须为基本数据类型
//		对于Get/Delete：除第一个参数外的其他参数均被作为查询参数处理，如果为一个结构体，则对结构体字段进行解析并确定是否必选，如果为基本类型则全部为可选参数;
//		对于Post/Patch/Put: 其最后一个参数必须为一个 struct指针，此参数会作为请求体进行处理，其他参数则=全部为可选的查询参数
//
//	2：返回值
//
//		有且仅有2个返回值 (XXX, error)
//		其中XXX会作为响应体模型，若error!=nil则返回错误; 如果返回值XXX=nil则无响应体
//
//	对于上述参数和返回值XXX，其数据类型不能是 接口，函数，通道，指针的指针;
//	只能是以下类型：~int, ~float, ~string, ~slice, ~struct, ~map, 结构体指针;
//	对于结构体类型，最好实现了 SchemaIface 接口
type BaseGroupRouter struct {
	// 基类实现不能包含任何路由方法
}

func (g *BaseGroupRouter) Prefix() string { return "" }

func (g *BaseGroupRouter) Tags() []string { return []string{} }

func (g *BaseGroupRouter) PathSchema() pathschema.RoutePathSchema {
	return pathschema.Default()
}

func (g *BaseGroupRouter) Path() map[string]string {
	return map[string]string{}
}

func (g *BaseGroupRouter) Summary() map[string]string {
	return map[string]string{}
}

func (g *BaseGroupRouter) Description() map[string]string {
	return map[string]string{}
}

func (g *BaseGroupRouter) InParamsName() map[string]map[int]string {
	return map[string]map[int]string{}
}

// =================================== 👇 路由组元数据 ===================================

const WebsocketMethod = "WS"
const HttpMethodMinimumLength = len(http.MethodGet)
const (
	FirstInParamOffset       = 1                      // 第一个有效参数的索引位置，由于结构体接收器处于第一位置
	FirstCustomInParamOffset = FirstInParamOffset + 1 // 第一个自定义参数的索引位置
	FirstOutParamOffset      = 0
	LastOutParamOffset       = 1 // 最后一个返回值参数的索引位置
	OutParamNum              = 2
)

const (
	FirstInParamName = "Context" // 第一个入参名称
	LastOutParamName = "error"   // 最后一个出参名称
)

var HttpMethods = []string{
	http.MethodGet,
	http.MethodPost,
	http.MethodPatch,
	http.MethodPut,
	http.MethodDelete,
	http.MethodOptions,
}

// IllegalResponseType 非法的返回值类型, 不支持指针的指针
var IllegalResponseType = append(openapi.IllegalRouteParamType, reflect.Ptr)

// IllegalLastInParamType 非法的请求体类型, 不支持指针的指针
var IllegalLastInParamType = append(openapi.IllegalRouteParamType, reflect.Ptr)

// GroupRouterMeta 反射构建路由组的元信息
type GroupRouterMeta struct {
	router      GroupRouter
	routerValue reflect.Value
	pkg         string `description:"结构体.包名"`
	routes      []*GroupRoute
	tags        []string
}

// NewGroupRouteMeta 构建一个路由组的主入口
func NewGroupRouteMeta(router GroupRouter) *GroupRouterMeta {
	r := &GroupRouterMeta{router: router}
	return r
}

func (r *GroupRouterMeta) Init() (err error) {
	err = r.Scan()
	if err != nil {
		return err
	}

	// 处理内部路由的文档等数据
	err = r.ScanInner()
	return
}

func (r *GroupRouterMeta) String() string { return r.pkg }

func (r *GroupRouterMeta) Scan() (err error) {
	r.routerValue = reflect.ValueOf(r.router)
	obj := reflect.TypeOf(r.router)

	// 路由组必须是结构体实现
	if obj.Kind() != reflect.Struct && obj.Kind() != reflect.Pointer {
		return fmt.Errorf("router: '%s' not a struct", obj.String())
	}

	// 记录包名
	if obj.Kind() == reflect.Ptr {
		r.pkg = obj.Elem().String()
	} else {
		r.pkg = obj.String()
	}

	r.routes = make([]*GroupRoute, 0)

	// 扫描tags
	r.scanTags()

	if err != nil {
		return err
	}
	// 扫描方法路由
	err = r.scanMethod()

	return
}

// ScanInner 处理内部路由 GroupRoute 的文档等数据
func (r *GroupRouterMeta) ScanInner() (err error) {
	for _, route := range r.routes {
		err = route.Init()
		if err != nil {
			return err
		}
	}

	return
}

func (r *GroupRouterMeta) Routes() []*GroupRoute { return r.routes }

// 扫描tags, 由于接口方法允许留空，此处需处理默认值
func (r *GroupRouterMeta) scanTags() {
	obj := reflect.TypeOf(r.router)
	if obj.Kind() == reflect.Pointer {
		obj = obj.Elem()
	}

	tags := r.router.Tags()
	if len(tags) == 0 {
		tags = append(tags, obj.Name())
	}
	r.tags = tags
}

func (r *GroupRouterMeta) scanPath(swagger *openapi.RouteSwagger, method reflect.Method) string {
	dv := pathschema.Format(r.router.Prefix(), swagger.RelativePath, r.router.PathSchema())

	if len(r.router.Path()) > 0 {
		// 此方式可存在路径参数
		v, ok := r.router.Path()[method.Name]
		if ok {
			dv = path.Join(r.router.Prefix(), v)
		}
	}

	return dv
}

func (r *GroupRouterMeta) scanSummary(swagger *openapi.RouteSwagger, method reflect.Method) string {
	dv := fmt.Sprintf("%s %s", swagger.Method, swagger.RelativePath)
	if len(r.router.Summary()) > 0 {
		v, ok := r.router.Summary()[method.Name]
		if ok {
			dv = v
		}
	}

	return dv
}

func (r *GroupRouterMeta) scanDescription(swagger *openapi.RouteSwagger, method reflect.Method) string {
	dv := r.scanSummary(swagger, method)
	if len(r.router.Description()) > 0 {
		v, ok := r.router.Description()[method.Name]
		if ok {
			dv = v
		}
	}

	return dv
}

// 获得自定义查询参数名
func (r *GroupRouterMeta) scanQueryName(method reflect.Method, param *openapi.RouteParam) string {
	methodName := method.Name

	if len(r.router.InParamsName()) > 0 {
		m, ok := r.router.InParamsName()[methodName]
		if ok {
			v, okk := m[param.Index]
			if okk {
				return v
			}
		}
	}

	return param.QueryName
}

// 反射方法
func (r *GroupRouterMeta) scanMethod() (err error) {
	obj := reflect.TypeOf(r.router) // 由于必须是指针接收器，因此obj应为指针类型
	for i := 0; i < obj.NumMethod(); i++ {
		method := obj.Method(i)
		swagger, isRoute := r.isRouteMethod(method)
		if !isRoute {
			continue
		}
		// 匹配到路由方法
		swagger.Url = r.scanPath(swagger, method)
		swagger.Summary = r.scanSummary(swagger, method)
		swagger.Description = r.scanDescription(swagger, method)
		swagger.Tags = append(r.tags)

		r.routes = append(r.routes, NewGroupRoute(swagger, method, r))
	}

	return nil
}

// 判断一个方法是不是路由对象
func (r *GroupRouterMeta) isRouteMethod(method reflect.Method) (*openapi.RouteSwagger, bool) {
	if len(method.Name) <= HttpMethodMinimumLength {
		// 长度不够
		return nil, false
	}

	if unicode.IsLower([]rune(method.Name)[0]) {
		// 非导出方法
		return nil, false
	}

	swagger := &openapi.RouteSwagger{}
	methodNameLength := len(method.Name)

	// 依次判断是哪一种方法
	for _, hm := range HttpMethods {
		offset := len(hm)
		if methodNameLength <= offset {
			continue // 长度不匹配
		}
		if strings.ToUpper(method.Name[:offset]) == hm {
			// 记录方法和路由
			swagger.Method = hm
			swagger.RelativePath = method.Name[offset:] // 方法在前，截取后半部分为路由
			break
		}

		if strings.ToUpper(method.Name[methodNameLength-offset:]) == hm {
			swagger.Method = hm
			swagger.RelativePath = method.Name[:methodNameLength-offset]
			break
		}
	}
	if swagger.Method == "" {
		// 方法名称不符合
		return nil, false
	}

	// 判断方法参数是否符合要求
	inParamNum := method.Type.NumIn()
	outParamNum := method.Type.NumOut()

	if inParamNum < FirstInParamOffset || outParamNum != OutParamNum {
		// 方法参数数量不对
		return nil, false
	}

	// 获取请求参数
	if method.Type.In(FirstInParamOffset).Elem().Name() != FirstInParamName || method.Type.Out(LastOutParamOffset).Name() != LastOutParamName {
		// 方法参数类型不符合
		return nil, false
	}
	// 如果有多个入参, 判断最后一个入参是否符合要求
	if inParamNum > FirstInParamOffset {
		lastInParam := method.Type.In(inParamNum - FirstInParamOffset)
		if lastInParam.Kind() == reflect.Pointer {
			// 通常情况是个结构体指针，此时获取实际的类型
			lastInParam = lastInParam.Elem()
		}
		for _, k := range IllegalLastInParamType {
			if lastInParam.Kind() == k {
				// 返回值的第一个参数不符合要求
				return nil, false
			}
		}
	}

	// 判断第一个返回值参数类型是否符合要求,返回值不允许为nil
	firstOutParam := method.Type.Out(FirstOutParamOffset)
	if firstOutParam.Kind() == reflect.Pointer {
		// 通常情况下会返回指针，此时获取实际的类型
		firstOutParam = firstOutParam.Elem()
	}
	firstOutParamKind := firstOutParam.Kind()
	for _, k := range IllegalResponseType {
		if firstOutParamKind == k {
			// 返回值的第一个参数不符合要求
			return nil, false
		}
	}

	// 全部符合要求
	return swagger, true
}

// GroupRoute 路由组路由定义
type GroupRoute struct {
	swagger        *openapi.RouteSwagger
	group          *GroupRouterMeta
	requestBinder  *ParamBinder          // 请求题校验器，不存在请求题则为 NothingBindMethod
	responseBinder *ParamBinder          // 响应体校验器，响应体肯定存在 ModelBindMethod
	outParams      *openapi.RouteParam   // 不包含最后一个 error, 因此只有一个出参
	queryParamMode QueryParamMode        // 查询参数的定义模式
	method         reflect.Method        // 路由方法所属的结构体方法, 用于API调用
	queryBinders   []*ParamBinder        // 查询参数，路径参数的校验器，不存在参数则为 NothingBindMethod
	inParams       []*openapi.RouteParam // 不包含第一个 Context 但包含最后一个“查询参数结构体”或“请求体”, 因此 handlerInNum - len(inParams) = 1
	index          int                   // 当前方法所属的结构体方法的偏移量
	structQuery    int                   // 结构体查询参数在 inParams 中的索引
	handlerInNum   int                   // 路由函数入参数量, 入参数量可以不固定,但第一个必须是 Context，如果>1:则最后一个视为请求体(Post/Patch/Post)或查询参数(Get/Delete)
	handlerOutNum  int                   // 路由函数出参数量, 出参数量始终为2,最后一个必须是 error
	getOrDelete    bool                  // GET 或 DELETE 方法
}

func (r *GroupRoute) Id() string { return r.swagger.Id() }

func NewGroupRoute(swagger *openapi.RouteSwagger, method reflect.Method, group *GroupRouterMeta) *GroupRoute {
	r := &GroupRoute{}
	r.method = method
	r.swagger = swagger
	r.group = group
	r.index = method.Index
	r.structQuery = -1 // 不存在

	r.queryBinders = make([]*ParamBinder, 0)

	return r
}

func (r *GroupRoute) Init() (err error) {
	r.handlerInNum = r.method.Type.NumIn() - FirstInParamOffset // 排除接收器
	r.handlerOutNum = OutParamNum                               // 返回值数量始终为2
	r.getOrDelete = utils.Has[string]([]string{http.MethodGet, http.MethodDelete}, r.swagger.Method)

	r.outParams = openapi.NewRouteParam(r.method.Type.Out(FirstOutParamOffset), FirstOutParamOffset, openapi.RouteParamResponse)
	for n := FirstCustomInParamOffset; n <= r.handlerInNum; n++ {
		r.inParams = append(r.inParams, openapi.NewRouteParam(r.method.Type.In(n), n, openapi.RouteParamQuery))
	}

	err = r.Scan()

	return
}

func (r *GroupRoute) Scan() (err error) {
	// 首先初始化参数
	for _, in := range r.inParams {
		err = in.Init()
		if err != nil {
			return err
		}
	}
	// 由于以下几个scan方法续需读取内部的反射数据, swagger 层面无法读取,因此在此层面进行解析

	links := []func() error{
		r.outParams.Init, // 解析响应体
		r.scanInParamsBefore,
		r.scanInParams, // 初始化模型文档
		r.scanOutParams,
		r.scanQueryParamMode,
		r.ScanInner, // 递归进入下层进行解析
		r.scanBinders,
	}

	for _, link := range links {
		err = link()
		if err != nil {
			return err
		}
	}

	return
}

// ScanInner 解析内部 openapi.RouteSwagger 数据
func (r *GroupRoute) ScanInner() (err error) {
	err = r.swagger.Init()
	return
}

func (r *GroupRoute) scanInParamsBefore() (err error) {
	// TODO: Future-231203.9: 限制POST/PATCH/PUT方法最多支持2个结构体参数
	for _, param := range r.inParams {
		switch param.SchemaType() {
		case openapi.ArrayType:
		case openapi.ObjectType:
			if param.IsTime {
				param.QueryName = r.group.scanQueryName(r.method, param)
			}

		default:
			param.QueryName = r.group.scanQueryName(r.method, param)
		}
	}
	return
}

// 从方法入参中初始化路由参数, 包含了查询参数，请求体参数
func (r *GroupRoute) scanInParams() (err error) {
	r.swagger.QueryFields = make([]*openapi.QModel, 0)
	if r.handlerInNum == FirstInParamOffset { // 只有一个参数,只能是 Context
		return nil
	}

	for index, param := range r.inParams {
		isLast := index == r.handlerInNum-FirstCustomInParamOffset
		switch param.SchemaType() {

		case openapi.ArrayType:
			if isLast && !r.getOrDelete { // // 最后一个参数, 是否可以断言为请求体
				r.swagger.RequestModel = openapi.NewBaseModelMeta(param)
			} else {
				// 方法不支持断言为请求体, 查询参数不支持数组
				return errors.New(fmt.Sprintf(
					"method: '%s' param: '%s', index: %d, query param not support array",
					r.group.pkg+"."+r.method.Name, param.Pkg, param.Index,
				))
			}

		case openapi.ObjectType:
			// 判断是否是时间类型, 时间类型全部解释为查询参数
			qm, ok := scanHelper.InferTimeParam(param)
			if ok {
				r.swagger.QueryFields = append(r.swagger.QueryFields, qm)
			} else {
				if !isLast { // 不是最后一个参数
					if r.getOrDelete {
						// GET/DELETE方法不支持多个结构体参数, 打印出结构体方法名，参数索引出从1开始, 排除接收器参数，直接取Index即可
						return errors.New(fmt.Sprintf(
							"method: '%s' param: '%s', index: %d cannot be a %s",
							r.group.pkg+"."+r.method.Name, param.Pkg, param.Index, param.SchemaType(),
						))
					} else {
						// POST/PATCH/PUT 方法，识别为结构体查询参数
						r.structQuery = index
						r.swagger.QueryFields = append(r.swagger.QueryFields, openapi.StructToQModels(param.CopyPrototype())...)
					}
				} else {
					// 最后一个参数, 对于GET/DELETE 视为查询参数, 结构体的每一个字段都将作为一个查询参数;
					// 对于 POST/PATCH/PUT 接口,如果是结构体或数组则作为请求体
					if r.getOrDelete {
						r.structQuery = index
						qms := scanHelper.InferObjectQueryParam(param)
						r.swagger.QueryFields = append(r.swagger.QueryFields, qms...)
					} else {
						r.swagger.RequestModel = openapi.NewBaseModelMeta(param)
					}
				}
			}

		default:
			// NOTICE: 此处无法获得方法的参数名，只能获得参数类型的名称
			r.swagger.QueryFields = append(r.swagger.QueryFields, scanHelper.InferBaseQueryParam(param, r.RouteType()))
		}
	}
	return nil
}

// 从方法出参中初始化路由响应体
func (r *GroupRoute) scanOutParams() (err error) {
	// RouteSwagger.Init -> ResponseModel.Init() 时会自行处理
	r.swagger.ResponseModel = openapi.NewBaseModelMeta(r.outParams)
	return err
}

// 此方法需在 scanInParams 执行完成之后执行
func (r *GroupRoute) scanQueryParamMode() (err error) {
	if r.handlerInNum > FirstInParamOffset { // 存在自定义参数
		r.queryParamMode = NoQueryParamMode
		return
	}

	var end int
	if utils.Has[string]([]string{http.MethodGet, http.MethodDelete}, r.swagger.Method) {
		end = len(r.inParams) // 掐头
	} else {
		end = len(r.inParams) - 1 // 掐头去尾，最后一个为请求体
	}

	var hasBase = len(r.inParams[:end]) > 1 // 仅能存在一个 struct 查询参数
	var hasStruct bool

	for _, param := range r.inParams[:end] {
		if param.SchemaType() == openapi.ObjectType {
			hasStruct = true
		} else {
			hasBase = true
		}
	}

	if hasBase && hasStruct {
		r.queryParamMode = MixQueryParamMode
	} else {
		if hasStruct {
			r.queryParamMode = StructQueryParamMode
		}
		if hasBase {
			r.queryParamMode = SimpleQueryParamMode
		}
	}

	return
}

// 此方法需在 scanInParams, scanOutParams, scanQueryParamMode 执行完成之后执行
func (r *GroupRoute) scanBinders() (err error) {
	r.responseBinder = scanHelper.InferResponseBinder(r.swagger.ResponseModel, r.RouteType())

	// 初始化请求体验证方法
	r.requestBinder = scanHelper.InferRequestBinder(r.swagger.RequestModel, r.RouteType())

	// 构建查询参数验证器
	for _, qmodel := range r.swagger.QueryFields {
		binder := scanHelper.InferQueryBinder(qmodel, r.RouteType())
		r.queryBinders = append(r.queryBinders, binder)
	}
	return
}

func (r *GroupRoute) RouteType() RouteType { return RouteTypeGroup }

func (r *GroupRoute) Swagger() *openapi.RouteSwagger {
	return r.swagger
}

func (r *GroupRoute) ResponseBinder() *ParamBinder { return r.responseBinder }

func (r *GroupRoute) RequestBinders() *ParamBinder { return r.requestBinder }

// QueryBinders 查询参数校验方法
func (r *GroupRoute) QueryBinders() []*ParamBinder { return r.queryBinders }

func (r *GroupRoute) HasStructQuery() bool { return r.structQuery != -1 }

// NewStructQuery 构造一个新的结构体查询参数实例
func (r *GroupRoute) NewStructQuery() any {
	var v reflect.Value
	if r.inParams[r.structQuery].IsPtr {
		v = reflect.New(r.inParams[r.structQuery].Prototype.Elem())
	} else {
		v = reflect.New(r.inParams[r.structQuery].Prototype)
	}

	return v.Interface()
}

func (r *GroupRoute) NewInParams(ctx *Context) []reflect.Value {
	params := make([]reflect.Value, len(r.inParams)+2) // 接收器 + *Context
	params[0] = r.group.routerValue                    // 接收器
	params[1] = reflect.ValueOf(ctx)

	// 处理入参
	for i, param := range r.inParams {
		var instance reflect.Value
		isLast := i == len(r.inParams)-1 // 是否是最后一个参数

		switch param.SchemaType() {

		case openapi.ArrayType: // 只能是请求体
			instance = reflect.ValueOf(ctx.requestModel)

		case openapi.ObjectType: // 查询参数或请求体
			// time.Time 类型只能是查询参数
			if param.IsTime {
				v := ctx.queryFields[param.QueryName] // 参数是必选的, 此时肯定存在,且已经做好了类型转换
				tt := v.(time.Time)
				instance = reflect.ValueOf(tt)
			} else {
				if isLast && !r.getOrDelete { // 最后一个参数, 可以断言为请求体
					instance = reflect.ValueOf(ctx.requestModel)
				} else {
					// 匹配到结构体查询参数
					instance = reflect.ValueOf(ctx.queryStruct)
				}
			}

		default: // 对于基本参数,只能是查询参数
			instance = param.NewNotStruct(ctx.queryFields[param.QueryName])
		}

		if param.IsPtr || param.IsTime {
			params[i+FirstCustomInParamOffset] = instance
		} else {
			params[i+FirstCustomInParamOffset] = instance.Elem()
		}
	}

	return params
}

func (r *GroupRoute) NewRequestModel() any {
	if r.swagger.Method == http.MethodGet || r.swagger.Method == http.MethodDelete {
		return nil
	}
	if r.swagger.RequestModel == nil {
		return nil
	}

	return r.swagger.RequestModel.Param.NewNotStruct(nil).Interface()
}

// Call 调用API, 并将响应结果写入 Response 内
func (r *GroupRoute) Call(ctx *Context) {
	params := r.NewInParams(ctx)
	result := r.method.Func.Call(params)
	// 是否存在错误
	last := result[LastOutParamOffset]
	if !last.IsValid() || last.IsNil() {
		// err=nil, 函数没有返回错误
		ctx.response.Content = result[FirstOutParamOffset].Interface()
	} else {
		err := last.Interface().(error)
		if ctx.response.StatusCode == 0 {
			// 允许上层自定义错误码
			ctx.response.StatusCode = DefaultErrorStatusCode
		}
		ctx.response.Content = err.Error()
	}
}

// ResponseValidate 仅校验“200的JSONResponse”
func (r *GroupRoute) ResponseValidate(c *Context, stopImmediately bool) []*openapi.ValidationError {
	if (c.response.StatusCode == http.StatusOK || c.response.StatusCode == 0) && c.response.Type == JsonResponseType {
		// 内部返回的 422 也不再校验
		var ves []*openapi.ValidationError
		// TODO: 此校验浪费性能, 尝试通过某种方式绕过
		_, ves = r.ResponseBinder().Method.Validate(c.routeCtx, c.response.Content)
		if len(ves) > 0 {
			ves[0].Ctx[modelDescLabel] = r.Swagger().ResponseModel.SchemaDesc()
		}
		return ves
	}
	return nil
}
