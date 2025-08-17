package fastapi

import (
	"sync"
)

// Response 路由返回值
// 也用于内部直接传递数据，对于 GroupRouteHandler 不应将其作为函数返回值
type Response struct {
	StatusCode int
	Content    any
}

var responsePool = &sync.Pool{New: func() any {
	return &Response{}
}}

func AcquireResponse() *Response {
	r := responsePool.Get().(*Response)
	r.StatusCode = 0

	return r
}

func ReleaseResponse(resp *Response) {
	resp.StatusCode = 0
	resp.Content = nil

	responsePool.Put(resp)
}
