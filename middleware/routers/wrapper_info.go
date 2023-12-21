package routers

import (
	"github.com/Chendemo12/fastapi"
	"github.com/Chendemo12/fastapi/pathschema"
)

// NewInfoRouter 用于获取后端服务基本信息的路由组
//
//	# Usage
//
//	router := NewInfoRouter(Config{})
//	app.IncludeRouter(router)
func NewInfoRouter(conf fastapi.Config) fastapi.GroupRouter {
	return &WrapperInfoRouter{
		Title:   conf.Title,
		Version: conf.Version,
		Desc:    conf.Description,
		Debug:   conf.Debug,
	}
}

type WrapperInfoRouter struct {
	fastapi.BaseRouter
	Title   string
	Version string
	Desc    string
	Debug   bool
}

func (r *WrapperInfoRouter) Prefix() string {
	return "/api"
}

func (r *WrapperInfoRouter) Tags() []string {
	return []string{"Base"}
}

func (r *WrapperInfoRouter) PathSchema() pathschema.RoutePathSchema {
	return pathschema.Default()
}

func (r *WrapperInfoRouter) Summary() map[string]string {
	return map[string]string{
		"GetTitle":       "获取软件名",
		"GetDescription": "获取软件描述信息",
		"GetVersion":     "获取软件版本号",
		"GetDebug":       "获取调试开关",
		"GetHeartbeat":   "心跳检测",
	}
}

func (r *WrapperInfoRouter) Description() map[string]string {
	return map[string]string{}
}

func (r *WrapperInfoRouter) GetTitle(c *fastapi.Context) (string, error) {
	return r.Title, nil
}

func (r *WrapperInfoRouter) GetDescription(c *fastapi.Context) (string, error) {
	return r.Desc, nil
}

func (r *WrapperInfoRouter) GetVersion(c *fastapi.Context) (string, error) {
	return r.Version, nil
}

func (r *WrapperInfoRouter) GetDebug(c *fastapi.Context) (bool, error) {
	return r.Debug, nil
}

func (r *WrapperInfoRouter) GetHeartbeat(c *fastapi.Context) (string, error) {
	return "pong", nil
}
