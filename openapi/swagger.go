package openapi

import (
	"github.com/Chendemo12/fastapi-tool/helper"
	"github.com/Chendemo12/fastapi/pathschema"
	"github.com/Chendemo12/fastapi/utils"
	"net/http"
	"reflect"
	"strconv"
	"strings"
)

// Contact 联系方式, 显示在 info 字段内部
// 无需重写序列化方法
type Contact struct {
	Name  string `json:"name" description:"姓名/名称"`
	Url   string `json:"url" description:"链接"`
	Email string `json:"email" description:"联系方式"`
}

// License 权利证书, 显示在 info 字段内部
// 无需重写序列化方法
type License struct {
	Name string `json:"name" description:"名称"`
	Url  string `json:"url" description:"链接"`
}

// Info 文档说明信息
// 无需重写序列化方法
type Info struct {
	Title          string  `json:"title" description:"显示在文档顶部的标题"`
	Version        string  `json:"version" description:"显示在标题右上角的程序版本号"`
	Description    string  `json:"description,omitempty" description:"显示在标题下方的说明"`
	Contact        Contact `json:"contact,omitempty" description:"联系方式"`
	License        License `json:"license,omitempty" description:"许可证"`
	TermsOfService string  `json:"termsOfService,omitempty" description:"服务条款(不常用)"`
}

// Reference 引用模型,用于模型字段和路由之间互相引用
type Reference struct {
	// 关联模型, 取值为 RefPrefix + modelName
	Name string `json:"-" description:"关联模型"`
}

func (r *Reference) MarshalJSON() ([]byte, error) {
	m := make(map[string]any)
	m[RefName] = RefPrefix + r.Name

	return helper.JsonMarshal(m)
}

// ComponentScheme openapi 的模型文档部分
type ComponentScheme struct {
	Model SchemaIface `json:"model" description:"模型定义"`
	Name  string      `json:"name" description:"模型名称，包含包名"`
}

// Components openapi 的模型部分
// 需要重写序列化方法
type Components struct {
	Scheme []*ComponentScheme `json:"scheme" description:"模型文档"`
}

// MarshalJSON 重载序列化方法
func (c *Components) MarshalJSON() ([]byte, error) {
	m := make(map[string]any)
	for _, v := range c.Scheme {
		m[v.Name] = v.Model.Schema() // 记录根模型
	}

	// 记录内置错误类型文档
	m[ValidationErrorDefinition.SchemaPkg()] = ValidationErrorDefinition.Schema()
	m[ValidationErrorResponseDefinition.SchemaPkg()] = ValidationErrorResponseDefinition.Schema()

	return helper.JsonMarshal(map[string]any{"schemas": m})
}

// AddModel 添加一个模型文档
func (c *Components) AddModel(m SchemaIface) {
	c.Scheme = append(c.Scheme, &ComponentScheme{
		Name:  m.SchemaPkg(),
		Model: m,
	})
}

type ParameterInType string

const (
	InQuery  ParameterInType = "query"
	InHeader ParameterInType = "header"
	InPath   ParameterInType = "path"
	InCookie ParameterInType = "cookie"
)

// ParameterBase 各种参数的基类
type ParameterBase struct {
	Name        string          `json:"name" description:"名称"`
	Description string          `json:"description,omitempty" description:"说明"`
	In          ParameterInType `json:"in" description:"参数位置"`
	Required    bool            `json:"required" description:"是否必须"`
	Deprecated  bool            `json:"deprecated" description:"是否禁用"`
}

type ParameterSchema struct {
	Type  DataType `json:"type" description:"数据类型"`
	Title string   `json:"title"`
}

// Parameter 路径参数或者查询参数
type Parameter struct {
	ParameterBase
	Default any              `json:"default,omitempty" description:"默认值"`
	Schema  *ParameterSchema `json:"schema,omitempty" description:"字段模型"`
}

