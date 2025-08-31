package ginWrapper

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"sync"
	"time"

	"github.com/Chendemo12/fastapi"
	"github.com/gin-gonic/gin"
)

var pool = &sync.Pool{New: func() any { return &GinContext{} }}

func AcquireCtx(c *gin.Context) *GinContext {
	obj := pool.Get().(*GinContext)
	obj.ctx = c

	return obj
}

func ReleaseCtx(c *GinContext) {
	c.ctx = nil
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
	switch method {
	case http.MethodGet:
		m.app.GET(path, func(c *gin.Context) {
			mCtx := AcquireCtx(c)
			defer ReleaseCtx(mCtx)

			err := handler(mCtx)
			if err != nil {
				// 通常情况下此方法不会返回错误，如果发生错误，错误通常为写流错误
				_ = c.Error(err)
				c.Abort()
			}
		})
	case http.MethodPost:
		m.app.POST(path, func(c *gin.Context) {
			mCtx := AcquireCtx(c)
			defer ReleaseCtx(mCtx)

			err := handler(mCtx)
			if err != nil {
				_ = c.Error(err)
				c.Abort()
			}
		})
	case http.MethodPatch:
		m.app.PATCH(path, func(c *gin.Context) {
			mCtx := AcquireCtx(c)
			defer ReleaseCtx(mCtx)

			err := handler(mCtx)
			if err != nil {
				_ = c.Error(err)
				c.Abort()
			}
		})
	case http.MethodPut:
		m.app.PUT(path, func(c *gin.Context) {
			mCtx := AcquireCtx(c)
			defer ReleaseCtx(mCtx)

			err := handler(mCtx)
			if err != nil {
				_ = c.Error(err)
				c.Abort()
			}
		})
	case http.MethodDelete:
		m.app.DELETE(path, func(c *gin.Context) {
			mCtx := AcquireCtx(c)
			defer ReleaseCtx(mCtx)

			err := handler(mCtx)
			if err != nil {
				_ = c.Error(err)
				c.Abort()
			}
		})
	default:
		return errors.New(fmt.Sprintf("unknow method:'%s' for path: '%s'", method, path))
	}

	return nil
}

type GinContext struct {
	ctx *gin.Context
}

func (c *GinContext) Method() string { return c.ctx.Request.Method }
func (c *GinContext) Path() string   { return c.ctx.FullPath() }

func (c *GinContext) Ctx() any { return c.ctx }

func (c *GinContext) Set(key string, value any) {
	c.ctx.Set(key, value)
}

func (c *GinContext) Get(key string) (value any, exists bool) {
	return c.ctx.Get(key)
}

func (c *GinContext) ClientIP() string { return c.ctx.RemoteIP() }

func (c *GinContext) ContentType() string {
	return c.ctx.ContentType()
}

// GetHeader 解析请求头参数
func (c *GinContext) GetHeader(key string) string {
	return c.ctx.GetHeader(key)
}

// Cookie 解析cookies参数
func (c *GinContext) Cookie(name string) (string, error) {
	return c.ctx.Cookie(name)
}

// Params 解析路径参数
func (c *GinContext) Params(key string, undefined ...string) string {
	value := c.ctx.Param(key)
	if value == "" && len(undefined) > 0 {
		return undefined[0]
	}
	return value
}

func (c *GinContext) Query(key string, undefined ...string) string {
	value := c.ctx.Query(key)
	if value == "" && len(undefined) > 0 {
		return undefined[0]
	}
	return value
}

func (c *GinContext) MultipartForm() (*multipart.Form, error) {
	return c.ctx.MultipartForm()
}

func (c *GinContext) ShouldBind(obj any) (validated bool, err error) {
	return true, c.ctx.ShouldBind(obj)
}

func (c *GinContext) Header(key, value string) {
	c.ctx.Header(key, value)
}

func (c *GinContext) SetCookie(cookie *http.Cookie) {
	c.ctx.SetCookie(cookie.Name, cookie.Value, cookie.MaxAge, cookie.Path, cookie.Domain, cookie.Secure, cookie.HttpOnly)
}

func (c *GinContext) Redirect(code int, location string) error {
	c.ctx.Redirect(code, location)
	return nil
}

func (c *GinContext) Status(statusCode int) {
	c.ctx.Status(statusCode)
}

func (c *GinContext) Write(p []byte) (int, error) {
	return c.ctx.Writer.Write(p)
}

func (c *GinContext) SendString(s string) error {
	c.ctx.String(http.StatusOK, "%s", s)
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
			_, err = c.ctx.Writer.Write(buf[:n])
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
		_, err = c.ctx.Writer.Write(buf.Bytes())
		if err != nil {
			return err
		}
	}
	return nil
}

func (c *GinContext) File(filepath string) error {
	c.ctx.File(filepath)
	return nil
}

func (c *GinContext) FileAttachment(filepath, filename string) error {
	c.ctx.FileAttachment(filepath, filename)
	return nil
}

func (c *GinContext) JSON(statusCode int, data any) error {
	c.ctx.JSON(statusCode, data)
	return nil
}

func (c *GinContext) FlushBody() {
	c.ctx.Writer.Flush()
}

func (c *GinContext) CloseNotify() <-chan bool {
	return c.ctx.Writer.CloseNotify()
}
