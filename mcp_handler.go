package fastapi

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/Chendemo12/fastapi/utils"
)

// =================================== JSON-RPC Types ===================================

type mcpJSONRPCRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id,omitempty"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

type mcpJSONRPCResponse struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id,omitempty"`
	Result  any             `json:"result,omitempty"`
	Error   *mcpRPCError    `json:"error,omitempty"`
}

type mcpRPCError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    any    `json:"data,omitempty"`
}

// Standard JSON-RPC error codes
const (
	errParse          = -32700
	errInvalidRequest = -32600
	errMethodNotFound = -32601
	errInvalidParams  = -32602
	errInternal       = -32603
)

// MCP header constants
const (
	mcpSessionHeader      = "mcp-session-id"
	mcpProtocolVersion    = "2024-11-05"
	defaultSessionTimeout = 10 * time.Minute
)

// =================================== MCP Session ===================================

type mcpSession struct {
	ID        string
	CreatedAt time.Time
}

type mcpSessionManager struct {
	mu       sync.RWMutex
	sessions map[string]*mcpSession
	timeout  time.Duration
}

func newMCPSessionManager(timeout time.Duration) *mcpSessionManager {
	return &mcpSessionManager{
		sessions: make(map[string]*mcpSession),
		timeout:  timeout,
	}
}

func (sm *mcpSessionManager) create() *mcpSession {
	id := generateSessionID()
	s := &mcpSession{ID: id, CreatedAt: time.Now()}
	sm.mu.Lock()
	sm.sessions[id] = s
	sm.mu.Unlock()
	return s
}

func (sm *mcpSessionManager) get(id string) (*mcpSession, bool) {
	sm.mu.RLock()
	s, ok := sm.sessions[id]
	sm.mu.RUnlock()
	if !ok {
		return nil, false
	}
	if time.Since(s.CreatedAt) > sm.timeout {
		sm.delete(id)
		return nil, false
	}
	return s, true
}

func (sm *mcpSessionManager) delete(id string) {
	sm.mu.Lock()
	delete(sm.sessions, id)
	sm.mu.Unlock()
}

func (sm *mcpSessionManager) cleanup() {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	now := time.Now()
	for id, s := range sm.sessions {
		if now.Sub(s.CreatedAt) > sm.timeout {
			delete(sm.sessions, id)
		}
	}
}

func generateSessionID() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

// =================================== MCP Capabilities ===================================

type mcpToolCapability struct {
	ListChanged bool `json:"listChanged,omitempty"`
}

type mcpServerCapabilities struct {
	Tools     *mcpToolCapability `json:"tools,omitempty"`
	Resources *mcpToolCapability `json:"resources,omitempty"`
	Prompts   *mcpToolCapability `json:"prompts,omitempty"`
}

type mcpServerInfo struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

type mcpInitializeResult struct {
	ProtocolVersion string                `json:"protocolVersion"`
	Capabilities    mcpServerCapabilities `json:"capabilities"`
	ServerInfo      mcpServerInfo         `json:"serverInfo"`
}

type mcpListToolsResult struct {
	Tools []*MCPToolMeta `json:"tools"`
}

type mcpListResourcesResult struct {
	Resources []*MCPResourceMeta `json:"resources"`
}

type mcpListPromptsResult struct {
	Prompts []*MCPPromptMeta `json:"prompts"`
}

type mcpReadResourceParams struct {
	URI string `json:"uri"`
}

type mcpGetPromptParams struct {
	Name      string         `json:"name"`
	Arguments map[string]any `json:"arguments,omitempty"`
}

type mcpCallToolParams struct {
	Name      string         `json:"name"`
	Arguments map[string]any `json:"arguments,omitempty"`
}

type mcpContentItem struct {
	Type string `json:"type"`
	Text string `json:"text,omitempty"`
	Data string `json:"data,omitempty"`
	URI  string `json:"uri,omitempty"`
}

type mcpToolCallResult struct {
	Content []mcpContentItem `json:"content"`
	IsError bool             `json:"isError,omitempty"`
}

type mcpResourceReadResult struct {
	Contents []mcpContentItem `json:"contents"`
}

type mcpPromptGetResult struct {
	Description string           `json:"description,omitempty"`
	Messages    []mcpContentItem `json:"messages"`
}

// =================================== MCPHandler ===================================

// MCPHandler handles JSON-RPC 2.0 requests for the MCP protocol.
// Transport: Streamable HTTP (single POST /mcp endpoint).
type MCPHandler struct {
	tools     []*MCPToolMeta
	resources []*MCPResourceMeta
	prompts   []*MCPPromptMeta

	toolByName    map[string]*MCPToolMeta
	resourceByURI map[string]*MCPResourceMeta
	promptByName  map[string]*MCPPromptMeta

	sessions  *mcpSessionManager
	authFuncs []func(c *Context) error
	appTitle  string
	appVer    string
}

