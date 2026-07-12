package fastapi

import (
	"encoding/json"
	"errors"
	"reflect"
	"testing"
	"time"

	"github.com/Chendemo12/fastapi/openapi"
)

// =================================== Test MCP Providers ===================================

// TestToolProvider is a simple MCP provider with various tool signatures.
type TestToolProvider struct {
	BaseMCPProvider
}

func (t *TestToolProvider) Prefix() string { return "test" }

// ToolEcho - simple tool with one string arg
func (t *TestToolProvider) ToolEcho(ctx *Context, message string) (string, error) {
	return message, nil
}

// ToolAdd - tool with two int args
func (t *TestToolProvider) ToolAdd(ctx *Context, a int, b int) (int, error) {
	return a + b, nil
}

// ToolNoArgs - tool with no additional args
func (t *TestToolProvider) ToolPing(ctx *Context) (string, error) {
	return "pong", nil
}

// NotAMCPMethod - shouldn't be discovered (no prefix)
func (t *TestToolProvider) GetSomething(ctx *Context) (string, error) {
	return "", nil
}

// =================================== Struct Request Tool ===================================

type CalcRequest struct {
	X float64 `json:"x" validate:"required" description:"first number"`
	Y float64 `json:"y" validate:"required" description:"second number"`
}

type CalcResponse struct {
	Result float64 `json:"result"`
}

type StructToolProvider struct {
	BaseMCPProvider
}

func (s *StructToolProvider) Prefix() string { return "math" }

func (s *StructToolProvider) ToolCalculate(ctx *Context, req CalcRequest) (*CalcResponse, error) {
	return &CalcResponse{Result: req.X + req.Y}, nil
}

// =================================== Resource & Prompt Provider ===================================

type SearchResult struct {
	Query string `json:"q"`
	Limit int    `json:"n"`
}

type ConfigData struct {
	Theme string `json:"theme"`
	Lang  string `json:"lang"`
}

type GreetingMsg struct {
	Text string `json:"text"`
}

type FullMCPProvider struct {
	BaseMCPProvider
}

func (f *FullMCPProvider) Prefix() string { return "app" }

func (f *FullMCPProvider) ToolSearch(ctx *Context, query string, limit int) (*SearchResult, error) {
	return &SearchResult{Query: query, Limit: limit}, nil
}

func (f *FullMCPProvider) ResourceConfig(ctx *Context) (*ConfigData, error) {
	return &ConfigData{Theme: "dark", Lang: "zh"}, nil
}

func (f *FullMCPProvider) ResourceURI() map[string]string {
	return map[string]string{
		"ResourceConfig": "config://app/settings",
	}
}

func (f *FullMCPProvider) PromptGreeting(ctx *Context, name string) (*GreetingMsg, error) {
	return &GreetingMsg{Text: "Hello " + name}, nil
}

// =================================== Tests ===================================

func TestClassifyMCPMethod(t *testing.T) {
	obj := reflect.TypeOf(&TestToolProvider{})

	tests := []struct {
		methodName string
		wantKind   MCPMethodKind
		wantSuffix string
	}{
		{"ToolEcho", MCPTool, "Echo"},
		{"ToolAdd", MCPTool, "Add"},
		{"ToolPing", MCPTool, "Ping"},
		{"GetSomething", "", ""},
	}

	for _, tc := range tests {
		m, ok := obj.MethodByName(tc.methodName)
		if !ok && tc.wantKind == "" {
			continue
		}
		if !ok {
			t.Errorf("method %s not found", tc.methodName)
			continue
		}
		kind, suffix := classifyMCPMethod(m)
		if kind != tc.wantKind || suffix != tc.wantSuffix {
			t.Errorf("%s: got (%q, %q), want (%q, %q)", tc.methodName, kind, suffix, tc.wantKind, tc.wantSuffix)
		}
	}

	// Also test FullMCPProvider for Resource/Prompt
	fullObj := reflect.TypeOf(&FullMCPProvider{})
	fullTests := []struct {
		methodName string
		wantKind   MCPMethodKind
		wantSuffix string
	}{
		{"ResourceConfig", MCPResource, "Config"},
		{"PromptGreeting", MCPPrompt, "Greeting"},
	}
	for _, tc := range fullTests {
		m, _ := fullObj.MethodByName(tc.methodName)
		kind, suffix := classifyMCPMethod(m)
		if kind != tc.wantKind || suffix != tc.wantSuffix {
			t.Errorf("%s: got (%q, %q), want (%q, %q)", tc.methodName, kind, suffix, tc.wantKind, tc.wantSuffix)
		}
	}
}