func (p *Parameter) FromQModel(model *QModel) *Parameter {
	p.Name = model.JsonName()
	p.Description = model.SchemaDesc()
	p.Required = model.IsRequired()
	p.Default = GetDefaultV(model.Tag, model.SchemaType())
	p.Schema = &ParameterSchema{
		Type:  model.SchemaType(),
		Title: model.SchemaTitle(),
	}

	if model.InPath {
		p.In = InPath
	} else {
		p.In = InQuery
	}

	return p
}

type ModelContentSchema interface {
	Schema() map[string]any
	SchemaType() DataType
	SchemaTitle() string
	SchemaPkg() string
}

// RequestBody 路由 请求体模型文档
type RequestBody struct {
	Content  *PathModelContent `json:"content,omitempty" description:"请求体模型"`
	Required bool              `json:"required" description:"是否必须"`
}

// PathModelContent 路由中请求体 RequestBody 和 响应体中返回值 Responses 模型
type PathModelContent struct {
	Schema   ModelContentSchema `json:"schema" description:"模型引用文档"`
	MIMEType string             `json:"-"`
}

// MarshalJSON 自定义序列化
func (p *PathModelContent) MarshalJSON() ([]byte, error) {
	m := make(map[string]any)
	switch p.Schema.SchemaType() {
	case ObjectType:
		m[p.MIMEType] = map[string]any{
			"schema": map[string]string{
				RefName: RefPrefix + p.Schema.SchemaPkg(),
			},
		}
	default:
		m[p.MIMEType] = map[string]any{"schema": p.Schema.Schema()}
	}

	return helper.JsonMarshal(m)
}

// Response 路由返回体，包含了返回状态码，状态码说明和返回值模型
type Response struct {
	Content     *PathModelContent `json:"content" description:"返回值模型"`
	Description string            `json:"description,omitempty" description:"说明"`
	StatusCode  int               `json:"-" description:"状态码"`
}

// Operation 路由HTTP方法: Get/Post/Patch/Delete 等操作方法
type Operation struct {
	Tags        []string `json:"tags,omitempty" description:"路由标签"`
	Summary     string   `json:"summary,omitempty" description:"摘要描述"`
	Description string   `json:"description,omitempty" description:"说明"`
	OperationId string   `json:"operationId,omitempty" description:"唯一ID"` // no use, keep
	// 路径参数和查询参数, 对于路径相同，方法不同的路由来说，其查询参数可以不一样，但其路径参数都是一样的
	Parameters []*Parameter `json:"parameters,omitempty" description:"路径参数和查询参数"`
	// 请求体，通过 MakeOperationRequestBody 构建
	RequestBody *RequestBody `json:"requestBody,omitempty" description:"请求体"`
	// 响应文档，对于任一个路由，均包含2个响应实例：200 + 422， 通过函数 MakeOperationResponses 构建
	Responses  []*Response `json:"responses,omitempty" description:"响应体"`
	Deprecated bool        `json:"deprecated,omitempty" description:"是否禁用"`
}

// MarshalJSON 重写序列化方法，修改 Responses 和 RequestBody 字段
func (o *Operation) MarshalJSON() ([]byte, error) {
	type OperationWithResponseMap struct {
		Responses map[int]*Response `json:"responses" description:"响应体"`
		Operation
	}

	orm := OperationWithResponseMap{}
	orm.Tags = o.Tags
	orm.Summary = o.Summary
	orm.Description = o.Description
	orm.OperationId = o.OperationId
	orm.Parameters = o.Parameters
	orm.RequestBody = o.RequestBody // TODO:
	orm.Deprecated = o.Deprecated

	orm.Responses = make(map[int]*Response)
	for _, r := range o.Responses {
		orm.Responses[r.StatusCode] = r
	}

	return helper.JsonMarshal(orm)
}

// RequestBodyFrom 从 *openapi.BaseModelMeta 转换成 openapi 的请求体 RequestBody
func (o *Operation) RequestBodyFrom(model *BaseModelMeta) *Operation {
	o.RequestBody = &RequestBody{}
	if model != nil {
		o.RequestBody.Required = model.IsRequired()
		o.RequestBody.Content = &PathModelContent{
			MIMEType: MIMEApplicationJSON,
			Schema:   model,
		}
	}

	return o
}

