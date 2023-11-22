package fastapi

import (
	"fmt"
	"net/http"
	"reflect"
	"strings"
	"unicode"
)

var HttpMethods = []string{
	http.MethodGet,
	http.MethodPost,
	http.MethodPatch,
	http.MethodPut,
	http.MethodDelete,
	http.MethodOptions,
}

const HttpMethodMinimumLength = len(http.MethodGet)
const FirstInParamOffset = 1 // 第一个有效参数的索引位置，由于结构体接收器处于第一位置
const FirstOutParamOffset = 0
const LastOutParamOffset = 1 // 最后一个返回值参数的索引位置

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

func (g *BaseRouter) Tags() []string { return []string{"DEFAULT"} }

func (g *BaseRouter) PathSchema() RoutePathSchema { return UnixDash{} }

func IsRouteMethod(method reflect.Method) (bool, *Route) {
	if !unicode.IsUpper([]rune(method.Name)[0]) {
		// 非导出方法
		return false, nil
	}

	if len(method.Name) <= HttpMethodMinimumLength {
		// 长度不够
		return false, nil
	}

	route := &Route{}
	methodNameLength := len(method.Name)

	// 依次判断是哪一种方法
	for _, hm := range HttpMethods {
		offset := len(hm)
		if strings.ToUpper(method.Name[:offset]) == hm || strings.ToUpper(method.Name[methodNameLength-offset:]) == hm {
			route.Method = hm
			break
		}
	}
	if route.Method == "" {
		// 方法名称不符合
		return false, nil
	}

	// 判断方法参数是否符合要求
	paramNum := method.Type.NumIn()
	respNum := method.Type.NumOut()

	if paramNum < 1 || respNum != 2 {
		// 方法参数数量不对
		return false, nil
	}

	// 获取请求参数
	if method.Type.In(FirstInParamOffset).Elem().Name() != "Context" || method.Type.Out(LastOutParamOffset).Name() != "error" {
		// 方法参数类型不符合
		return false, nil
	}

	// 获取返回值参数
	viType := []reflect.Kind{
		reflect.Interface,
		reflect.Func,
		reflect.Ptr,
		reflect.Chan,
		reflect.UnsafePointer,
	}

	for _, k := range viType {
		elemType := method.Type.Out(FirstOutParamOffset)
		if elemType.Kind() == reflect.Pointer {
			elemType = elemType.Elem()
		}

		if elemType.Kind() == k {
			// 返回值的第一个参数不符合要求
			return false, nil
		}
	}

	route.handleInNum = paramNum
	route.handleOutNum = respNum
	return true, route
}

func IncludeRouter(router RouterGroup) {
	// TODO: 反射方法生成 [] Route
	obj := reflect.TypeOf(router)

	// 路由组必须是结构体实现
	if obj.Kind() != reflect.Struct && obj.Kind() != reflect.Ptr {
		panic("router not a struct, " + obj.String())
	}
	// 反射其方法
	for i := 0; i < obj.NumMethod(); i++ {
		isRoute, route := IsRouteMethod(obj.Method(i))
		if !isRoute {
			continue
		}
		url := router.PathSchema().Format(router.Prefix(), obj.Method(i).Name)
		fmt.Println(route.Method, url)
	}
}
