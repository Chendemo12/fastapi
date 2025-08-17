package fastapi

import (
	"io"
	"mime/multipart"
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
// 此接口定义的方法无需全部实现
//
//  1. Method 和 Path 方法必须实现且不可返回空值，否则将导致 panic
//  2. 对于 MuxContext 缺少的方法，可通过直接调用 Ctx 来实现
//  3. 对于 Set / Get 方法，如果实际的Context未提供，则通过 Context.Get/Context.Set 代替
//  4. GetHeader, Cookie, Query, Params 是必须实现的方法
//  5. ShouldBind 和 BodyParser + Validate 必须实现一个，如果请求体不是JSON时则重写此方法，同时 CustomShouldBindMethod 需要返回 true
type MuxContext interface {
	Method() string // [重要方法]获得当前请求方法，取值为 http.MethodGet, http.MethodPost 等
	Path() string   // [重要方法]获的当前请求的路由模式，而非请求Url

	Ctx() any                                // 原始的 Context
	Set(key string, value any)               // Set用于存储专门用于此上下文的新键/值对，如果以前没有使用c.Keys，它也会延迟初始化它
	Get(key string) (value any, exists bool) // 从上下文中读取键/值对

	// === 与请求体有关方法

	ClientIP() string                              // 获得客户端IP
	ContentType() string                           // 请求体的 Content-Type
	GetHeader(key string) string                   // 读取请求头
	Cookie(name string) (string, error)            // 读取cookie
	Params(key string, undefined ...string) string // 读取路径参数
	Query(key string, undefined ...string) string  // 读取查询参数
	MultipartForm() (*multipart.Form, error)       // 读取 multipart/form-data 数据
	// ShouldBind 绑定请求体到obj上，如果已完成数据校验则返回true，error 应为validator.ValidationErrors 类型
	ShouldBind(obj any) (validated bool, err error)

	// === 与响应有关方法

	Header(key, value string)                       // 添加响应头 [!!注意是添加响应头，而非读取]
	SetCookie(cookie *http.Cookie)                  // 添加cookie
	Redirect(code int, location string) error       // 重定向
	Status(code int)                                // 设置响应状态码
	SendString(s string) error                      // 写字符串到响应体,当此方法执行完毕时应中断后续流程
	JSON(code int, data any) error                  // 写入json响应体
	SendStream(stream io.Reader, size ...int) error // 写入消息流到响应体
	File(filepath string) error                     // 返回文件
	FileAttachment(filepath, filename string) error // 将指定的文件以有效的方式写入主体流, 在客户端，文件通常会以给定的文件名下载
	Write(p []byte) (int, error)                    // 写入响应字节流,当此方法执行完毕时应中断后续流程
}