// ResponseFrom 从 *openapi.BaseModelMeta 转换成 openapi 的响应实例
func (o *Operation) ResponseFrom(model *BaseModelMeta) *Operation {
	if model == nil { // 若返回值为空，则设置为空
		model = &BaseModelMeta{}
	}

	m := make([]*Response, 2) // 200 + 422
	// 200 接口处注册的返回值
	m[0] = &Response{
		StatusCode:  http.StatusOK,
		Description: http.StatusText(http.StatusOK),
		Content: &PathModelContent{
			MIMEType: MIMEApplicationJSON,
			Schema:   model,
		},
	}
	// 422 所有接口默认携带的请求体校验错误返回值
	m[1] = &Response{
		StatusCode:  http.StatusUnprocessableEntity,
		Description: http.StatusText(http.StatusUnprocessableEntity),
		Content: &PathModelContent{
			MIMEType: MIMEApplicationJSON,
			Schema:   &ValidationError{},
		},
	}

	o.Responses = m
	return o
}

// PathItem 路由选项，由于同一个路由可以存在不同的操作方法，因此此选项可以存在多个 Operation
type PathItem struct {
	Get    *Operation `json:"get,omitempty" description:"GET方法"`
	Put    *Operation `json:"put,omitempty" description:"PUT方法"`
	Post   *Operation `json:"post,omitempty" description:"POST方法"`
	Patch  *Operation `json:"patch,omitempty" description:"PATCH方法"`
	Delete *Operation `json:"delete,omitempty" description:"DELETE方法"`
	Head   *Operation `json:"head,omitempty" description:"header方法"`
	Trace  *Operation `json:"trace,omitempty" description:"trace方法"`
	Path   string     `json:"-" description:"请求绝对路径"`
}

// Paths openapi 的路由部分
// 需要重写序列化方法
type Paths struct {
	Paths []*PathItem
}

func (p *Paths) AddItem(item *PathItem) {
	p.Paths = append(p.Paths, item)
}

// MarshalJSON 重载序列化方法
func (p *Paths) MarshalJSON() ([]byte, error) {
	m := make(map[string]any)
	for _, v := range p.Paths {
		m[v.Path] = v
	}

	return helper.JsonMarshal(m)
}

// OpenApi 模型类, 移除 FastApi 中不常用的属性
type OpenApi struct {
	Info        *Info       `json:"info,omitempty" description:"联系信息"`
	Components  *Components `json:"components" description:"模型文档"`
	Paths       *Paths      `json:"paths" description:"路由列表,同一路由存在多个方法文档"`
	Version     string      `json:"openapi" description:"Open API版本号"`
	cache       []byte
	initialized bool
}

// NewOpenApi 构造一个新的 OpenApi 文档
func NewOpenApi(title, version, description string) *OpenApi {
	return &OpenApi{
		Version: ApiVersion,
		Info: &Info{
			Title:          title,
			Version:        version,
			Description:    description,
			TermsOfService: "",
			Contact: Contact{
				Name:  "FastApi",
				Url:   "github.com/Chendemo12/fastapi",
				Email: "chendemo12@gmail.com",
			},
			License: License{
				Name: "FastApi",
				Url:  "github.com/Chendemo12/fastapi",
			},
		},
		Components:  &Components{Scheme: make([]*ComponentScheme, 0)},
		Paths:       &Paths{Paths: make([]*PathItem, 0)},
		initialized: false,
		cache:       make([]byte, 0),
	}
}

func (o *OpenApi) AddLicense(info License) *OpenApi {
	o.Info.License.Url = info.Url
	o.Info.License.Name = info.Name

	return o
}

func (o *OpenApi) AddContact(info Contact) *OpenApi {
	o.Info.Contact.Url = info.Url
	o.Info.Contact.Name = info.Name
	o.Info.Contact.Email = info.Email

	return o
}

