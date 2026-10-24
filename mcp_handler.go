package fastapi

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/Chendemo12/fastapi/utils"
)

// =================================== JSON-RPC Types ===================================

type mcpJSONRPCRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id,omitempty"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

// isNotification reports whether the request is a JSON-RPC notification,
// i.e. it has no "id" member or its method belongs to the notifications namespace.
// A request with an explicit null id is still a request and receives a response.
func (r *mcpJSONRPCRequest) isNotification() bool {
	if r.Method == "" {
		// Malformed request: it must receive an error response.
		return false
	}
	if len(r.ID) == 0 {
		return true
	}
	return len(r.Method) > len("notifications/") && r.Method[:len("notifications/")] == "notifications/"
}

// isValidJSONRPCID reports whether the raw id is a valid JSON-RPC id:
// String, Number, or Null (JSON-RPC 2.0 requires one of these).
func isValidJSONRPCID(id json.RawMessage) bool {
	if len(id) == 0 {
		return false
	}
	b := id[0]
	if b == '"' || b == 'n' { // string / null
		return true
	}
	if b == '-' || (b >= '0' && b <= '9') { // number
		return true
	}
	return false
}

type mcpJSONRPCResponse struct {
	JSONRPC string `json:"jsonrpc"`
	// ID must always be present in responses; it is null when unknown.
	ID     json.RawMessage `json:"id"`
	Result any             `json:"result,omitempty"`
	Error  *mcpRPCError    `json:"error,omitempty"`
}

type mcpRPCError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    any    `json:"data,omitempty"`
}

// Standard JSON-RPC error codes plus MCP-defined server error codes.
const (
	errParse            = -32700
	errInvalidRequest   = -32600
	errMethodNotFound   = -32601
	errInvalidParams    = -32602
	errInternal         = -32603
	errResourceNotFound = -32002
)

// MCP protocol constants.
const (
	// mcpProtocolVersion is the latest protocol revision supported by this server.
	// Streamable HTTP was introduced in 2025-03-26.
	mcpProtocolVersion = "2025-03-26"
)

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

type mcpInitializeParams struct {
	ProtocolVersion string `json:"protocolVersion"`
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
	Type     string `json:"type"`
	Text     string `json:"text,omitempty"`
	Data     string `json:"data,omitempty"`
	MimeType string `json:"mimeType,omitempty"`
	URI      string `json:"uri,omitempty"`
}

type mcpToolCallResult struct {
	Content []mcpContentItem `json:"content"`
	IsError bool             `json:"isError,omitempty"`
}

// mcpResourceContents follows the MCP ResourceContents shape:
// either "text" or "blob" is present.
type mcpResourceContents struct {
	URI      string `json:"uri"`
	MimeType string `json:"mimeType,omitempty"`
	Text     string `json:"text,omitempty"`
	Blob     string `json:"blob,omitempty"`
}

type mcpResourceReadResult struct {
	Contents []mcpResourceContents `json:"contents"`
}

// mcpPromptMessage follows the MCP PromptMessage shape.
type mcpPromptMessage struct {
	Role    string         `json:"role"`
	Content mcpContentItem `json:"content"`
}

type mcpPromptGetResult struct {
	Description string             `json:"description,omitempty"`
	Messages    []mcpPromptMessage `json:"messages"`
}

// =================================== MCPHandler ===================================

