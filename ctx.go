package fastapi

import (
	"context"
	"sync"
	"time"

	"github.com/Chendemo12/fastapi/openapi"
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
	// 热点字段 — 每次请求频繁访问
	mux         MuxContext        `description:"适配器自身, 实现 MuxContext"`
	response    *Response         `description:"返回值,以减少函数间复制的开销"`
	pathFields  map[string]string `description:"路径参数"`
	queryFields map[string]any    `description:"查询参数"`
	queryStruct any               `description:"结构体查询参数"`

	// 温点字段 — 部分请求访问
	requestModel any            `description:"请求体"`
	file         *File          `description:"文件"`
	locker       *sync.RWMutex  `description:"保护 Keys map"`
	sseOnce      sync.Once      `description:"SSE 初始化, 值类型零分配重置"`
	Keys         map[string]any `description:"每个请求专有的K/V"`
}

// NewContext 创建一个预分配好内部字段的 Context，供中间件 pool 使用
func NewContext() *Context {
	return &Context{
		pathFields:  make(map[string]string),
		queryFields: make(map[string]any),
		locker:      &sync.RWMutex{},
	}
}

// initContext 重置 Context 的可变字段，由 Wrapper.Handler 调用
func (c *Context) initContext(mux MuxContext) {
	c.mux = mux
	c.response = AcquireResponse()
	c.file = nil
	c.sseOnce = sync.Once{}
}

// resetContext 清理 Context 字段，由 Wrapper.Handler 通过 defer 调用
func (c *Context) resetContext() {
	ReleaseResponse(c.response)

	c.mux = nil
	c.requestModel = nil
	c.queryStruct = nil
	c.file = nil
	c.response = nil

	for k := range c.pathFields {
		delete(c.pathFields, k)
	}
	for k := range c.queryFields {
		delete(c.queryFields, k)
	}

	c.Keys = nil
}

// ================================ 公共方法 ================================

// MuxContext 获取web引擎的上下文
func (c *Context) MuxContext() MuxContext { return c.mux }

// MX shortcut web引擎的上下文
func (c *Context) MX() any { return c.mux.Ctx() }

// Context 获取当前请求的 context.Context，由底层 HTTP 引擎提供，
// 随请求结束自动取消。
func (c *Context) Context() context.Context {
	if c.mux != nil {
		return c.mux.RequestContext()
	}
	return nil
}

// Done 监听请求 Context 是否完成退出
func (c *Context) Done() <-chan struct{} {
	if c.mux != nil {
		return c.mux.Done()
	}
	return nil
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

	return c.mux.Query(name, undefined...)
}

// PathField 获取路径参数
// 对于已经在路由处定义的路径参数，首先从 Context.pathFields 内部读取；
// 对于没有定义的其他查询参数则调用低层 MuxContext 进行解析
func (c *Context) PathField(name string, undefined ...string) string {
	v, ok := c.pathFields[name]
	if ok {
		return v
	}

	return c.mux.Params(name, undefined...)
}

// Set 存储一个键值对，延迟初始化 ！仅当 MuxContext 未实现此类方法时采用！
// Set is used to store a new key/value pair exclusively for this context.
// It also lazy initializes  c.Keys if it was not used previously.
func (c *Context) Set(key string, value any) {
	c.locker.Lock()
	defer c.locker.Unlock()
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
	c.locker.RLock()
	defer c.locker.RUnlock()
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

// SSE 向客户端发送SSE数据，无需设置响应头，也无需关心消息结尾的换行符
func (c *Context) SSE(message *SSE) (err error) {
	if !(message != nil && len(message.Data) > 0) {
		return ErrSSEMessageEmpty
	}

	c.sseOnce.Do(func() {
		// 设置消息头
		c.mux.Header(openapi.HeaderContentType, string(openapi.MIMEEventStreamCharsetUTF8))
		c.mux.Header("Cache-Control", "no-cache")
		c.mux.Header("Connection", "keep-alive")
	})

	return c.mux.SSE(message)
}

// SSEKeepAlive 启动SSE保活, 通过周期性的向客户端发送注释消息，从而实现保活效果
func (c *Context) SSEKeepAlive(ctx context.Context, interval time.Duration) error {
	c.sseOnce.Do(func() {
		// 设置消息头
		c.mux.Header(openapi.HeaderContentType, string(openapi.MIMEEventStreamCharsetUTF8))
		c.mux.Header("Cache-Control", "no-cache")
		c.mux.Header("Connection", "keep-alive")
	})

	ticker := time.NewTicker(interval)
	var err error
	var msg = &SSE{Comment: "keep-alive"}

	for {
		select {
		case <-c.mux.Done():
			// 检测到路由结束
			return nil
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			err = c.mux.SSE(msg)
			if err != nil {
				return err
			}
		}
	}
}

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
