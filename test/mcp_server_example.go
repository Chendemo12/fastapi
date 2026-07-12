//go:build ignore

// MCP Server 完整示例，演示如何创建一个带认证的 MCP 服务。
//
// 运行方式：
//
//	go run test/mcp_server_example.go
//
// 启动后 MCP 端点为: http://localhost:8080/mcp
// Swagger 文档:    http://localhost:8080/docs
package main

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/Chendemo12/fastapi"
	"github.com/Chendemo12/fastapi/middleware/fiberWrapper"
)

// ============================================================================
// 数据模型
// ============================================================================

type User struct {
	ID    int64  `json:"id" description:"用户ID"`
	Name  string `json:"name" description:"用户名称"`
	Email string `json:"email" description:"邮箱地址"`
}

type CreateUserReq struct {
	Name  string `json:"name" validate:"required" description:"用户名称"`
	Email string `json:"email" validate:"required" description:"邮箱地址"`
}

type CalcReq struct {
	X float64 `json:"x" validate:"required" description:"第一个数"`
	Y float64 `json:"y" validate:"required" description:"第二个数"`
}

type CalcResp struct {
	Result float64 `json:"result"`
}

type ServerInfo struct {
	Name    string `json:"name"`
	Version string `json:"version"`
	Status  string `json:"status"`
}

type SystemConfig struct {
	Version string `json:"version"`
	Theme   string `json:"theme"`
	Lang    string `json:"lang"`
}

type GreetingMsg struct {
	Text string `json:"text"`
}

// ============================================================================
// 公开 MCP Provider（无需认证）
// ============================================================================

type PublicTools struct {
	fastapi.BaseMCPProvider
}

func (p *PublicTools) Prefix() string { return "public" }

func (p *PublicTools) Tags() []string { return []string{"public", "utility"} }

// ToolCalculate 基本运算工具
func (p *PublicTools) ToolCalculate(c *fastapi.Context, req CalcReq) (*CalcResp, error) {
	return &CalcResp{Result: req.X + req.Y}, nil
}

// ToolPing 健康检查
func (p *PublicTools) ToolPing(c *fastapi.Context) (string, error) {
	return "pong", nil
}

// ResourceServerInfo 只读的系统信息
func (p *PublicTools) ResourceServerInfo(c *fastapi.Context) (*ServerInfo, error) {
	return &ServerInfo{
		Name:    "fastapi-mcp-server",
		Version: "1.0.0",
		Status:  "running",
	}, nil
}

func (p *PublicTools) ResourceURI() map[string]string {
	return map[string]string{
		"ResourceServerInfo": "info://public/server",
	}
}

// PromptGreeting 欢迎语模板
func (p *PublicTools) PromptGreeting(c *fastapi.Context, name string) (*GreetingMsg, error) {
	return &GreetingMsg{Text: fmt.Sprintf("你好 %s，欢迎使用 FastAPI MCP Server！", name)}, nil
}

// ============================================================================
// 认证 MCP Provider（需要 Bearer Token）
// ============================================================================

type SecureTools struct {
	fastapi.BaseMCPProvider
}

func (s *SecureTools) Prefix() string { return "secure" }

func (s *SecureTools) Tags() []string { return []string{"admin", "secure"} }

// AuthFunc 认证校验 — 从 HTTP Header 读取 Authorization
func (s *SecureTools) AuthFunc(c *fastapi.Context) error {
	token := c.MuxContext().GetHeader("Authorization")
	if token == "" {
		return errors.New("未提供认证令牌")
	}
	if token != "Bearer my-secret-token" {
		return errors.New("认证令牌无效")
	}
	return nil
}

// ToolListUsers 查询用户列表（认证）
func (s *SecureTools) ToolListUsers(c *fastapi.Context) ([]*User, error) {
	return []*User{
		{ID: 1, Name: "张三", Email: "zhangsan@example.com"},
		{ID: 2, Name: "李四", Email: "lisi@example.com"},
	}, nil
}

// ToolCreateUser 创建用户（认证）
func (s *SecureTools) ToolCreateUser(c *fastapi.Context, req CreateUserReq) (*User, error) {
	return &User{
		ID:    100,
		Name:  req.Name,
		Email: req.Email,
	}, nil
}

// ToolDeleteUser 删除用户（认证）
func (s *SecureTools) ToolDeleteUser(c *fastapi.Context, userId int64) (string, error) {
	return fmt.Sprintf("用户 %d 已删除", userId), nil
}

// ResourceSystemConfig 系统配置（认证资源）
func (s *SecureTools) ResourceSystemConfig(c *fastapi.Context) (*SystemConfig, error) {
	return &SystemConfig{
		Version: "2.0.0",
		Theme:   "dark",
		Lang:    "zh-CN",
	}, nil
}

func (s *SecureTools) ResourceURI() map[string]string {
	return map[string]string{
		"ResourceSystemConfig": "config://secure/system",
	}
}

// PromptAdminHelp 管理员帮助模板（认证）
func (s *SecureTools) PromptAdminHelp(c *fastapi.Context) (*GreetingMsg, error) {
	return &GreetingMsg{
		Text: "管理员可用命令：list_users, create_user, delete_user",
	}, nil
}

// ============================================================================
// HTTP 路由（RESTful API + MCP 并存）
// ============================================================================

type HTTPRouter struct {
	fastapi.BaseGroupRouter
}

func (r *HTTPRouter) Prefix() string { return "/api/v1" }

func (r *HTTPRouter) GetHealth(c *fastapi.Context) (string, error) {
	return `{"status":"ok"}`, nil
}

// ============================================================================
// 启动入口
// ============================================================================

func main() {
	// 创建应用
	app := fastapi.New(fastapi.Config{
		Title:       "FastAPI MCP Server",
		Version:     "2.0.0",
		Description: "集成 MCP 协议的 FastAPI Server，支持公开和认证两类 MCP Provider",
	})

	// 设置 Fiber 引擎
	fiberEngine := fiberWrapper.Default()
	app.SetMux(fiberEngine)

	// 注册 HTTP 路由
	app.IncludeRouter(&HTTPRouter{})

	// 注册 MCP Provider
	app.IncludeMCP(&PublicTools{}) // 公开接口，无需认证
	app.IncludeMCP(&SecureTools{}) // 管理接口，需要 Bearer Token

	// 请求日志钩子（对所有请求生效，包括 MCP）
	app.UseBeforeWrite(func(c *fastapi.Context) {
		fmt.Printf("[%s] %s  %d\n",
			http.MethodPost, c.MuxContext().Path(), c.Response().StatusCode)
	})

	// 启动服务
	fmt.Println("MCP Server 启动于: http://localhost:8080")
	fmt.Println("  - MCP 端点:  http://localhost:8080/mcp")
	fmt.Println("  - Swagger:   http://localhost:8080/docs")
	fmt.Println("  - API 健康:   http://localhost:8080/api/v1/health")
	app.Run("0.0.0.0", "8080")
}
