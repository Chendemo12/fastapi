# MCP (Model Context Protocol) 支持

Go-FastApi 内置了 MCP 能力，允许将 Go 结构体方法通过反射暴露为 MCP 的 **Tool**、**Resource**、**Prompt**，供 AI Agent（Claude Desktop、Cursor、Cline 等）直接调用。

MCP 端点采用 **Streamable HTTP** 传输协议，单端点 `POST /mcp`，通过 `mcp-session-id` header 管理会话。

## 快速开始

```go
package main

import (
    "github.com/Chendemo12/fastapi"
    "github.com/Chendemo12/fastapi/middleware/fiberWrapper"
)

// 1. 定义一个 MCP Provider，嵌入 BaseMCPProvider
type MyMCPTools struct {
    fastapi.BaseMCPProvider
}

func (t *MyMCPTools) Prefix() string { return "myapp" }

// 2. 定义 Tool 方法：前缀 "Tool"，签名 (c *Context, [params...]) (any, error)
func (t *MyMCPTools) ToolGetUser(c *fastapi.Context, userId int) (*User, error) {
    return &User{ID: userId, Name: "test"}, nil
}

func (t *MyMCPTools) ToolCreateUser(c *fastapi.Context, req CreateUserReq) (*User, error) {
    // 业务逻辑...
    return &User{ID: 1, Name: req.Name}, nil
}

// 3. 注册并启动
func main() {
    app := fastapi.New(fastapi.Config{
        Title:   "My MCP Server",
        Version: "1.0.0",
    })
    app.SetMux(fiberWrapper.Default())
    app.IncludeMCP(&MyMCPTools{})
    app.Run("0.0.0.0", "8080")
}
```

MCP 客户端连接: `http://localhost:8080/mcp`

## MCPProvider 接口

```go
type MCPProvider interface {
    Prefix() string                          // 命名空间前缀，会添加到 tool|resource|prompt 名称前
    Tags() []string                          // 分类标签
    PathSchema() pathschema.RoutePathSchema  // 方法名→名称的转换规则，默认 snake_case（全小写+下划线）
    Summary() map[string]string              // 方法名 → 摘要
    Description() map[string]string          // 方法名 → 详细描述
    ResourceURI() map[string]string          // 方法名 → Resource URI（仅 Resource 需要）
}
```

嵌入 `BaseMCPProvider` 即可获得所有方法的默认实现。

## 方法命名与签名

### 命名规则

通过方法名前缀区分 MCP 能力类型，仿 HTTP 路由的前缀识别模式：

| 前缀 | MCP 类型 | 示例方法名 | 生成名称 |
|---|---|---|---|
| `Tool` | Tool | `ToolGetUser` | `myapp_get_user` |
| `Resource` | Resource | `ResourceConfig` | `myapp/config` |
| `Prompt` | Prompt | `PromptGreeting` | `myapp.greeting` |

前缀后的部分（suffix）通过 `PathSchema()` 转换为最终名称。

默认使用 `pathschema.LowerCaseUnderline`（`NewComposition(&LowerCase{}, &Underline{})`），即分词后全小写 + 下划线连接（snake_case）：

| 示例方法名 | 分词 | 连接 | 生成名称 |
|---|---|---|---|
| `ToolGetUser` | `["get", "user"]` | `_` | `get_user` |
| `ResourceAppConfig` | `["app", "config"]` | `_` | `app_config` |
| `ToolAdd` | `["add"]` | `_` | `add` |

可通过重写 `PathSchema()` 自定义规则，支持 `LowerCaseDash`、`LowerCamelCase` 等所有 `pathschema` 内置方案。

### 签名规则（三种类型统一）

```go
func (p *Provider) ToolXxx(c *Context, [params...]) (any, error)
```

| 要求 | 说明 |
|---|---|
| 第一个入参 | 必须是 `*fastapi.Context` |
| 额外入参 | 作为 Tool 的输入参数（inputSchema） |
| 返回值数量 | 有且仅有 2 个：`(any, error)` |
| 最后一个返回值 | 必须是 `error` |
| 返回值类型限制 | 不能是 `interface`、`func`、`chan`、`map`、`unsafe.Pointer` |

**参数展开规则**：
- **单一 struct 参数**：struct 的 JSON 字段自动展开为 inputSchema 的 properties，支持 `json`、`validate`、`description` 标签
- **多个基本类型参数**：每个参数成为独立 property，名称为 `{类型}_{索引}`（如 `int_2`），**推荐使用 struct 参数以获得有意义的参数名**

## Tool（工具）

Tool 是可供 AI 调用的函数。对应 MCP 的 `tools/list` 和 `tools/call`。

```go
type MyTools struct {
    fastapi.BaseMCPProvider
}

func (t *MyTools) Prefix() string { return "app" }

// 无参数 Tool
func (t *MyTools) ToolPing(c *fastapi.Context) (string, error) {
    return "pong", nil
}

// 基本类型参数（参数名为 int_2, int_3 — 反射限制）
func (t *MyTools) ToolAdd(c *fastapi.Context, a int, b int) (int, error) {
    return a + b, nil
}

// struct 参数（推荐 — 字段 json tag 即为参数名）
type CalcReq struct {
    X float64 `json:"x" validate:"required" description:"第一个数"`
    Y float64 `json:"y" validate:"required" description:"第二个数"`
}

func (t *MyTools) ToolCalculate(c *fastapi.Context, req CalcReq) (*CalcResp, error) {
    return &CalcResp{Result: req.X + req.Y}, nil
}
```

