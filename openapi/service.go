package openapi

import (
	"github.com/Chendemo12/fastapi/pathschema"
	"net/http"
	"reflect"
	"runtime"
	"strconv"
	"strings"
)

// IsAnonymousStruct 是否是匿名(未声明)的结构体
func IsAnonymousStruct(fieldType reflect.Type) bool {
	if fieldType.Kind() == reflect.Ptr {
		return fieldType.Elem().Name() == ""
	}
	return fieldType.Name() == ""
}

// GetElementType 获取实际元素的反射类型
func GetElementType(rt reflect.Type) reflect.Type {
	var fieldType reflect.Type

	switch rt.Kind() {
	case reflect.Ptr, reflect.Slice, reflect.Array:
		fieldType = rt.Elem()
	default:
		fieldType = rt
	}

	return fieldType
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

// ReflectObjectType 获取任意对象的类型，若为指针，则反射具体的类型
func ReflectObjectType(obj any) reflect.Type {
	rt := reflect.TypeOf(obj)
	if rt.Kind() == reflect.Ptr {
		rt = rt.Elem()
	}
	return rt
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

// ReflectFuncName 反射获得函数名或方法名
func ReflectFuncName(handler any) string {
	funcName := runtime.FuncForPC(reflect.ValueOf(handler).Pointer()).Name()
	parts := strings.Split(funcName, ".")
	funcName = parts[len(parts)-1]
	return funcName
}

// IsFieldRequired 从tag中判断此字段是否是必须的
func IsFieldRequired(tag reflect.StructTag) bool {
	for _, name := range []string{GinValidateTagName, DefaultValidateTagName} {
		bindings := strings.Split(QueryFieldTag(tag, name, ""), ",") // binding 存在多个值
		for i := 0; i < len(bindings); i++ {
			if strings.TrimSpace(bindings[i]) == DefaultParamRequiredLabel {
				return true
			}
		}
	}

	return false
}

// GetDefaultV 从Tag中提取字段默认值
func GetDefaultV(tag reflect.StructTag, otype DataType) (v any) {
	defaultV := QueryFieldTag(tag, DefaultValueTagNam, "")

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

// QueryFieldTag 查找struct字段的Tag
//
//	@param	tag			reflect.StructTag	字段的Tag
//	@param	label		string				要查找的标签
//	@param	undefined	string				当查找的标签不存在时返回的默认值
//	@return	string 查找到的标签值, 不存在则返回提供的默认值
func QueryFieldTag(tag reflect.StructTag, label string, undefined string) string {
	if tag == "" {
		return undefined
	}
	if v := tag.Get(label); v != "" {
		return v
	}
	return undefined
}

// QueryJsonName 查询字段定义的json名称
func QueryJsonName(tag reflect.StructTag, undefined string) string {
	if tag == "" {
		return undefined
	}
	if v := tag.Get(DefaultJsonTagName); v != "" {
		return strings.TrimSpace(strings.Split(v, ",")[0])
	}
	return undefined
}

// MakeOperationRequestBody 将路由中的 *openapi.BaseModelMeta 转换成 openapi 的请求体 RequestBody
func MakeOperationRequestBody(model *BaseModelMeta) *RequestBody {
	if model == nil {
		return &RequestBody{}
	}

	return &RequestBody{
		Required: model.IsRequired(),
		Content: &PathModelContent{
			MIMEType: MIMEApplicationJSON,
			Schema:   model,
		},
	}
}

// MakeOperationResponses 将路由中的 *openapi.BaseModelMeta 转换成 openapi 的返回体 []*Response
func MakeOperationResponses(model *BaseModelMeta) []*Response {
	if model == nil { // 若返回值为空，则设置为字符串
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
	m[1] = Resp422

	return m
}

func QModelToParameter(model *QModel) *Parameter {
	p := &Parameter{
		ParameterBase: ParameterBase{
			Name:        model.SchemaPkg(),
			Description: model.SchemaDesc(),
			In:          InQuery,
			Required:    model.IsRequired(),
			Deprecated:  false,
		},
		Schema: &ParameterSchema{
			Type:  model.SchemaType(),
			Title: model.SchemaTitle(),
		},
		Default: GetDefaultV(model.Tag, model.SchemaType()),
	}

	if model.InPath {
		p.In = InPath
	}

	return p
}

func getModelNames(fieldMeta *BaseModelField, fieldType reflect.Type) (string, string) {
	var pkg, name string
	if IsAnonymousStruct(fieldType) {
		// 未命名的结构体类型, 没有名称, 分配包名和名称
		name = fieldMeta.Name + "Model"
		//pkg = fieldMeta.Pkg + AnonymousModelNameConnector + name
		pkg = fieldMeta.Pkg
	} else {
		pkg = fieldType.String() // 关联模型
		name = fieldType.Name()
	}

	return pkg, name
}
