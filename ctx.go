package fastapi

import (
	"context"
	"sync"
	"time"

	"github.com/Chendemo12/fastapi/utils"
	"github.com/go-playground/validator/v10"
)

// Context 路由上下文信息, 也是钩子函数的操作句柄
//
// 此结构体内包含了响应体 Response 以减少在路由处理过程中的内存分配和复制
//
//	注意: 当一个路由被执行完毕时, 路由函数中的 Context 将被立刻释放回收, 因此在return之后对
//	Context 的任何引用都是不对的, 若需在return之后监听 Context.Context() 则应该显式的复制或派生
type Context struct {
	muxCtx      MuxContext         `description:"路由器Context"`
	appCtx      context.Context    `description:"根context"`
	routeCtx    context.Context    `description:"获取针对此次请求的唯一context"`
	routeCancel context.CancelFunc `description:"获取针对此次请求的唯一取消函数"`
	// 存储路径参数, 路径参数类型全部为字符串类型, 路径参数都是肯定存在的
	pathFields map[string]string `description:"路径参数"`
	// 对于查询参数，参数类型会按照以下规则进行转换：
	// 	int 	=> int64
	// 	uint 	=> uint64
	// 	float 	=> float64
	//	string 	=> string
	// 	bool 	=> bool
	queryFields  map[string]any `description:"查询参数, 仅记录存在值的查询参数"`
	queryStruct  any            `description:"结构体查询参数"`
	requestModel any            `description:"请求体"`
	response     *Response      `description:"返回值,以减少函数间复制的开销"`
	// This mutex protects Keys map.
	mu sync.RWMutex
	// 每个请求专有的K/V
	Keys map[string]any
}

// 申请一个 Context 并初始化
func (f *Wrapper) acquireCtx(ctx MuxContext) *Context {
	c := f.pool.Get().(*Context)
	// 初始化各种参数
	c.muxCtx = ctx
	c.response = AcquireResponse()
	// 为每一个路由创建一个独立的ctx, 允许不启用此功能
	if !f.conf.ContextAutomaticDerivationDisabled {
		c.routeCtx, c.routeCancel = context.WithCancel(f.ctx)
	}
	c.appCtx = f.ctx
	c.pathFields = map[string]string{}
	c.queryFields = map[string]any{}
	c.mu = sync.RWMutex{}

	return c
}

// 释放并归还 Context
func (f *Wrapper) releaseCtx(ctx *Context) {
	ReleaseResponse(ctx.response)

	ctx.muxCtx = nil
	ctx.appCtx = nil
	ctx.routeCtx = nil
	ctx.routeCancel = nil
	ctx.requestModel = nil
	ctx.response = nil // 释放内存

	ctx.pathFields = nil
	ctx.queryFields = nil
	ctx.Keys = nil

	f.pool.Put(ctx)
}

// ================================ 公共方法 ================================

// MuxContext 获取web引擎的上下文
func (c *Context) MuxContext() MuxContext { return c.muxCtx }

// MX shortcut web引擎的上下文
func (c *Context) MX() any { return c.muxCtx.Ctx() }

// Context 针对此次请求的唯一context, 当路由执行完毕返回时,将会自动关闭
// <如果 ContextAutomaticDerivationDisabled = true 则异常>
//
// 为每一个请求创建一个新的 context.Context 其代价是非常高的，因此允许通过设置关闭此功能
//
//	@return	context.Context 当前请求的唯一context
func (c *Context) Context() context.Context { return c.routeCtx }

// RootContext 根context
// 当禁用了context自动派生功能 <ContextAutomaticDerivationDisabled = true>，但又需要一个context时，可获得路由器Wrapper的context
func (c *Context) RootContext() context.Context { return c.appCtx }

// Done 监听 Context 是否完成退出
// <如果 ContextAutomaticDerivationDisabled = true 则异常>
//
//	@return	chan struct{} 是否退出
func (c *Context) Done() <-chan struct{} {
	if c.routeCtx != nil {
		return c.routeCtx.Done()
	} else {
		return nil
	}
}