生成的 Input Schema（`ToolCalculate`）：

```json
{
    "type": "object",
    "properties": {
        "x": {"title": "X", "type": "number", "description": "第一个数"},
        "y": {"title": "Y", "type": "number", "description": "第二个数"}
    },
    "required": ["x", "y"]
}
```

## Resource（资源）

Resource 是可通过 URI 读取的数据源。对应 MCP 的 `resources/list` 和 `resources/read`。

```go
type MyResources struct {
    fastapi.BaseMCPProvider
}

func (r *MyResources) Prefix() string { return "app" }

// 自定义 Resource URI
func (r *MyResources) ResourceURI() map[string]string {
    return map[string]string{
        "ResourceConfig": "config://app/settings",
    }
}

type AppConfig struct {
    Theme string `json:"theme"`
    Lang  string `json:"lang"`
}

func (r *MyResources) ResourceConfig(c *fastapi.Context) (*AppConfig, error) {
    return &AppConfig{Theme: "dark", Lang: "zh"}, nil
}
```

URI 默认从方法名推导为 `resource://{prefix}/{name}`，可通过 `ResourceURI()` 覆盖。

## Prompt（提示模板）

Prompt 是预定义的提示模板。对应 MCP 的 `prompts/list` 和 `prompts/get`。

```go
type MyPrompts struct {
    fastapi.BaseMCPProvider
}

func (p *MyPrompts) Prefix() string { return "app" }

type GreetingMsg struct {
    Text string `json:"text"`
}

func (p *MyPrompts) PromptGreeting(c *fastapi.Context, name string) (*GreetingMsg, error) {
    return &GreetingMsg{Text: "Hello " + name}, nil
}
```

Prompt 的参数会自动注册为 MCP Prompt Arguments。

## 与 HTTP 路由并存

MCP Provider 和 HTTP 路由（GroupRouter）可以同时注册，互不干扰：

```go
app := fastapi.New(fastapi.Config{Title: "My App", Version: "1.0.0"})
app.SetMux(fiberWrapper.Default())

// HTTP 路由
app.IncludeRouter(&MyHTTPRouter{})

// MCP 能力
app.IncludeMCP(&MyMCPTools{})

app.Run("0.0.0.0", "8080")
```

此时：
- HTTP API 服务在 `http://localhost:8080`
- MCP 端点在 `http://localhost:8080/mcp`
- Swagger 文档在 `http://localhost:8080/docs`

## MCP 客户端配置

### Claude Desktop

```json
{
    "mcpServers": {
        "my-fastapi-server": {
            "url": "http://localhost:8080/mcp"
        }
    }
}
```

### Cursor

```json
{
    "mcpServers": {
        "my-fastapi-server": {
            "url": "http://localhost:8080/mcp"
        }
    }
}
```

### MCP Inspector（调试工具）

```bash
npx @modelcontextprotocol/inspector
```

输入 `http://localhost:8080/mcp` 即可查看所有 Tool/Resource/Prompt 并交互调试。

## 协议交互流程

```
Client                                    Server
  │
  │── POST /mcp
  │   {"jsonrpc":"2.0","id":1,"method":"initialize",...}
  │                                         → 创建 session
  │◀── 200 OK  mcp-session-id: <uuid>
  │    {"jsonrpc":"2.0","id":1,"result":{"capabilities":{"tools":{},"resources":{},"prompts":{}}}}
  │
  │── POST /mcp  (带 mcp-session-id header)
  │   {"jsonrpc":"2.0","id":2,"method":"tools/list"}
  │◀── {"jsonrpc":"2.0","id":2,"result":{"tools":[...]}}
  │
  │── POST /mcp  (带 mcp-session-id header)
  │   {"jsonrpc":"2.0","id":3,"method":"tools/call",
  │    "params":{"name":"myapp_get_user","arguments":{"userId":42}}}
  │◀── {"jsonrpc":"2.0","id":3,"result":{"content":[{"type":"text","text":"{...}"}]}}
```

## 配置

```go
type MCPConfig struct {
    Endpoint       string        // MCP 端点路径，默认 "/mcp"
    SessionTimeout time.Duration // 会话超时，默认 10 分钟
    ToolNaming     string        // 工具命名策略，默认 "summary"（可选 "operation"）
}
```

> **注意**：当前 MCP 配置通过 `MCPProvider` 接口方法控制（`Prefix`, `PathSchema`, `Summary`, `Description` 等），无需额外的 `MCPConfig` 参数。

## 已知限制

1. **函数参数名不可获取**：Go 反射无法获取函数参数名（如 `ToolAdd(c *Context, a int, b int)` 中的 `a` 和 `b`）。基本类型参数的 MCP 属性名使用 `{类型}_{位置索引}`（如 `int_2`、`int_3`）。**推荐使用 struct 参数**，通过 JSON tag 指定有意义的参数名。

2. **不支持 map 返回值**：受框架类型校验限制，Tool/Resource/Prompt 的返回值不能是 `map` 类型，需使用 struct。

3. **不支持 SSE 和文件类型的返回值**：MCP Tool 目前只支持 JSON 序列化的返回值（struct、基本类型、slice），不支持 `*FileResponse` 或 `*SSE`。
