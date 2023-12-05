package ginEngine

import (
	"github.com/gin-gonic/gin"
	_ "github.com/gin-gonic/gin"
)

type GinMux struct {
	App *gin.Engine
}

type GinContext struct {
	ctx *gin.Context
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

func (c *GinContext) Query(key string) string {
	return c.ctx.Query(key)
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
