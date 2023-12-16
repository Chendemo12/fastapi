package fastapi

import (
	"context"
	"github.com/Chendemo12/fastapi-tool/helper"
	"github.com/Chendemo12/fastapi-tool/logger"
	"github.com/Chendemo12/fastapi/openapi"
	"github.com/go-playground/validator/v10"
	"io"
	"net/http"
	"time"
)

// Context 路由上下文信息, 也是钩子函数的操作句柄
//
// 此结构体内包含了响应体 Response 以减少在路由处理过程中的内存分配和复制
//
//	注意: 当一个路由被执行完毕时, 路由函数中的 Context 将被立刻释放回收, 因此在return之后对
//	Context 的任何引用都是不对的, 若需在return之后监听 Context.DisposableCtx() 则应该显式的复制或派生
type Context struct {
	svc         *Service           `description:"service"`
	muxCtx      MuxContext         `description:"路由器Context"`
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
}

// 申请一个 Context 并初始化
func (f *Wrapper) acquireCtx(ctx MuxContext) *Context {
	c := f.pool.Get().(*Context)
	// 初始化各种参数
	c.muxCtx = ctx
	c.response = AcquireResponse()
	// 为每一个路由创建一个独立的ctx, 允许不启用此功能
	if !f.conf.ContextAutomaticDerivationDisabled {
		c.routeCtx, c.routeCancel = context.WithCancel(f.service.ctx)
	}
	c.pathFields = map[string]string{}
	c.queryFields = map[string]any{}

	return c
}

// 释放并归还 Context
func (f *Wrapper) releaseCtx(ctx *Context) {
	ReleaseResponse(ctx.response)

	ctx.muxCtx = nil
	ctx.routeCtx = nil
	ctx.routeCancel = nil
	ctx.requestModel = nil
	ctx.response = nil // 释放内存

	ctx.pathFields = nil
	ctx.queryFields = nil

	f.pool.Put(ctx)
}

// ================================ 公共方法 ================================

func (c *Context) Deadline() (deadline time.Time, ok bool) {
	if c.routeCtx != nil {
		return c.routeCtx.Deadline()
	}
	return c.svc.ctx.Deadline()
}

func (c *Context) Err() error {
	if c.routeCtx != nil {
		return c.routeCtx.Err()
	}
	return nil
}

func (c *Context) Value(key any) any {
	if c.routeCtx != nil {
		return c.routeCtx.Value(key)
	}
	return nil
}

// Service 获取 Wrapper 的 Service 服务依赖信息
//
//	@return	Service 服务依赖信息
func (c *Context) Service() *Service { return c.svc }

// MuxContext 获取web引擎的上下文
func (c *Context) MuxContext() MuxContext { return c.muxCtx }

// DisposableCtx 针对此次请求的唯一context, 当路由执行完毕返回时,将会自动关闭
// <如果 ContextAutomaticDerivationDisabled = true 则异常>
// 为每一个请求创建一个新的 context.Context 其代价是非常高的，因此允许通过设置关闭此功能
//
//	@return	context.Context 唯一context
func (c *Context) DisposableCtx() context.Context { return c.routeCtx }

// Done 监听 DisposableCtx 是否完成退出
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

// Logger 获取注册的日志句柄
func (c *Context) Logger() logger.Iface { return dLog }

// Query 获取查询参数
// 对于已经在路由处定义的查询参数，应首先从 Context.queryFields 内部读取；
// 对于没有定义的其他查询参数则调用低层 MuxContext 进行解析
func (c *Context) Query(name string, undefined ...string) any {
	v, ok := c.queryFields[name]
	if ok {
		return v
	}

	return c.muxCtx.Query(name, undefined...)
}

// PathField 获取路径参数
// 对于已经在路由处定义的路径参数，应首先从 Context.pathFields 内部读取；
// 对于没有定义的其他查询参数则调用低层 MuxContext 进行解析
//
// 对于查询参数，参数类型会按照以下规则进行转换：
//
//	int 	=> int64
//	uint 	=> uint64
//	float 	=> float64
//	string 	=> string
//	bool 	=> bool
func (c *Context) PathField(name string, undefined ...string) string {
	v, ok := c.pathFields[name]
	if ok {
		return v
	}

	return c.muxCtx.Query(name, undefined...)
}

// ================================ 路由组路由方法 ================================

// Status 允许路由组路由函数在error=nil时修改响应状态码
// 由于路由组路由函数 GroupRouteHandler 签名的限制；当error=nil时状态码默认为500，error!=nil时默认为200
// 允许通过此方法进行修改
func (c *Context) Status(code int) {
	c.response.StatusCode = code
}

