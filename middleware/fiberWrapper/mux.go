package fiberWrapper

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"runtime"
	"sync"
	"time"

	"github.com/Chendemo12/fastapi"
	"github.com/Chendemo12/fastapi/utils"
	"github.com/gofiber/fiber/v2"
	echo "github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/recover"
)

var pool = &sync.Pool{New: func() any {
	return &FiberContext{Context: *fastapi.NewContext()}
}}

func AcquireCtx(c *fiber.Ctx) *FiberContext {
	obj := pool.Get().(*FiberContext)
	obj.fiberCtx = c
	obj.once = sync.Once{}

	return obj
}

func ReleaseCtx(c *FiberContext) {
	c.fiberCtx = nil
	pool.Put(c)
}

type FiberMux struct {
	app *fiber.App
}

// NewWrapper 创建App实例
func NewWrapper(app *fiber.App) *FiberMux {
	return &FiberMux{
		app: app,
	}
}

// Default 默认的fiber.app，已做好基本的参数配置
func Default(cf ...fiber.Config) *FiberMux {
	var conf fiber.Config
	if len(cf) == 0 {
		conf = fiber.Config{
			Prefork:       false,                   // 多进程模式
			CaseSensitive: true,                    // 区分路由大小写
			StrictRouting: true,                    // 严格路由
			ServerHeader:  "FastApi",               // 服务器头
			AppName:       "fastapi.fiber",         // 设置为 Response.Header.Server 属性
			ColorScheme:   fiber.DefaultColors,     // 彩色输出
			JSONEncoder:   utils.JsonMarshal,       // json序列化器
			JSONDecoder:   utils.JsonUnmarshal,     // json解码器
			ErrorHandler:  customFiberErrorHandler, // 设置自定义错误处理
			BodyLimit:     100 * 1024 * 1024,       // 设置请求体最大为 100MB
		}
	} else {
		conf = cf[0]
	}
	app := fiber.New(conf)

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

func (m *FiberMux) Listen(addr string) error {
	return m.app.Listen(addr)
}

func (m *FiberMux) ShutdownWithTimeout(timeout time.Duration) error {
	return m.app.ShutdownWithTimeout(timeout)
}

func (m *FiberMux) BindRoute(method, path string, handler fastapi.MuxHandler) error {
	wrapper := func(ctx *fiber.Ctx) error {
		mCtx := AcquireCtx(ctx)
		defer ReleaseCtx(mCtx)
		return handler(mCtx)
	}

	switch method {
	case http.MethodGet:
		m.app.Get(path, wrapper)
	case http.MethodPost:
		m.app.Post(path, wrapper)
	case http.MethodDelete:
		m.app.Delete(path, wrapper)
	case http.MethodPatch:
		m.app.Patch(path, wrapper)
	case http.MethodPut:
		m.app.Put(path, wrapper)
	default:
		return fmt.Errorf("unknown method: '%s' for path: '%s'", method, path)
	}

	return nil
}

type FiberContext struct {
	fastapi.Context            // 嵌入，共用内存
	fiberCtx        *fiber.Ctx // 原始 fiber 上下文
	once            sync.Once
	sseChan         chan *fastapi.SSE
}

func (c *FiberContext) FastApiContext() *fastapi.Context { return &c.Context }

func (c *FiberContext) Method() string { return c.fiberCtx.Method() }

func (c *FiberContext) Path() string { return c.fiberCtx.Route().Path }

func (c *FiberContext) Ctx() any { return c.fiberCtx }

func (c *FiberContext) Done() <-chan struct{} {
	return c.fiberCtx.Context().Done()
}

func (c *FiberContext) RequestContext() context.Context {
	return c.fiberCtx.Context()
}

func (c *FiberContext) ClientIP() string { return c.fiberCtx.IP() }

func (c *FiberContext) Query(key string, undefined ...string) string {
	return c.fiberCtx.Query(key, undefined...)
}

func (c *FiberContext) Params(key string, undefined ...string) string {
	return c.fiberCtx.Params(key, undefined...)
}

func (c *FiberContext) MultipartForm() (*multipart.Form, error) {
	return c.fiberCtx.MultipartForm()
}

// GetHeader 获取请求头, 当key不存在时返回空字符串
func (c *FiberContext) GetHeader(key string) string {
	return c.fiberCtx.Get(key)
}

func (c *FiberContext) Cookie(name string) (string, error) {
	return c.fiberCtx.Cookies(name, ""), nil
}

func (c *FiberContext) ContentType() string {
	return string(c.fiberCtx.Context().Request.Header.ContentType())
}

func (c *FiberContext) ShouldBind(obj any) (validated bool, err error) {
	// fiber 没有校验方法，因此需返回 false
	return false, c.fiberCtx.BodyParser(obj)
}

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
	c.fiberCtx.Cookie(ck)
}