func TestMCPProviderMetaScan(t *testing.T) {
	provider := &TestToolProvider{}
	meta := NewMCPProviderMeta(provider)

	err := meta.Init()
	if err != nil {
		t.Fatalf("Init failed: %v", err)
	}

	if len(meta.Tools()) != 3 {
		t.Errorf("expected 3 tools, got %d", len(meta.Tools()))
	}

	if len(meta.Resources()) != 0 {
		t.Errorf("expected 0 resources, got %d", len(meta.Resources()))
	}

	if len(meta.Prompts()) != 0 {
		t.Errorf("expected 0 prompts, got %d", len(meta.Prompts()))
	}

	// Check tool names
	names := make(map[string]bool)
	for _, tool := range meta.Tools() {
		names[tool.Name] = true
	}

	expected := []string{"test_echo", "test_add", "test_ping"}
	for _, name := range expected {
		if !names[name] {
			t.Errorf("expected tool %q not found, got: %v", name, names)
		}
	}
}

func TestMCPToolMetaInputSchema(t *testing.T) {
	provider := &TestToolProvider{}
	meta := NewMCPProviderMeta(provider)
	err := meta.Init()
	if err != nil {
		t.Fatalf("Init failed: %v", err)
	}

	// Find ToolAdd
	var addTool *MCPToolMeta
	for _, tool := range meta.Tools() {
		if tool.Name == "test_add" {
			addTool = tool
			break
		}
	}
	if addTool == nil {
		t.Fatal("ToolAdd not found")
	}

	// Check input schema
	schema := addTool.InputSchema
	if schema["type"] != "object" {
		t.Errorf("expected type 'object', got %v", schema["type"])
	}

	props, ok := schema["properties"].(map[string]any)
	if !ok {
		t.Fatal("expected properties map")
	}

	// Go reflection cannot get param names, so properties use QueryName (type_index)
	// a int → "int_2", b int → "int_3"
	if len(props) != 2 {
		t.Errorf("expected 2 properties, got %d: %v", len(props), props)
	}

	required, _ := schema["required"].([]string)
	if len(required) != 2 {
		t.Errorf("expected 2 required fields, got %d", len(required))
	}

	// Check property types
	for _, key := range required {
		prop, ok := props[key].(map[string]any)
		if !ok {
			t.Errorf("property %q is not an object", key)
			continue
		}
		if prop["type"] != "integer" {
			t.Errorf("expected property %q type 'integer', got %v", key, prop["type"])
		}
	}
}

func TestMCPToolMetaStructInputSchema(t *testing.T) {
	provider := &StructToolProvider{}
	meta := NewMCPProviderMeta(provider)
	err := meta.Init()
	if err != nil {
		t.Fatalf("Init failed: %v", err)
	}

	if len(meta.Tools()) != 1 {
		t.Fatalf("expected 1 tool, got %d", len(meta.Tools()))
	}

	tool := meta.Tools()[0]
	if tool.Name != "math_calculate" {
		t.Errorf("expected name 'math_calculate', got %q", tool.Name)
	}

	schema := tool.InputSchema
	props, ok := schema["properties"].(map[string]any)
	if !ok {
		t.Fatal("expected properties map")
	}

	if _, ok := props["x"]; !ok {
		t.Error("expected 'x' property")
	}
	if _, ok := props["y"]; !ok {
		t.Error("expected 'y' property")
	}

	// Check x property
	xProp := props["x"].(map[string]any)
	if xProp["type"] != "number" {
		t.Errorf("expected 'x' type 'number', got %v", xProp["type"])
	}
}

func TestMCPToolMetaCall(t *testing.T) {
	provider := &TestToolProvider{}
	meta := NewMCPProviderMeta(provider)
	err := meta.Init()
	if err != nil {
		t.Fatalf("Init failed: %v", err)
	}

	// Test ToolEcho
	var echoTool *MCPToolMeta
	for _, tool := range meta.Tools() {
		if tool.Name == "test_echo" {
			echoTool = tool
			break
		}
	}
	if echoTool == nil {
		t.Fatal("ToolEcho not found")
	}

	ctx := NewContext()
	ctx.initContext(nil)

	// Go reflection cannot get param names; use QueryName from RouteParam
	// message string at index 2 → "string_2"
	args := echoTool.NewCallArgs(ctx, map[string]any{"string_2": "hello world"})
	result, err := echoTool.Call(args)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if result != "hello world" {
		t.Errorf("expected 'hello world', got %v", result)
	}

	// Test ToolAdd
	var addTool *MCPToolMeta
	for _, tool := range meta.Tools() {
		if tool.Name == "test_add" {
			addTool = tool
			break
		}
	}
	if addTool == nil {
		t.Fatal("ToolAdd not found")
	}

	// a int at index 2 → "int_2", b int at index 3 → "int_3"
	args = addTool.NewCallArgs(ctx, map[string]any{"int_2": float64(3), "int_3": float64(4)})
	result, err = addTool.Call(args)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if result != 7 {
		t.Errorf("expected 7, got %v", result)
	}
}

