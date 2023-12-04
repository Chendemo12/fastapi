package fastapi

import (
	"bytes"
	"github.com/Chendemo12/fastapi/openapi"
	"github.com/gofiber/fiber/v2"
)

const staticPrefix = "internal/static/"

// 创建openapi文档
func (f *FastApi) createOpenApiDoc() *FastApi {
	f.service.openApi = openapi.NewOpenApi(f.Config().Title, f.Config().Version, f.Config().Description)
	f.registerRouteDoc().registerRouteHandle()

	return f
}

// 生成模型定义
func (f *FastApi) registerRouteDoc() *FastApi {
	// 注册路由组数据模型
	for _, group := range f.groupRouters {
		for _, route := range group.Routes() {
			f.service.openApi.RegisterFrom(route.Swagger())
		}
	}

	return f
}

// 注册 swagger 的文档路由
func (f *FastApi) registerRouteHandle() *FastApi {
	// openapi 获取路由定义
	f.engine.Get("/openapi.json", func(c *fiber.Ctx) error {
		c.Set(openapi.HeaderContentType, openapi.MIMEApplicationJSONCharsetUTF8)
		return c.SendStream(bytes.NewReader(f.service.openApi.Schema()))
	})

	// docs 在线调试页面
	f.engine.Get("/docs", func(c *fiber.Ctx) error {
		c.Set(openapi.HeaderContentType, openapi.MIMETextHTMLCharsetUTF8)
		return c.SendString(openapi.MakeSwaggerUiHtml(
			f.Config().Title,
			openapi.JsonUrl,
			openapi.SwaggerJsName,
			openapi.SwaggerCssName,
			openapi.FaviconName,
		))
	})

	// redoc 纯文档页面
	f.engine.Get("/redoc", func(c *fiber.Ctx) error {
		c.Set(openapi.HeaderContentType, openapi.MIMETextHTMLCharsetUTF8)
		return c.SendString(openapi.MakeRedocUiHtml(
			f.Config().Title,
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
