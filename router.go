package fastapi

import (
	"fmt"
	"net/http"
	"reflect"

	"github.com/Chendemo12/fastapi/openapi"
)

type RouteType string

const (
	RouteTypeGroup   RouteType = "GroupRoute"
	RouteTypeGeneric RouteType = "GenericRouteMeta"
)

// QueryParamMode 查询参数的定义模式，不同模式决定了查询参数的校验方式
// 对于泛型路由来说，仅存在 结构体查询参数 StructQueryParamMode 一种形式;
// 对于路由组路由来说，三种形式都存在
type QueryParamMode string

const (
	// NoQueryParamMode 不存在查询参数 = 0
	NoQueryParamMode QueryParamMode = "NoQueryParamMode"
	// Deprecated:SimpleQueryParamMode 只有基本数据类型的简单查询参数类型，不包含结构体类型的查询参数 = 1
	SimpleQueryParamMode QueryParamMode = "SimpleQueryParamMode"
	// StructQueryParamMode 以结构体形式定义的查询参数模式 = 4
	StructQueryParamMode QueryParamMode = "StructQueryParamMode"
	// Deprecated:MixQueryParamMode 二种形式都有的混合模式 = 7
	MixQueryParamMode QueryParamMode = "MixQueryParamMode"
)

const DefaultErrorStatusCode = http.StatusInternalServerError

// Scanner 元数据接口
// Init -> Scan -> ScanInner -> Init 级联初始化
type Scanner interface {
	Init() (err error)      // 初始化元数据对象
	Scan() (err error)      // 扫描并初始化自己
	ScanInner() (err error) // 扫描并初始化自己包含的字节点,通过 child.Init() 实现
}

// RouteIface 路由定义
// 路由组接口定义或泛型接口定义都需实现此接口
type RouteIface interface {
	Scanner
	Id() string
	RouteType() RouteType
	Swagger() *openapi.RouteSwagger           // 路由文档
	QueryBinders() []*ParamBinder             // 查询参数的处理接口(查询参数名:处理接口)，每一个查询参数都必须绑定一个 ParamBinder
	RequestBinders() *ParamBinder             // 请求体的处理接口,请求体也只有一个
	ResponseBinder() *ParamBinder             // 响应体的处理接口,响应体只有一个
	NewInParams(ctx *Context) []reflect.Value // 创建一个完整的函数入参实例列表, 此方法会在完成请求参数校验 RequestBinders，QueryBinders 之后执行
	NewStructQuery() any                      // 创建一个结构体查询参数实例,对于POST/PATCH/PUT, 即为 NewInParams 的最后一个元素; 对于GET/DELETE则为nil
	NewRequestModel() any                     // 创建一个请求体实例,对于POST/PATCH/PUT, 即为 NewInParams 的最后一个元素; 对于GET/DELETE则为nil
	HasStructQuery() bool                     // 是否存在结构体查询参数，如果存在则会调用 NewStructQuery 获得结构体实例
	HasFileRequest() bool                     // 是否存在上传文件
	Call(in []reflect.Value) []reflect.Value  // 调用API
}

// ParamBinder 参数验证模型
type ParamBinder struct {
	Method         ModelBindMethod        `json:"-"`
	QModel         *openapi.QModel        `json:"-"`
	RequestModel   *openapi.BaseModelMeta `json:"-"`
	ResponseModel  *openapi.BaseModelMeta `json:"-"`
	Title          string                 `json:"-"`
	RouteParamType openapi.RouteParamType `json:"-"`
}

// BaseModel 基本数据模型, 对于上层的路由定义其请求体和响应体都应为继承此结构体的结构体
// 在 OpenApi 文档模型中,此模型的类型始终为 "object"
// 对于 BaseModel 其字段仍然可能会是 BaseModel
type BaseModel struct{}

// SchemaDesc 结构体文档注释
func (b *BaseModel) SchemaDesc() string { return openapi.InnerModelsName[0] }

// SchemaType 模型类型
func (b *BaseModel) SchemaType() openapi.DataType { return openapi.ObjectType }

