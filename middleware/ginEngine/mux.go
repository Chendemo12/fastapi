package ginEngine

import (
	"github.com/gin-gonic/gin"
	_ "github.com/gin-gonic/gin"
)

type GinContext struct {
	c *gin.Context
}
