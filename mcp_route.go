package fastapi

import (
	"encoding/json"
	"reflect"
	"unicode"

	"github.com/Chendemo12/fastapi/openapi"
	"github.com/Chendemo12/fastapi/utils"
)

// =================================== MCPToolMeta ===================================

// MCPToolMeta holds metadata for a single MCP Tool, analogous to GroupRoute.
type MCPToolMeta struct {
	method   reflect.Method
	provider *MCPProviderMeta

	Name        string         `json:"name"`
	Description string         `json:"description,omitempty"`
	InputSchema map[string]any `json:"inputSchema"`

	inParams     []*openapi.RouteParam // all params after *Context
	outParam     *openapi.RouteParam   // first return value
	index        int                   // method.Index for calling
	handlerInNum int                   // num input params (excl. receiver)
}

// NewMCPToolMeta creates a new tool metadata instance.
func NewMCPToolMeta(method reflect.Method, provider *MCPProviderMeta, suffix string) *MCPToolMeta {
	name := provider.deriveFullName(MCPTool, suffix)
	return &MCPToolMeta{
		method:   method,
		provider: provider,
		index:    method.Index,
		Name:     name,
	}
}

// Init initializes the tool metadata: builds param descriptors, input schema.
func (t *MCPToolMeta) Init() (err error) {
	t.handlerInNum = t.method.Type.NumIn() // includes receiver

	// Set description from provider
	summary := t.provider.deriveSummary(t.method.Name, t.Name)
	t.Description = t.provider.deriveDescription(t.method.Name, summary)

	// Build inParams for all params after *Context
	t.inParams = make([]*openapi.RouteParam, 0)
	for i := MCPFirstInParamOffset + 1; i < t.handlerInNum; i++ {
		param := openapi.NewRouteParam(t.method.Type.In(i), i, openapi.RouteParamQuery)
		err = param.Init()
		if err != nil {
			return err
		}
		t.inParams = append(t.inParams, param)
	}

	// Build the output param (first return value)
	t.outParam = openapi.NewRouteParam(t.method.Type.Out(MCPFirstOutParamOffset), 0, openapi.RouteParamResponse)
	err = t.outParam.Init()
	if err != nil {
		return err
	}

	// Build JSON Schema input schema
	t.InputSchema = t.buildInputSchema()
	return nil
}

// buildInputSchema generates a JSON Schema object for the tool's input arguments.
func (t *MCPToolMeta) buildInputSchema() map[string]any {
	schema := map[string]any{
		"type":       "object",
		"properties": make(map[string]any),
	}

	if len(t.inParams) == 0 {
		return schema
	}

	// If only one struct param (and it's the last one), use its fields as schema properties
	isLastStruct := len(t.inParams) == 1 &&
		t.inParams[0].SchemaType() == openapi.ObjectType &&
		!t.inParams[0].IsTime &&
		!t.inParams[0].IsFile

	if isLastStruct {
		return t.expandStructParam(t.inParams[0])
	}

	// Multiple params: each becomes a top-level property
	required := make([]string, 0)
	properties := schema["properties"].(map[string]any)

	for _, param := range t.inParams {
		propName := t.paramPropertyName(param)
		properties[propName] = t.paramToJSONSchema(param)
		// All explicit function params are required in MCP tools
		required = append(required, propName)
	}

	schema["required"] = required
	return schema
}

// paramPropertyName returns the property name for a RouteParam.
// Uses QueryName since Go reflection cannot provide function parameter names.
func (t *MCPToolMeta) paramPropertyName(param *openapi.RouteParam) string {
	return param.QueryName
}

// paramToJSONSchema converts a RouteParam to a JSON Schema property object.
func (t *MCPToolMeta) paramToJSONSchema(param *openapi.RouteParam) map[string]any {
	prop := map[string]any{
		"title": param.Name,
		"type":  goKindToJSONType(param.PrototypeKind),
	}

	if param.IsTime {
		prop["type"] = "string"
		prop["format"] = "date-time"
	}

	if param.IsPtr {
		// Ptr types are not required in the flattened params case,
		// but all function-level params are required. We just note the nullable.
	}

	return prop
}