func (b *BaseModel) IsRequired() bool { return true }

// ================================================================================

var scanHelper = &ScanHelper{}

type ScanHelper struct{}

// InferBinderMethod 利用反射推断参数的校验器
func (s ScanHelper) InferBinderMethod(param openapi.SchemaIface, prototypeKind reflect.Kind, modelType openapi.RouteParamType) ModelBindMethod {
	if param == nil {
		return &NothingBindMethod{}
	}

	var binder ModelBindMethod
	switch prototypeKind {

	case reflect.Int, reflect.Int64:
		binder = &IntBindMethod{
			Title:   param.SchemaTitle(),
			Kind:    prototypeKind,
			Maximum: openapi.IntMaximum,
			Minimum: openapi.IntMinimum,
		}
	case reflect.Int8:
		binder = &IntBindMethod{
			Title:   param.SchemaTitle(),
			Kind:    prototypeKind,
			Maximum: openapi.Int8Maximum,
			Minimum: openapi.Int8Minimum,
		}
	case reflect.Int16:
		binder = &IntBindMethod{
			Title:   param.SchemaTitle(),
			Kind:    prototypeKind,
			Maximum: openapi.Int16Maximum,
			Minimum: openapi.Int16Minimum,
		}
	case reflect.Int32:
		binder = &IntBindMethod{
			Title:   param.SchemaTitle(),
			Kind:    prototypeKind,
			Maximum: openapi.Int32Maximum,
			Minimum: openapi.Int32Minimum,
		}

	case reflect.Uint, reflect.Uint64:
		binder = &UintBindMethod{
			Title:   param.SchemaTitle(),
			Kind:    prototypeKind,
			Maximum: openapi.UintMaximum,
			Minimum: openapi.UintMinimum,
		}
	case reflect.Uint8:
		binder = &UintBindMethod{
			Title:   param.SchemaTitle(),
			Kind:    prototypeKind,
			Maximum: openapi.Uint8Maximum,
			Minimum: openapi.Uint8Minimum,
		}
	case reflect.Uint16:
		binder = &UintBindMethod{
			Title:   param.SchemaTitle(),
			Kind:    prototypeKind,
			Maximum: openapi.Uint16Maximum,
			Minimum: openapi.Uint16Minimum,
		}
	case reflect.Uint32:
		binder = &UintBindMethod{
			Title:   param.SchemaTitle(),
			Kind:    prototypeKind,
			Maximum: openapi.Uint32Maximum,
			Minimum: openapi.Uint32Minimum,
		}
	case reflect.Bool:
		binder = &BoolBindMethod{Title: param.SchemaTitle()}
	case reflect.Float32, reflect.Float64:
		binder = &FloatBindMethod{
			Title: param.SchemaTitle(),
			Kind:  prototypeKind,
		}
	case reflect.String:
		binder = &NothingBindMethod{}
	case reflect.Struct:
		if modelType == openapi.RouteParamResponse {
			binder = &JsonBindMethod[any]{Title: param.SchemaTitle(), RouteParamType: modelType}
		} else {
			binder = &JsonBindMethod[any]{Title: param.SchemaTitle(), RouteParamType: modelType}
		}

	default:
		binder = &NothingBindMethod{}
	}

	return binder
}

// InferResponseBinder 推断响应体的校验器
func (s ScanHelper) InferResponseBinder(model *openapi.BaseModelMeta, routeType RouteType) *ParamBinder {
	if model == nil {
		return &ParamBinder{
			Title:          "",
			RouteParamType: openapi.RouteParamResponse,
			ResponseModel:  nil,
			Method:         &NothingBindMethod{},
		}
	}

	binder := &ParamBinder{
		Title:          model.SchemaTitle(),
		RouteParamType: openapi.RouteParamResponse,
		ResponseModel:  model,
	}

	if model.SchemaType().IsBaseType() {
		if routeType == RouteTypeGroup {
			// 对于结构体路由组，其他类型的参数, 函数签名就已经保证了类型的正确性,无需手动校验
			binder.Method = &NothingBindMethod{}
		} else {
			// 推断 InferBinderMethod
			binder.Method = &NothingBindMethod{}
		}

	} else {
		binder.Method = s.InferBinderMethod(
			model.Param,
			model.Param.ElemKind(), openapi.RouteParamResponse,
		)
	}

	return binder
}

