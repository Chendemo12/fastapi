package fiberEngine

import (
	"errors"
	"fmt"
	"github.com/Chendemo12/fastapi"
	"github.com/Chendemo12/fastapi-tool/helper"
	"github.com/gofiber/fiber/v2"
	echo "github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"net"
	"net/http"
	"runtime"
	"sync"
	"time"
)

type FiberMux struct {
	one     *sync.Once
	App     *fiber.App
	Title   string
	Version string
	// fiber自定义错误处理函数
	ErrorHandler fiber.ErrorHandler
	// StackTraceHandlerFunc 错误堆栈处理函数, 即 recover 方法
	StackTraceHandlerFunc func(c *fiber.Ctx, e any)
}

// New 创建App实例
func New(title, version string) *FiberMux {
	return &FiberMux{
		one:                   &sync.Once{},
		Title:                 title,
		Version:               version,
		ErrorHandler:          customFiberErrorHandler,
		StackTraceHandlerFunc: customRecoverHandler,
	}
}

func (m *FiberMux) Listen(addr string) error {
	return m.App.Listen(addr)
}

func (m *FiberMux) ShutdownWithTimeout(timeout time.Duration) error {
	return m.App.ShutdownWithTimeout(timeout)
}

func (m *FiberMux) SetErrorHandler(handler any) {
	m.ErrorHandler = func(ctx *fiber.Ctx, err error) error {
		return nil
	}
}

func (m *FiberMux) SetRecoverHandler(handler any) {
	// TODO:
	m.StackTraceHandlerFunc = func(c *fiber.Ctx, e any) {
		return
	}
}

func (m *FiberMux) BindRoute(method, path string, handler func(ctx fastapi.MuxCtx) error) error {
	m.one.Do(func() {
		app := fiber.New(fiber.Config{
			Prefork:       false,                      // core.MultipleProcessEnabled, // 多进程模式
			CaseSensitive: true,                       // 区分路由大小写
			StrictRouting: true,                       // 严格路由
			ServerHeader:  m.Title,                    // 服务器头
			AppName:       m.Title + " v" + m.Version, // 设置为 Response.Header.Server 属性
			ColorScheme:   fiber.DefaultColors,        // 彩色输出
			JSONEncoder:   helper.JsonMarshal,         // json序列化器
			JSONDecoder:   helper.JsonUnmarshal,       // json解码器
			ErrorHandler:  m.ErrorHandler,             // 设置自定义错误处理
		})

		// 输出API访问日志
		echoConfig := echo.ConfigDefault
		echoConfig.TimeFormat = time.DateTime
		echoConfig.Format = "${time}    ${method}\t${path} ${status}\n"
		app.Use(echo.New(echoConfig))

		// 自定义全局 recover 方法
		app.Use(recover.New(recover.Config{
			EnableStackTrace: true,
			// StackTraceHandler: 处理堆栈跟踪的函数, 若留空，则默认将整个错误堆栈输出到控制台,
			// 并在处理完成后将错误流转到 fiber.ErrorHandler
			StackTraceHandler: m.StackTraceHandlerFunc,
		}))

		m.App = app
	})

	switch method {
	case http.MethodGet:
		m.App.Get(path, func(ctx *fiber.Ctx) error {
			return handler(&FiberContext{ctx: ctx})
		})
	case http.MethodPost:
		m.App.Post(path, func(ctx *fiber.Ctx) error {
			return handler(&FiberContext{ctx: ctx})
		})
	case http.MethodDelete:
		m.App.Delete(path, func(ctx *fiber.Ctx) error {
			return handler(&FiberContext{ctx: ctx})
		})
	case http.MethodPatch:
		m.App.Patch(path, func(ctx *fiber.Ctx) error {
			return handler(&FiberContext{ctx: ctx})
		})
	case http.MethodPut:
		m.App.Put(path, func(ctx *fiber.Ctx) error {
			return handler(&FiberContext{ctx: ctx})
		})
	}

	return errors.New(fmt.Sprintf("unknow method:'%s' for path: '%s'", method, path))
}

type FiberContext struct {
	ctx *fiber.Ctx
}

func (c *FiberContext) Header(key, value string) {
	//TODO implement me
	panic("implement me")
}

func (c *FiberContext) Redirect(code int, location string) error {
	//TODO implement me
	panic("implement me")
}

func (c *FiberContext) JSONP(code int, data any) error {
	//TODO implement me
	panic("implement me")
}

func (c *FiberContext) File(filepath string) error {
	//TODO implement me
	panic("implement me")
}

func (c *FiberContext) FileAttachment(filepath, filename string) error {
	//TODO implement me
	panic("implement me")
}

func (c *FiberContext) RemoteAddr() net.Addr {
	//TODO implement me
	panic("implement me")
}

func (c *FiberContext) Set(key string, value any) {
	//TODO implement me
	panic("implement me")
}

func (c *FiberContext) XML(content any) error {
	return c.ctx.XML(content)
}

func (c *FiberContext) SendString(s string) error {
	return c.ctx.SendString(s)
}

func (c *FiberContext) SetHeader(key, value string) {
	c.ctx.Set(key, value)
}

func (c *FiberContext) Method() string {
	return c.ctx.Method()
}

func (c *FiberContext) Path() string {
	return c.ctx.Route().Path
}

func (c *FiberContext) BodyParser(model any) error {
	return c.ctx.BodyParser(model)
}

func (c *FiberContext) Query(key string) string {
	return c.ctx.Query(key, "")
}

func (c *FiberContext) Status(statusCode int) {
	c.ctx.Status(statusCode)
}

func (c *FiberContext) Write(p []byte) (int, error) {
	return c.ctx.Write(p)
}

func (c *FiberContext) JSON(statusCode int, data any) error {
	return c.ctx.Status(statusCode).JSON(data)
}

// customRecoverHandler 自定义 recover 错误处理函数
func customRecoverHandler(c *fiber.Ctx, e any) {
	buf := make([]byte, 1024)
	buf = buf[:runtime.Stack(buf, true)]
	_ = helper.CombineStrings(
		"Request RelativePath: ", c.Path(), fmt.Sprintf(", Error: %v, \n", e), string(buf),
	)
	//wrapper.Service().Logger().Error(msg)
}

// customFiberErrorHandler 自定义fiber接口错误处理函数
func customFiberErrorHandler(c *fiber.Ctx, e error) error {
	//wrapper.logger.Warn(helper.CombineStrings(
	//	"error happened during: '",
	//	c.Method(), ": ", c.RelativePath(),
	//	"', Msg: ", e.Error(),
	//))
	return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
		"code": fiber.StatusBadRequest,
		"msg":  e.Error()},
	)
}