// expandStructParam flattens a struct parameter's fields into JSON Schema properties.
// Reuses openapi.StructToQModels.
func (t *MCPToolMeta) expandStructParam(param *openapi.RouteParam) map[string]any {
	qms := openapi.StructToQModels(param.CopyPrototype())

	schema := map[string]any{
		"type":       "object",
		"properties": make(map[string]any),
	}

	required := make([]string, 0)
	properties := schema["properties"].(map[string]any)

	for _, qm := range qms {
		// Init must be called to parse json/query tags and set JsonName
		_ = qm.Init()
		prop := map[string]any{
			"title": qm.Name,
			"type":  string(qm.SchemaType()),
		}
		if qm.IsTime {
			prop["type"] = "string"
			prop["format"] = "date-time"
		}
		if qm.SchemaDesc() != "" {
			prop["description"] = qm.SchemaDesc()
		}
		properties[qm.JsonName()] = prop
		if qm.IsRequired() {
			required = append(required, qm.JsonName())
		}
	}

	if len(required) > 0 {
		schema["required"] = required
	}
	return schema
}

// NewCallArgs constructs the []reflect.Value needed for method.Func.Call().
// arguments is the JSON-RPC arguments map (string -> any).
func (t *MCPToolMeta) NewCallArgs(ctx *Context, arguments map[string]any) []reflect.Value {
	params := make([]reflect.Value, t.handlerInNum)
	params[0] = t.provider.providerValue // receiver
	params[1] = reflect.ValueOf(ctx)     // *Context

	for i, param := range t.inParams {
		paramIdx := MCPFirstInParamOffset + 1 + i // actual index in method params

		isLast := i == len(t.inParams)-1

		if param.SchemaType() == openapi.ObjectType && !param.IsTime && !param.IsFile {
			// Struct param
			params[paramIdx] = t.argumentsToStruct(param, arguments)
		} else {
			// Basic type param
			propName := t.paramPropertyName(param)
			argValue, exists := arguments[propName]
			if !exists {
				argValue = arguments[param.QueryName] // fallback to query name
			}
			if argValue != nil {
				params[paramIdx] = t.convertArgument(param, argValue)
			} else {
				params[paramIdx] = t.zeroValueForParam(param)
			}
		}
		_ = isLast
	}

	return params
}

// argumentsToStruct converts a map of arguments into a Go struct value via JSON round-trip.
func (t *MCPToolMeta) argumentsToStruct(param *openapi.RouteParam, args map[string]any) reflect.Value {
	rt := param.CopyPrototype()

	var newStruct reflect.Value
	if param.IsPtr {
		newStruct = reflect.New(rt.Elem())
	} else {
		newStruct = reflect.New(rt)
	}

	if len(args) == 0 {
		if !param.IsPtr {
			return newStruct.Elem()
		}
		return newStruct
	}

	// Use JSON round-trip for reliable type conversion: map -> JSON -> struct
	jsonBytes, err := json.Marshal(args)
	if err != nil {
		if !param.IsPtr {
			return newStruct.Elem()
		}
		return newStruct
	}

	if param.IsPtr {
		utils.JsonUnmarshal(jsonBytes, newStruct.Interface())
		return newStruct
	}
	utils.JsonUnmarshal(jsonBytes, newStruct.Interface())
	return newStruct.Elem()
}

// convertArgument converts a JSON-decoded value to a reflect.Value suitable for the param.
func (t *MCPToolMeta) convertArgument(param *openapi.RouteParam, value any) reflect.Value {
	val := reflect.ValueOf(value)

	// If types match, return directly
	if val.Type() == param.Prototype {
		return val
	}
	if param.IsPtr && val.Type() == param.Prototype.Elem() {
		ptr := reflect.New(param.Prototype.Elem())
		ptr.Elem().Set(val)
		return ptr
	}

	// Try type conversion via JSON round-trip for complex cases
	jsonBytes, err := json.Marshal(value)
	if err != nil {
		return t.zeroValueForParam(param)
	}

	var newVal reflect.Value
	if param.IsPtr {
		newVal = reflect.New(param.Prototype.Elem())
	} else {
		newVal = reflect.New(param.Prototype)
	}

	utils.JsonUnmarshal(jsonBytes, newVal.Interface())

	if param.IsPtr {
		return newVal
	}
	return newVal.Elem()
}