func (s ScanHelper) InferRequestBinder(model *openapi.BaseModelMeta, routeType RouteType) *ParamBinder {
	if model == nil {
		return &ParamBinder{
			Title:          "",
			RouteParamType: openapi.RouteParamRequest,
			ResponseModel:  nil,
			Method:         &NothingBindMethod{},
		}
	}

	return &ParamBinder{
		Title:          model.SchemaTitle(),
		RouteParamType: openapi.RouteParamRequest,
		RequestModel:   model,
		Method:         s.InferBinderMethod(model.Param, model.Param.ElemKind(), openapi.RouteParamRequest),
	}
}

func (s ScanHelper) InferQueryBinder(qmodel *openapi.QModel, routeType RouteType) *ParamBinder {
	binder := &ParamBinder{
		Title:          qmodel.SchemaTitle(),
		QModel:         qmodel,
		RouteParamType: openapi.RouteParamQuery,
		RequestModel:   nil,
		ResponseModel:  nil,
	}

	if qmodel.IsTime {
		binder.Method = &DateTimeBindMethod{Title: qmodel.SchemaTitle()}
	} else {
		binder.Method = scanHelper.InferBinderMethod(qmodel, qmodel.Kind, openapi.RouteParamQuery)
	}

	return binder
}

// InferBaseQueryParam 推断基本类型的查询参数
func (s ScanHelper) InferBaseQueryParam(param *openapi.RouteParam, routeType RouteType) *openapi.QModel {
	name := param.QueryName // 手动指定一个查询参数名称
	qmodel := &openapi.QModel{
		Name:     name,
		DataType: param.SchemaType(),
		Kind:     param.PrototypeKind,
		InPath:   false,
		InStruct: false,
	}
	if routeType == RouteTypeGroup { // 路由组：对于函数参数类型的查询参数,全部为必选的
		qmodel.Tag = reflect.StructTag(fmt.Sprintf(`json:"%s" %s:"%s" %s:"%s"`,
			name, openapi.QueryTagName, name, openapi.ValidateTagName, openapi.ParamRequiredLabel))
	} else { // 范型路由推断为可选的
		qmodel.Tag = reflect.StructTag(fmt.Sprintf(`json:"%s,omitempty" %s:"%s"`,
			name, openapi.QueryTagName, name))
	}

	return qmodel
}

func (s ScanHelper) InferTimeParam(param *openapi.RouteParam) (*openapi.QModel, bool) {
	if param.SchemaPkg() == openapi.TimePkg {
		// 时间类型
		return &openapi.QModel{
			Name: param.QueryName, // 手动指定一个查询参数名称
			Tag: reflect.StructTag(fmt.Sprintf(`json:"%s" %s:"%s" %s:"%s"`,
				param.QueryName, openapi.QueryTagName, param.QueryName, openapi.ValidateTagName, openapi.ParamRequiredLabel)), // 对于函数参数类型的查询参数,全部为必选的
			DataType: openapi.StringType,
			Kind:     param.PrototypeKind,
			InPath:   false,
			InStruct: false,
			IsTime:   true,
		}, true
	}

	return nil, false
}

// InferObjectQueryParam 推断结构体类型的查询参数
func (s ScanHelper) InferObjectQueryParam(param *openapi.RouteParam) []*openapi.QModel {
	var qms []*openapi.QModel
	qm, ok := s.InferTimeParam(param)
	if ok {
		qms = append(qms, qm)
	} else {
		// 对于结构体查询参数, 结构体的每一个字段都将作为一个查询参数
		qms = append(qms, openapi.StructToQModels(param.CopyPrototype())...)
	}

	return qms
}
