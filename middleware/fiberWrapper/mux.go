package fiberWrapper

import (
	"errors"
	"fmt"
	"github.com/Chendemo12/fastapi"
	"github.com/Chendemo12/fastapi-tool/helper"
	"github.com/gofiber/fiber/v2"
	echo "github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"io"
	"net/http"
	"runtime"
	"sync"
	"time"
)

type FiberMux struct {
	app  *fiber.App
	pool *sync.Pool
}

// NewWrapper 创建App实例
func NewWrapper(app *fiber.App) *FiberMux {
	return &FiberMux{
		app:  app,
		pool: &sync.Pool{New: func() any { return &FiberContext{} }},
	}
}

// Default 默认的fiber.app，已做好基本的参数配置
func Default() *FiberMux {
	app := fiber.New(fiber.Config{
		Prefork:       false,                   // 多进程模式
		CaseSensitive: true,                    // 区分路由大小写
		StrictRouting: true,                    // 严格路由
		ServerHeader:  "FastApi",               // 服务器头
		AppName:       "fastapi.fiber",         // 设置为 Response.Header.Server 属性
		ColorScheme:   fiber.DefaultColors,     // 彩色输出
		JSONEncoder:   helper.JsonMarshal,      // json序列化器
		JSONDecoder:   helper.JsonUnmarshal,    // json解码器
		ErrorHandler:  customFiberErrorHandler, // 设置自定义错误处理
	})

	// 输出API访问日志
	echoConfig := echo.ConfigDefault
	echoConfig.TimeFormat = time.DateTime
	echoConfig.Format = "${time}    ${method}\t${path}    ${status}\n"
	app.Use(echo.New(echoConfig))

	// 自定义全局 recover 方法
	app.Use(recover.New(recover.Config{
		EnableStackTrace: true,
		// StackTraceHandler: 处理堆栈跟踪的函数, 若留空，则默认将整个错误堆栈输出到控制台,
		// 并在处理完成后将错误流转到 fiber.ErrorHandler
		StackTraceHandler: customRecoverHandler,
	}))

	return NewWrapper(app)
}

func (m *FiberMux) App() *fiber.App { return m.app }

func (m *FiberMux) AcquireCtx(c *fiber.Ctx) *FiberContext {
	obj := m.pool.Get().(*FiberContext)
	obj.ctx = c

	return obj
}

func (m *FiberMux) ReleaseCtx(c *FiberContext) {
	c.ctx = nil
	m.pool.Put(c)
}

func (m *FiberMux) Listen(addr string) error {
	return m.app.Listen(addr)
}

func (m *FiberMux) ShutdownWithTimeout(timeout time.Duration) error {
	return m.app.ShutdownWithTimeout(timeout)
}

func (m *FiberMux) BindRoute(method, path string, handler fastapi.MuxHandler) error {
	switch method {
	case http.MethodGet:
		m.app.Get(path, func(ctx *fiber.Ctx) error {
			return handler(&FiberContext{ctx: ctx})
		})
	case http.MethodPost:
		m.app.Post(path, func(ctx *fiber.Ctx) error {
			return handler(&FiberContext{ctx: ctx})
		})
	case http.MethodDelete:
		m.app.Delete(path, func(ctx *fiber.Ctx) error {
			return handler(&FiberContext{ctx: ctx})
		})
	case http.MethodPatch:
		m.app.Patch(path, func(ctx *fiber.Ctx) error {
			return handler(&FiberContext{ctx: ctx})
		})
	case http.MethodPut:
		m.app.Put(path, func(ctx *fiber.Ctx) error {
			return handler(&FiberContext{ctx: ctx})
		})
	default:
		return errors.New(fmt.Sprintf("unknow method:'%s' for path: '%s'", method, path))
	}

	return nil
}

type FiberContext struct {
	ctx *fiber.Ctx
}

func (c *FiberContext) Method() string {
	return c.ctx.Method()
}

func (c *FiberContext) Path() string {
	return c.ctx.Route().Path
}

func (c *FiberContext) Query(key string, undefined ...string) string {
	return c.ctx.Query(key, undefined...)
}

func (c *FiberContext) Params(key string, undefined ...string) string {
	return c.ctx.Params(key, undefined...)
}

func (c *FiberContext) ContentType() string {
	return string(c.ctx.Context().Request.Header.ContentType())
}

