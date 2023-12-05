package fastapi

import "time"

// EngineMux WEB服务器
type EngineMux interface {
	// Listen 启动http server
	Listen(addr string) error
	// ShutdownWithTimeout 优雅关闭
	ShutdownWithTimeout(timeout time.Duration) error
	// SetErrorHandler 设置错误处理方法
	SetErrorHandler(handler any)
	// SetRecoverHandler 设置全局recovery方法
	SetRecoverHandler(handler any)
	// BindRoute 注册路由
	BindRoute(method, path string, handler func(ctx MuxCtx) error) error
}

// MuxCtx Web引擎的 Context，例如 fiber.Ctx, gin.Context
type MuxCtx interface {
	Method() string              // 获得当前请求方法，取为为 http.MethodGet, http.MethodPost 等
	Path() string                // 获的当前请求的路由模式，而非请求Url
	SetHeader(key, value string) // 添加响应头
	BodyParser(model any) error  // 解析请求体
	Query(key string) string     // 解析查询参数
	Status(statusCode int)       // 设置响应状态码
	Write(p []byte) (int, error) // 写入响应字节流,当此方法执行完毕时应中断后续流程
	JSON(statusCode int, data any) error
	XML(content any) error
	SendString(s string) error // 写入json响应体,当此方法执行完毕时应中断后续流程
}
