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
	QueryBinders() []ModelBinder              // 查询参数的处理接口(查询参数名:处理接口)，每一个查询参数都必须绑定一个 ParamBinder
	RequestBinders() ModelBinder              // 请求体的处理接口,请求体也只有一个,内部已处理文件+表单
	ResponseBinder() ModelBinder              // 响应体的处理接口,响应体只有一个
	NewInParams(ctx *Context) []reflect.Value // 创建一个完整的函数入参实例列表, 此方法会在完成请求参数校验之后执行
	NewStructQuery() any                      // 创建一个结构体查询参数实例,对于POST/PATCH/PUT, 即为 NewInParams 的最后一个元素; 对于GET/DELETE则为nil
	NewRequestModel() any                     // 创建一个请求体实例,对于POST/PATCH/PUT, 即为 NewInParams 的最后一个元素; 对于GET/DELETE则为nil
	HasStructQuery() bool                     // 是否存在结构体查询参数，如果存在则会调用 NewStructQuery 获得结构体实例
	HasFileRequest() bool                     // 是否存在上传文件
	Call(in []reflect.Value) []reflect.Value  // 调用API
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

// InferParamBinder 利用反射推断查询参数、路径参数、cookies参数等的校验器
// ！！！！！ 不能推断请求体参数 ！！！！！
// ！！！！！ 不能推断响应体参数 ！！！！！
func (s ScanHelper) InferParamBinder(param openapi.SchemaIface, prototypeKind reflect.Kind, paramType openapi.RouteParamType) ModelBinder {
	nothing := NewNothingModelBinder(param, paramType)
	if param == nil {
		return nothing
	}

	var binder ModelBinder
	switch prototypeKind {

	case reflect.Int:
		binder = &IntModelBinder[int]{
			modelName: param.SchemaTitle(),
			paramType: paramType,
			Maximum:   openapi.IntMaximum,
			Minimum:   openapi.IntMinimum,
		}
	case reflect.Int64:
		binder = &IntModelBinder[int64]{
			modelName: param.SchemaTitle(),
			paramType: paramType,
			Maximum:   openapi.Int64Maximum,
			Minimum:   openapi.Int64Minimum,
		}
	case reflect.Int8:
		binder = &IntModelBinder[int8]{
			modelName: param.SchemaTitle(),
			paramType: paramType,
			Maximum:   openapi.Int8Maximum,
			Minimum:   openapi.Int8Minimum,
		}
	case reflect.Int16:
		binder = &IntModelBinder[int16]{
			modelName: param.SchemaTitle(),
			paramType: paramType,
			Maximum:   openapi.Int16Maximum,
			Minimum:   openapi.Int16Minimum,
		}
	case reflect.Int32:
		binder = &IntModelBinder[int32]{
			modelName: param.SchemaTitle(),
			paramType: paramType,
			Maximum:   openapi.Int32Maximum,
			Minimum:   openapi.Int32Minimum,
		}

	case reflect.Uint:
		binder = &UintModelBinder[uint]{
			modelName: param.SchemaTitle(),
			paramType: paramType,
			Maximum:   openapi.UintMaximum,
			Minimum:   openapi.UintMinimum,
		}
	case reflect.Uint64:
		binder = &UintModelBinder[uint64]{
			modelName: param.SchemaTitle(),
			paramType: paramType,
			Maximum:   openapi.UintMaximum,
			Minimum:   openapi.UintMinimum,
		}
	case reflect.Uint8:
		binder = &UintModelBinder[uint8]{
			modelName: param.SchemaTitle(),
			paramType: paramType,
			Maximum:   openapi.Uint8Maximum,
			Minimum:   openapi.Uint8Minimum,
		}
	case reflect.Uint16:
		binder = &UintModelBinder[uint16]{
			modelName: param.SchemaTitle(),
			paramType: paramType,
			Maximum:   openapi.Uint16Maximum,
			Minimum:   openapi.Uint16Minimum,
		}
	case reflect.Uint32:
		binder = &UintModelBinder[uint32]{
			modelName: param.SchemaTitle(),
			paramType: paramType,
			Maximum:   openapi.Uint32Maximum,
			Minimum:   openapi.Uint32Minimum,
		}
	case reflect.Bool:
		binder = &BoolModelBinder{modelName: param.SchemaTitle(), paramType: paramType}

	case reflect.Float32:
		binder = &FloatModelBinder[float32]{modelName: param.SchemaTitle(), paramType: paramType}
	case reflect.Float64:
		binder = &FloatModelBinder[float64]{modelName: param.SchemaTitle(), paramType: paramType}

	case reflect.String:
		binder = nothing

	case reflect.Struct:
		binder = &JsonModelBinder[any]{modelName: param.SchemaTitle(), paramType: paramType}

	default:
		binder = nothing
	}

	return binder
}

func (s ScanHelper) InferRequestBinder(swagger *openapi.RouteSwagger) ModelBinder {
	var binder ModelBinder = &NothingModelBinder{modelName: "", paramType: openapi.RouteParamRequest}

	if swagger.RequestContentType != openapi.MIMEApplicationJSON && swagger.RequestContentType != openapi.MIMEApplicationJSONCharsetUTF8 && swagger.RequestContentType != openapi.MIMEMultipartForm {
		// 暂不支持非json和multiform-data的请求参数验证
		return binder
	}

	if swagger.Method == http.MethodPost || swagger.Method == http.MethodPut || swagger.Method == http.MethodPatch {
		if swagger.RequestFile {
			// 存在上传文件定义，则从 multiform-data 中获取上传参数
			if swagger.RequestModel != nil && swagger.RequestModel.SchemaPkg() != openapi.NoneRequestPkg { // file + json
				b := &FileWithParamModelBinder{}
				b.paramType = openapi.RouteParamRequest
				b.modelName = swagger.RequestModel.JsonName()
				binder = b
			} else {
				// modelName 为固定值，固定为 openapi.MultipartFormFileName
				binder = &FileModelBinder{openapi.MultipartFormFileName}
			}
		} else {
			// 处理特殊类型 fastapi.None
			if swagger.RequestModel != nil && swagger.RequestModel.SchemaPkg() != openapi.NoneRequestPkg { // 此情况基本不存在
				binder = &RequestModelBinder{modelName: swagger.RequestModel.SchemaTitle()}
			}
		}
	}

	// get/delete 方法没有请求体
	return binder
}

// InferResponseBinder 推断响应体的校验器,
// 当返回值为
//
//	基本数据类型时，	binder=NothingModelBinder
//	struct 时，		binder=JsonModelBinder
//	FileResponse 时，binder=NothingModelBinder
func (s ScanHelper) InferResponseBinder(model *openapi.BaseModelMeta, routeType RouteType) ModelBinder {
	var binder ModelBinder = NewNothingModelBinder(model, openapi.RouteParamResponse)
	if model == nil {
		return binder
	}

	// 对于非struct类型,函数签名就已经保证了类型的正确性,无需手动校验
	if !model.SchemaType().IsBaseType() {
		if !model.Param.IsFile && !model.Param.IsSSE {
			binder = s.InferParamBinder(model.Param, model.Param.ElemKind(), openapi.RouteParamResponse)
		}
	}

	return binder
}

func (s ScanHelper) InferQueryBinder(qmodel *openapi.QModel, routeType RouteType) ModelBinder {
	var binder ModelBinder

	if qmodel.IsTime {
		binder = &DateTimeModelBinder{modelName: qmodel.SchemaTitle(), paramType: openapi.RouteParamQuery}
	} else {
		binder = scanHelper.InferParamBinder(qmodel, qmodel.Kind, openapi.RouteParamQuery)
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