// AddDefinition 手动添加一个模型文档
func (o *OpenApi) AddDefinition(meta SchemaIface) *OpenApi {
	o.Components.AddModel(meta)
	return o
}

// 查询路由对象, 不存在则新建
func (o *OpenApi) getPath(path string) *PathItem {
	// 修改路径格式为FastApi路径格式, 主要区别在于用"{}"标识路径参数,而非":"
	path = ToFastApiRoutePath(path) // 修改路径格式

	for _, item := range o.Paths.Paths {
		if item.Path == path {
			return item
		}
	}

	item := &PathItem{
		Path:   path,
		Get:    nil,
		Put:    nil,
		Post:   nil,
		Patch:  nil,
		Delete: nil,
		Head:   nil,
		Trace:  nil,
	}
	o.Paths.Paths = append(o.Paths.Paths, item)

	return item
}

func (o *OpenApi) modelFrom(swagger *RouteSwagger) {
	if swagger.RequestModel != nil {
		o.AddDefinition(swagger.RequestModel)
		// 生成模型，处理嵌入类型
		for _, inner := range swagger.RequestModel.InnerSchema() {
			o.AddDefinition(inner)
		}
		// 处理数组类型
		if swagger.RequestModel.itemModel != nil {
			o.AddDefinition(swagger.RequestModel.itemModel)
			for _, inner := range swagger.RequestModel.itemModel.InnerSchema() {
				o.AddDefinition(inner)
			}
		}
	}

	if swagger.ResponseModel != nil {
		o.AddDefinition(swagger.ResponseModel)
		// 生成模型，处理嵌入类型
		for _, inner := range swagger.ResponseModel.InnerSchema() {
			o.AddDefinition(inner)
		}
		// 处理数组类型
		if swagger.ResponseModel.itemModel != nil {
			o.AddDefinition(swagger.ResponseModel.itemModel)
			for _, inner := range swagger.ResponseModel.itemModel.InnerSchema() {
				o.AddDefinition(inner)
			}
		}
	}
}

func (o *OpenApi) pathFrom(swagger *RouteSwagger) {
	// 存在相同路径，不同方法的路由选项
	item := o.getPath(swagger.Url)

	// 构造路径参数
	pathParams := make([]*Parameter, len(swagger.PathFields))
	for no, q := range swagger.PathFields {
		p := &Parameter{}
		p.FromQModel(q)
		p.Deprecated = swagger.Deprecated

		pathParams[no] = p
	}

	// 构造查询参数
	queryParams := make([]*Parameter, len(swagger.QueryFields))
	for no, q := range swagger.QueryFields {
		p := &Parameter{}
		p.FromQModel(q)
		p.Deprecated = swagger.Deprecated
		queryParams[no] = p
	}

	// 构造操作符
	operation := &Operation{
		Summary:     swagger.Summary,
		Description: swagger.Description,
		Tags:        swagger.Tags,
		Parameters:  append(pathParams, queryParams...),
		Deprecated:  swagger.Deprecated,
	}
	if utils.Has[string]([]string{http.MethodGet, http.MethodDelete}, swagger.Method) {
		// GET/DELETE 无请求体，不显示
		operation.RequestBody = nil
	} else {
		operation.RequestBodyFrom(swagger.RequestModel)
	}
	operation.ResponseFrom(swagger.ResponseModel)

	// 绑定到操作方法
	switch swagger.Method {

	case http.MethodPost:
		item.Post = operation
	case http.MethodPut:
		item.Put = operation
	case http.MethodDelete:
		item.Delete = operation
	case http.MethodPatch:
		item.Patch = operation
	case http.MethodHead:
		item.Head = operation
	case http.MethodTrace:
		item.Trace = operation

	default:
		item.Get = operation
	}
}

// RegisterFrom home point
func (o *OpenApi) RegisterFrom(swagger *RouteSwagger) *OpenApi {
	o.modelFrom(swagger)
	o.pathFrom(swagger)

	return o
}

