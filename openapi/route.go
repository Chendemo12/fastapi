package openapi

import (
	"errors"
	"fmt"
	"github.com/Chendemo12/fastapi-tool/helper"
	"github.com/Chendemo12/fastapi/pathschema"
	"github.com/Chendemo12/fastapi/utils"
	"reflect"
	"strings"
)

// RouteSwagger 路由文档定义，所有类型的路由均包含此部分
type RouteSwagger struct {
	Url           string         `json:"url" description:"完整请求路由"` // 此路由为函数定义时的路由
	RelativePath  string         `json:"relative_path" description:"相对路由"`
	Method        string         `json:"method" description:"请求方法"`
	Summary       string         `json:"summary" description:"摘要描述"`
	Description   string         `json:"description" description:"详细描述"`
	Tags          []string       `json:"tags" description:"路由标签"`
	RequestModel  *BaseModelMeta `description:"请求体元数据"` // 请求体也只有一个, 当 method 为 GET和DELETE 时无请求体
	ResponseModel *BaseModelMeta `description:"响应体元数据"` // 响应体只有一个
	QueryFields   []*QModel      `json:"-" description:"查询参数"`
	PathFields    []*QModel      `json:"-" description:"路径参数"`
	Deprecated    bool           `json:"deprecated" description:"是否禁用"`
	api           string         // 用作唯一标识
}

func (r *RouteSwagger) Init() (err error) {
	r.api = CreateRouteIdentify(r.Method, r.Url)
	// 由于查询参数和请求体需要从方法入参中提取, 以及响应体需要从方法出参中提取,因此在上层进行解析
	if r.ResponseModel == nil { // 返回值不允许为nil, 此处错误为上层忘记初始化模型参数
		return errors.New("ResponseModel is not init")
	}

	// 请求体可以为nil
	err = r.Scan()

	return
}

func (r *RouteSwagger) Scan() (err error) {
	err = r.scanPath()
	if err != nil {
		return err
	}

	err = r.ScanInner()

	return
}

// ScanInner 解析内部模型数据
func (r *RouteSwagger) ScanInner() (err error) {
	for _, qmodel := range r.QueryFields {
		err = qmodel.Init()
		if err != nil {
			return err
		}
	}

	for _, qmodel := range r.PathFields {
		err = qmodel.Init()
		if err != nil {
			return err
		}
	}

	if r.RequestModel != nil {
		err = r.RequestModel.Init()
		if err != nil {
			return err
		}
	}

	if r.ResponseModel != nil {
		err = r.ResponseModel.Init()
	}

	return
}

func (r *RouteSwagger) scanPath() (err error) {
	// 提取路由中的路径参数
	// 通过结构体方法名称确定路由的方式无法包含路径参数的, 但是如果定义了重载方法 GroupRouter.Path() 则可以包含路径参数
	for _, p := range strings.Split(r.Url, pathschema.PathSeparator) {
		if p == "" {
			continue
		}

		// GET: /api/clipboard/:day?num  	=> day为路径参数,num为查询参数
		// GET: /api/clipboard/:day/?num	=> day为路径参数,num为查询参数
		qm := &QModel{DataType: StringType, InPath: false}

		if strings.HasPrefix(p, pathschema.OptionalQueryParamPrefix) {
			// 发现查询参数
			// 通过路由组定义的方法即便通过了重载也不应包含查询参数, 但泛型路由定义方式将支持
			qm.Name = p[1:]
			// 通过Tag标识这不是一个必须的参数
			qm.Tag = reflect.StructTag(`json:"` + qm.Name + `,omitempty"`)

			r.QueryFields = append(r.QueryFields, qm)
			continue
		}

		if strings.HasPrefix(p, pathschema.PathParamPrefix) {
			// 识别到路径参数
			qm.InPath = true
			qm.Name = p[1:]
			// 通过Tag标识这是一个必须的参数: `validate:"required"`
			qm.Tag = reflect.StructTag(fmt.Sprintf(`json:"%s" %s:"%s"`,
				qm.Name, ValidateTagName, ParamRequiredLabel))

			r.PathFields = append(r.PathFields, qm)
			continue
		}
	}

	return
}

func (r *RouteSwagger) Id() string { return r.api }

