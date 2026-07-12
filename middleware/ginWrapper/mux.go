package ginWrapper

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"sync"
	"time"

	"github.com/Chendemo12/fastapi"
	"github.com/gin-gonic/gin"
)

var pool = &sync.Pool{New: func() any {
	return &GinContext{Context: *fastapi.NewContext()}
}}

func AcquireCtx(c *gin.Context) *GinContext {
	obj := pool.Get().(*GinContext)
	obj.ginCtx = c

	return obj
}

func ReleaseCtx(c *GinContext) {
	c.ginCtx = nil
	pool.Put(c)
}

type GinMux struct {
	app *gin.Engine
	srv *http.Server
}

func Default() *GinMux {
	app := gin.Default()
	return NewWrapper(app)
}

// NewWrapper 创建App实例
func NewWrapper(app *gin.Engine) *GinMux {
	return &GinMux{
		app: app,
	}
}

func (m *GinMux) App() *gin.Engine { return m.app }

func (m *GinMux) Listen(addr string) error {
	srv := &http.Server{
		Addr:    addr,
		Handler: m.app.Handler(),
	}
	m.srv = srv
	return srv.ListenAndServe()
}

func (m *GinMux) ShutdownWithTimeout(timeout time.Duration) error {
	if m.srv == nil {
		return nil
	}

	// 关闭服务器
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	if err := m.srv.Shutdown(ctx); err != nil {
		return err
	}

	return nil
}

func (m *GinMux) BindRoute(method, path string, handler fastapi.MuxHandler) error {
	wrapper := func(c *gin.Context) {
		mCtx := AcquireCtx(c)
		defer ReleaseCtx(mCtx)

		err := handler(mCtx)
		if err != nil {
			_ = c.Error(err)
			c.Abort()
		}
	}

	switch method {
	case http.MethodGet:
		m.app.GET(path, wrapper)
	case http.MethodPost:
		m.app.POST(path, wrapper)
	case http.MethodPatch:
		m.app.PATCH(path, wrapper)
	case http.MethodPut:
		m.app.PUT(path, wrapper)
	case http.MethodDelete:
		m.app.DELETE(path, wrapper)
	default:
		return fmt.Errorf("unknown method: '%s' for path: '%s'", method, path)
	}

	return nil
}

type GinContext struct {
	fastapi.Context              // 嵌入，共用内存
	ginCtx          *gin.Context // 原始 gin 上下文
}

func (c *GinContext) FastApiContext() *fastapi.Context { return &c.Context }

func (c *GinContext) Method() string { return c.ginCtx.Request.Method }
func (c *GinContext) Path() string   { return c.ginCtx.FullPath() }

func (c *GinContext) Ctx() any { return c.ginCtx }

func (c *GinContext) Done() <-chan struct{} { return c.ginCtx.Done() }

func (c *GinContext) RequestContext() context.Context {
	return c.ginCtx.Request.Context()
}

func (c *GinContext) Set(key string, value any) {
	c.ginCtx.Set(key, value)
}

func (c *GinContext) Get(key string) (value any, exists bool) {
	return c.ginCtx.Get(key)
}

func (c *GinContext) ClientIP() string { return c.ginCtx.RemoteIP() }

func (c *GinContext) ContentType() string {
	return c.ginCtx.ContentType()
}

// GetHeader 解析请求头参数
func (c *GinContext) GetHeader(key string) string {
	return c.ginCtx.GetHeader(key)
}

// Cookie 解析cookies参数
func (c *GinContext) Cookie(name string) (string, error) {
	return c.ginCtx.Cookie(name)
}

// Params 解析路径参数
func (c *GinContext) Params(key string, undefined ...string) string {
	value := c.ginCtx.Param(key)
	if value == "" && len(undefined) > 0 {
		return undefined[0]
	}
	return value
}

func (c *GinContext) Query(key string, undefined ...string) string {
	value := c.ginCtx.Query(key)
	if value == "" && len(undefined) > 0 {
		return undefined[0]
	}
	return value
}

func (c *GinContext) MultipartForm() (*multipart.Form, error) {
	return c.ginCtx.MultipartForm()
}

func (c *GinContext) ShouldBind(obj any) (validated bool, err error) {
	return true, c.ginCtx.ShouldBind(obj)
}

func (c *GinContext) Header(key, value string) {
	c.ginCtx.Header(key, value)
}

func (c *GinContext) SetCookie(cookie *http.Cookie) {
	c.ginCtx.SetCookie(cookie.Name, cookie.Value, cookie.MaxAge, cookie.Path, cookie.Domain, cookie.Secure, cookie.HttpOnly)
}

func (c *GinContext) Redirect(code int, location string) error {
	c.ginCtx.Redirect(code, location)
	return nil
}

func (c *GinContext) Status(statusCode int) {
	c.ginCtx.Status(statusCode)
}

func (c *GinContext) Write(p []byte) (int, error) {
	return c.ginCtx.Writer.Write(p)
}

func (c *GinContext) SendString(s string) error {
	c.ginCtx.String(http.StatusOK, "%s", s)
	return nil
}

func (c *GinContext) SendStream(stream io.Reader, size ...int) error {
	if len(size) > 0 && size[0] >= 0 {
		// 读取指定字节数
		buf := make([]byte, size[0])
		for {
			n, err := stream.Read(buf)
			if err != nil {
				if err == io.EOF {
					break
				}
				return err
			}
			_, err = c.ginCtx.Writer.Write(buf[:n])
			if err != nil {
				return err
			}
		}
	} else {
		// 读取全部
		buf := new(bytes.Buffer)
		_, err := buf.ReadFrom(stream)
		if err != nil && err != io.EOF {
			return err
		}
		_, err = c.ginCtx.Writer.Write(buf.Bytes())
		if err != nil {
			return err
		}
	}
	return nil
}

func (c *GinContext) File(filepath string) error {
	c.ginCtx.File(filepath)
	return nil
}

func (c *GinContext) FileAttachment(filepath, filename string) error {
	c.ginCtx.FileAttachment(filepath, filename)
	return nil
}

func (c *GinContext) JSON(statusCode int, data any) error {
	c.ginCtx.JSON(statusCode, data)
	return nil
}

func (c *GinContext) SSE(message *fastapi.SSE) (err error) {
	_, err = c.ginCtx.Writer.WriteString(message.ToBuilder().String())
	if err != nil {
		return
	}
	c.ginCtx.Writer.Flush()

	return
}