// ContentType 允许路由组路由函数修改响应MIME
// 由于路由组路由函数 GroupRouteHandler 签名的限制；其返回ContentType默认为"application/json; charset=utf-8"
// 允许通过此方法进行修改
func (c *Context) ContentType(contentType string) {
	// TODO：目前无法生效
	c.response.ContentType = contentType
}

// ================================ 范型路由方法 ================================

// Validator 获取请求体验证器
func (c *Context) Validator() *validator.Validate { return defaultValidator }

// JSONResponse 仅支持可以json序列化的响应体 (校验返回值)
//
// 对于结构体类型: 其返回值为序列化后的json
// 对于基本数据类型: 其返回值为实际数值
//
//	@param	statusCode	int	响应状态码
//	@param	content		any	可以json序列化的类型
//	@return	resp *Response response返回体
func (c *Context) JSONResponse(statusCode int, content any) *Response {
	c.response.StatusCode = statusCode
	c.response.Content = content

	// 通过校验
	return c.response
}

// OKResponse 返回状态码为200的 JSONResponse (校验返回值)
//
//	@param	content	any	可以json序列化的类型
//	@return	resp *Response response返回体
func (c *Context) OKResponse(content any) *Response {
	c.response.Content = content

	return c.response
}

// StringResponse 返回值为字符串对象 (不校验返回值)
//
//	@param	content	string	字符串文本
//	@return	resp *Response response返回体
func (c *Context) StringResponse(content string) *Response {
	c.response.Content = content

	return c.response
}

// StreamResponse 返回值为字节流对象 (不校验返回值)
//
//	@param	reader	io.Reader	字节流
//	@param	mime	string		返回头媒体资源类型信息,	缺省则为	"text/plain"
//	@return	resp *Response response返回体
func (c *Context) StreamResponse(reader io.Reader, mime ...string) *Response {
	c.response.Content = reader
	c.response.Type = StreamResponseType

	if len(mime) > 0 {
		c.response.ContentType = mime[0]
	} else {
		c.response.ContentType = openapi.MIMETextPlain
	}

	return c.response
}

// FileResponse 返回值为文件对象，如：照片视频文件流等, 若文件不存在，则状态码置为404 (不校验返回值)
//
//	@param	filepath	string	文件路径
//	@return	resp *Response response返回体
func (c *Context) FileResponse(filepath string) *Response {
	c.response.Content = filepath
	c.response.Type = FileResponseType

	return c.response
}

// ErrorResponse 返回一个服务器错误 (不校验返回值)
//
//	@param	content	any	错误消息
//	@return	resp *Response response返回体
func (c *Context) ErrorResponse(content any) *Response {
	c.response.StatusCode = http.StatusInternalServerError
	c.response.Content = content
	c.response.Type = ErrResponseType

	return c.response
}

// HTMLResponse 返回一段HTML文本 (不校验返回值)
//
//	@param	statusCode	int		响应状态码
//	@param	content		string	HTML文本字符串
//	@return	resp *Response response返回体
func (c *Context) HTMLResponse(statusCode int, content []byte) *Response {
	c.response.StatusCode = statusCode
	c.response.Content = content
	c.response.ContentType = openapi.MIMETextHTMLCharsetUTF8
	c.response.Type = HtmlResponseType

	return c.response
}

// AnyResponse 自定义响应体,响应体可是任意类型 (不校验返回值)
//
//	@param	statusCode	int		响应状态码
//	@param	content		any		响应体
//	@param	contentType	[]string	响应头MIME, 默认值为“application/json; charset=utf-8”
//	@return	resp *Response response返回体
func (c *Context) AnyResponse(statusCode int, content any, contentType ...string) *Response {
	var ct string
	if len(contentType) > 0 {
		ct = contentType[0]
	} else {
		ct = openapi.MIMEApplicationJSONCharsetUTF8
	}
	c.response.StatusCode = statusCode
	c.response.Content = content
	c.response.ContentType = ct
	c.response.Type = CustomResponseType

	return c.response
}

// ================================ SHORTCUTS ================================

// F 合并字符串
func (c *Context) F(s ...string) string { return helper.CombineStrings(s...) }

func (c *Context) Marshal(obj any) ([]byte, error) { return helper.JsonMarshal(obj) }

func (c *Context) Unmarshal(data []byte, v interface{}) error {
	return helper.JsonUnmarshal(data, v)
}
