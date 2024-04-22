package fastapi

import (
	"io"
	"net/http"
	"time"
)

type MuxHandler func(c MuxContext) error

// MuxWrapper WEB服务器包装器接口
// 为兼容不同的 server 引擎，需要对其二次包装
type MuxWrapper interface {
	// Listen 启动http server
	Listen(addr string) error
	// ShutdownWithTimeout 优雅关闭
	ShutdownWithTimeout(timeout time.Duration) error
	// BindRoute 注册路由
	BindRoute(method, path string, handler MuxHandler) error
}

// MuxContext Web引擎的 Context，例如 fiber.Ctx, gin.Context
type MuxContext interface {
	Method() string                                // [重要方法]获得当前请求方法，取为为 http.MethodGet, http.MethodPost 等
	Path() string                                  // [重要方法]获的当前请求的路由模式，而非请求Url
	Ctx() any                                      // 原始的 Context
	ContentType() string                           // Content-Type
	Header(key, value string)                      // 添加响应头
	Cookie(name string) (string, error)            // 读取cookie
	Query(key string, undefined ...string) string  // 解析查询参数
	Params(key string, undefined ...string) string // 解析路径参数

	BodyParser(obj any) error       // 反序列化到 obj 上,作用等同于Unmarshal
	Validate(obj any) error         // 校验请求体, error应为 validator.ValidationErrors 类型
	ShouldBind(obj any) error       // 反序列化并校验请求体 = BodyParser + Validate, error 应为validator.ValidationErrors 类型
	BindQuery(obj any) error        // 绑定请求参数到 obj 上, error应为 validator.ValidationErrors 类型
	ShouldBindNotImplemented() bool // mux 没有实现 ShouldBind 方法, 此情况下会采用替代实现
	BindQueryNotImplemented() bool  // mux 没有实现 BindQuery 方法, 此情况下会采用替代实现

	Set(key string, value any)                     // Set用于存储专门用于此上下文的新键/值对，如果以前没有使用c.Keys，它也会延迟初始化它
	Get(key string, defaultValue ...string) string // 从上下文中读取键/值对
	SetCookie(cookie *http.Cookie)                 // 添加cookie
	Redirect(code int, location string) error      // 重定向

	ClientIP() string // ClientIP实现了一个最佳努力算法来返回真实的客户端IP。

	Status(code int)                                               // 设置响应状态码
	Write(p []byte) (int, error)                                   // 写入响应字节流,当此方法执行完毕时应中断后续流程
	SendString(s string) error                                     // 写字符串到响应体,当此方法执行完毕时应中断后续流程
	SendStream(stream io.Reader, size ...int) error                // 写入消息流到响应体
	JSON(code int, data any) error                                 // 写入json响应体
	JSONP(code int, data any) error                                // JSONP 支持
	Render(name string, bind interface{}, layouts ...string) error // 用于返回HTML
	XML(code int, obj any) error                                   // 写入XML
	YAML(code int, obj any) error                                  // 写入YAML
	TOML(code int, obj any) error                                  // 写入TOML
	File(filepath string) error                                    // 返回文件
	FileAttachment(filepath, filename string) error                // 将指定的文件以有效的方式写入主体流, 在客户端，文件通常会以给定的文件名下载
}