func TestMCPToolMetaCallStructArg(t *testing.T) {
	provider := &StructToolProvider{}
	meta := NewMCPProviderMeta(provider)
	err := meta.Init()
	if err != nil {
		t.Fatalf("Init failed: %v", err)
	}

	tool := meta.Tools()[0]
	ctx := NewContext()
	ctx.initContext(nil)

	args := tool.NewCallArgs(ctx, map[string]any{"x": float64(1.5), "y": float64(2.5)})
	result, err := tool.Call(args)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	resp, ok := result.(*CalcResponse)
	if !ok {
		t.Fatalf("expected *CalcResponse, got %T", result)
	}
	if resp.Result != 4.0 {
		t.Errorf("expected 4.0, got %v", resp.Result)
	}
}

func TestFullMCPProvider(t *testing.T) {
	provider := &FullMCPProvider{}
	meta := NewMCPProviderMeta(provider)
	err := meta.Init()
	if err != nil {
		t.Fatalf("Init failed: %v", err)
	}

	if len(meta.Tools()) != 1 {
		t.Errorf("expected 1 tool, got %d", len(meta.Tools()))
	}
	if len(meta.Resources()) != 1 {
		t.Errorf("expected 1 resource, got %d", len(meta.Resources()))
	}
	if len(meta.Prompts()) != 1 {
		t.Errorf("expected 1 prompt, got %d", len(meta.Prompts()))
	}

	// Check resource URI override
	res := meta.Resources()[0]
	if res.URI != "config://app/settings" {
		t.Errorf("expected URI 'config://app/settings', got %q", res.URI)
	}

	// Check prompt arguments (uses QueryName since Go reflection can't get param names)
	prompt := meta.Prompts()[0]
	if len(prompt.Arguments) != 1 {
		t.Errorf("expected 1 prompt argument, got %d", len(prompt.Arguments))
	}
	// name string at index 2 → QueryName = "string_2"
	if prompt.Arguments[0].Name != "string_2" {
		t.Errorf("expected argument name 'string_2', got %q", prompt.Arguments[0].Name)
	}
}

func TestMCPHandlerToolsList(t *testing.T) {
	provider := &TestToolProvider{}
	meta := NewMCPProviderMeta(provider)
	err := meta.Init()
	if err != nil {
		t.Fatalf("Init failed: %v", err)
	}

	handler := NewMCPHandler([]*MCPProviderMeta{meta}, "TestApp", "1.0.0")

	if len(handler.tools) != 3 {
		t.Errorf("expected 3 tools, got %d", len(handler.tools))
	}
	if len(handler.resources) != 0 {
		t.Errorf("expected 0 resources, got %d", len(handler.resources))
	}
	if len(handler.prompts) != 0 {
		t.Errorf("expected 0 prompts, got %d", len(handler.prompts))
	}

	// Check toolByName
	if _, ok := handler.toolByName["test_echo"]; !ok {
		t.Error("expected 'test_echo' in toolByName")
	}
	if _, ok := handler.toolByName["test_add"]; !ok {
		t.Error("expected 'test_add' in toolByName")
	}
}

func TestJSONRPCRequestParsing(t *testing.T) {
	body := `{"jsonrpc":"2.0","id":1,"method":"tools/list"}`
	var req mcpJSONRPCRequest
	err := json.Unmarshal([]byte(body), &req)
	if err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}

	if req.JSONRPC != "2.0" {
		t.Errorf("expected jsonrpc '2.0', got %q", req.JSONRPC)
	}
	if req.Method != "tools/list" {
		t.Errorf("expected method 'tools/list', got %q", req.Method)
	}
}

func TestJSONRPCResponseSerialization(t *testing.T) {
	resp := mcpJSONRPCResponse{
		JSONRPC: "2.0",
		ID:      json.RawMessage("1"),
		Result:  map[string]any{"tools": []any{}},
	}

	data, err := json.Marshal(resp)
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}

	var parsed map[string]any
	err = json.Unmarshal(data, &parsed)
	if err != nil {
		t.Fatalf("unmarshal back failed: %v", err)
	}

	if parsed["jsonrpc"] != "2.0" {
		t.Errorf("expected jsonrpc '2.0'")
	}
}

