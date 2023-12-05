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
	PathFields  map[string]string  `json:"path_fields,omitempty"`  // 路径参数
	QueryFields map[string]string  `json:"query_fields,omitempty"` // 查询参数
	svc         *Service           `description:"flask-go service"`
	muxCtx      MuxCtx             `description:"路由器Context"`
	routeCtx    context.Context    `description:"获取针对此次请求的唯一context"`
	routeCancel context.CancelFunc `description:"获取针对此次请求的唯一取消函数"`
	response    *Response          `description:"返回值,以减少函数间复制的开销"`
}

func (c *Context) Deadline() (deadline time.Time, ok bool) {
	if c.routeCtx != nil {
		return c.routeCtx.Deadline()
	}
	return time.Time{}, false
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

// Service 获取 FastApi 的 Service 服务依赖信息
//
//	@return	Service 服务依赖信息
func (c *Context) Service() *Service { return c.svc }

// MuxCtx 获取web引擎的上下文
func (c *Context) MuxCtx() MuxCtx { return c.muxCtx }

// DisposableCtx 针对此次请求的唯一context, 当路由执行完毕返回时,将会自动关闭
// 为每一个请求创建一个新的 context.Context 其代价是非常高的，因此允许通过设置关闭此功能
//
//	@return	context.Context 唯一context
func (c *Context) DisposableCtx() context.Context { return c.routeCtx }

// Done 监听 DisposableCtx 是否完成退出
//
//	@return	chan struct{} 是否退出
func (c *Context) Done() <-chan struct{} { return c.routeCtx.Done() }

// Logger 获取日志句柄
func (c *Context) Logger() logger.Iface { return c.svc.Logger() }

// Validator 获取请求体验证器
func (c *Context) Validator() *validator.Validate { return c.svc.validate }

// Validate 结构体验证
//
//	@param	stc	any	需要校验的结构体
//	@param	ctx	any	当校验不通过时需要返回给客户端的附加信息，仅第一个有效
//	@return
func (c *Context) Validate(stc any, ctx ...map[string]any) *Response {
	return c.svc.Validate(stc, ctx...)
}

// Query 获取查询参数
func (c *Context) Query(name string, undefined ...string) string {
	v, ok := c.QueryFields[name]
	if ok {
		return v
	}

	if len(undefined) > 0 {
		return undefined[0]
	}

	return ""
}

// PathField 获取路径参数
func (c *Context) PathField(name string, undefined ...string) string {
	v, ok := c.PathFields[name]
	if ok {
		return v
	}

	if len(undefined) > 0 {
		return undefined[0]
	}

	return ""
}

// BodyParser 序列化请求体
func (c *Context) BodyParser(a any) *Response {
	if err := c.MuxCtx().BodyParser(a); err != nil { // 请求的表单序列化错误
		c.Logger().Error(err)
		return validationErrorResponse(jsoniterUnmarshalErrorToValidationError(err))
	}

	return nil
}

// ShouldBindJSON 绑定并校验参数是否正确
func (c *Context) ShouldBindJSON(stc any) *Response {
	if err := c.BodyParser(stc); err != nil {
		return err
	}
	if resp := c.Validate(stc, whereClientError); resp != nil {
		return resp
	}

	return nil
}

// StringResponse 返回值为字符串对象 (校验返回值)
//
//	@param	content	string	字符串文本
//	@return	resp *Response response返回体
func (c *Context) StringResponse(content string) *Response {
	c.response = &Response{
		StatusCode: http.StatusOK, Content: content, Type: StringResponseType,
	}

	return c.response
}

// JSONResponse 仅支持可以json序列化的响应体 (校验返回值)
//
// 对于结构体类型: 其返回值为序列化后的json
// 对于基本数据类型: 其返回值为实际数值
// 对于数组类型: 若其子元素为Uint8,则自动转换为 StreamResponse 以避免转义错误,但应显式的返回 StreamResponse
//
//	@param	statusCode	int	响应状态码
//	@param	content		any	可以json序列化的类型
//	@return	resp *Response response返回体
func (c *Context) JSONResponse(statusCode int, content any) *Response {
	c.response = &Response{
		StatusCode: statusCode, Content: content, Type: JsonResponseType,
	}

	// 通过校验
	return c.response
}

// OKResponse 返回状态码为200的 JSONResponse (校验返回值)
//
//	@param	content	any	可以json序列化的类型
//	@return	resp *Response response返回体
func (c *Context) OKResponse(content any) *Response { return c.JSONResponse(http.StatusOK, content) }

// StreamResponse 返回值为字节流对象 (不校验返回值)
//
//	@param	reader	io.Reader	字节流
//	@param	mime	string		返回头媒体资源类型信息,	缺省则为	"text/plain"
//	@return	resp *Response response返回体
func (c *Context) StreamResponse(reader io.Reader, mime ...string) *Response {
	c.response = &Response{
		StatusCode: http.StatusOK, Content: reader, Type: StreamResponseType, ContentType: openapi.MIMETextPlain,
	}
	if len(mime) > 0 {
		c.response.ContentType = mime[0]
	}

	return c.response
}

// FileResponse 返回值为文件对象，如：照片视频文件流等, 若文件不存在，则状态码置为404 (不校验返回值)
//
//	@param	filepath	string	文件路径
//	@return	resp *Response response返回体
func (c *Context) FileResponse(filepath string) *Response {
	c.response = &Response{
		StatusCode: http.StatusOK, Content: filepath, Type: FileResponseType,
	}
	return c.response
}

// ErrorResponse 返回一个服务器错误 (不校验返回值)
//
//	@param	content	any	错误消息
//	@return	resp *Response response返回体
func (c *Context) ErrorResponse(content any) *Response {
	c.response = &Response{
		StatusCode: http.StatusInternalServerError, Content: content, Type: ErrResponseType,
	}
	return c.response
}

// HTMLResponse 返回一段HTML文本 (不校验返回值)
//
//	@param	statusCode	int		响应状态码
//	@param	content		string	HTML文本字符串
//	@return	resp *Response response返回体
func (c *Context) HTMLResponse(statusCode int, context string) *Response {
	c.response = &Response{
		Type:        HtmlResponseType,
		StatusCode:  statusCode,
		Content:     context,
		ContentType: openapi.MIMETextHTMLCharsetUTF8,
	}
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

	c.response = &Response{
		StatusCode: statusCode, Content: content, ContentType: ct,
		Type: CustomResponseType,
	}
	return c.response
}

// ================================ SHORTCUTS ================================

// F 合并字符串
func (c *Context) F(s ...string) string { return helper.CombineStrings(s...) }

func (c *Context) Marshal(obj any) ([]byte, error) { return helper.JsonMarshal(obj) }

func (c *Context) Unmarshal(data []byte, v interface{}) error {
	return helper.JsonUnmarshal(data, v)
}
