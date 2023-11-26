package fastapi

import (
	"errors"
	"fmt"
	"github.com/Chendemo12/fastapi/openapi"
	"github.com/Chendemo12/fastapi/pathschema"
	"github.com/Chendemo12/fastapi/utils"
	"net/http"
	"reflect"
	"strings"
	"unicode"
)

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

// GroupRouter 结构体路由组定义
// 用法：首先实现此接口，然后通过调用 FastApi.IncludeRoute 方法进行注册绑定
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
}

// BaseRouter (面向对象式)路由组基类
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
type BaseRouter struct {
	// 基类实现不能包含任何路由方法
}

func (g *BaseRouter) Prefix() string { return "" }

func (g *BaseRouter) Tags() []string { return []string{} }

func (g *BaseRouter) PathSchema() pathschema.RoutePathSchema {
	return pathschema.Default()
}

func (g *BaseRouter) Path() map[string]string {
	return map[string]string{}
}

func (g *BaseRouter) Summary() map[string]string {
	return map[string]string{}
}

func (g *BaseRouter) Description() map[string]string {
	return map[string]string{}
}

// GroupRoute 路由组路由定义
type GroupRoute struct {
	swagger *openapi.RouteSwagger
	method  reflect.Method // 路由方法所属的结构体方法, 用于API调用
	index   int            // 当前方法所属的结构体方法的偏移量
	// 路由函数入参数量, 入参数量可以不固定,但第一个必须是 Context
	// 如果>1:则最后一个视为请求体(Post/Patch/Post)或查询参数(Get/Delete)
	handlerInNum int
	// 路由函数出参数量, 出参数量始终为2,最后一个必须是 error
	handlerOutNum int
	inParams      []*openapi.RouteParam // 不包含第一个 Context, 因此 handlerInNum - len(inParams) = 1
	outParams     *openapi.RouteParam   // 不包含最后一个 error, 因此只有一个出参
}

func (r *GroupRoute) Id() string { return r.swagger.Id() }

func NewGroupRoute(swagger *openapi.RouteSwagger, method reflect.Method, group *GroupRouterMeta) *GroupRoute {
	r := &GroupRoute{}
	r.method = method
	r.swagger = swagger
	//r.group = group
	r.index = method.Index

	return r
}