func (t *MCPToolMeta) zeroValueForParam(param *openapi.RouteParam) reflect.Value {
	if param.IsPtr {
		return reflect.New(param.Prototype.Elem())
	}
	return reflect.New(param.Prototype).Elem()
}

// Call executes the method and returns the result.
func (t *MCPToolMeta) Call(in []reflect.Value) (result any, err error) {
	out := t.method.Func.Call(in)
	if !out[MCPLastOutParamOffset].IsNil() {
		err = out[MCPLastOutParamOffset].Interface().(error)
	}
	result = out[MCPFirstOutParamOffset].Interface()
	return
}

// HasArgs returns whether this tool expects any arguments.
func (t *MCPToolMeta) HasArgs() bool { return len(t.inParams) > 0 }

// =================================== MCPResourceMeta ===================================

// MCPResourceMeta holds metadata for a single MCP Resource.
type MCPResourceMeta struct {
	method   reflect.Method
	provider *MCPProviderMeta

	Name        string `json:"name"`
	URI         string `json:"uri"`
	Description string `json:"description,omitempty"`
	MimeType    string `json:"mimeType,omitempty"`

	inParams     []*openapi.RouteParam
	outParam     *openapi.RouteParam
	index        int
	handlerInNum int
}

// NewMCPResourceMeta creates a new resource metadata instance.
func NewMCPResourceMeta(method reflect.Method, provider *MCPProviderMeta, suffix string) *MCPResourceMeta {
	name := provider.deriveFullName(MCPResource, suffix)
	uri := provider.deriveResourceURI(method.Name, suffix)
	return &MCPResourceMeta{
		method:   method,
		provider: provider,
		index:    method.Index,
		Name:     name,
		URI:      uri,
	}
}

// Init initializes the resource metadata.
func (r *MCPResourceMeta) Init() (err error) {
	r.handlerInNum = r.method.Type.NumIn()

	summary := r.provider.deriveSummary(r.method.Name, r.Name)
	r.Description = r.provider.deriveDescription(r.method.Name, summary)

	// Build inParams
	r.inParams = make([]*openapi.RouteParam, 0)
	for i := MCPFirstInParamOffset + 1; i < r.handlerInNum; i++ {
		param := openapi.NewRouteParam(r.method.Type.In(i), i, openapi.RouteParamQuery)
		err = param.Init()
		if err != nil {
			return err
		}
		r.inParams = append(r.inParams, param)
	}

	// Build outParam and infer MIME type
	r.outParam = openapi.NewRouteParam(r.method.Type.Out(MCPFirstOutParamOffset), 0, openapi.RouteParamResponse)
	err = r.outParam.Init()
	if err != nil {
		return err
	}
	r.MimeType = r.inferMimeType()

	return nil
}

func (r *MCPResourceMeta) inferMimeType() string {
	if r.outParam.IsFile {
		return "application/octet-stream"
	}
	switch r.outParam.SchemaType() {
	case openapi.StringType:
		return "text/plain"
	default:
		return "application/json"
	}
}

// NewCallArgs constructs reflect.Value args from the arguments map.
func (r *MCPResourceMeta) NewCallArgs(ctx *Context, arguments map[string]any) []reflect.Value {
	// Reuse the tool's implementation for now - same pattern
	t := &MCPToolMeta{
		method:       r.method,
		provider:     r.provider,
		inParams:     r.inParams,
		handlerInNum: r.handlerInNum,
	}
	return t.NewCallArgs(ctx, arguments)
}

// Call executes the resource method.
func (r *MCPResourceMeta) Call(in []reflect.Value) (result any, err error) {
	out := r.method.Func.Call(in)
	if !out[MCPLastOutParamOffset].IsNil() {
		err = out[MCPLastOutParamOffset].Interface().(error)
	}
	result = out[MCPFirstOutParamOffset].Interface()
	return
}

