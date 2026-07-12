package fastapi

import (
	"fmt"
	"reflect"
	"strings"
	"unicode"

	"github.com/Chendemo12/fastapi/openapi"
	"github.com/Chendemo12/fastapi/pathschema"
)

// MCPMethodKind distinguishes the three MCP primitive types.
type MCPMethodKind string

func (k MCPMethodKind) String() string { return string(k) }

const (
	MCPTool     MCPMethodKind = "Tool"
	MCPResource MCPMethodKind = "Resource"
	MCPPrompt   MCPMethodKind = "Prompt"
)

const (
	MCPMethodNameMinLength = len("ToolX") // prefix + at least 1 char
	MCPFirstInParamOffset  = 1            // first param after receiver = *Context
	MCPOutParamNum         = 2            // (any, error)
	MCPFirstOutParamOffset = 0
	MCPLastOutParamOffset  = 1
	MCPFirstInParamName    = "Context"
	MCPLastOutParamName    = "error"
)

// MCPProvider is the user-facing interface for defining MCP capabilities
// on a struct, mirroring GroupRouter for HTTP routes.
//
// Usage: embed BaseMCPProvider and define methods with Tool/Resource/Prompt prefixes.
type MCPProvider interface {
	// Prefix serves as a namespace prefix for tool/resource/prompt names.
	// If empty, no prefix is applied.
	Prefix() string

	// Tags for categorization.
	Tags() []string

	// PathSchema controls how method-name suffixes are converted to names.
	PathSchema() pathschema.RoutePathSchema

	// Summary maps method-name -> summary string.
	Summary() map[string]string

	// Description maps method-name -> description string.
	Description() map[string]string

	// ResourceURI maps method-name -> URI template for Resource methods.
	// If not defined, a default URI is derived from Prefix + method suffix.
	ResourceURI() map[string]string

	// AuthFunc provides authentication for this MCP provider's requests.
	// Return nil to allow the request; return error to reject it with a JSON-RPC error.
	// The provided *Context has a valid MuxContext for reading headers.
	AuthFunc(c *Context) error
}

// BaseMCPProvider provides default implementations for MCPProvider.
// Embed this in your struct and override methods as needed.
type BaseMCPProvider struct{}

func (b *BaseMCPProvider) Prefix() string { return "" }

func (b *BaseMCPProvider) Tags() []string { return nil }

func (b *BaseMCPProvider) PathSchema() pathschema.RoutePathSchema {
	return pathschema.LowerCaseUnderline()
}

func (b *BaseMCPProvider) Summary() map[string]string { return nil }

func (b *BaseMCPProvider) Description() map[string]string { return nil }

func (b *BaseMCPProvider) ResourceURI() map[string]string { return nil }

func (b *BaseMCPProvider) AuthFunc(c *Context) error { return nil }

// =================================== MCPProviderMeta ===================================

// MCPProviderMeta scans a struct's methods via reflection and builds MCP metadata.
// Mirrors GroupRouterMeta.
type MCPProviderMeta struct {
	provider      MCPProvider
	providerValue reflect.Value
	pkg           string
	tools         []*MCPToolMeta
	resources     []*MCPResourceMeta
	prompts       []*MCPPromptMeta
	tags          []string
}

// NewMCPProviderMeta creates a new scanner for an MCP provider struct.
func NewMCPProviderMeta(provider MCPProvider) *MCPProviderMeta {
	return &MCPProviderMeta{provider: provider}
}

func (m *MCPProviderMeta) String() string { return m.pkg }

// Init performs full initialization: Scan + ScanInner.
func (m *MCPProviderMeta) Init() (err error) {
	err = m.Scan()
	if err != nil {
		return err
	}
	err = m.ScanInner()
	return
}

// Scan reflects the struct and discovers MCP methods.
func (m *MCPProviderMeta) Scan() (err error) {
	m.providerValue = reflect.ValueOf(m.provider)
	obj := reflect.TypeOf(m.provider)

	if obj.Kind() != reflect.Struct && obj.Kind() != reflect.Ptr {
		return fmt.Errorf("mcp provider: '%s' not a struct", obj.String())
	}

	if obj.Kind() == reflect.Ptr {
		m.pkg = obj.Elem().String()
	} else {
		m.pkg = obj.String()
	}

	m.tools = make([]*MCPToolMeta, 0)
	m.resources = make([]*MCPResourceMeta, 0)
	m.prompts = make([]*MCPPromptMeta, 0)

	m.scanTags()
	err = m.scanMethod()
	return
}

