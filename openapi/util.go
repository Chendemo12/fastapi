package openapi

import (
	"net/http"
	"reflect"
	"strconv"
	"strings"
)

// StringsToInts 将字符串数组转换成int数组, 简单实现
//
//	@param	strs	[]string	输入字符串数组
//	@return	[]int 输出int数组
func StringsToInts(strs []string) []int {
	ints := make([]int, 0)

	for _, s := range strs {
		i, err := strconv.Atoi(s)
		if err != nil {
			continue
		}
		ints = append(ints, i)
	}

	return ints
}

// StringsToFloats 将字符串数组转换成float64数组, 简单实现
//
//	@param	strs	[]string	输入字符串数组
//	@return	[]float64 输出float64数组
func StringsToFloats(strs []string) []float64 {
	floats := make([]float64, len(strs))

	for _, s := range strs {
		i, err := strconv.ParseFloat(s, 10)
		if err != nil {
			continue
		}
		floats = append(floats, i)
	}

	return floats
}

// MakeOperationRequestBody 将路由中的 *openapi.Metadata 转换成 openapi 的请求体 RequestBody
func MakeOperationRequestBody(model *Metadata) *RequestBody {
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

// MakeOperationResponses 将路由中的 *openapi.Metadata 转换成 openapi 的返回体 []*Response
func MakeOperationResponses(model *Metadata) []*Response {
	if model == nil { // 若返回值为空，则设置为字符串
		model = String
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

// FastApiRoutePath 将 fiber.App 格式的路径转换成 FastApi 格式的路径
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
func FastApiRoutePath(path string) string {
	paths := strings.Split(path, PathSeparator) // 路径字符
	// 查找路径中的参数
	for i := 0; i < len(paths); i++ {
		if strings.HasPrefix(paths[i], PathParamPrefix) {
			// 识别到路径参数
			if strings.HasSuffix(paths[i], OptionalPathParamSuffix) {
				// 可选路径参数
				paths[i] = "{" + paths[i][1:len(paths[i])-1] + "}"
			} else {
				paths[i] = "{" + paths[i][1:] + "}"
			}
		}
	}

	return strings.Join(paths, PathSeparator)
}

func QModelToParameter(model *QModel) *Parameter {
	p := &Parameter{
		ParameterBase: ParameterBase{
			Name:        model.SchemaName(),
			Description: model.SchemaDesc(),
			In:          InQuery,
			Required:    model.IsRequired(),
			Deprecated:  false,
		},
		Schema: &ParameterSchema{
			Type:  model.SchemaType(),
			Title: model.Title,
		},
		Default: GetDefaultV(model.Tag, model.SchemaType()),
	}

	if model.InPath {
		p.In = InPath
	}

	return p
}

// ReflectObjectType 获取任意对象的类型，若为指针，则反射具体的类型
func ReflectObjectType(obj any) reflect.Type {
	rt := reflect.TypeOf(obj)
	if rt.Kind() == reflect.Ptr {
		rt = rt.Elem()
	}
	return rt
}

// ReflectKindToOType 转换reflect.Kind为swagger类型说明
//
//	@param	ReflectKind	reflect.Kind	反射类型
func ReflectKindToOType(kind reflect.Kind) (name DataType) {
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
	for _, name := range []string{"binding", "validate"} {
		bindings := strings.Split(QueryFieldTag(tag, name, ""), ",") // binding 存在多个值
		for i := 0; i < len(bindings); i++ {
			if strings.TrimSpace(bindings[i]) == "required" {
				return true
			}
		}
	}

	return false
}

// GetDefaultV 从Tag中提取字段默认值
func GetDefaultV(tag reflect.StructTag, otype DataType) (v any) {
	defaultV := QueryFieldTag(tag, "default", "")

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

// IsArray 判断一个对象是否是数组类型
func IsArray(object any) bool {
	if object == nil {
		return false
	}
	return ReflectKindToOType(reflect.TypeOf(object).Kind()) == ArrayType
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
	if v := tag.Get("json"); v != "" {
		return strings.TrimSpace(strings.Split(v, ",")[0])
	}
	return undefined
}
