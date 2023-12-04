package fastapi

type EngineMux interface {
	Shutdown() error
	Listen(addr string) error
	SetErrorHandler(handler any)
	SetRecoverHandler(handler any)
	BodyParser(ctx MuxCtx, model any) error
	BindRoute(method, path string, handler func(ctx MuxCtx) *Response) error
}

type MuxCtx interface {
	Path() string
	Method() string
}
