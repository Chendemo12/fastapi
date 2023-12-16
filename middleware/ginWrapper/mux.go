package ginWrapper

import (
	"github.com/gin-gonic/gin"
	_ "github.com/gin-gonic/gin"
	"io"
	"net/http"
)

type GinMux struct {
	App *gin.Engine
}

type GinContext struct {
	ctx *gin.Context
}

func (c *GinContext) BindQuery(obj any) {
	//TODO implement me
	panic("implement me")
}

func (c *GinContext) BindQueryNotImplemented() bool {
	//TODO implement me
	panic("implement me")
}

func (c *GinContext) Header(key, value string) {
	//TODO implement me
	panic("implement me")
}

func (c *GinContext) Cookie(name string) (string, error) {
	//TODO implement me
	panic("implement me")
}

func (c *GinContext) Query(key string, undefined ...string) string {
	//TODO implement me
	panic("implement me")
}

func (c *GinContext) Params(key string, undefined ...string) string {
	//TODO implement me
	panic("implement me")
}

func (c *GinContext) Bind(obj any) error {
	//TODO implement me
	panic("implement me")
}

func (c *GinContext) Validate(obj any) error {
	//TODO implement me
	panic("implement me")
}

func (c *GinContext) ShouldBind(obj any) error {
	return c.ctx.ShouldBind(obj)
}

func (c *GinContext) ValidateNotImplemented() bool {
	return false
}

func (c *GinContext) Set(key string, value any) {
	//TODO implement me
	panic("implement me")
}

func (c *GinContext) Get(key string, defaultValue ...string) string {
	//TODO implement me
	panic("implement me")
}

func (c *GinContext) SetCookie(cookie *http.Cookie) {
	//TODO implement me
	panic("implement me")
}

func (c *GinContext) Redirect(code int, location string) error {
	//TODO implement me
	panic("implement me")
}

func (c *GinContext) ClientIP() string { return c.ctx.RemoteIP() }

func (c *GinContext) SendString(s string) error {
	//TODO implement me
	panic("implement me")
}

func (c *GinContext) SendStream(stream io.Reader, size ...int) error {
	//TODO implement me
	panic("implement me")
}

func (c *GinContext) JSONP(code int, data any) error {
	//TODO implement me
	panic("implement me")
}

func (c *GinContext) Render(name string, bind interface{}, layouts ...string) error {
	//TODO implement me
	panic("implement me")
}

func (c *GinContext) XML(code int, obj any) error {
	//TODO implement me
	panic("implement me")
}

func (c *GinContext) YAML(code int, obj any) error {
	//TODO implement me
	panic("implement me")
}

func (c *GinContext) TOML(code int, obj any) error {
	//TODO implement me
	panic("implement me")
}

func (c *GinContext) File(filepath string) error {
	//TODO implement me
	panic("implement me")
}

func (c *GinContext) FileAttachment(filepath, filename string) error {
	//TODO implement me
	panic("implement me")
}

func (c *GinContext) SetHeader(key, value string) {
	c.ctx.Header(key, value)
}

func (c *GinContext) Method() string {
	//TODO implement me
	panic("implement me")
}

func (c *GinContext) Path() string {
	//TODO implement me
	panic("implement me")
}

func (c *GinContext) BodyParser(model any) error {
	//TODO implement me
	panic("implement me")
}

func (c *GinContext) Status(statusCode int) {
	c.ctx.Status(statusCode)
}

func (c *GinContext) Write(p []byte) (int, error) {
	return 0, nil
}

func (c *GinContext) JSON(statusCode int, data any) error {
	c.ctx.JSON(statusCode, data)
	return nil
}