// MCPHandler handles JSON-RPC 2.0 requests for the MCP protocol.
// Transport: Streamable HTTP (POST /mcp), stateless: no Mcp-Session-Id is
// issued or required.
type MCPHandler struct {
	tools     []*MCPToolMeta
	resources []*MCPResourceMeta
	prompts   []*MCPPromptMeta

	toolByName    map[string]*MCPToolMeta
	resourceByURI map[string]*MCPResourceMeta
	promptByName  map[string]*MCPPromptMeta

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
		authFuncs:     make([]func(c *Context) error, 0),
		appTitle:      title,
		appVer:        version,
	}

	for _, p := range providers {
		h.tools = append(h.tools, p.Tools()...)
		h.resources = append(h.resources, p.Resources()...)
		h.prompts = append(h.prompts, p.Prompts()...)

		for _, t := range p.Tools() {
			if _, exists := h.toolByName[t.Name]; exists {
				Warnf("mcp: duplicate tool name '%s' ignored", t.Name)
				continue
			}
			h.toolByName[t.Name] = t
		}
		for _, r := range p.Resources() {
			if _, exists := h.resourceByURI[r.URI]; exists {
				Warnf("mcp: duplicate resource uri '%s' ignored", r.URI)
				continue
			}
			h.resourceByURI[r.URI] = r
		}
		for _, pr := range p.Prompts() {
			if _, exists := h.promptByName[pr.Name]; exists {
				Warnf("mcp: duplicate prompt name '%s' ignored", pr.Name)
				continue
			}
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
	// Read the raw body once so that both single requests and JSON-RPC batches
	// (arrays) can be handled.
	var raw json.RawMessage
	if _, err := ctx.ShouldBind(&raw); err != nil {
		return h.writeJSONRPCErrorStatus(ctx, nil, errParse, "Parse error: "+err.Error(), http.StatusBadRequest)
	}

	payload := bytes.TrimSpace(raw)
	if len(payload) == 0 {
		return h.writeJSONRPCErrorStatus(ctx, nil, errParse, "Parse error: empty body", http.StatusBadRequest)
	}

	if payload[0] == '[' {
		return h.serveBatch(ctx, payload)
	}

	var request mcpJSONRPCRequest
	if err := json.Unmarshal(payload, &request); err != nil {
		return h.writeJSONRPCErrorStatus(ctx, nil, errParse, "Parse error: "+err.Error(), http.StatusBadRequest)
	}

	return h.serveOne(ctx, &request)
}

// ServeMCPGet handles GET /mcp. This server does not offer a server-to-client
// SSE stream, so it responds with 405 as allowed by the Streamable HTTP spec.
func (h *MCPHandler) ServeMCPGet(ctx MuxContext) error {
	return ctx.JSON(http.StatusMethodNotAllowed, map[string]string{
		"error": "SSE stream is not supported",
	})
}

// ServeMCPDelete handles DELETE /mcp. The server is stateless and does not
// allow clients to terminate sessions, so it responds with 405.
func (h *MCPHandler) ServeMCPDelete(ctx MuxContext) error {
	return ctx.JSON(http.StatusMethodNotAllowed, map[string]string{
		"error": "sessions are not supported",
	})
}

// Shutdown performs cleanup (call on application shutdown).
// The handler is stateless, so there is nothing to clean up.
func (h *MCPHandler) Shutdown() {}

// serveOne validates and dispatches a single JSON-RPC message.
func (h *MCPHandler) serveOne(ctx MuxContext, request *mcpJSONRPCRequest) error {
	if request.JSONRPC != "2.0" {
		return h.writeJSONRPCErrorStatus(ctx, request.ID, errInvalidRequest,
			"Invalid Request: jsonrpc must be 2.0", http.StatusBadRequest)
	}

	// JSON-RPC notifications never receive a response body.
	if request.isNotification() {
		ctx.Status(http.StatusAccepted)
		return nil
	}

	if !isValidJSONRPCID(request.ID) {
		return h.writeJSONRPCErrorStatus(ctx, nil, errInvalidRequest,
			"Invalid Request: id must be a string, number, or null", http.StatusBadRequest)
	}

	if resp := h.authenticate(ctx, request.ID); resp != nil {
		return h.writeResponse(ctx, resp)
	}

	return h.writeResponse(ctx, h.dispatch(ctx, request))
}

// serveBatch validates and dispatches a JSON-RPC batch.
func (h *MCPHandler) serveBatch(ctx MuxContext, payload []byte) error {
	var requests []mcpJSONRPCRequest
	if err := json.Unmarshal(payload, &requests); err != nil {
		return h.writeJSONRPCErrorStatus(ctx, nil, errParse, "Parse error: "+err.Error(), http.StatusBadRequest)
	}
	if len(requests) == 0 {
		return h.writeJSONRPCErrorStatus(ctx, nil, errInvalidRequest,
			"Invalid Request: empty batch", http.StatusBadRequest)
	}

	responses := make([]*mcpJSONRPCResponse, 0, len(requests))
	for i := range requests {
		req := &requests[i]

		if req.JSONRPC != "2.0" {
			responses = append(responses, mcpErrorResponse(req.ID, errInvalidRequest,
				"Invalid Request: jsonrpc must be 2.0"))
			continue
		}
		// The initialize request MUST NOT be part of a batch.
		if req.Method == "initialize" {
			responses = append(responses, mcpErrorResponse(req.ID, errInvalidRequest,
				"Invalid Request: initialize must not be batched"))
			continue
		}
		// Notifications produce no response entry.
		if req.isNotification() {
			continue
		}
		if !isValidJSONRPCID(req.ID) {
			responses = append(responses, mcpErrorResponse(nil, errInvalidRequest,
				"Invalid Request: id must be a string, number, or null"))
			continue
		}
		if resp := h.authenticate(ctx, req.ID); resp != nil {
			responses = append(responses, resp)
			continue
		}
		responses = append(responses, h.dispatch(ctx, req))
	}

	if len(responses) == 0 {
		ctx.Status(http.StatusAccepted)
		return nil
	}
	return ctx.JSON(http.StatusOK, responses)
}

// authenticate runs all provider auth funcs. It returns a JSON-RPC error
// response when authentication fails, or nil on success.
func (h *MCPHandler) authenticate(ctx MuxContext, id json.RawMessage) *mcpJSONRPCResponse {
	if len(h.authFuncs) == 0 {
		return nil
	}
	c := NewContext()
	c.initContext(ctx)
	defer c.resetContext()
	for _, auth := range h.authFuncs {
		if err := auth(c); err != nil {
			return mcpErrorResponse(id, errInternal, err.Error())
		}
	}
	return nil
}

// dispatch routes a request to the corresponding handler and builds a
// JSON-RPC response. Notifications must be filtered out by the caller.
func (h *MCPHandler) dispatch(ctx MuxContext, req *mcpJSONRPCRequest) *mcpJSONRPCResponse {
	switch req.Method {
	case "initialize":
		return h.handleInitialize(req)
	case "ping":
		// MCP ping MUST return an empty result object.
		return mcpResultResponse(req.ID, struct{}{})
	case "tools/list":
		return h.handleToolsList(req)
	case "tools/call":
		return h.handleToolsCall(ctx, req)
	case "resources/list":
		return h.handleResourcesList(req)
	case "resources/read":
		return h.handleResourcesRead(ctx, req)
	case "prompts/list":
		return h.handlePromptsList(req)
	case "prompts/get":
		return h.handlePromptsGet(ctx, req)
	default:
		return mcpErrorResponse(req.ID, errMethodNotFound,
			fmt.Sprintf("Method not found: %s", req.Method))
	}
}

// =================================== Handler Methods ===================================

func (h *MCPHandler) handleInitialize(req *mcpJSONRPCRequest) *mcpJSONRPCResponse {
	// Stateless mode: never issue an Mcp-Session-Id header.
	var params mcpInitializeParams
	_ = json.Unmarshal(req.Params, &params)

	result := mcpInitializeResult{
		ProtocolVersion: negotiateProtocolVersion(params.ProtocolVersion),
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

	return mcpResultResponse(req.ID, result)
}

// negotiateProtocolVersion echoes the client's requested version when it is
// supported, otherwise falls back to the latest version supported by server.
func negotiateProtocolVersion(requested string) string {
	if requested == mcpProtocolVersion {
		return requested
	}
	return mcpProtocolVersion
}

func (h *MCPHandler) handleToolsList(req *mcpJSONRPCRequest) *mcpJSONRPCResponse {
	result := mcpListToolsResult{Tools: h.tools}
	if result.Tools == nil {
		result.Tools = make([]*MCPToolMeta, 0)
	}
	return mcpResultResponse(req.ID, result)
}

func (h *MCPHandler) handleToolsCall(ctx MuxContext, req *mcpJSONRPCRequest) *mcpJSONRPCResponse {
	var params mcpCallToolParams
	if err := json.Unmarshal(req.Params, &params); err != nil {
		return mcpErrorResponse(req.ID, errInvalidParams, "Invalid params")
	}

	tool, ok := h.toolByName[params.Name]
	if !ok {
		return mcpErrorResponse(req.ID, errInvalidParams,
			fmt.Sprintf("Unknown tool: %s", params.Name))
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
		return mcpResultResponse(req.ID, mcpToolCallResult{
			Content: []mcpContentItem{{
				Type: "text",
				Text: err.Error(),
			}},
			IsError: true,
		})
	}

	resultJSON, _ := utils.JsonMarshal(result)

	return mcpResultResponse(req.ID, mcpToolCallResult{
		Content: []mcpContentItem{{
			Type: "text",
			Text: string(resultJSON),
		}},
	})
}

func (h *MCPHandler) handleResourcesList(req *mcpJSONRPCRequest) *mcpJSONRPCResponse {
	result := mcpListResourcesResult{Resources: h.resources}
	if result.Resources == nil {
		result.Resources = make([]*MCPResourceMeta, 0)
	}
	return mcpResultResponse(req.ID, result)
}

func (h *MCPHandler) handleResourcesRead(ctx MuxContext, req *mcpJSONRPCRequest) *mcpJSONRPCResponse {
	var params mcpReadResourceParams
	if err := json.Unmarshal(req.Params, &params); err != nil {
		return mcpErrorResponse(req.ID, errInvalidParams, "Invalid params")
	}

	res, ok := h.resourceByURI[params.URI]
	if !ok {
		return mcpErrorResponseWithData(req.ID, errResourceNotFound,
			"Resource not found", map[string]any{"uri": params.URI})
	}

	c := NewContext()
	c.initContext(ctx)
	defer c.resetContext()

	args := res.NewCallArgs(c, nil)
	result, err := res.Call(args)
	if err != nil {
		return mcpErrorResponse(req.ID, errInternal, err.Error())
	}

	resultJSON, _ := utils.JsonMarshal(result)

	return mcpResultResponse(req.ID, mcpResourceReadResult{
		Contents: []mcpResourceContents{{
			URI:      res.URI,
			MimeType: res.MimeType,
			Text:     string(resultJSON),
		}},
	})
}

func (h *MCPHandler) handlePromptsList(req *mcpJSONRPCRequest) *mcpJSONRPCResponse {
	result := mcpListPromptsResult{Prompts: h.prompts}
	if result.Prompts == nil {
		result.Prompts = make([]*MCPPromptMeta, 0)
	}
	return mcpResultResponse(req.ID, result)
}

func (h *MCPHandler) handlePromptsGet(ctx MuxContext, req *mcpJSONRPCRequest) *mcpJSONRPCResponse {
	var params mcpGetPromptParams
	if err := json.Unmarshal(req.Params, &params); err != nil {
		return mcpErrorResponse(req.ID, errInvalidParams, "Invalid params")
	}

	prompt, ok := h.promptByName[params.Name]
	if !ok {
		return mcpErrorResponse(req.ID, errInvalidParams,
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
		return mcpErrorResponse(req.ID, errInternal, err.Error())
	}

	resultJSON, _ := utils.JsonMarshal(result)

	return mcpResultResponse(req.ID, mcpPromptGetResult{
		Description: prompt.Description,
		Messages: []mcpPromptMessage{{
			Role: "user",
			Content: mcpContentItem{
				Type: "text",
				Text: string(resultJSON),
			},
		}},
	})
}

// =================================== Response Helpers ===================================

func mcpResultResponse(id json.RawMessage, result any) *mcpJSONRPCResponse {
	return &mcpJSONRPCResponse{
		JSONRPC: "2.0",
		ID:      id,
		Result:  result,
	}
}

func mcpErrorResponse(id json.RawMessage, code int, message string) *mcpJSONRPCResponse {
	return mcpErrorResponseWithData(id, code, message, nil)
}

func mcpErrorResponseWithData(id json.RawMessage, code int, message string, data any) *mcpJSONRPCResponse {
	return &mcpJSONRPCResponse{
		JSONRPC: "2.0",
		ID:      id,
		Error: &mcpRPCError{
			Code:    code,
			Message: message,
			Data:    data,
		},
	}
}

func (h *MCPHandler) writeResponse(ctx MuxContext, resp *mcpJSONRPCResponse) error {
	if resp == nil {
		ctx.Status(http.StatusAccepted)
		return nil
	}
	return ctx.JSON(http.StatusOK, resp)
}

func (h *MCPHandler) writeJSONRPCErrorStatus(ctx MuxContext, id json.RawMessage, code int, message string, status int) error {
	return ctx.JSON(status, mcpErrorResponse(id, code, message))
}