// ScanInner initializes all discovered method metadata.
func (m *MCPProviderMeta) ScanInner() (err error) {
	for _, tool := range m.tools {
		err = tool.Init()
		if err != nil {
			return err
		}
	}
	for _, res := range m.resources {
		err = res.Init()
		if err != nil {
			return err
		}
	}
	for _, prompt := range m.prompts {
		err = prompt.Init()
		if err != nil {
			return err
		}
	}
	return
}

// Accessors

func (m *MCPProviderMeta) Tools() []*MCPToolMeta         { return m.tools }
func (m *MCPProviderMeta) Resources() []*MCPResourceMeta { return m.resources }
func (m *MCPProviderMeta) Prompts() []*MCPPromptMeta     { return m.prompts }
func (m *MCPProviderMeta) Tags() []string                { return m.tags }

func (m *MCPProviderMeta) scanTags() {
	obj := reflect.TypeOf(m.provider)
	if obj.Kind() == reflect.Ptr {
		obj = obj.Elem()
	}
	tags := m.provider.Tags()
	if len(tags) == 0 {
		tags = append(tags, obj.Name())
	}
	m.tags = tags
}

// scanMethod iterates all exported methods and dispatches by prefix.
func (m *MCPProviderMeta) scanMethod() (err error) {
	obj := reflect.TypeOf(m.provider)
	for i := 0; i < obj.NumMethod(); i++ {
		method := obj.Method(i)

		// Must be exported
		if unicode.IsLower([]rune(method.Name)[0]) {
			continue
		}

		kind, suffix := classifyMCPMethod(method)
		if kind == "" {
			continue
		}

		switch kind {
		case MCPTool:
			tool, ok := m.newToolMeta(method, suffix)
			if ok {
				m.tools = append(m.tools, tool)
			}
		case MCPResource:
			res, ok := m.newResourceMeta(method, suffix)
			if ok {
				m.resources = append(m.resources, res)
			}
		case MCPPrompt:
			prompt, ok := m.newPromptMeta(method, suffix)
			if ok {
				m.prompts = append(m.prompts, prompt)
			}
		}
	}
	return nil
}

// mcpInterfaceMethods are method names on the MCPProvider interface itself.
// These must not be classified as MCP capabilities.
var mcpInterfaceMethods = map[string]bool{
	"Prefix":      true,
	"Tags":        true,
	"PathSchema":  true,
	"Summary":     true,
	"Description": true,
	"ResourceURI": true,
}

// classifyMCPMethod checks the method name prefix and returns the kind and suffix.
func classifyMCPMethod(method reflect.Method) (MCPMethodKind, string) {
	if len(method.Name) < MCPMethodNameMinLength {
		return "", ""
	}

	// Skip MCPProvider interface methods
	if mcpInterfaceMethods[method.Name] {
		return "", ""
	}

	prefixes := []struct {
		prefix string
		kind   MCPMethodKind
	}{
		{"Tool", MCPTool},
		{"Resource", MCPResource},
		{"Prompt", MCPPrompt},
	}

	for _, p := range prefixes {
		if len(method.Name) > len(p.prefix) && strings.HasPrefix(method.Name, p.prefix) {
			suffix := method.Name[len(p.prefix):]
			return p.kind, suffix
		}
	}
	return "", ""
}

// validateMCPSignature checks that the method conforms to the MCP signature pattern:
//
//	func (p *Provider) Xxx(c *Context, [params...]) (any, error)
func (m *MCPProviderMeta) validateMCPSignature(method reflect.Method) error {
	inNum := method.Type.NumIn()   // includes receiver
	outNum := method.Type.NumOut() // must be 2

	// 1. Must have at least receiver + *Context
	if inNum < MCPFirstInParamOffset+1 {
		return fmt.Errorf("mcp method '%s' must have at least *Context as first param", method.Name)
	}

	// 2. First param (after receiver) must be *Context
	firstParam := method.Type.In(MCPFirstInParamOffset)
	if firstParam.Kind() != reflect.Ptr || firstParam.Elem().Name() != MCPFirstInParamName {
		return fmt.Errorf("mcp method '%s' first param must be *Context, got %s", method.Name, firstParam)
	}

	// 3. Must return exactly 2 values
	if outNum != MCPOutParamNum {
		return fmt.Errorf("mcp method '%s' must return exactly (any, error), got %d values", method.Name, outNum)
	}

	// 4. Last return must be error
	lastOut := method.Type.Out(MCPLastOutParamOffset)
	if lastOut.Name() != MCPLastOutParamName {
		return fmt.Errorf("mcp method '%s' last return must be error, got %s", method.Name, lastOut)
	}

	// 5. Check all additional params are valid types
	for i := MCPFirstInParamOffset + 1; i < inNum; i++ {
		param := method.Type.In(i)
		kind := param.Kind()
		if kind == reflect.Ptr {
			kind = param.Elem().Kind()
		}
		for _, illegal := range openapi.IllegalRouteParamType {
			if kind == illegal {
				return fmt.Errorf("mcp method '%s' param %d has illegal type %s", method.Name, i, kind)
			}
		}
	}

	// 6. First return value must not be illegal type
	firstOut := method.Type.Out(MCPFirstOutParamOffset)
	firstOutKind := firstOut.Kind()
	if firstOutKind == reflect.Ptr {
		firstOutKind = firstOut.Elem().Kind()
	}
	for _, illegal := range openapi.IllegalRouteParamType {
		if firstOutKind == illegal {
			return fmt.Errorf("mcp method '%s' first return value has illegal type %s", method.Name, firstOutKind)
		}
	}

	return nil
}

