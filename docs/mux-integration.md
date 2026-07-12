# Mux 集成指南

本文档说明如何为 Go-FastApi 实现一个新的 HTTP 引擎适配器（Mux wrapper），使其能够运行在 Fiber、Gin 以外的 Web 框架上（如
Echo、Net/HTTP 等）。

## 1. 架构概述

Go-FastApi 是一个装饰器/适配层，不绑定特定的 HTTP 引擎。它通过 `MuxWrapper` 和 `MuxContext` 两个接口与底层框架交互：

```
             HTTP 请求
                 │
                 v
┌────────────────────────────────┐
│  原生引擎 (Fiber/Gin/Echo/...)  │
└────────────────┬───────────────┘
                 │  调用 BindRoute 注册的 wrapper 函数
                 v
┌─────────────────────────────────┐
│           MuxContext 适配层      │  ← 你需要实现的接口
└────────────────┬────────────────┘
                 │  调用 MuxHandler (Wrapper.Handler)
                 v
┌─────────────────────────────────┐
│         fastapi.Wrapper         │  ← 路由匹配、参数校验、响应序列化
└────────────────┬────────────────┘
                 │
                 v
            路由处理函数
```

## 2. 需要实现的接口

### 2.1 MuxWrapper — 服务器生命周期

`fastapi.MuxWrapper` 定义服务器级别的操作：

```go
type MuxWrapper interface {
Listen(addr string) error
ShutdownWithTimeout(timeout time.Duration) error
BindRoute(method, path string, handler MuxHandler) error
}
```

- **Listen** — 启动 HTTP 服务，监听指定地址
- **ShutdownWithTimeout** — 优雅关闭，在指定时间内完成正在处理的请求
- **BindRoute** — 将 `(method, path, handler)` 注册到引擎的路由表；`handler` 是 `func(c MuxContext) error` 类型的
  MuxHandler

### 2.2 MuxContext — 请求上下文

`fastapi.MuxContext` 定义单次请求的全部操作。详细接口定义见 `mux.go`。核心需要实现的方法：

| 类别   | 方法                                               | 说明                              |
|------|--------------------------------------------------|---------------------------------|
| 标识   | `Method()`                                       | 必须实现，不可返回空值，取值为 `http.Method*`  |
| 标识   | `Path()`                                         | 必须实现，不可返回空值，取值为静态路径，非动态路由       |
| 读取请求 | `Params()`, `Query()`, `GetHeader()`, `Cookie()` | 路径参数、查询参数、请求头、Cookie            |
| 请求体  | `ShouldBind(obj)`                                | 反序列化请求体到结构体                     |
| 写入响应 | `JSON()`, `SendString()`, `Status()`, `Header()` | 写 JSON、纯文本、状态码、响应头              |
| 内部   | `FastApiContext() *Context`                      | 返回内嵌的 `fastapi.Context` 见`生命周期` |

## 3. Context 生命周期（关键）

### 3.1 核心原则

- **Mux 层** 只负责池化管理和原生 context 引用
- **Wrapper 层** (`initContext`/`resetContext`) 是 `fastapi.Context` 生命周期的唯一管理者

### 3.2 结构体设计

MuxContext 实现必须**值嵌入** `fastapi.Context`，并维护一个 `sync.Pool`：

```go
type EchoContext struct {
    fastapi.Context      // 值嵌入，共享内存
    echoCtx echo.Context // 原生引擎的 context 引用
}

func (c *EchoContext) FastApiContext() *fastapi.Context { return &c.Context }

var pool = &sync.Pool{New: func () any {
    return &EchoContext{Context: *fastapi.NewContext()}
}}
```

`fastapi.NewContext()` 预分配了 `pathFields` 和 `queryFields` map，避免每次请求重新分配。

### 3.3 AcquireCtx / ReleaseCtx

```go
func AcquireCtx(c echo.Context) *EchoContext {
    obj := pool.Get().(*EchoContext)
    obj.echoCtx = c
    // 注意：不调用 initContext —— 由 Wrapper.Handler 统一管理
    return obj
}

func ReleaseCtx(c *EchoContext) {
    // 只负责 Mux 专属的清理：原生引用 + 归还池
    c.echoCtx = nil
    pool.Put(c)
}
```

内嵌 Context 的 `resetContext` 由 `Wrapper.Handler` 在入口通过 `defer` 调用，Mux wrapper 无需关心。

### 3.4 完整的请求生命周期

下面是一次 HTTP 请求经过两层生命周期管理的完整流程：

```
AcquireCtx(nativeCtx)
    │  pool.Get(), 设置原生引用 (不调用 initContext — response=nil)
    │
    v
handler(mCtx) → Wrapper.Handler
    │
    ├─ FastApiContext().initContext(mux, appCtx, autoCtx)
    │     └─ response = AcquireResponse()  ← 唯一分配点
    │
    ├─ 校验 (路径参数、查询参数、请求体)
    ├─ 路由处理函数
    ├─ 响应校验
    ├─ beforeWrite 钩子
    │
    ├─ resetContext (defer)
    │     └─ ReleaseResponse(response) → response=nil
    │
    v
控制返回 BindRoute wrapper：
    └─ ReleaseCtx(mCtx) → 原生引用 = nil, pool.Put()
    
    注意：resetContext 已在 Wrapper.Handler 的 defer 中调用，不在 BindRoute 层重复。
```

### 3.5 设计说明