// =================================== MCPPromptMeta ===================================

// MCPPromptMeta holds metadata for a single MCP Prompt.
type MCPPromptMeta struct {
	method   reflect.Method
	provider *MCPProviderMeta

	Name        string               `json:"name"`
	Description string               `json:"description,omitempty"`
	Arguments   []PromptArgumentMeta `json:"arguments,omitempty"`

	inParams     []*openapi.RouteParam
	outParam     *openapi.RouteParam
	index        int
	handlerInNum int
}

// PromptArgumentMeta describes a prompt argument.
type PromptArgumentMeta struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	Required    bool   `json:"required"`
}

// NewMCPPromptMeta creates a new prompt metadata instance.
func NewMCPPromptMeta(method reflect.Method, provider *MCPProviderMeta, suffix string) *MCPPromptMeta {
	name := provider.deriveFullName(MCPPrompt, suffix)
	return &MCPPromptMeta{
		method:   method,
		provider: provider,
		index:    method.Index,
		Name:     name,
	}
}

// Init initializes the prompt metadata.
func (p *MCPPromptMeta) Init() (err error) {
	p.handlerInNum = p.method.Type.NumIn()

	summary := p.provider.deriveSummary(p.method.Name, p.Name)
	p.Description = p.provider.deriveDescription(p.method.Name, summary)

	// Build inParams
	p.inParams = make([]*openapi.RouteParam, 0)
	for i := MCPFirstInParamOffset + 1; i < p.handlerInNum; i++ {
		param := openapi.NewRouteParam(p.method.Type.In(i), i, openapi.RouteParamQuery)
		err = param.Init()
		if err != nil {
			return err
		}
		p.inParams = append(p.inParams, param)
	}

	// Build prompt arguments from params
	p.Arguments = make([]PromptArgumentMeta, len(p.inParams))
	for i, param := range p.inParams {
		p.Arguments[i] = PromptArgumentMeta{
			Name:        param.QueryName,
			Description: param.Name,
			Required:    true,
		}
	}

	// outParam
	p.outParam = openapi.NewRouteParam(p.method.Type.Out(MCPFirstOutParamOffset), 0, openapi.RouteParamResponse)
	err = p.outParam.Init()

	return
}

// NewCallArgs constructs reflect.Value args from the arguments map.
func (p *MCPPromptMeta) NewCallArgs(ctx *Context, arguments map[string]any) []reflect.Value {
	t := &MCPToolMeta{
		method:       p.method,
		provider:     p.provider,
		inParams:     p.inParams,
		handlerInNum: p.handlerInNum,
	}
	return t.NewCallArgs(ctx, arguments)
}

// Call executes the prompt method.
func (p *MCPPromptMeta) Call(in []reflect.Value) (result any, err error) {
	out := p.method.Func.Call(in)
	if !out[MCPLastOutParamOffset].IsNil() {
		err = out[MCPLastOutParamOffset].Interface().(error)
	}
	result = out[MCPFirstOutParamOffset].Interface()
	return
}

// =================================== Helpers ===================================

// goKindToJSONType maps a reflect.Kind to a JSON Schema type string.
func goKindToJSONType(kind reflect.Kind) string {
	switch kind {
	case reflect.Bool:
		return "boolean"
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return "integer"
	case reflect.Float32, reflect.Float64:
		return "number"
	case reflect.String:
		return "string"
	case reflect.Struct:
		return "object"
	case reflect.Slice, reflect.Array:
		return "array"
	case reflect.Map:
		return "object"
	case reflect.Ptr:
		return "object"
	default:
		return "string"
	}
}

// toSnakeCase converts a CamelCase string to snake_case.
func toSnakeCase(s string) string {
	var result []rune
	for i, r := range s {
		if unicode.IsUpper(r) {
			if i > 0 {
				result = append(result, '_')
			}
			result = append(result, unicode.ToLower(r))
		} else {
			result = append(result, r)
		}
	}
	return string(result)
}