// deriveMCPName converts a method suffix to an MCP-compatible name using PathSchema.
// Unlike pathschema.Format (which is for HTTP URLs), this produces a clean name
// suitable for MCP tool/resource/prompt identifiers.
func (m *MCPProviderMeta) deriveMCPName(suffix string) string {
	schema := m.provider.PathSchema()
	if schema == nil {
		schema = pathschema.Default()
	}
	spans := schema.Split(suffix)
	name := strings.Join(spans, schema.Connector())
	return strings.ToLower(name)
}

// deriveFullName combines prefix and derived name.
func (m *MCPProviderMeta) deriveFullName(kind MCPMethodKind, suffix string) string {
	name := m.deriveMCPName(suffix)
	prefix := m.provider.Prefix()
	if prefix == "" {
		return name
	}
	switch kind {
	case MCPTool:
		return prefix + "_" + name
	case MCPResource:
		return prefix + "/" + name
	case MCPPrompt:
		return prefix + "." + name
	}
	return name
}

// deriveResourceURI creates a URI for a resource method.
func (m *MCPProviderMeta) deriveResourceURI(methodName, suffix string) string {
	if uri, ok := m.provider.ResourceURI()[methodName]; ok {
		return uri
	}
	prefix := m.provider.Prefix()
	name := m.deriveMCPName(suffix)
	if prefix != "" {
		return fmt.Sprintf("resource://%s/%s", prefix, name)
	}
	return fmt.Sprintf("resource://default/%s", name)
}

// deriveSummary returns the summary for a method.
func (m *MCPProviderMeta) deriveSummary(methodName, name string) string {
	summaries := m.provider.Summary()
	if summaries != nil {
		if v, ok := summaries[methodName]; ok {
			return v
		}
	}
	return name
}

// deriveDescription returns the description for a method.
func (m *MCPProviderMeta) deriveDescription(methodName, summary string) string {
	descriptions := m.provider.Description()
	if descriptions != nil {
		if v, ok := descriptions[methodName]; ok {
			return v
		}
	}
	return summary
}

// factory methods for creating metadata objects

func (m *MCPProviderMeta) newToolMeta(method reflect.Method, suffix string) (*MCPToolMeta, bool) {
	if err := m.validateMCPSignature(method); err != nil {
		Warnf("mcp-provider: '%s' tool '%s' validation failed: %v", m.pkg, method.Name, err)
		return nil, false
	}
	tool := NewMCPToolMeta(method, m, suffix)
	return tool, true
}

func (m *MCPProviderMeta) newResourceMeta(method reflect.Method, suffix string) (*MCPResourceMeta, bool) {
	if err := m.validateMCPSignature(method); err != nil {
		Warnf("mcp-provider: '%s' resource '%s' validation failed: %v", m.pkg, method.Name, err)
		return nil, false
	}
	res := NewMCPResourceMeta(method, m, suffix)
	return res, true
}

func (m *MCPProviderMeta) newPromptMeta(method reflect.Method, suffix string) (*MCPPromptMeta, bool) {
	if err := m.validateMCPSignature(method); err != nil {
		Warnf("mcp-provider: '%s' prompt '%s' validation failed: %v", m.pkg, method.Name, err)
		return nil, false
	}
	prompt := NewMCPPromptMeta(method, m, suffix)
	return prompt, true
}
