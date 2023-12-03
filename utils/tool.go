package utils

import (
	"reflect"
	"runtime"
	"strconv"
	"strings"
)

// StringsToInts 将字符串数组转换成int数组, 简单实现
//
//	@param	strs	[]string	输入字符串数组
//	@return	[]int 	输出int数组
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
//	@param	strs		[]string	输入字符串数组
//	@return	[]float64 	输出float64数组
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

// ReflectObjectType 获取任意对象的类型，若为指针，则反射具体的类型
func ReflectObjectType(obj any) reflect.Type {
	rt := reflect.TypeOf(obj)
	if rt.Kind() == reflect.Ptr {
		rt = rt.Elem()
	}
	return rt
}

// ReflectFuncName 反射获得函数名或方法名
func ReflectFuncName(handler any) string {
	funcName := runtime.FuncForPC(reflect.ValueOf(handler).Pointer()).Name()
	parts := strings.Split(funcName, ".")
	funcName = parts[len(parts)-1]
	return funcName
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
