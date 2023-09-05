package openapi

import (
	"github.com/Chendemo12/fastapi-tool/helper"
	"github.com/Chendemo12/fastapi/godantic"
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
	// 关联模型, 取值为 godantic.RefPrefix + modelName
	Name string `json:"-" description:"关联模型"`
}

func (r *Reference) MarshalJSON() ([]byte, error) {
	m := make(map[string]any)
	m[godantic.RefName] = godantic.RefPrefix + r.Name

	return helper.JsonMarshal(m)
}

// ComponentScheme openapi 的模型文档部分
type ComponentScheme struct {
	Model *godantic.Metadata `json:"model" description:"模型定义"`
	Name  string             `json:"name" description:"模型名称，包含包名"`
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

		// 生成模型，处理嵌入类型
		for _, innerF := range v.Model.InnerFields() {
			exist, innerM := innerF.ToMetadata()
			if exist { // 发现子模型
				m[innerF.SchemaName()] = innerM.Schema() // 对于未命名结构体，给其指定一个结构体名称
			}
		}
	}

	// 记录内置错误类型文档
	m[validationErrorDefinition.SchemaName()] = validationErrorDefinition.Schema()
	m[validationErrorResponseDefinition.SchemaName()] = validationErrorResponseDefinition.Schema()

	return helper.JsonMarshal(map[string]any{"schemas": m})
}

// AddModel 添加一个模型文档
func (c *Components) AddModel(m *godantic.Metadata) {
	c.Scheme = append(c.Scheme, &ComponentScheme{
		Name:  m.SchemaName(),
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
	Type  godantic.OpenApiDataType `json:"type" description:"数据类型"`
	Title string                   `json:"title"`
}

// Parameter 路径参数或者查询参数
type Parameter struct {
	Default any              `json:"default,omitempty" description:"默认值"`
	Schema  *ParameterSchema `json:"schema,omitempty" description:"字段模型"`
	ParameterBase
}

type ModelContentSchema interface {
	SchemaType() godantic.OpenApiDataType
	Schema() map[string]any
	SchemaName(exclude ...bool) string
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
	case godantic.ObjectType:
		m[p.MIMEType] = map[string]any{
			"schema": map[string]string{
				godantic.RefName: godantic.RefPrefix + p.Schema.SchemaName(),
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

// AddDefinition 添加一个模型文档
func (o *OpenApi) AddDefinition(meta *godantic.Metadata) *OpenApi {
	o.Components.AddModel(meta)
	return o
}

// QueryPathItem 查询路由对象, 不存在则新建
func (o *OpenApi) QueryPathItem(path string) *PathItem {
	path = FastApiRoutePath(path) // 修改路径格式

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
