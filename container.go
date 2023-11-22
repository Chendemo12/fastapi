package fastapi

import (
	"github.com/Chendemo12/fastapi/openapi"
	"github.com/gofiber/fiber/v2"
)

// Container 一个路由实例
type Container struct {
	kind int // 1: 泛型路由，2：路由组
}

type RouteIface interface {
	ResponseModel() *openapi.Metadata                // 响应体元数据
	RequestModel() *openapi.Metadata                 // 请求体元数据
	requestValidate() RouteModelValidateHandlerFunc  // 请求体校验函数
	responseValidate() RouteModelValidateHandlerFunc // 返回值校验函数
	Description() string                             // 详细描述
	Summary() string                                 // 摘要描述
	Method() string                                  // 请求方法
	RelativePath() string                            // 相对路由
	Tags() []string                                  // 路由标签
	QueryFields() []*openapi.QModel                  // 查询参数
	Handlers() []fiber.Handler                       // 处理函数
	Dependencies() []DependencyFunc                  // 依赖
	PathFields() []*openapi.QModel                   // 路径参数
	Deprecated() bool                                // 是否禁用
}

// RouteSwagger 路由组文档定义，所有路由实现的相同部分
type RouteSwagger struct {
	ResponseModel *openapi.Metadata `description:"响应体元数据"`
	RequestModel  *openapi.Metadata `description:"请求体元数据"`
	Description   string            `json:"description" description:"详细描述"`
	Summary       string            `json:"summary" description:"摘要描述"`
	Method        string            `json:"method" description:"请求方法"`
	RelativePath  string            `json:"relative_path" description:"相对路由"`
	Tags          []string          `json:"tags" description:"路由标签"`
	QueryFields   []*openapi.QModel `json:"-" description:"查询参数"`
	PathFields    []*openapi.QModel `json:"-" description:"路径参数"`
}
