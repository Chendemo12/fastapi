package fastapi

import (
	"sync"
)

type ResponseType int

const (
	JsonResponseType ResponseType = iota + 1
	StringResponseType
	StreamResponseType
	FileResponseType
	HtmlResponseType
	AnyResponseType
)

// Response 范型路由返回值
// 也用于内部直接传递数据，对于 GroupRouteHandler 不应将其作为函数返回值
type Response struct {
	Content any `json:"content" description:"响应体"`
	// 默认情况下由 Type 类型决定
	//
	//	对于 JSON 类型（默认类型）, 为 application/json; charset=utf-8 不可更改
	//	对于 String 类型, 默认为 text/plain; charset=utf-8; 可以修改
	//	对于 Html 类型, 默认为 text/html; charset=utf-8; 不可修改
	//	对于 File 类型, 默认为 text/plain; 可以修改
	//	对于 Stream 类型, 无默认值; 可以修改
	ContentType string       `json:"-" description:"响应类型"`
	Type        ResponseType `json:"-" description:"返回体类型"`
	StatusCode  int          `json:"-" description:"响应状态码"`
}

var responsePool = &sync.Pool{New: func() any {
	return &Response{}
}}

func AcquireResponse() *Response {
	r := responsePool.Get().(*Response)
	r.Type = JsonResponseType
	r.StatusCode = 0

	return r
}

func ReleaseResponse(resp *Response) {
	resp.Type = 0
	resp.ContentType = ""
	resp.StatusCode = 0
	resp.Content = nil

	responsePool.Put(resp)
}