// Query 获取查询参数
// 对于已经在路由处定义的查询参数，首先从 Context.queryFields 内部读取
// 对于没有定义的其他查询参数则调用低层 MuxContext 进行解析
//
//	对于路由处定义的查询参数，参数类型会按照以下规则进行转换：
//
//	int 	=> int64
//	uint 	=> uint64
//	float 	=> float64
//	string 	=> string
//	bool 	=> bool
func (c *Context) Query(name string, undefined ...string) any {
	v, ok := c.queryFields[name]
	if ok {
		return v
	}

	return c.muxCtx.Query(name, undefined...)
}

// PathField 获取路径参数
// 对于已经在路由处定义的路径参数，首先从 Context.pathFields 内部读取；
// 对于没有定义的其他查询参数则调用低层 MuxContext 进行解析
func (c *Context) PathField(name string, undefined ...string) string {
	v, ok := c.pathFields[name]
	if ok {
		return v
	}

	return c.muxCtx.Params(name, undefined...)
}

// Set 存储一个键值对，延迟初始化 ！仅当 MuxContext 未实现此类方法时采用！
// Set is used to store a new key/value pair exclusively for this context.
// It also lazy initializes  c.Keys if it was not used previously.
func (c *Context) Set(key string, value any) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.Keys == nil {
		c.Keys = make(map[string]any)
	}

	c.Keys[key] = value
}

// Get 从上下文中读取键值, ie: (value, true).
// 如果不存在则返回 (nil, false)
// Get returns the value for the given key, ie: (value, true).
// If the value does not exist it returns (nil, false)
func (c *Context) Get(key string) (value any, exists bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	value, exists = c.Keys[key]
	return
}

// MustGet 从上下文中读取键值，如果不存在则panic
// MustGet returns the value for the given key if it exists, otherwise it panics.
func (c *Context) MustGet(key string) any {
	if value, exists := c.Get(key); exists {
		return value
	}
	panic("Key \"" + key + "\" does not exist")
}

// GetString 以字符串形式读取键值
// GetString returns the value associated with the key as a string.
func (c *Context) GetString(key string) (s string) {
	if val, ok := c.Get(key); ok && val != nil {
		s, _ = val.(string)
	}
	return
}

// GetBool 以bool形式读取键值
// GetBool returns the value associated with the key as a boolean.
func (c *Context) GetBool(key string) (b bool) {
	if val, ok := c.Get(key); ok && val != nil {
		b, _ = val.(bool)
	}
	return
}

// GetInt64 以int64形式读取键值
// GetInt64 returns the value associated with the key as an integer.
func (c *Context) GetInt64(key string) (i64 int64) {
	if val, ok := c.Get(key); ok && val != nil {
		i64, _ = val.(int64)
	}
	return
}

// GetUint64 以uint64形式读取键值
// GetUint64 returns the value associated with the key as an unsigned integer.
func (c *Context) GetUint64(key string) (ui64 uint64) {
	if val, ok := c.Get(key); ok && val != nil {
		ui64, _ = val.(uint64)
	}
	return
}

// GetTime 以time形式读取键值
// GetTime returns the value associated with the key as time.
func (c *Context) GetTime(key string) (t time.Time) {
	if val, ok := c.Get(key); ok && val != nil {
		t, _ = val.(time.Time)
	}
	return
}

// Response 响应体，配合 Wrapper.UseBeforeWrite 实现在依赖函数中读取响应体内容，以进行日志记录等 ！慎重对 Response 进行修改！
func (c *Context) Response() *Response { return c.response }

// ================================ 路由组路由方法 ================================

// Status 修改成功响应的状态码
// 由于路由组路由函数 GroupRouteHandler 签名的限制；当error!=nil时状态码默认为500，error==nil时默认为200
// 允许通过此方法修改当error=nil的响应状态码
func (c *Context) Status(code int) {
	c.response.StatusCode = code
}

// ================================ 范型路由方法 ================================

// Validator 获取请求体验证器
func (c *Context) Validator() *validator.Validate { return defaultValidator }

// ================================ SHORTCUTS ================================

// F 合并字符串
func (c *Context) F(s ...string) string { return utils.CombineStrings(s...) }

func (c *Context) Marshal(obj any) ([]byte, error) { return utils.JsonMarshal(obj) }

func (c *Context) Unmarshal(data []byte, v interface{}) error {
	return utils.JsonUnmarshal(data, v)
}