func (r *GroupRoute) Init() (err error) {
	r.handlerInNum = r.method.Type.NumIn() - FirstInParamOffset // 排除接收器
	r.handlerOutNum = OutParamNum                               // 返回值数量始终为2

	r.outParams = openapi.NewRouteParam(r.method.Type.Out(FirstOutParamOffset), FirstOutParamOffset)
	for n := FirstCustomInParamOffset; n <= r.handlerInNum; n++ {
		r.inParams = append(r.inParams, openapi.NewRouteParam(r.method.Type.In(n), n))
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
	// 解析响应体
	err = r.outParams.Init()
	if err != nil {
		return err
	}

	// 初始化模型文档
	err = r.scanInParams()
	if err != nil {
		return err
	}
	err = r.scanOutParams()
	if err != nil {
		return err
	}

	err = r.ScanInner()
	return
}

func (r *GroupRoute) ScanInner() (err error) {
	err = r.swagger.Init()
	return
}

// 从方法入参中初始化路由参数, 包含了查询参数，请求体参数
func (r *GroupRoute) scanInParams() (err error) {
	r.swagger.QueryFields = make([]*openapi.QModel, 0)
	if r.handlerInNum == FirstInParamOffset { // 只有一个参数,只能是 Context
		return nil
	}

	if r.handlerInNum > FirstInParamOffset { // 存在自定义参数
		// 处理查询参数
		for index, param := range r.inParams[:r.handlerInNum-1-1] {
			switch param.Type {
			case openapi.ObjectType, openapi.ArrayType:
				return errors.New(fmt.Sprintf("param: %s, index: %d cannot be a %s",
					param.Pkg, index+FirstInParamOffset, param.Type))
			default:
				// 掐头去尾,获得查询参数,必须为基本数据类型
				// NOTICE: 此处无法获得方法的参数名，只能获得参数类型的名称
				r.swagger.QueryFields = append(r.swagger.QueryFields, &openapi.QModel{
					Name:   CreateQueryFieldName(param.Prototype, index), // 手动指定一个查询参数名称
					Tag:    "",
					Type:   param.Type,
					InPath: false,
				})
			}
		}
		// 入参最后一个视为请求体或查询参数
		lastInParam := r.inParams[r.handlerInNum-FirstCustomInParamOffset]
		if utils.Has[string]([]string{http.MethodGet, http.MethodDelete}, r.swagger.Method) {
			// 作为查询参数
			switch lastInParam.Type {
			case openapi.ObjectType:
				// 如果为结构体,则结构体的每一个字段都将作为一个查询参数
				// TODO Future-231126.3: 请求体不支持time.Time;
				r.swagger.QueryFields = append(r.swagger.QueryFields, openapi.StructToQModels(lastInParam.CopyPrototype())...)
			case openapi.ArrayType:
				// TODO Future-231126.6: 查询参数考虑是否要支持数组
			default:
				r.swagger.QueryFields = append(r.swagger.QueryFields, &openapi.QModel{
					Name:   CreateQueryFieldName(lastInParam.Prototype, r.handlerInNum), // 手动指定一个查询参数名称
					Tag:    "",
					Type:   lastInParam.Type,
					InPath: false,
				})
			}
		} else { // 作为请求体
			r.swagger.RequestModel = openapi.NewBaseModelMeta(lastInParam)
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

func (r *GroupRoute) Type() RouteType { return GroupRouteType }

func (r *GroupRoute) Swagger() *openapi.RouteSwagger {
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

func (r *GroupRoute) NewRequestModel() reflect.Value {
	if r.swagger.RequestModel != nil {
		req := r.inParams[r.handlerInNum-1]
		var rt reflect.Type
		if req.IsPtr {
			rt = req.CopyPrototype().Elem()
		} else {
			rt = req.CopyPrototype()
		}
		newValue := reflect.New(rt).Interface()
		reqParam := reflect.ValueOf(newValue)

		return reqParam
	}
	return reflect.Value{}
}

func (r *GroupRoute) Call() {
	//TODO implement me
	// result := method.Func.Call([]reflect.Value{reflect.ValueOf(newValue)})
	panic("implement me")
}

// =================================== 👇 路由组元数据 ===================================

// Scanner 元数据接口
// Init -> Scan -> ScanInner -> Init 级联初始化
type Scanner interface {
	Init() (err error)      // 初始化元数据对象
	Scan() (err error)      // 扫描并初始化自己
	ScanInner() (err error) // 扫描并初始化自己包含的字节点,通过 child.Init() 实现
}

// GroupRouterMeta 反射构建路由组的元信息
type GroupRouterMeta struct {
	router GroupRouter
	routes []*GroupRoute
	pkg    string // 包名.结构体名
	tags   []string
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

func (r *GroupRouterMeta) Id() string { return r.pkg }

func (r *GroupRouterMeta) Scan() (err error) {
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

// ScanInner 处理内部路由的文档等数据
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
			dv = v
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
	dv := fmt.Sprintf("%s %s", swagger.Method, swagger.RelativePath)
	if len(r.router.Description()) > 0 {
		v, ok := r.router.Description()[method.Name]
		if ok {
			dv = v
		}
	}

	return dv
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

	// 判断第一个返回值参数类型是否符合要求
	// TODO Future-231126.1: 返回值不允许为nil, see RouteParam.Init()
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

func CreateQueryFieldName(rt reflect.Type, index int) string {
	if rt.Kind() == reflect.Ptr {
		rt = rt.Elem()
	}

	return fmt.Sprintf("%s%d", rt.Name(), index)
}
