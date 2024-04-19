package fiberWrapper

import (
	"github.com/Chendemo12/fastapi/openapi"
	"github.com/gofiber/fiber/v2"
	"regexp"
)

func DefaultCORS(c *fiber.Ctx) error {
	c.Set("Access-Control-Allow-Origin", "*")
	c.Set("Access-Control-Allow-Headers", "*")
	c.Set("Access-Control-Allow-Credentials", "false")
	c.Set("Access-Control-Allow-Methods", "GET,POST,OPTIONS,DELETE,PATCH")

	if c.Method() == fiber.MethodOptions {
		c.Status(fiber.StatusOK)
		return nil
	}
	return c.Next()
}

var FastApiExcludePaths = []string{
	openapi.DocumentUrl,
	openapi.ReDocumentUrl,
	"/" + openapi.SwaggerCssName,
	"/" + openapi.FaviconName,
	"/" + openapi.FaviconIcoName,
	"/" + openapi.SwaggerJsName,
	"/" + openapi.RedocJsName,
	"/" + openapi.JsonUrl,
}

// NewAuthInterceptor 请求认证拦截器，验证请求是否需要认证，如果需要认证，则执行拦截器，否则继续执行
//	@param	excludePaths	[]string			排除的路径，如果请求路径匹配这些路径，则不执行拦截器
//	@param	itp				func(c *fiber.Ctx)	error	拦截器函数
//	@return	fiber.Handler
func NewAuthInterceptor(excludePaths []string, itp func(c *fiber.Ctx) error) func(c *fiber.Ctx) error {
	excludeExps := make([]regexp.Regexp, 0, len(excludePaths)+len(FastApiExcludePaths))
	for _, excludePattern := range excludePaths {
		excludeExps = append(excludeExps, *regexp.MustCompile(excludePattern))
	}
	for _, excludePattern := range FastApiExcludePaths {
		excludeExps = append(excludeExps, *regexp.MustCompile(excludePattern))
	}

	return func(c *fiber.Ctx) error {
		_path := c.Path() // 此处不能使用 c.Route().Path
		for _, excludeExp := range excludeExps {
			if excludeExp.MatchString(_path) {
				return c.Next()
			}
		}
		return itp(c)
	}
}
