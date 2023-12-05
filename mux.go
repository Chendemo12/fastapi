package fastapi

type EngineMux interface {
	Shutdown() error
	Listen(addr string) error
	SetErrorHandler(handler any)
	SetRecoverHandler(handler any)
	BodyParser(ctx MuxCtx, model any) error
	BindRoute(method, path string, handler func(ctx MuxCtx) error) error
}

// MuxCtx Web引擎的 Context，例如fiber.Ctx, gin.Context
type MuxCtx interface {
	Method() string          // 当前请求方法，取为为 http.MethodGet, http.MethodPost 等
	Path() string            // 获取当前请求的路由模式，而非请求Url
	Query(key string) string // 解析查询参数
	Status(code int)
	Send(code int, data any) error
	JSON(code int, obj any) error
	Write(p []byte) (int, error) // 写入消息响应
}