func (c *FiberContext) Status(statusCode int) { c.fiberCtx.Status(statusCode) }

func (c *FiberContext) SendStream(stream io.Reader, size ...int) error {
	return c.fiberCtx.SendStream(stream, size...)
}

func (c *FiberContext) Header(key, value string) { c.fiberCtx.Set(key, value) }

func (c *FiberContext) Redirect(code int, location string) error {
	return c.fiberCtx.Redirect(location, code)
}

func (c *FiberContext) File(filepath string) error {
	return c.fiberCtx.SendFile(filepath)
}

func (c *FiberContext) FileAttachment(filepath, filename string) error {
	c.fiberCtx.Attachment(filename)
	return c.fiberCtx.SendFile(filepath)
}

func (c *FiberContext) SendString(s string) error {
	return c.fiberCtx.SendString(s)
}

func (c *FiberContext) Write(p []byte) (int, error) {
	return c.fiberCtx.Write(p)
}

func (c *FiberContext) JSON(statusCode int, data any) error {
	return c.fiberCtx.Status(statusCode).JSON(data)
}

func (c *FiberContext) SSE(message *fastapi.SSE) (err error) {
	c.once.Do(func() {
		c.sseChan = make(chan *fastapi.SSE, 1)
		ctx := c.fiberCtx    // 在 goroutine 启动前捕获，防止 ReleaseCtx 置 nil
		done := c.Done()     // 在 goroutine 启动前捕获 channel
		sseChan := c.sseChan // 捕获 channel 引用，防止池复用后被覆盖

		go func() {
			ctx.Context().SetBodyStreamWriter(func(w *bufio.Writer) {
				// 立即刷新头部 (Flush the headers first)
				_ = w.Flush()

				for {
					select {
					case <-done:
						return
					case sse := <-sseChan:
						fastapi.Warnf("receive message")
						_, err = w.Write([]byte(sse.ToBuilder().String()))
						if err != nil {
							fastapi.Errorf("[sse] write stream failed, %s", err)
						} else {
							err = w.Flush() // todo: not work
							if err != nil {
								fastapi.Errorf("[sse] flush stream failed, %s", err)
							}
						}
					}
				}
			})
		}()
	})

	c.sseChan <- message

	return nil
}

// customRecoverHandler fiber自定义错误处理函数
func customRecoverHandler(c *fiber.Ctx, e any) {
	buf := make([]byte, 1024)
	buf = buf[:runtime.Stack(buf, true)]
	fastapi.Errorf("%s %s failed, Error: %s", c.Method(), c.Path(), string(buf))
}

// customFiberErrorHandler 自定义fiber接口错误处理函数
func customFiberErrorHandler(c *fiber.Ctx, e error) error {
	fastapi.Warnf("%s %s, Error: %s", c.Method(), c.Path(), e.Error())
	return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
		"code": fiber.StatusBadRequest,
		"msg":  e.Error()},
	)
}
