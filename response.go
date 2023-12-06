package fastapi

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

// Response 路由返回值
type Response struct {
	Content     any          `json:"content"`     // 响应体
	ContentType string       `json:"contentType"` // 响应类型,默认为 application/json
	Type        ResponseType `json:"type"`        // 返回体类型
	StatusCode  int          `json:"status_code"` // 响应状态码
}
