package fastapi

import (
	"fmt"
	"net/http"
	"reflect"
	"strings"
	"unicode"
)

const WebsocketMethod = "WS"
const HttpMethodMinimumLength = len(http.MethodGet)
const (
	FirstInParamOffset  = 1 // 第一个有效参数的索引位置，由于结构体接收器处于第一位置
	FirstOutParamOffset = 0
	LastOutParamOffset  = 1 // 最后一个返回值参数的索引位置
	OutParamNum         = 2
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

var IllegalResponseType = []reflect.Kind{
	reflect.Interface,
	reflect.Func,
	reflect.Ptr,
	reflect.Chan,
	reflect.UnsafePointer,
}

// RouterGroup 结构体路由组定义
// 用法：首先实现此接口，然后通过调用 FastApi.IncludeRoute 方法进行注册绑定
type RouterGroup interface {
	// Prefix 路由组前缀，无需考虑是否以/开头或结尾
	// 如果为空则通过 PathSchema 方案进行格式化
	Prefix() string
	// Tags 标签，如果为空则设为结构体名称的大驼峰形式，去掉可能存在的http方法名
	Tags() []string
	// PathSchema 路由解析规则，对路由前缀和路由地址都有效
	PathSchema() RoutePathSchema
}

func (f *FastApi) IncludeRoute(router RouterGroup) *FastApi {
	return f
}

// BaseRouter (面向对象式)路由组基类
// 需实现 RouterGroup 接口
//
// 其中以 Get,Post,Delete,Patch,Put 字符串(不区分大小写)开头或结尾并以 (XXX, error)形式为返回值的方法会被作为路由处理
// 其url的确定按照 RoutePathSchema 接口进行解析。
//
// 对于作为路由的方法签名有如下要求：
//
//	1：参数：
//
//		第一个参数必须为 *Context
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
	// 基类实现不能包含路由方法
}

func (g *BaseRouter) Prefix() string { return "" }

func (g *BaseRouter) Tags() []string { return []string{} }

func (g *BaseRouter) PathSchema() RoutePathSchema { return UnixDash{} }

type GroupRouteMeta struct {
	method reflect.Method
	route  *GroupRoute
}

// GroupRouterMeta 反射构建路由组的元信息
type GroupRouterMeta struct {
	router RouterGroup
	pkg    string // 包名.结构体名
	routes []*GroupRouteMeta
}

// ============================================================================

func IsRouteMethod(method reflect.Method) (*RouteSwagger, bool) {
	if len(method.Name) <= HttpMethodMinimumLength {
		// 长度不够
		return nil, false
	}

	if unicode.IsLower([]rune(method.Name)[0]) {
		// 非导出方法
		return nil, false
	}

	swagger := &RouteSwagger{}
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
			swagger.RelativePath = method.Name[:offset]
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

	// 判断第一个返回值参数类型是否符合要求
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

// TODO: 考虑是否要修改为 GroupRouteMeta 的方法

// ScanGroupRouterTags 扫描tags, 由于接口方法允许留空，此处需处理默认值
func ScanGroupRouterTags(obj reflect.Type, router RouterGroup) []string {
	if obj.Kind() == reflect.Pointer {
		obj = obj.Elem()
	}
	tags := router.Tags()
	if len(tags) == 0 {
		tags = append(tags, obj.Name())
	}
	return tags
}

func ScanGroupRouter(router RouterGroup) *GroupRouterMeta {
	obj := reflect.TypeOf(router)

	// 路由组必须是结构体实现
	if obj.Kind() != reflect.Struct && obj.Kind() != reflect.Pointer {
		panic("router not a struct, " + obj.String())
	}

	groupMeta := &GroupRouterMeta{
		router: router,
		pkg:    obj.String(),
		routes: make([]*GroupRouteMeta, 0),
	}
	// 扫描tags
	tags := ScanGroupRouterTags(obj, router)
	// 扫描方法路由
	ScanGroupRouterMethod(obj, groupMeta, tags)

	return groupMeta
}

// ScanGroupRouterMethod 反射方法，由于必须是指针接收器，因此obj应为指针类型
func ScanGroupRouterMethod(obj reflect.Type, groupMeta *GroupRouterMeta, tags []string) {
	for i := 0; i < obj.NumMethod(); i++ {
		method := obj.Method(i)
		swagger, isRoute := IsRouteMethod(method)
		if !isRoute {
			continue
		}
		// 匹配到路由方法
		swagger.Url = groupMeta.router.PathSchema().Format(groupMeta.router.Prefix(), swagger.RelativePath)
		swagger.Summary = fmt.Sprintf("%s %s", swagger.Method, swagger.RelativePath)
		swagger.Tags = tags
		route := &GroupRoute{swagger: swagger, outParams: method.Type.Out(FirstOutParamOffset)}
		route.handlerInNum = method.Type.NumIn() - FirstInParamOffset // 排除接收器
		route.handlerOutNum = OutParamNum                             // 返回值数量始终为2

		for n := FirstInParamOffset; n <= route.handlerInNum; n++ {
			route.inParams = append(route.inParams, method.Type.In(n))
		}

		groupMeta.routes = append(groupMeta.routes, &GroupRouteMeta{
			method: method,
			route:  route,
		})
	}
}

func ScanGroupRoute(groupMeta *GroupRouterMeta) {
	for _, routeMeta := range groupMeta.routes {
		for _, model := range routeMeta.route.inParams {
			if model != nil {
				//routeMeta.route.swagger.RequestModel = openapi.BaseModelToMetadata(model)
			}
			println(model.String())
		}
		if routeMeta.route.outParams != nil {
			println(routeMeta.route.outParams.String())
		}
	}
}
