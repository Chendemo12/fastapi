package fastapi

func Get[T SchemaIface](path string, handle func(c *Context, query T) *Response, opt ...Option) *GRoute[T] {
	return &GRoute[T]{}
}

func Delete[T SchemaIface](path string, handle func(c *Context, query T) *Response, opt ...Option) *GRoute[T] {
	return &GRoute[T]{}
}

func Post[T SchemaIface](path string, handle func(c *Context, query T) *Response, opt ...Option) *GRoute[T] {
	var prototype T
	g := &GRoute[T]{
		handle:    handle,
		prototype: prototype,
	}
	// 添加到全局数组
	g.RelativePath = path
	return g
}

func Patch[T SchemaIface](path string, handle func(c *Context, query T) *Response, opt ...Option) *GRoute[T] {
	var prototype T
	g := &GRoute[T]{
		handle:    handle,
		prototype: prototype,
	}
	// 添加到全局数组
	g.RelativePath = path
	return g
}

type GRoute[T SchemaIface] struct {
	Route
	prototype T
	handle    func(c *Context, params T) *Response
}

// ================ example ================

type Clipboard struct {
	BaseModel
	Text string
}

func WriteClipboard(c *Context, req *Clipboard) *Response {
	return c.OKResponse("")
}

func init() {
	Post[*Clipboard]("/api/clipboard", WriteClipboard, Option{})
}