// RecreateDocs 重建Swagger 文档
func (o *OpenApi) RecreateDocs() *OpenApi {
	bs, err := helper.JsonMarshal(o)
	if err == nil {
		o.cache = bs
	}

	o.initialized = true
	return o
}

// Schema Swagger 文档, 并非完全符合 OpenApi 文档规范
func (o *OpenApi) Schema() []byte {
	if !o.initialized {
		o.RecreateDocs()
	}

	return o.cache
}

// ToFastApiRoutePath 将 fiber.App 格式的路径转换成 FastApi 格式的路径
//
//	Example:
//	必选路径参数：
//		Input: "/api/rcst/:no"
//		Output: "/api/rcst/{no}"
//	可选路径参数：
//		Input: "/api/rcst/:no?"
//		Output: "/api/rcst/{no}"
//	常规路径：
//		Input: "/api/rcst/no"
//		Output: "/api/rcst/no"
func ToFastApiRoutePath(path string) string {
	paths := strings.Split(path, pathschema.PathSeparator) // 路径字符
	// 查找路径中的参数
	for i := 0; i < len(paths); i++ {
		if strings.HasPrefix(paths[i], pathschema.PathParamPrefix) {
			// 识别到路径参数
			if strings.HasSuffix(paths[i], pathschema.OptionalQueryParamPrefix) {
				// 可选路径参数
				paths[i] = "{" + paths[i][1:len(paths[i])-1] + "}"
			} else {
				paths[i] = "{" + paths[i][1:] + "}"
			}
		}
	}

	return strings.Join(paths, pathschema.PathSeparator)
}

// ReflectKindToType 转换reflect.Kind为swagger类型说明
//
//	@param	ReflectKind	reflect.Kind	反射类型,不进一步对指针类型进行上浮
func ReflectKindToType(kind reflect.Kind) (name DataType) {
	switch kind {

	case reflect.Array, reflect.Slice, reflect.Chan:
		name = ArrayType
	case reflect.String:
		name = StringType
	case reflect.Bool:
		name = BoolType
	default:
		if reflect.Bool < kind && kind <= reflect.Uint64 {
			name = IntegerType
		} else if reflect.Float32 <= kind && kind <= reflect.Complex128 {
			name = NumberType
		} else {
			name = ObjectType
		}
	}

	return
}

// IsFieldRequired 从tag中判断此字段是否是必须的
func IsFieldRequired(tag reflect.StructTag) bool {
	bindings := strings.Split(utils.QueryFieldTag(tag, ValidateTagName, ""), ",") // binding 存在多个值
	for i := 0; i < len(bindings); i++ {
		if strings.TrimSpace(bindings[i]) == ParamRequiredLabel {
			return true
		}
	}

	return false
}

// GetDefaultV 从Tag中提取字段默认值
func GetDefaultV(tag reflect.StructTag, otype DataType) (v any) {
	defaultV := utils.QueryFieldTag(tag, DefaultValueTagNam, "")

	if defaultV == "" {
		v = nil
	} else { // 存在默认值
		switch otype {

		case StringType:
			v = defaultV
		case IntegerType:
			v, _ = strconv.Atoi(defaultV)
		case NumberType:
			v, _ = strconv.ParseFloat(defaultV, 64)
		case BoolType:
			v, _ = strconv.ParseBool(defaultV)
		default:
			v = defaultV
		}
	}
	return
}

// 针对结构体字段仍然是结构体或数组的情况，如果目标是个匿名对象则人为分配个名称，反之则获取实际的名称
func assignModelNames(fieldMeta *BaseModelField, fieldType reflect.Type) (string, string) {
	var pkg, name string

	if utils.IsAnonymousStruct(fieldType) {
		// 未命名的结构体类型, 没有名称, 分配包名和名称
		name = fieldMeta.Name + "Model"
		//pkg = fieldMeta._pkg + AnonymousModelNameConnector + name
		pkg = fieldMeta.Pkg
	} else {
		pkg = fieldType.String() // 关联模型
		name = fieldType.Name()
	}

	return pkg, name
}
