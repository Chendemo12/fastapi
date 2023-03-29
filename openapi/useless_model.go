package openapi

// ServerVariable 服务器变量(不常用)
type ServerVariable struct {
	Default     string   `json:"default" description:"默认值"`
	Description string   `json:"description" description:"说明"`
	Enum        []string `json:"enum" description:"可选项"`
}

// Server 服务器配置信息(不常用)
type Server struct {
	Url         string           `json:"url" description:"链接"`
	Description string           `json:"description" description:"说明"`
	Variables   []ServerVariable `json:"variables" description:""`
}

// Encoding 编码(不常用)
type Encoding struct {
	ContentType   string
	Style         string
	Headers       []Header
	Explode       bool
	AllowReserved bool
}

// MediaType 媒体类型(不常用)
type MediaType struct {
	Encoding map[string]Encoding
}

// Header 请求头参数,通常与认证相关(不常用)
type Header struct {
	Content     map[string]MediaType `json:"content" description:""`
	Description string               `json:"description" description:"说明"`
	Required    bool                 `json:"required" description:"是否必须"`
	Deprecated  bool                 `json:"deprecated" description:"是否禁用"`
}

type APIKeyIn string

const (
	APIKeyInQuery  APIKeyIn = "query"
	APIKeyInHeader APIKeyIn = "header"
	APIKeyInCookie APIKeyIn = "cookie"
)

type Link struct {
	Description  string         `json:"description" description:"说明"`
	OperationRef string         `json:"operationRef" description:""`
	OperationId  string         `json:"operationId" description:"唯一ID"`
	Parameters   map[string]any `json:"parameters" description:"路由参数"`
	RequestBody  RequestBody    `json:"requestBody" description:"请求"`
	Server       Server         `json:"server" description:""`
}

func (l Link) Alias() string { return "" }

type Tag struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

func (t Tag) Schema() map[string]string {
	return map[string]string{
		"name":        t.Name,
		"description": t.Description,
	}
}
