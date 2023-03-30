package fastapi

import (
	"context"
	"github.com/Chendemo12/fastapi/godantic"
	"github.com/Chendemo12/fastapi/logger"
	"github.com/Chendemo12/fastapi/openapi"
	"github.com/Chendemo12/fastapi/tool"
	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
	"io"
	"net/http"
)

const ( // json序列化错误, 关键信息的序号
	jsoniterUnmarshalErrorSeparator = "|" // 序列化错误信息分割符, 定义于 validator/validator_instance.orSeparator
	jsonErrorFieldMsgIndex          = 0   // 错误原因
	jsonErrorFieldNameFormIndex     = 1   // 序列化错误的字段和值
	jsonErrorFormIndex              = 3   // 接收到的数据
)

// ------------------------------------------------------------------------------------

// RouteMeta 记录创建的路由对象，用于其后的请求和响应校验
type RouteMeta struct {
	Get    *Route
	Post   *Route
	Patch  *Route
	Delete *Route
	Put    *Route
	Any    *Route
	Path   string `json:"path" description:"绝对路由"`
}

// UserService 自定义服务依赖信息
type UserService interface {
	Config() any // 获取配置文件
}

// Service FastApi 全局服务依赖信息
// 此对象由FastApi启动时自动创建，此对象不应被修改，组合和嵌入，
// 但可通过 setUserSVC 接口设置自定义的上下文信息，并在每一个路由钩子函数中可得
type Service struct {
	logger    logger.Iface        `description:"日志对象"`
	userSVC   UserService         `description:"自定义服务依赖"`
	ctx       context.Context     `description:"根Context"`
	validate  *validator.Validate `description:"请求体验证包"`
	openApi   *openapi.OpenApi    `description:"模型文档"`
	cancel    context.CancelFunc  `description:"取消函数"`
	scheduler *Scheduler          `description:"定时任务"`
	addr      string              `description:"绑定地址"`
	cache     []*RouteMeta        `description:"用于数据校验的路由信息"`
}

// 查询自定义路由
//
//	@param	method	string	请求方法
//	@param	path	string	请求路由
//	@return	*Route 自定义路由对象
func (s *Service) queryRoute(method string, path string) (route *Route) {
	for i := 0; i < len(s.cache); i++ {
		if s.cache[i].Path == path {
			switch method {
			case http.MethodGet:
				route = s.cache[i].Get
			case http.MethodPut:
				route = s.cache[i].Put
			case http.MethodPatch:
				route = s.cache[i].Patch
			case http.MethodDelete:
				route = s.cache[i].Delete
			case http.MethodPost:
				route = s.cache[i].Post
			default:
				route = nil
			}
			break
		}
	}

	return
}

func (s *Service) queryRouteMeta(path string) *RouteMeta {
	for i := 0; i < len(s.cache); i++ {
		if s.cache[i].Path == path {
			return s.cache[i]
		}
	}

	// 不存在则创建
	meta := &RouteMeta{Path: path}
	s.cache = append(s.cache, meta)
	return meta
}

// 设置一个自定义服务信息
//
//	@param	service	UserService	服务依赖
func (s *Service) setUserSVC(svc UserService) *Service {
	s.userSVC = svc
	return s
}

// 替换日志句柄
//
//	@param	logger	logger.Iface	日志句柄
func (s *Service) setLogger(logger logger.Iface) *Service {
	s.logger = logger
	return s
}

// Addr 绑定地址
//
//	@return	string 绑定地址
func (s *Service) Addr() string { return s.addr }

// RootCtx 根 context
//
//	@return	context.Context 整个服务的根 context
func (s *Service) RootCtx() context.Context { return s.ctx }

// Logger 获取日志句柄
func (s *Service) Logger() logger.Iface { return s.logger }

// Done 监听程序是否退出或正在关闭，仅当程序关闭时解除阻塞
func (s *Service) Done() <-chan struct{} { return s.ctx.Done() }

// Scheduler 获取内部调度器
func (s *Service) Scheduler() *Scheduler { return s.scheduler }

