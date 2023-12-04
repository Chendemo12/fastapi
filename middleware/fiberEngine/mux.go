package fiberEngine

import (
	"fmt"
	"github.com/Chendemo12/fastapi-tool/helper"
	"github.com/gofiber/fiber/v2"
	"runtime"
)

type FiberMux struct {
}

func (f *FiberMux) BindRoute(method, path string, handler any) error {
	//TODO implement me
	panic("implement me")
}

func (f *FiberMux) BodyParser(ctx any, model any) error {
	//TODO implement me
	panic("implement me")
}

func (f *FiberMux) Shutdown() error {
	//TODO implement me
	panic("implement me")
}

func (f *FiberMux) Listen(addr string) error {
	//TODO implement me
	panic("implement me")
}

func (f *FiberMux) SetErrorHandler(handler any) {
	//TODO implement me
	panic("implement me")
}

func (f *FiberMux) SetRecoverHandler(handler any) {
	//TODO implement me
	panic("implement me")
}

func New(title, version string) *FiberMux { return &FiberMux{} }

// f 创建 fiber.App 已做了基本的中间件配置
func f(title, version string) *fiber.App {
	// fc fiber.ErrorHandler
	if fiberErrorHandler == nil {
		fiberErrorHandler = customFiberErrorHandler
	}

	if recoverHandler == nil {
		recoverHandler = customRecoverHandler
	}

	// 创建App实例
	app := fiber.New(fiber.Config{
		Prefork:       false,                  // core.MultipleProcessEnabled, // 多进程模式
		CaseSensitive: true,                   // 区分路由大小写
		StrictRouting: true,                   // 严格路由
		ServerHeader:  title,                  // 服务器头
		AppName:       title + " v" + version, // 设置为 Response.Header.Server 属性
		ColorScheme:   fiber.DefaultColors,    // 彩色输出
		JSONEncoder:   helper.JsonMarshal,     // json序列化器
		JSONDecoder:   helper.JsonUnmarshal,   // json解码器
		ErrorHandler:  fiberErrorHandler,      // 设置自定义错误处理
	})

	// 输出API访问日志
	echoConfig := echo.ConfigDefault
	echoConfig.TimeFormat = "2006/01/02 15:04:05"
	echoConfig.Format = "${time}    ${method}${path} ${status}\n"
	app.Use(echo.New(echoConfig))

	// 自定义全局 recover 方法
	app.Use(recover.New(recover.Config{
		EnableStackTrace: true,
		// StackTraceHandler: 处理堆栈跟踪的函数, 若留空，则默认将整个错误堆栈输出到控制台,
		// 并在处理完成后将错误流转到 fiber.ErrorHandler
		StackTraceHandler: recoverHandler,
	}))

	return app
}

// customRecoverHandler 自定义 recover 错误处理函数
func customRecoverHandler(c *fiber.Ctx, e any) {
	buf := make([]byte, 1024)
	buf = buf[:runtime.Stack(buf, true)]
	msg := helper.CombineStrings(
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
