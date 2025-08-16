package fastapi

import (
	"fmt"
	"net/http"

	"github.com/Chendemo12/fastapi/openapi"
)

const staticPrefix = "internal/static/"

// 生成模型定义
func (f *Wrapper) registerRouteDoc() *Wrapper {
	// 注册路由组数据模型
	for _, group := range f.groupRouters {
		for _, route := range group.Routes() {
			f.openApi.RegisterFrom(route.Swagger())
		}
	}

	return f
}

// 注册 swagger 的文档路由
func (f *Wrapper) registerRouteHandle() *Wrapper {
	// =========== docs 在线调试页面
	err := f.Mux().BindRoute(http.MethodGet, openapi.DocumentUrl,
		func(ctx MuxContext) error {
			ctx.Header(openapi.HeaderContentType, string(openapi.MIMETextHTMLCharsetUTF8))
			return ctx.SendString(openapi.MakeSwaggerUiHtml(
				f.Config().Title,
				openapi.JsonUrl,
				openapi.SwaggerJsName,
				openapi.SwaggerCssName,
				openapi.FaviconName,
			))
		},
	)
	if err != nil {
		panic(fmt.Sprintf("bind openapi failed, method: 'GET', path: '%s', error: %v", openapi.DocumentUrl, err))
	}

	// =========== openapi 获取路由定义
	err = f.Mux().BindRoute(http.MethodGet, openapi.JsonUrl,
		func(ctx MuxContext) error {
			ctx.Header(openapi.HeaderContentType, string(openapi.MIMEApplicationJSONCharsetUTF8))
			_, err := ctx.Write(f.openApi.Schema())
			return err
		},
	)
	if err != nil {
		panic(fmt.Sprintf("bind openapi failed, method: 'GET', path: '%s', error: %v", openapi.JsonUrl, err))
	}

	// =========== redoc 纯文档页面
	err = f.Mux().BindRoute(http.MethodGet, openapi.ReDocumentUrl,
		func(ctx MuxContext) error {
			ctx.Header(openapi.HeaderContentType, string(openapi.MIMETextHTMLCharsetUTF8))
			return ctx.SendString(openapi.MakeRedocUiHtml(
				f.Config().Title,
				openapi.JsonUrl,
				openapi.RedocJsName,
				openapi.FaviconName,
			))
		},
	)
	if err != nil {
		panic(fmt.Sprintf("bind openapi failed, method: 'GET', path: '%s', error: %v", openapi.ReDocumentUrl, err))
	}

	// =========== 创建静态资源文件
	err = f.Mux().BindRoute(http.MethodGet, openapi.FaviconIcoName, querySwaggerFaviconIco)
	err = f.Mux().BindRoute(http.MethodGet, openapi.FaviconName, querySwaggerFaviconPng)
	err = f.Mux().BindRoute(http.MethodGet, openapi.SwaggerCssName, queryDocsUiCSS)
	err = f.Mux().BindRoute(http.MethodGet, openapi.SwaggerJsName, queryDocsUiJS)
	err = f.Mux().BindRoute(http.MethodGet, openapi.RedocJsName, queryRedocUiJS)

	return f
}

// 挂载 png 图标资源
func querySwaggerFaviconPng(c MuxContext) error {
	b, err := openapi.Asset(staticPrefix + openapi.FaviconName)
	if err != nil {
		return c.Redirect(http.StatusFound, openapi.SwaggerFaviconUrl) // 加载错误，重定向
	}

	// use asset data
	_, err = c.Write(b)
	return err
}

// 挂载 ico 图标资源
func querySwaggerFaviconIco(c MuxContext) error {
	b, err := openapi.Asset(staticPrefix + openapi.FaviconIcoName)
	if err != nil {
		return c.Redirect(http.StatusFound, openapi.SwaggerFaviconUrl)
	}

	_, err = c.Write(b)
	return err
}

// 挂载 docs/css 资源
func queryDocsUiCSS(c MuxContext) error {
	b, err := openapi.Asset(staticPrefix + openapi.SwaggerCssName)
	if err != nil {
		return c.Redirect(http.StatusFound, openapi.SwaggerCssUrl)
	}

	c.Status(http.StatusOK)
	c.Header(openapi.HeaderContentType, string(openapi.MIMETextCSSCharsetUTF8))

	_, err = c.Write(b)
	return err
}

// 挂载 docs/js 资源
func queryDocsUiJS(c MuxContext) error {
	b, err := openapi.Asset(staticPrefix + openapi.SwaggerJsName)
	if err != nil {
		return c.Redirect(http.StatusFound, openapi.SwaggerJsUrl)
	}

	c.Status(http.StatusOK)
	c.Header(openapi.HeaderContentType, string(openapi.MIMETextJavaScriptCharsetUTF8))

	_, err = c.Write(b)
	return err
}

// 挂载 redoc/js 资源
func queryRedocUiJS(c MuxContext) error {
	b, err := openapi.Asset(staticPrefix + openapi.RedocJsName)
	if err != nil {
		return c.Redirect(http.StatusFound, openapi.RedocJsUrl)
	}

	c.Status(http.StatusOK)
	c.Header(openapi.HeaderContentType, string(openapi.MIMETextJavaScriptCharsetUTF8))

	_, err = c.Write(b)
	return err
}
