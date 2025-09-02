package test

import (
	"testing"

	"github.com/Chendemo12/fastapi"
	"github.com/Chendemo12/fastapi/middleware/ginWrapper"
)

func CreateApp(mux fastapi.MuxWrapper) *fastapi.Wrapper {
	// 可选的 fastapi.Config 参数
	app := fastapi.New(fastapi.Config{
		Version:     "v1.0.0",
		Description: "这是一段Http服务描述信息，会显示在openApi文档的顶部",
		Title:       "FastApi Example",
	})
	app.SetMux(mux)

	// 注册路由
	app.IncludeRouter(&ExampleRouter{})
	// 自定义错误格式
	app.SetRouteErrorFormatter(FormatErrorMessage, fastapi.RouteErrorOpt{
		StatusCode:   400,
		ResponseMode: &ErrorMessage{},
		Description:  "此状态代表服务器处理中遇上了错误",
	})
	app.UsePrevious(BeforeValidate)
	app.UseBeforeWrite(PrintRequestLog)

	return app
}

func TestExampleRouter(t *testing.T) {
	mux := ginWrapper.Default()
	app := CreateApp(mux)

	app.Run("0.0.0.0", "8090") // 阻塞运行
}