// Validate 结构体验证
//
//	@param	stc	any	需要校验的结构体
//	@param	ctx	any	当校验不通过时需要返回给客户端的附加信息，仅第一个有效
//	@return
func (s *Service) Validate(stc any, ctx ...map[string]any) *Response {
	err := s.validate.StructCtx(s.ctx, stc)
	if err != nil { // 模型验证错误
		err, _ := err.(validator.ValidationErrors) // validator的校验错误信息

		if nums := len(err); nums == 0 {
			return validationErrorResponse()
		} else {
			ves := make([]*godantic.ValidationError, nums) // 自定义的错误信息
			for i := 0; i < nums; i++ {
				ves[i] = &godantic.ValidationError{
					Loc:  []string{"body", err[i].Field()},
					Msg:  err[i].Error(),
					Type: err[i].Type().String(),
				}
				if len(ctx) > 0 {
					ves[i].Ctx = ctx[0]
				}
			}
			return validationErrorResponse(ves...)
		}
	}

	return nil
}

// ------------------------------------------------------------------------------------

// Context 路由上下文信息, 也是钩子函数的操作句柄
//
// 此结构体内包含了相应体 response 以减少在路由处理过程中的内存分配和复制
//
//	注意: 当一个路由被执行完毕时, 路由函数 HandlerFunc 中的 Context 将被立刻释放回收, 因此在return之后对
//
// Context 的任何引用都是不对的, 若需在return之后监听 Context.DisposableCtx() 则应该显式的复制或派生
type Context struct {
	PathFields  map[string]string  `json:"path_fields,omitempty"`  // 路径参数
	QueryFields map[string]string  `json:"query_fields,omitempty"` // 查询参数
	RequestBody any                `json:"request_body,omitempty"` // 请求体，初始值为1
	svc         *Service           `description:"flask-go service"`
	ec          *fiber.Ctx         `description:"engine context"`
	route       *Route             `description:"用于请求体和响应体校验"`
	routeCtx    context.Context    `description:"获取针对此次请求的唯一context"`
	routeCancel context.CancelFunc `description:"获取针对此次请求的唯一取消函数"`
	response    *Response          `description:"返回值,以减少函数间复制的开销"`
}

// Service 获取 FastApi 的 Service 服务依赖信息
//
//	@return	Service 服务依赖信息
func (c *Context) Service() *Service { return c.svc }

// EngineCtx 获取web引擎的上下文 Service
//
//	@return	*fiber.Ctx fiber.App 的上下文信息
func (c *Context) EngineCtx() *fiber.Ctx { return c.ec }

// UserSVC 获取自定义服务依赖
func (c *Context) UserSVC() UserService { return c.svc.userSVC }

// DisposableCtx 针对此次请求的唯一context, 当路由执行完毕返回时,将会自动关闭
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

// BodyParser 序列化请求体
//
//	@param	c	*fiber.Ctx	fiber上下文
//	@param	a	any			请求体指针
//	@return	*Response 错误信息,若为nil 则序列化成功
func (c *Context) BodyParser(a any) *Response {
	if err := c.EngineCtx().BodyParser(a); err != nil { // 请求的表单序列化错误
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
		ContentType: fiber.MIMETextHTML,
	}
	return c.response
}

// AdvancedResponse 高级返回值，允许返回一个函数，以实现任意类型的返回 (不校验返回值)
//
//	@param	statusCode	int				响应状态码
//	@param	content		fiber.Handler	钩子函数
//	@return	resp *Response response返回体
func (c *Context) AdvancedResponse(statusCode int, content fiber.Handler) *Response {
	c.response = &Response{
		Type:        AdvancedResponseType,
		StatusCode:  statusCode,
		Content:     content,
		ContentType: "",
	}
	return c.response
}

// AnyResponse 自定义响应体,响应体可是任意类型 (不校验返回值)
//
//	@param	statusCode	int		响应状态码
//	@param	content		any		响应体
//	@param	contentType	string	响应头MIME
//	@return	resp *Response response返回体
func (c *Context) AnyResponse(statusCode int, content any, contentType string) *Response {
	c.response = &Response{
		StatusCode: statusCode, Content: content, ContentType: contentType,
		Type: CustomResponseType,
	}
	return c.response
}

// ================================ SHORTCUTS ================================

// F 合并字符串
func (c *Context) F(s ...string) string            { return tool.CombineStrings(s...) }
func (c *Context) Marshal(obj any) ([]byte, error) { return tool.Marshal(obj) }
func (c *Context) Unmarshal(data []byte, v interface{}) error {
	return tool.Unmarshal(data, v)
}