func TestGoKindToJSONType(t *testing.T) {
	tests := []struct {
		kind reflect.Kind
		want string
	}{
		{reflect.Bool, "boolean"},
		{reflect.Int, "integer"},
		{reflect.Int64, "integer"},
		{reflect.Float64, "number"},
		{reflect.String, "string"},
		{reflect.Struct, "object"},
		{reflect.Slice, "array"},
	}

	for _, tc := range tests {
		got := goKindToJSONType(tc.kind)
		if got != tc.want {
			t.Errorf("goKindToJSONType(%v) = %q, want %q", tc.kind, got, tc.want)
		}
	}
}

func TestMCPSessionManager(t *testing.T) {
	sm := newMCPSessionManager(100 * time.Millisecond)

	s := sm.create()
	if s.ID == "" {
		t.Error("expected non-empty session ID")
	}

	got, ok := sm.get(s.ID)
	if !ok {
		t.Error("expected session to exist")
	}
	if got.ID != s.ID {
		t.Error("session IDs don't match")
	}

	// Test deletion
	sm.delete(s.ID)
	_, ok = sm.get(s.ID)
	if ok {
		t.Error("expected session to be deleted")
	}
}

func TestMCPToolMetaHasArgs(t *testing.T) {
	provider := &TestToolProvider{}
	meta := NewMCPProviderMeta(provider)
	_ = meta.Init()

	for _, tool := range meta.Tools() {
		switch tool.Name {
		case "test_ping":
			if tool.HasArgs() {
				t.Errorf("ToolPing should have no args")
			}
		case "test_echo":
			if !tool.HasArgs() {
				t.Errorf("ToolEcho should have args")
			}
		case "test_add":
			if !tool.HasArgs() {
				t.Errorf("ToolAdd should have args")
			}
		}
	}
}

// =================================== Validate OpenAPI reuse ===================================

func TestRouteParamReuseForMCP(t *testing.T) {
	// Verify that RouteParam works correctly when used in MCP context
	rt := reflect.TypeOf("")
	param := openapi.NewRouteParam(rt, 1, openapi.RouteParamQuery)
	err := param.Init()
	if err != nil {
		t.Fatalf("RouteParam.Init failed: %v", err)
	}

	if param.SchemaType() != openapi.StringType {
		t.Errorf("expected StringType, got %v", param.SchemaType())
	}
	if param.PrototypeKind != reflect.String {
		t.Errorf("expected reflect.String, got %v", param.PrototypeKind)
	}
}

// =================================== MCP Auth Tests ===================================

// RejectAuthProvider always rejects.
type RejectAuthProvider struct {
	BaseMCPProvider
}

func (r *RejectAuthProvider) Prefix() string { return "reject" }

func (r *RejectAuthProvider) ToolPing(c *Context) (string, error) {
	return "pong", nil
}

func (r *RejectAuthProvider) AuthFunc(c *Context) error {
	return errors.New("access denied")
}

func TestMCPProviderAuthFuncReject(t *testing.T) {
	provider := &RejectAuthProvider{}
	meta := NewMCPProviderMeta(provider)
	err := meta.Init()
	if err != nil {
		t.Fatalf("Init failed: %v", err)
	}

	handler := NewMCPHandler([]*MCPProviderMeta{meta}, "Test", "1.0.0")
	if len(handler.authFuncs) != 1 {
		t.Fatalf("expected 1 auth func, got %d", len(handler.authFuncs))
	}

	c := NewContext()
	c.initContext(nil)
	if err := handler.authFuncs[0](c); err == nil {
		t.Error("expected auth to fail, but got nil")
	}
}

func TestMCPProviderAuthFuncDefault(t *testing.T) {
	// BaseMCPProvider has nil auth (AuthFunc returns nil)
	provider := &TestToolProvider{}
	meta := NewMCPProviderMeta(provider)
	err := meta.Init()
	if err != nil {
		t.Fatalf("Init failed: %v", err)
	}

	handler := NewMCPHandler([]*MCPProviderMeta{meta}, "Test", "1.0.0")
	if len(handler.authFuncs) != 1 {
		t.Fatalf("expected 1 auth func, got %d", len(handler.authFuncs))
	}

	c := NewContext()
	c.initContext(nil)
	if err := handler.authFuncs[0](c); err != nil {
		t.Errorf("default auth should pass, got: %v", err)
	}
}