// NewMCPHandler creates a new MCP handler from provider metadata.
func NewMCPHandler(providers []*MCPProviderMeta, title, version string) *MCPHandler {
	h := &MCPHandler{
		toolByName:    make(map[string]*MCPToolMeta),
		resourceByURI: make(map[string]*MCPResourceMeta),
		promptByName:  make(map[string]*MCPPromptMeta),
		sessions:      newMCPSessionManager(defaultSessionTimeout),
		authFuncs:     make([]func(c *Context) error, 0),
		appTitle:      title,
		appVer:        version,
	}

	for _, p := range providers {
		h.tools = append(h.tools, p.Tools()...)
		h.resources = append(h.resources, p.Resources()...)
		h.prompts = append(h.prompts, p.Prompts()...)

		for _, t := range p.Tools() {
			h.toolByName[t.Name] = t
		}
		for _, r := range p.Resources() {
			h.resourceByURI[r.URI] = r
		}
		for _, pr := range p.Prompts() {
			h.promptByName[pr.Name] = pr
		}

		// Collect auth func per provider
		if p.provider != nil {
			h.authFuncs = append(h.authFuncs, p.provider.AuthFunc)
		}
	}
	return h
}

// ServeMCP is the HTTP handler for the POST /mcp endpoint.
func (h *MCPHandler) ServeMCP(ctx MuxContext) error {
	// Read the raw JSON-RPC request body
	var request mcpJSONRPCRequest
	_, err := ctx.ShouldBind(&request)
	if err != nil {
		return h.writeJSONRPCError(ctx, nil, errParse, "Parse error: "+err.Error())
	}

	if request.JSONRPC != "2.0" {
		return h.writeJSONRPCError(ctx, request.ID, errInvalidRequest, "Invalid Request: jsonrpc must be 2.0")
	}

	// Session validation (non-initialize requests must carry a session ID)
	if request.Method != "initialize" {
		sessionID := ctx.GetHeader(mcpSessionHeader)
		if sessionID == "" {
			return h.writeJSONRPCError(ctx, request.ID, errInvalidRequest,
				"Session required. Call initialize first.")
		}
		if _, ok := h.sessions.get(sessionID); !ok {
			return h.writeJSONRPCError(ctx, request.ID, errInvalidRequest,
				"Session expired or not found. Call initialize first.")
		}
	}

	// Run auth funcs (for non-initialize requests)
	if len(h.authFuncs) > 0 {
		c := NewContext()
		c.initContext(ctx)
		defer c.resetContext()
		for _, auth := range h.authFuncs {
			if err := auth(c); err != nil {
				return h.writeJSONRPCError(ctx, request.ID, errInternal, err.Error())
			}
		}
	}

	// Dispatch by method
	switch request.Method {
	case "initialize":
		return h.handleInitialize(ctx, &request)
	case "tools/list":
		return h.handleToolsList(ctx, &request)
	case "tools/call":
		return h.handleToolsCall(ctx, &request)
	case "resources/list":
		return h.handleResourcesList(ctx, &request)
	case "resources/read":
		return h.handleResourcesRead(ctx, &request)
	case "prompts/list":
		return h.handlePromptsList(ctx, &request)
	case "prompts/get":
		return h.handlePromptsGet(ctx, &request)
	default:
		return h.writeJSONRPCError(ctx, request.ID, errMethodNotFound,
			fmt.Sprintf("Method not found: %s", request.Method))
	}
}

// Shutdown performs cleanup (call on application shutdown).
func (h *MCPHandler) Shutdown() {
	h.sessions.cleanup()
}

// =================================== Handler Methods ===================================

func (h *MCPHandler) handleInitialize(ctx MuxContext, req *mcpJSONRPCRequest) error {
	session := h.sessions.create()
	ctx.Header(mcpSessionHeader, session.ID)

	result := mcpInitializeResult{
		ProtocolVersion: mcpProtocolVersion,
		Capabilities: mcpServerCapabilities{
			Tools:     &mcpToolCapability{},
			Resources: &mcpToolCapability{},
			Prompts:   &mcpToolCapability{},
		},
		ServerInfo: mcpServerInfo{
			Name:    h.appTitle,
			Version: h.appVer,
		},
	}

	return h.writeJSONRPCResult(ctx, req.ID, result, http.StatusOK)
}

func (h *MCPHandler) handleToolsList(ctx MuxContext, req *mcpJSONRPCRequest) error {
	result := mcpListToolsResult{Tools: h.tools}
	if result.Tools == nil {
		result.Tools = make([]*MCPToolMeta, 0)
	}
	return h.writeJSONRPCResult(ctx, req.ID, result, http.StatusOK)
}