- `initContext` **只在 `Wrapper.Handler` 中调用一次**：分配 Response、设置 mux、按需派生 request context
- `resetContext` 在 `Wrapper.Handler` 入口通过 `defer` 调用，覆盖所有路由（含 route not found 分支）
- `ReleaseResponse` 内置 nil 守护，`resetContext` 可安全重入
- 绕过 `Wrapper.Handler` 的路由（如 Swagger UI、MCP）不会调用 `initContext`，`response` 始终为 nil；Mux 层仅通过 `ReleaseCtx` 归还池

## 4. BindRoute 实现模板

```go
func (m *EchoMux) BindRoute(method, path string, handler fastapi.MuxHandler) error {
    wrapper := func (c echo.Context) error {
        mCtx := AcquireCtx(c)
        defer ReleaseCtx(mCtx)  // 归还池
        return handler(mCtx)
    }

    switch method {
    case http.MethodGet:
        m.app.GET(path, wrapper)
    case http.MethodPost:
        m.app.POST(path, wrapper)
        // ... 其他 HTTP 方法
    default:
        return fmt.Errorf("unsupported method: %s", method)
    }

    return nil
}
```

> `resetContext` 不在此处调用 —— 它由 `Wrapper.Handler` 内部的 `defer` 统一管理。

### 关于错误处理

原生框架的 handler 签名可能不同。常见处理方式：

- Fiber (`func(*fiber.Ctx) error`)：直接返回 error
- Gin (`func(*gin.Context)`)：无返回值，需调用 `c.Error(err)` + `c.Abort()`
- Echo (`func(echo.Context) error`)：直接返回 error

`BindRoute` 的 wrapper 需要做适配：

```go
// Gin 风格（无返回值）
wrapper := func (c *gin.Context) {
    mCtx := AcquireCtx(c)
    defer ReleaseCtx(mCtx)
    if err := handler(mCtx); err != nil {  
        _ = c.Error(err)
        c.Abort()
    }
}

// Fiber / Echo 风格（返回 error）
wrapper := func (c *fiber.Ctx) error {
    mCtx := AcquireCtx(c)
    defer ReleaseCtx(mCtx)
    return handler(mCtx)
}
```

## 5. 完整模板

以下是在 `middleware/echoWrapper/mux.go` 中实现 Mux 集成的最简模板：

```go
package echoWrapper

import (
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/Chendemo12/fastapi"
	"github.com/labstack/echo/v4"
)

// ===================== Pool =====================

var pool = &sync.Pool{New: func() any {
	return &EchoContext{Context: *fastapi.NewContext()}
}}

func AcquireCtx(c echo.Context) *EchoContext {
	obj := pool.Get().(*EchoContext)
	obj.echoCtx = c
	// initContext 由 Wrapper.Handler 统一管理
	return obj
}

func ReleaseCtx(c *EchoContext) {
	c.echoCtx = nil
	pool.Put(c)
}

// ===================== Mux Wrapper =====================

type EchoMux struct {
	app *echo.Echo
}

func New(app *echo.Echo) *EchoMux {
	return &EchoMux{app: app}
}

func (m *EchoMux) Listen(addr string) error {
	return m.app.Start(addr)
}

func (m *EchoMux) ShutdownWithTimeout(d time.Duration) error {
	ctx, cancel := context.WithTimeout(context.Background(), d)
	defer cancel()
	return m.app.Shutdown(ctx)
}

func (m *EchoMux) BindRoute(method, path string, handler fastapi.MuxHandler) error {
	wrapper := func(c echo.Context) error {
		mCtx := AcquireCtx(c)
		defer ReleaseCtx(mCtx) // 归还池
		return handler(mCtx)
	}

	switch method {
	case http.MethodGet:
		m.app.GET(path, wrapper)
	case http.MethodPost:
		m.app.POST(path, wrapper)
	case http.MethodPut:
		m.app.PUT(path, wrapper)
	case http.MethodDelete:
		m.app.DELETE(path, wrapper)
	case http.MethodPatch:
		m.app.PATCH(path, wrapper)
	default:
		return fmt.Errorf("unsupported method: %s", method)
	}
	return nil
}

// ===================== Mux Context =====================

type EchoContext struct {
	fastapi.Context
	echoCtx echo.Context
}

func (c *EchoContext) FastApiContext() *fastapi.Context { return &c.Context }

func (c *EchoContext) Method() string { return c.echoCtx.Request().Method }
func (c *EchoContext) Path() string   { return c.echoCtx.Path() }
func (c *EchoContext) Ctx() any       { return c.echoCtx }

func (c *EchoContext) Params(key string, undefined ...string) string {
	v := c.echoCtx.Param(key)
	if v == "" && len(undefined) > 0 {
		return undefined[0]
	}
	return v
}

func (c *EchoContext) Query(key string, undefined ...string) string {
	v := c.echoCtx.QueryParam(key)
	if v == "" && len(undefined) > 0 {
		return undefined[0]
	}
	return v
}

func (c *EchoContext) ShouldBind(obj any) (bool, error) {
	return false, c.echoCtx.Bind(obj)
}

func (c *EchoContext) JSON(code int, data any) error {
	return c.echoCtx.JSON(code, data)
}

func (c *EchoContext) Status(code int) {
	c.echoCtx.Response().Status = code
}

func (c *EchoContext) Header(key, value string) {
	c.echoCtx.Response().Header().Set(key, value)
}

// ... 其余接口方法
```

## 6. 参考实现

完整的 Mux 适配器实现请参考：

- [middleware/fiberWrapper/mux.go](../middleware/fiberWrapper/mux.go) — Fiber 适配器（包含 SSE 流式响应实现）
- [middleware/ginWrapper/mux.go](../middleware/ginWrapper/mux.go) — Gin 适配器

两个实现结构完全对称，可作为新引擎适配的对照参考。
