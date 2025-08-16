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
func NewInfoRouter(conf *fastapi.Config, prefix ...string) fastapi.GroupRouter {
	r := &WrapperInfoRouter{
		prefix:  "/api/base",
		Tag:     []string{"Base"},
		Title:   conf.Title,
		Version: conf.Version,
		Desc:    conf.Description,
	}
	if len(prefix) > 0 {
		r.prefix = prefix[0]
	}

	return r
}

type WrapperInfoRouter struct {
	fastapi.BaseGroupRouter
	prefix  string
	Title   string
	Version string
	Desc    string
	Tag     []string
}

func (r *WrapperInfoRouter) Prefix() string {
	return r.prefix
}

func (r *WrapperInfoRouter) Tags() []string { return r.Tag }

func (r *WrapperInfoRouter) PathSchema() pathschema.RoutePathSchema {
	return pathschema.Default()
}

func (r *WrapperInfoRouter) Summary() map[string]string {
	return map[string]string{
		"GetTitle":       "获取软件名",
		"GetDescription": "获取软件描述信息",
		"GetVersion":     "获取软件版本号",
		"GetHeartbeat":   "心跳检测",
	}
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

func (r *WrapperInfoRouter) GetHeartbeat(c *fastapi.Context) (string, error) {
	return "pong", nil
}
