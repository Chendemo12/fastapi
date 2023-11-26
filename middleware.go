package fastapi

import (
	"github.com/Chendemo12/fastapi/pathschema"
	"github.com/gofiber/fiber/v2"
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

type BaseGroupRouter struct {
	Title   string
	Version string
	Desc    string
	Debug   bool
}

func (r *BaseGroupRouter) Prefix() string {
	return "/api"
}

func (r *BaseGroupRouter) Tags() []string {
	return []string{"Base"}
}

func (r *BaseGroupRouter) PathSchema() pathschema.RoutePathSchema {
	return pathschema.Default()
}

func (r *BaseGroupRouter) Summary() map[string]string {
	return map[string]string{
		"GetTitle":       "获取软件名",
		"GetDescription": "获取软件描述信息",
		"GetVersion":     "获取软件版本号",
		"GetDebug":       "获取调试开关",
		"GetHeartbeat":   "心跳检测",
	}
}

func (r *BaseGroupRouter) Description() map[string]string {
	return map[string]string{}
}

func (r *BaseGroupRouter) GetTitle(c *Context) (string, error) {
	return r.Title, nil
}

func (r *BaseGroupRouter) GetDescription(c *Context) (string, error) {
	return r.Desc, nil
}

func (r *BaseGroupRouter) GetVersion(c *Context) (string, error) {
	return r.Version, nil
}

func (r *BaseGroupRouter) GetDebug(c *Context) (bool, error) {
	return r.Debug, nil
}

func (r *BaseGroupRouter) GetHeartbeat(c *Context) (string, error) {
	return "pong", nil
}
