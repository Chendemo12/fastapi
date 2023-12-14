package fastapi

import (
	"github.com/Chendemo12/fastapi/openapi"
	"net/http"
	"reflect"
)

// GenericRouteHandler 泛型路由函数定义，其中 param 定义为实际的泛型类型
type GenericRouteHandler func(c *Context, param any) *Response

// Option 泛型接口定义可选项
// NOTICE: 231126 暂不可用
type Option struct {
	Summary       string              `json:"summary" description:"摘要描述"`
	ResponseModel openapi.ModelSchema `json:"response_model" description:"响应体模型"`
	RequestModel  openapi.ModelSchema `json:"request_model" description:"请求体模型"`
	Params        openapi.ModelSchema `json:"params" description:"查询参数,结构体"`
	Description   string              `json:"description" description:"路由描述"`
	Tags          []string            `json:"tags" description:"路由标签"`
	Deprecated    bool                `json:"deprecated" description:"是否禁用"`
}

func cleanOpts(opts ...Option) *Option {
	opt := &Option{
		Summary:       "",
		Params:        nil,
		RequestModel:  nil,
		ResponseModel: nil,
		Description:   "",
		Tags:          make([]string, 0),
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
	}

	return opt
}

// GenericRoute 泛型路由定义
type GenericRoute[T openapi.ModelSchema] struct {
	swagger   *openapi.RouteSwagger
	prototype T
	handler   func(c *Context, params T) *Response // good
}

func (r *GenericRoute[T]) Id() string { return r.swagger.Id() }

func (r *GenericRoute[T]) RouteType() RouteType {
	return RouteTypeGeneric
}

func (r *GenericRoute[T]) Swagger() *openapi.RouteSwagger {
	return r.swagger
}

func (r *GenericRoute[T]) ResponseBinder() *ParamBinder {
	//TODO implement me
	panic("implement me")
}

func (r *GenericRoute[T]) RequestBinders() *ParamBinder {
	//TODO implement me
	panic("implement me")
}

func (r *GenericRoute[T]) QueryBinders() []*ParamBinder {
	//TODO implement me
	panic("implement me")
}

func (r *GenericRoute[T]) NewInParams(ctx *Context) []reflect.Value {
	//TODO implement me
	panic("implement me")
}

func (r *GenericRoute[T]) NewRequestModel() any {
	return nil
}

func (r *GenericRoute[T]) NewQueryModel() any {
	return nil
}

func (r *GenericRoute[T]) Call(ctx *Context) {
	//TODO implement me
	panic("implement me")
}

func (r *GenericRoute[T]) Init() (err error) {
	//TODO implement me
	panic("implement me")
}

func (r *GenericRoute[T]) Scan() (err error) {
	//TODO implement me
	panic("implement me")
}

func (r *GenericRoute[T]) ScanInner() (err error) {
	//TODO implement me
	panic("implement me")
}

// Get TODO Future-231126.5: 泛型路由注册
func Get[T openapi.ModelSchema](path string, handler func(c *Context, query T) *Response, opt ...Option) *GenericRoute[T] {
	var prototype T
	g := &GenericRoute[T]{
		handler:   handler,
		prototype: prototype,
	}
	// 添加到全局数组
	g.swagger.RelativePath = path
	g.swagger.Method = http.MethodGet
	return g
}

func Delete[T openapi.ModelSchema](path string, handler func(c *Context, query T) *Response, opt ...Option) *GenericRoute[T] {
	var prototype T
	g := &GenericRoute[T]{
		handler:   handler,
		prototype: prototype,
	}
	// 添加到全局数组
	g.swagger.RelativePath = path
	g.swagger.Method = http.MethodDelete
	return g
}

func Post[T openapi.ModelSchema](path string, handler func(c *Context, req T) *Response, opt ...Option) *GenericRoute[T] {
	var prototype T
	g := &GenericRoute[T]{
		handler:   handler,
		prototype: prototype,
	}
	// 添加到全局数组
	g.swagger.RelativePath = path
	g.swagger.Method = http.MethodPost
	return g
}

func Patch[T openapi.ModelSchema](path string, handler func(c *Context, req T) *Response, opt ...Option) *GenericRoute[T] {
	var prototype T
	g := &GenericRoute[T]{
		handler:   handler,
		prototype: prototype,
	}
	// 添加到全局数组
	g.swagger.RelativePath = path
	g.swagger.Method = http.MethodPatch
	return g
}

// =================================== 👇 路由组元数据 ===================================

// GenericRouterMeta 统一记录所有的泛型路由
type GenericRouterMeta[T openapi.ModelSchema] struct {
	routes []*GenericRoute[T]
}
