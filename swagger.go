package fastapi

import (
	"bytes"
	"net/http"

	"github.com/Chendemo12/fastapi/openapi"
	"github.com/gofiber/fiber/v2"
)

const staticPrefix = "internal/static/"

// 创建openapi文档
func (f *FastApi) createOpenApiDoc() *FastApi {
	f.service.openApi = openapi.NewOpenApi(f.title, f.version, f.Description())

	f.registerModels().registerPaths().createSwaggerRoutes()

	return f
}

// 生成模型定义
func (f *FastApi) registerModels() *FastApi {
	// 注册路由组数据模型
	for _, group := range f.groupRouters {
		for _, route := range group.Routes() {
			route.swagger.RegisterTo(f.service.openApi.AddDefinition)
		}
	}

	return f
}

// 生成路由定义
func (f *FastApi) registerPaths() *FastApi {
	for _, group := range f.groupRouters {
		for _, route := range group.Routes() {
			// TODO:
			route.Swagger()
		}
	}

	//for _, route := range f.groupRouters {
	//	if route.Get != nil {
	//		routeToPathItem(route.Path, route.Get, f.service.openApi)
	//	}
	//	if route.Post != nil {
	//		routeToPathItem(route.Path, route.Post, f.service.openApi)
	//	}
	//	if route.Patch != nil {
	//		routeToPathItem(route.Path, route.Patch, f.service.openApi)
	//	}
	//	if route.Delete != nil {
	//		routeToPathItem(route.Path, route.Delete, f.service.openApi)
	//	}
	//	if route.Put != nil {
	//		routeToPathItem(route.Path, route.Put, f.service.openApi)
	//	}
	//}

	return f
}

// 注册 swagger 的文档路由
func (f *FastApi) createSwaggerRoutes() *FastApi {
	// openapi 获取路由定义
	f.engine.Get("/openapi.json", func(c *fiber.Ctx) error {
		c.Set(openapi.HeaderContentType, openapi.MIMEApplicationJSONCharsetUTF8)
		return c.SendStream(bytes.NewReader(f.service.openApi.Schema()))
	})

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

	// 创建静态资源文件
	f.engine.Get(openapi.FaviconIcoName, querySwaggerFaviconIco)
	f.engine.Get(openapi.FaviconName, querySwaggerFaviconPng)

	f.engine.Get(openapi.SwaggerCssName, queryDocsUiCSS)
	f.engine.Get(openapi.SwaggerJsName, queryDocsUiJS)

	f.engine.Get(openapi.RedocJsName, queryRedocUiJS)

	return f
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

func routeToPathItem(path string, route RouteIface, api *openapi.OpenApi) {
	// 存在相同路径，不同方法的路由选项
	item := api.AddPathItem(path)

	// 构造路径参数
	pathParams := make([]*openapi.Parameter, len(route.Swagger().PathFields))
	for no, q := range route.Swagger().PathFields {
		p := openapi.QModelToParameter(q)
		p.Deprecated = route.Swagger().Deprecated

		pathParams[no] = p
	}

	// 构造查询参数
	queryParams := make([]*openapi.Parameter, len(route.Swagger().QueryFields))
	for no, q := range route.Swagger().QueryFields {
		p := openapi.QModelToParameter(q)
		p.Deprecated = route.Swagger().Deprecated
		queryParams[no] = p
	}

	// 构造操作符
	operation := &openapi.Operation{
		Summary:     route.Swagger().Summary,
		Description: route.Swagger().Description,
		Tags:        route.Swagger().Tags,
		Parameters:  append(pathParams, queryParams...),
		RequestBody: openapi.MakeOperationRequestBody(route.Swagger().RequestModel),
		Responses:   openapi.MakeOperationResponses(route.Swagger().ResponseModel),
		Deprecated:  route.Swagger().Deprecated,
	}

	// 绑定到操作方法
	switch route.Swagger().Method {

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