// RouteParam 路由参数, 具体包含查询参数,路径参数,请求体参数和响应体参数
type RouteParam struct {
	Prototype      reflect.Type   // 直接反射后获取的类型,未提取指针指向的类型
	PrototypeKind  reflect.Kind   // 原始类型的参数类型
	IsPtr          bool           // 标识 Prototype 是否是指针类型
	IsNil          bool           // TODO Future: what to do
	Name           string         // 名称
	Pkg            string         // 包含包名,如果是结构体则为: 包名.结构体名, 处理了指针
	DataType       DataType       // 如果是指针,则为指针指向的类型定义
	RouteParamType RouteParamType // 参数路由类型, 并非完全准确, 只在限制范围内访问
	Index          int            // 参数处于方法中的原始位置,可通过 method.RouteType.In(Index) 或 method.RouteType.Out(Index) 反向获得此参数
	T              ModelSchema    // TODO Future-231126.5: 泛型路由注册
}

func NewRouteParam(rt reflect.Type, index int, paramType RouteParamType) *RouteParam {
	r := &RouteParam{}
	r.Prototype = rt
	r.PrototypeKind = rt.Kind()
	r.IsPtr = rt.Kind() == reflect.Ptr
	r.Index = index
	r.RouteParamType = paramType

	return r
}

func (r *RouteParam) Init() (err error) {
	if r.IsPtr { // 指针类型
		r.DataType = ReflectKindToType(r.Prototype.Elem().Kind())
		r.Name = r.Prototype.Elem().Name()
		r.Pkg = r.Prototype.Elem().String()
	} else {
		r.DataType = ReflectKindToType(r.PrototypeKind)
		r.Name = r.Prototype.Name()
		r.Pkg = r.Prototype.String()
	}

	// 对于[]object 形式，修改其模型名称
	if r.DataType == ArrayType {
		elem := r.Prototype.Elem()
		if elem.Kind() == reflect.Ptr {
			elem = elem.Elem()
		}

		r.Name = utils.Pluralize(elem.Name())
		ss := strings.Split(elem.String(), ".")
		r.Pkg = strings.Join(ss[:len(ss)-1], "") + "." + utils.Pluralize(elem.Name())
	}
	if strings.HasPrefix(r.Pkg, "struct {") || r.Prototype.Name() == "" {
		// 对于匿名字段, 此处无法重命名，只能由外部重命名, 通过 Rename 方法重命名
	}
	return nil
}

func (r *RouteParam) Scan() (err error) { return }

func (r *RouteParam) ScanInner() (err error) { return }

func (r *RouteParam) Rename(pkg, name string) {
	r.Pkg = pkg
	r.Name = name
}

// CopyPrototype 将原始模型的 reflect.Type 拷贝一份
func (r *RouteParam) CopyPrototype() reflect.Type {
	var rt reflect.Type
	rt = r.Prototype
	return rt
}

func (r *RouteParam) SchemaTitle() string { return r.Name }

func (r *RouteParam) SchemaPkg() string { return r.Pkg }

func (r *RouteParam) JsonName() string { return r.Name }

func (r *RouteParam) SchemaDesc() string { return "" }

func (r *RouteParam) SchemaType() DataType { return r.DataType }

func (r *RouteParam) IsRequired() bool { return true }

func (r *RouteParam) Schema() (m map[string]any) { return map[string]any{} }

// InnerSchema 内部字段模型文档
func (r *RouteParam) InnerSchema() []SchemaIface {
	m := make([]SchemaIface, 0)
	return m
}

// NewNotStruct 通过反射创建一个(非struct类型)新的参数实例
func (r *RouteParam) NewNotStruct(value any) reflect.Value {
	var v reflect.Value
	if r.IsPtr {
		v = reflect.New(r.Prototype.Elem())
	} else {
		v = reflect.New(r.Prototype)
	}

	// 此时v是个指针类型
	if v.Elem().CanSet() {
		switch v.Elem().Kind() {
		case reflect.Bool:
			v.Elem().SetBool(value.(bool))
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			v.Elem().SetInt(value.(int64))
		case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
			v.Elem().SetUint(value.(uint64))
		case reflect.Float64, reflect.Float32:
			v.Elem().SetFloat(value.(float64))
		case reflect.String:
			v.Elem().SetString(value.(string))
		case reflect.Slice, reflect.Array:
			// array
		default:
		}
	}
	return v
}

// QueryName 获得查询参数名称
// 当其作为查询参数时，由于无法反射到参数的名称，因此手动分配一个名称
func (r *RouteParam) QueryName() string {
	return fmt.Sprintf("%s%d", r.Name, r.Index)
}

// RouteParamType 路由参数类型
type RouteParamType string

const (
	RouteParamQuery    RouteParamType = "query"
	RouteParamPath     RouteParamType = "path"
	RouteParamRequest  RouteParamType = "requestBody"
	RouteParamResponse RouteParamType = "responseBody"
)

// CreateRouteIdentify 获得一个路由对象的唯一标识
func CreateRouteIdentify(method, url string) string {
	return helper.CombineStrings(method, RouteMethodSeparator, url)
}