func (h *MCPHandler) handleToolsCall(ctx MuxContext, req *mcpJSONRPCRequest) error {
	var params mcpCallToolParams
	if err := json.Unmarshal(req.Params, &params); err != nil {
		return h.writeJSONRPCError(ctx, req.ID, errInvalidParams, "Invalid params")
	}

	tool, ok := h.toolByName[params.Name]
	if !ok {
		return h.writeJSONRPCError(ctx, req.ID, errInvalidParams,
			fmt.Sprintf("Tool not found: %s", params.Name))
	}

	if params.Arguments == nil {
		params.Arguments = make(map[string]any)
	}

	// Create context for MCP tool execution
	c := NewContext()
	c.initContext(ctx)
	defer c.resetContext()

	args := tool.NewCallArgs(c, params.Arguments)
	result, err := tool.Call(args)
	if err != nil {
		return h.writeJSONRPCResult(ctx, req.ID, mcpToolCallResult{
			Content: []mcpContentItem{{
				Type: "text",
				Text: err.Error(),
			}},
			IsError: true,
		}, http.StatusOK)
	}

	resultJSON, _ := utils.JsonMarshal(result)

	return h.writeJSONRPCResult(ctx, req.ID, mcpToolCallResult{
		Content: []mcpContentItem{{
			Type: "text",
			Text: string(resultJSON),
		}},
	}, http.StatusOK)
}

func (h *MCPHandler) handleResourcesList(ctx MuxContext, req *mcpJSONRPCRequest) error {
	result := mcpListResourcesResult{Resources: h.resources}
	if result.Resources == nil {
		result.Resources = make([]*MCPResourceMeta, 0)
	}
	return h.writeJSONRPCResult(ctx, req.ID, result, http.StatusOK)
}

func (h *MCPHandler) handleResourcesRead(ctx MuxContext, req *mcpJSONRPCRequest) error {
	var params mcpReadResourceParams
	if err := json.Unmarshal(req.Params, &params); err != nil {
		return h.writeJSONRPCError(ctx, req.ID, errInvalidParams, "Invalid params")
	}

	res, ok := h.resourceByURI[params.URI]
	if !ok {
		return h.writeJSONRPCError(ctx, req.ID, errInvalidParams,
			fmt.Sprintf("Resource not found: %s", params.URI))
	}

	c := NewContext()
	c.initContext(ctx)
	defer c.resetContext()

	args := res.NewCallArgs(c, nil)
	result, err := res.Call(args)
	if err != nil {
		return h.writeJSONRPCError(ctx, req.ID, errInternal, err.Error())
	}

	resultJSON, _ := utils.JsonMarshal(result)

	return h.writeJSONRPCResult(ctx, req.ID, mcpResourceReadResult{
		Contents: []mcpContentItem{{
			URI:  res.URI,
			Type: "text",
			Text: string(resultJSON),
		}},
	}, http.StatusOK)
}

func (h *MCPHandler) handlePromptsList(ctx MuxContext, req *mcpJSONRPCRequest) error {
	result := mcpListPromptsResult{Prompts: h.prompts}
	if result.Prompts == nil {
		result.Prompts = make([]*MCPPromptMeta, 0)
	}
	return h.writeJSONRPCResult(ctx, req.ID, result, http.StatusOK)
}

func (h *MCPHandler) handlePromptsGet(ctx MuxContext, req *mcpJSONRPCRequest) error {
	var params mcpGetPromptParams
	if err := json.Unmarshal(req.Params, &params); err != nil {
		return h.writeJSONRPCError(ctx, req.ID, errInvalidParams, "Invalid params")
	}

	prompt, ok := h.promptByName[params.Name]
	if !ok {
		return h.writeJSONRPCError(ctx, req.ID, errInvalidParams,
			fmt.Sprintf("Prompt not found: %s", params.Name))
	}

	if params.Arguments == nil {
		params.Arguments = make(map[string]any)
	}

	c := NewContext()
	c.initContext(ctx)
	defer c.resetContext()

	args := prompt.NewCallArgs(c, params.Arguments)
	result, err := prompt.Call(args)
	if err != nil {
		return h.writeJSONRPCError(ctx, req.ID, errInternal, err.Error())
	}

	resultJSON, _ := utils.JsonMarshal(result)

	return h.writeJSONRPCResult(ctx, req.ID, mcpPromptGetResult{
		Description: prompt.Description,
		Messages: []mcpContentItem{{
			Type: "text",
			Text: string(resultJSON),
		}},
	}, http.StatusOK)
}

// =================================== Response Helpers ===================================

func (h *MCPHandler) writeJSONRPCResult(ctx MuxContext, id json.RawMessage, result any, status int) error {
	resp := mcpJSONRPCResponse{
		JSONRPC: "2.0",
		ID:      id,
		Result:  result,
	}
	return ctx.JSON(status, resp)
}

func (h *MCPHandler) writeJSONRPCError(ctx MuxContext, id json.RawMessage, code int, message string) error {
	resp := mcpJSONRPCResponse{
		JSONRPC: "2.0",
		ID:      id,
		Error: &mcpRPCError{
			Code:    code,
			Message: message,
		},
	}
	return ctx.JSON(http.StatusOK, resp)
}