func (c *FiberContext) BindQuery(obj any) error {
	return nil
}

func (c *FiberContext) BindQueryNotImplemented() bool {
	return true
}

func (c *FiberContext) BodyParser(obj any) error { return c.ctx.BodyParser(obj) }

func (c *FiberContext) Validate(obj any) error { return nil }

func (c *FiberContext) ShouldBind(obj any) error { return nil }

func (c *FiberContext) ShouldBindNotImplemented() bool { return true }

func (c *FiberContext) SetCookie(cookie *http.Cookie) {
	ck := &fiber.Cookie{
		Name:        cookie.Name,
		Value:       cookie.Value,
		Path:        cookie.Path,
		Domain:      cookie.Domain,
		MaxAge:      cookie.MaxAge,
		Expires:     cookie.Expires,
		Secure:      cookie.Secure,
		HTTPOnly:    cookie.HttpOnly,
		SessionOnly: false,
	}

	switch cookie.SameSite {
	case http.SameSiteDefaultMode:
		ck.SameSite = fiber.CookieSameSiteDisabled
	case http.SameSiteLaxMode:
		ck.SameSite = fiber.CookieSameSiteLaxMode
	case http.SameSiteStrictMode:
		ck.SameSite = fiber.CookieSameSiteStrictMode
	case http.SameSiteNoneMode:
		ck.SameSite = fiber.CookieSameSiteNoneMode
	}
	c.ctx.Cookie(ck)
}

func (c *FiberContext) Cookie(name string) (string, error) {
	//TODO implement me
	panic("implement me")
}

func (c *FiberContext) Get(key string, defaultValue ...string) string {
	//TODO implement me
	panic("implement me")
}

func (c *FiberContext) Set(key string, value any) {
	//TODO implement me
	panic("implement me")
}

func (c *FiberContext) ClientIP() string { return c.ctx.IP() }

func (c *FiberContext) Status(statusCode int) { c.ctx.Status(statusCode) }

func (c *FiberContext) SendStream(stream io.Reader, size ...int) error {
	return c.ctx.SendStream(stream, size...)
}

func (c *FiberContext) Render(name string, bind interface{}, layouts ...string) error {
	//TODO implement me
	panic("implement me")
}

func (c *FiberContext) YAML(code int, obj any) error {
	//TODO implement me
	panic("implement me")
}

func (c *FiberContext) TOML(code int, obj any) error {
	return nil
}

func (c *FiberContext) Header(key, value string) { c.ctx.Set(key, value) }

func (c *FiberContext) Redirect(code int, location string) error {
	return c.ctx.Redirect(location, code)
}

func (c *FiberContext) JSONP(code int, data any) error {
	c.Status(code)
	return c.ctx.JSONP(data)
}

func (c *FiberContext) File(filepath string) error {
	//TODO implement me
	panic("implement me")
}

func (c *FiberContext) FileAttachment(filepath, filename string) error {
	//TODO implement me
	panic("implement me")
}

func (c *FiberContext) XML(code int, content any) error {
	c.Status(code)
	return c.ctx.XML(content)
}

func (c *FiberContext) SendString(s string) error {
	return c.ctx.SendString(s)
}

func (c *FiberContext) Write(p []byte) (int, error) {
	return c.ctx.Write(p)
}

func (c *FiberContext) JSON(statusCode int, data any) error {
	return c.ctx.Status(statusCode).JSON(data)
}

// customRecoverHandler fiber自定义错误处理函数
func customRecoverHandler(c *fiber.Ctx, e any) {
	buf := make([]byte, 1024)
	buf = buf[:runtime.Stack(buf, true)]
	msg := helper.CombineStrings(
		"Request RelativePath: ", c.Path(), fmt.Sprintf(", Error: %v, \n", e), string(buf),
	)
	fastapi.Logger().Error(msg)
}

// customFiberErrorHandler 自定义fiber接口错误处理函数
func customFiberErrorHandler(c *fiber.Ctx, e error) error {
	fastapi.Logger().Warn(helper.CombineStrings(
		"error happened during: '",
		c.Method(), ": ", c.Path(),
		"', Msg: ", e.Error(),
	))
	return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
		"code": fiber.StatusBadRequest,
		"msg":  e.Error()},
	)
}
