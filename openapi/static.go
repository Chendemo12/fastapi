package openapi

import (
	"github.com/Chendemo12/fastapi/godantic"
	"net/http"
)

const ApiVersion = "3.1.0"

// 用于swagger的一些静态文件，来自FastApi
const (
	SwaggerCssName    = "swagger-ui.css"
	FaviconName       = "favicon.png"
	FaviconIcoName    = "favicon.ico"
	SwaggerJsName     = "swagger-ui-bundle.js"
	RedocJsName       = "redoc.standalone.js"
	JsonUrl           = "openapi.json"
	SwaggerFaviconUrl = "https://fastapi.tiangolo.com/img/" + FaviconName
	SwaggerCssUrl     = "https://cdn.jsdelivr.net/npm/swagger-ui-dist@5/" + SwaggerCssName
	SwaggerJsUrl      = "https://cdn.jsdelivr.net/npm/swagger-ui-dist@5/" + SwaggerJsName
	RedocJsUrl        = "https://cdn.jsdelivr.net/npm/redoc@next/bundles/" + RedocJsName
)

const (
	PathParamPrefix         = ":" // 路径参数起始字符
	PathSeparator           = "/" // 路径分隔符
	OptionalPathParamSuffix = "?" // 可选路径参数结束字符
)

const HeaderContentType = "Content-Type"
const (
	MIMETextXML                    string = "text/xml"
	MIMETextHTML                   string = "text/html"
	MIMETextPlain                  string = "text/plain"
	MIMETextCSS                    string = "text/css"
	MIMETextJavaScript             string = "text/javascript"
	MIMEApplicationXML             string = "application/xml"
	MIMEApplicationJSON            string = "application/json"
	MIMEApplicationForm            string = "application/x-www-form-urlencoded"
	MIMEOctetStream                string = "application/octet-stream"
	MIMEMultipartForm              string = "multipart/form-data"
	MIMETextXMLCharsetUTF8         string = "text/xml; charset=utf-8"
	MIMETextHTMLCharsetUTF8        string = "text/html; charset=utf-8"
	MIMETextPlainCharsetUTF8       string = "text/plain; charset=utf-8"
	MIMETextCSSCharsetUTF8         string = "text/css; charset=utf-8"
	MIMETextJavaScriptCharsetUTF8  string = "text/javascript; charset=utf-8"
	MIMEApplicationXMLCharsetUTF8  string = "application/xml; charset=utf-8"
	MIMEApplicationJSONCharsetUTF8 string = "application/json; charset=utf-8"
)

// 422 表单验证错误模型
var validationErrorDefinition = godantic.ValidationError{}

// 请求体相应体错误消息
var validationErrorResponseDefinition = godantic.HTTPValidationError{}

var Resp422 = &Response{
	StatusCode:  http.StatusUnprocessableEntity,
	Description: http.StatusText(http.StatusUnprocessableEntity),
	Content: &PathModelContent{
		MIMEType: MIMEApplicationJSON,
		Schema:   &godantic.ValidationError{},
	},
}
