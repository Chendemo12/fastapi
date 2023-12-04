package basic

import (
	"github.com/Chendemo12/fastapi"
	"github.com/Chendemo12/fastapi/pathschema"
)

// NewBaseRouter 用于获取后端服务基本信息的路由组
//
//	# Usage
//
//	router := NewBaseRouter("FastApi", "1.0.0", "FastApi application", false)
//	app.IncludeRouter(router)
func NewBaseRouter(title, version, desc string, debug bool) fastapi.GroupRouter {
	return &BaseGroupRouter{
		Title:   title,
		Version: version,
		Desc:    desc,
		Debug:   debug,
	}
}

type BaseGroupRouter struct {
	fastapi.BaseRouter
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

func (r *BaseGroupRouter) GetTitle(c *fastapi.Context) (string, error) {
	return r.Title, nil
}

func (r *BaseGroupRouter) GetDescription(c *fastapi.Context) (string, error) {
	return r.Desc, nil
}

func (r *BaseGroupRouter) GetVersion(c *fastapi.Context) (string, error) {
	return r.Version, nil
}

func (r *BaseGroupRouter) GetDebug(c *fastapi.Context) (bool, error) {
	return r.Debug, nil
}

func (r *BaseGroupRouter) GetHeartbeat(c *fastapi.Context) (string, error) {
	return "pong", nil
}
