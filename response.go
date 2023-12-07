package fastapi

import (
	"github.com/Chendemo12/fastapi/openapi"
	"net/http"
	"sync"
)

type ResponseType int

const (
	CustomResponseType ResponseType = iota + 1
	JsonResponseType
	StringResponseType
	StreamResponseType
	FileResponseType
	ErrResponseType
	HtmlResponseType
	AdvancedResponseType
)

// Response 范型路由返回值
// 也用于内部直接传递数据，对于 GroupRouteHandler 不应将其作为函数返回值
type Response struct {
	Content     any          `json:"content" description:"响应体"`
	ContentType string       `json:"-" description:"响应类型,默认为 application/json"`
	Type        ResponseType `json:"-" description:"返回体类型"`
	StatusCode  int          `json:"-" description:"响应状态码"`
}

var responsePool = &sync.Pool{New: func() any {
	return &Response{}
}}

func AcquireResponse() *Response {
	r := responsePool.Get().(*Response)
	r.Type = JsonResponseType
	r.ContentType = openapi.MIMEApplicationJSONCharsetUTF8
	r.StatusCode = http.StatusOK

	return r
}

func ReleaseResponse(resp *Response) {
	resp.Type = 0
	resp.ContentType = ""
	resp.StatusCode = 0
	resp.Content = nil

	responsePool.Put(resp)
}
