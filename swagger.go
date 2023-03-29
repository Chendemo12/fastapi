package fastapi

import (
	"bytes"
	"github.com/Chendemo12/fastapi/internal/core"
	"github.com/Chendemo12/fastapi/openapi"
	"github.com/Chendemo12/fastapi/tool"
	"github.com/gofiber/fiber/v2"
	"net/http"
)

const staticPrefix = "internal/static/"

func (f *FastApi) createOpenApiDoc() {
	// 不允许创建swag文档
	if tool.All(!core.IsDebug(), core.SwaggerDisabled) {
		return
	}

	f.service.openApi = openapi.NewOpenApi(f.title, f.version, f.Description())

	f.createDefines()
	f.createPaths()
	f.createSwaggerRoutes()
	f.createStaticRoutes()
}

// 注册 swagger 的文档路由
func (f *FastApi) createSwaggerRoutes() {
	// docs 在线调试页面
	f.engine.Get("/docs", func(c *fiber.Ctx) error {
		c.Set(openapi.HeaderContentType, openapi.MIMETextHTML)
		return c.SendString(openapi.MakeSwaggerUiHtml(
			f.title,
			openapi.JsonUrl,
			openapi.SwaggerJsName,
			openapi.SwaggerCssName,
			openapi.FaviconName,
		))
	})

	// redoc 纯文档页面
	f.engine.Get("/redoc", func(c *fiber.Ctx) error {
		c.Set(openapi.HeaderContentType, openapi.MIMETextHTML)
		return c.SendString(openapi.MakeRedocUiHtml(
			f.title,
			openapi.JsonUrl,
			openapi.RedocJsName,
			openapi.FaviconName,
		))
	})

	// openapi 获取路由定义
	f.engine.Get("/openapi.json", func(c *fiber.Ctx) error {
		c.Set(openapi.HeaderContentType, openapi.MIMEApplicationJSONCharsetUTF8)
		return c.SendStream(bytes.NewReader(f.service.openApi.Schema()))
	})
}

// 生成模型定义
func (f *FastApi) createDefines() {
	for _, router := range f.APIRouters() {
		for _, route := range router.Routes() {
			if route.RequestModel != nil {
				// 内部会处理嵌入类型
				f.service.openApi.AddDefinition(route.RequestModel)
			}
			if route.ResponseModel != nil {
				f.service.openApi.AddDefinition(route.ResponseModel)
			}
		}
	}
}

// 生成路由定义
func (f *FastApi) createPaths() {
	for _, route := range f.service.cache {
		if route.Get != nil {
			routeToPathItem(route.Path, route.Get, f.service.openApi)
		}
		if route.Post != nil {
			routeToPathItem(route.Path, route.Post, f.service.openApi)
		}
		if route.Patch != nil {
			routeToPathItem(route.Path, route.Patch, f.service.openApi)
		}
		if route.Delete != nil {
			routeToPathItem(route.Path, route.Delete, f.service.openApi)
		}
		if route.Put != nil {
			routeToPathItem(route.Path, route.Put, f.service.openApi)
		}
	}
}

// 创建静态资源文件
func (f *FastApi) createStaticRoutes() {
	f.engine.Get(openapi.FaviconIcoName, querySwaggerFaviconIco)
	f.engine.Get(openapi.FaviconName, querySwaggerFaviconPng)

	f.engine.Get(openapi.SwaggerCssName, queryDocsUiCSS)
	f.engine.Get(openapi.SwaggerJsName, queryDocsUiJS)

	f.engine.Get(openapi.RedocJsName, queryRedocUiJS)
}

// 挂载 png 图标资源
func querySwaggerFaviconPng(c *fiber.Ctx) error {
	b, err := openapi.Asset(staticPrefix + openapi.FaviconName)
	if err != nil {
		return c.Redirect(openapi.SwaggerFaviconUrl) // 加载错误，重定向
	}
	// use asset data
	return c.SendStream(bytes.NewReader(b))
}

// 挂载 ico 图标资源
func querySwaggerFaviconIco(c *fiber.Ctx) error {
	b, err := openapi.Asset(staticPrefix + openapi.FaviconIcoName)
	if err != nil {
		return c.Redirect(openapi.SwaggerFaviconUrl)
	}
	return c.SendStream(bytes.NewReader(b))
}

// 挂载 docs/css 资源
func queryDocsUiCSS(c *fiber.Ctx) error {
	b, err := openapi.Asset(staticPrefix + openapi.SwaggerCssName)
	if err != nil {
		return c.Redirect(openapi.SwaggerCssUrl)
	}
	c.Status(200).Set(openapi.HeaderContentType, openapi.MIMETextCSSCharsetUTF8)
	return c.SendStream(bytes.NewReader(b))
}

// 挂载 docs/js 资源
func queryDocsUiJS(c *fiber.Ctx) error {
	b, err := openapi.Asset(staticPrefix + openapi.SwaggerJsName)
	if err != nil {
		return c.Redirect(openapi.SwaggerJsUrl)
	}
	c.Status(200).Set(openapi.HeaderContentType, openapi.MIMETextJavaScriptCharsetUTF8)
	return c.SendStream(bytes.NewReader(b))
}

// 挂载 redoc/js 资源
func queryRedocUiJS(c *fiber.Ctx) error {
	b, err := openapi.Asset(staticPrefix + openapi.RedocJsName)
	if err != nil {
		return c.Redirect(openapi.RedocJsUrl)
	}
	c.Status(200).Set(openapi.HeaderContentType, openapi.MIMETextJavaScriptCharsetUTF8)
	return c.SendStream(bytes.NewReader(b))
}

func routeToPathItem(path string, route *Route, api *openapi.OpenApi) {
	// 存在相同路径，不同方法的路由选项
	item := api.QueryPathItem(path)

	// 构造路径参数
	pathParams := make([]*openapi.Parameter, len(route.PathFields))
	for no, q := range route.PathFields {
		p := openapi.QModelToParameter(q)
		p.Deprecated = route.deprecated

		pathParams[no] = p
	}

	// 构造查询参数
	queryParams := make([]*openapi.Parameter, len(route.QueryFields))
	for no, q := range route.QueryFields {
		p := openapi.QModelToParameter(q)
		p.Deprecated = route.deprecated
		queryParams[no] = p
	}

	// 构造操作符
	operation := &openapi.Operation{
		Summary:     route.Summary,
		Description: route.Description,
		Tags:        route.Tags,
		Parameters:  append(pathParams, queryParams...),
		RequestBody: openapi.MakeOperationRequestBody(route.RequestModel),
		Responses:   openapi.MakeOperationResponses(route.ResponseModel),
		Deprecated:  route.deprecated,
	}

	// 绑定到操作方法
	switch route.Method {

	case http.MethodPost:
		item.Post = operation
	case http.MethodPut:
		item.Put = operation
	case http.MethodDelete:
		item.Delete = operation
	case http.MethodPatch:
		item.Patch = operation
	case http.MethodHead:
		item.Head = operation
	case http.MethodTrace:
		item.Trace = operation

	default:
		item.Get = operation
	}
}
