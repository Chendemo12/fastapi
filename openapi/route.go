package openapi

import (
	"errors"
	"fmt"
	"github.com/Chendemo12/fastapi/pathschema"
	"github.com/Chendemo12/fastapi/utils"
	"reflect"
	"strconv"
	"strings"
)

// RouteParamType 路由参数类型
type RouteParamType string

const (
	RouteParamQuery    RouteParamType = "query"
	RouteParamPath     RouteParamType = "path"
	RouteParamRequest  RouteParamType = "requestBody"
	RouteParamResponse RouteParamType = "responseBody"
)

const (
	SchemaDescMethodName string = "SchemaDesc"
	SchemaTypeMethodName string = "SchemaType"
)

// RouteSwagger 路由文档定义，所有类型的路由均包含此部分
type RouteSwagger struct {
	RequestModel        *BaseModelMeta `description:"请求体元数据"`
	ResponseModel       *BaseModelMeta `description:"响应体元数据"`
	Summary             string         `json:"summary" description:"摘要描述"`
	Url                 string         `json:"url" description:"完整请求路由"`
	Description         string         `json:"description" description:"详细描述"`
	Method              string         `json:"method" description:"请求方法"`
	RelativePath        string         `json:"relative_path" description:"相对路由"`
	RequestContentType  string         `json:"request_content_type,omitempty" description:"请求体类型, 仅在 application/json 情况下才进行请求体校验"`
	ResponseContentType string         `json:"response_content_type,omitempty" description:"响应体类型, 仅在 application/json 情况下才进行响应体校验"`
	Api                 string         `description:"用作唯一标识"`
	Tags                []string       `json:"tags" description:"路由标签"`
	PathFields          []*QModel      `json:"-" description:"路径参数"`
	QueryFields         []*QModel      `json:"-" description:"查询参数"`
	UploadFile          SchemaIface    `json:"-" description:"是否是上传文件,不能与RequestModel共存"`
	Deprecated          bool           `json:"deprecated" description:"是否禁用"`
}

func (r *RouteSwagger) Init() (err error) {
	r.RequestContentType = MIMEApplicationJSON
	r.ResponseContentType = MIMEApplicationJSON
	r.Api = CreateRouteIdentify(r.Method, r.Url)

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
		qm := &QModel{DataType: StringType, Kind: reflect.String, InPath: false, InStruct: false}

		if strings.HasPrefix(p, pathschema.OptionalQueryParamPrefix) {
			// 发现查询参数
			// 通过路由组定义的方法即便通过了重载也不应包含查询参数, 但泛型路由定义方式将支持
			qm.Name = p[1:]
			// 通过Tag标识这不是一个必须的参数, `json:"eventType,omitempty" query:"eventType"`
			qm.Tag = reflect.StructTag(fmt.Sprintf(`json:"%s,omitempty" %s:"%s"`, qm.Name, QueryTagName, qm.Name))

			r.QueryFields = append(r.QueryFields, qm)
			continue
		}

		if strings.HasPrefix(p, pathschema.PathParamPrefix) {
			// 识别到路径参数
			qm.InPath = true
			qm.Name = p[1:]
			// 通过Tag标识这是一个必须的参数: `json:"eventType" validate:"required"`
			qm.Tag = reflect.StructTag(fmt.Sprintf(`json:"%s" %s:"%s"`,
				qm.Name, ValidateTagName, ParamRequiredLabel))

			r.PathFields = append(r.PathFields, qm)
			continue
		}
	}

	return
}

func (r *RouteSwagger) Id() string { return r.Api }

// RouteParam 路由参数的原始类型信息（由反射获得）
// 具体包含查询参数,路径参数,请求体参数和响应体参数
type RouteParam struct {
	Prototype      reflect.Type   `description:"直接反射后获取的类型,未提取指针指向的类型"`
	T              ModelSchema    `description:"泛型路由"`
	Name           string         `description:"名称"`
	Pkg            string         `description:"包含包名,如果是结构体则为: 包名.结构体名, 处理了指针"`
	QueryName      string         `description:"作为查询参数时显示的名称,由于无法通过反射获得参数的名称，因此此名称为手动分配"`
	DataType       DataType       `description:"如果是指针,则为指针指向的类型定义"`
	RouteParamType RouteParamType `description:"参数路由类型, 并非完全准确, 只在限制范围内访问"`
	PrototypeKind  reflect.Kind   `description:"原始类型的参数类型"`
	Index          int            `description:"参数处于方法中的原始位置,可通过 method.RouteType.In(Index) 或 method.RouteType.Out(Index) 反向获得此参数"`
	IsPtr          bool           `description:"标识 Prototype 是否是指针类型"`
	IsTime         bool           `description:"是否是时间类型"`
	IsNil          bool           `description:"todo"`
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

// Init 初始化类型反射信息，此方法在所有 Scanner.Init 之前调用
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

	r.IsTime = r.Pkg == TimePkg

	// 当其作为查询参数时，手动分配一个名称
	r.QueryName = fmt.Sprintf("%s%s%d", r.Name, CustomQueryNameConnector, r.Index)
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
		default:
		}
	}
	return v
}

// ElemKind 获得子元素的真实类型
// 如果是指针类型,则上浮获得实际元素的类型; 如果是数组/切片类型, 则获得元素的类型
// 如果是 [][]any 类型的,则为[]any类型
func (r *RouteParam) ElemKind() reflect.Kind {
	rt := r.CopyPrototype()
	if rt.Kind() == reflect.Array || rt.Kind() == reflect.Slice {
		return rt.Elem().Kind()
	}

	for rt.Kind() == reflect.Ptr {
		rt = rt.Elem()
	}

	return rt.Kind()
}

// ReflectCallSchemaDesc 反射调用结构体的 SchemaDesc 方法
func ReflectCallSchemaDesc(re reflect.Type) string {
	method, found := re.MethodByName(SchemaDescMethodName)
	if found {
		// 创建一个的实例
		var rt = re
		var desc string
		if rt.Kind() == reflect.Ptr {
			rt = rt.Elem()
			// 指针类型
			newValue := reflect.New(rt).Interface()
			result := method.Func.Call([]reflect.Value{reflect.ValueOf(newValue)})
			desc = result[0].String()
		} else {
			newValue := reflect.New(rt).Interface()
			result := method.Func.Call([]reflect.Value{reflect.ValueOf(newValue).Elem()})
			desc = result[0].String()
		}

		return desc
	} else {
		return ""
	}
}

// CreateRouteIdentify 获得一个路由对象的唯一标识
func CreateRouteIdentify(method, url string) string {
	return utils.CombineStrings(method, RouteMethodSeparator, url)
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
