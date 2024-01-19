package utils

import (
	"reflect"
	"runtime"
	"strconv"
	"strings"

	"encoding/base64"
	"unsafe"

	jsoniter "github.com/json-iterator/go"
)

const hexTable = "0123456789abcdef"

//goland:noinspection GoUnusedGlobalVariable
var ( // 替换json标准库，提供更好的性能
	// 与标准库 100%兼容的配置
	json = jsoniter.ConfigCompatibleWithStandardLibrary
	// FasterJson 更快的配置，浮点数仅能保留6位小数, 且不能序列化HTML
	FasterJson          = jsoniter.ConfigFastest
	FasterJsonMarshal   = FasterJson.Marshal
	FasterJsonUnmarshal = FasterJson.Unmarshal
	// DefaultJson 默认配置
	DefaultJson       = FasterJson
	JsonMarshal       = DefaultJson.Marshal
	JsonUnmarshal     = DefaultJson.Unmarshal
	JsonMarshalIndent = DefaultJson.MarshalIndent
	JsonNewDecoder    = DefaultJson.NewDecoder
	JsonNewEncoder    = DefaultJson.NewEncoder
)

// SetJsonEngine 修改默认的JSON配置
func SetJsonEngine(api jsoniter.API) {
	DefaultJson = api
}

//goland:noinspection GoUnusedGlobalVariable
var (
	F           = CombineStrings
	StringsJoin = CombineStrings
)

// B2S 将[]byte转换为字符串,(就地修改)零内存分配
func B2S(b []byte) string {
	return *(*string)(unsafe.Pointer(&b))
}

// S2B 将字符串转换为[]byte,(就地修改)零内存分配
func S2B(s string) (b []byte) {
	bh := (*reflect.SliceHeader)(unsafe.Pointer(&b))
	sh := (*reflect.StringHeader)(unsafe.Pointer(&s))
	bh.Data = sh.Data
	bh.Cap = sh.Len
	bh.Len = sh.Len

	return b
}

// HexBeautify 格式化显示十六进制
func HexBeautify(src []byte) string {
	if len(src) == 0 {
		return ""
	}

	length := len(src) * 3 // 一个byte用2个字符+1个空格表示
	dst := make([]byte, length)

	j := 0
	for _, v := range src {
		dst[j] = hexTable[v>>4]
		dst[j+1] = hexTable[v&0x0f]
		dst[j+2] = 32 // 空格
		j += 3
	}
	// 去除末尾的空格
	return B2S(dst)[:length-1]
}

// CombineStrings 合并字符串, 实现等同于strings.Join()，只是少了判断分隔符
func CombineStrings(elems ...string) string {
	switch len(elems) {
	case 0:
		return ""
	case 1:
		return elems[0]
	}
	n := 0
	for i := 0; i < len(elems); i++ {
		n += len(elems[i])
	}

	var b strings.Builder
	b.Grow(n)
	b.WriteString(elems[0])
	for _, s := range elems[1:] {
		b.WriteString(s)
	}
	return b.String()
}

// WordCapitalize 单词首字母大写
//
//	@param	word	string	单词
//	@return	string 首字母大写的单词
func WordCapitalize(word string) string {
	return strings.ToUpper(word)[:1] + strings.ToLower(word[1:])
}

// Base64Encode base64编码
//
//	@param	data	[]byte	字节流
//	@return	string base64字符串
func Base64Encode(data []byte) string {
	return base64.StdEncoding.EncodeToString(data)
}

// Base64Decode base64解码
//
//	@param	data	string	base64字符串
func Base64Decode(data string) ([]byte, error) {
	return base64.StdEncoding.DecodeString(data)
}

// MapToString 将字典转换成字符串显示
func MapToString(object map[string]any) string {
	if bytes, err := JsonMarshal(&object); err != nil {
		return ""
	} else {
		return string(bytes)
	}
}

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

// Pluralize 获得一个单词的复数形式，按照语法规则进行变换
func Pluralize(word string) string {
	if word == "" {
		return word
	}

	// 定义一些特殊情况的规则
	irregulars := map[string]string{
		"man":   "men",
		"woman": "women",
		"child": "children",
		"Man":   "Men",
		"Woman": "Women",
		"Child": "Children",
	}

	// 检查特殊规则
	if plural, ok := irregulars[word]; ok {
		return plural
	}

	// 检查以 "s", "x", "z", "ch", "sh" 结尾的单词
	if strings.HasSuffix(word, "s") || strings.HasSuffix(word, "x") || strings.HasSuffix(word, "z") ||
		strings.HasSuffix(word, "ch") || strings.HasSuffix(word, "sh") {
		return word + "es"
	}

	// 检查以辅音字母 + "y" 结尾的单词
	if strings.HasSuffix(word, "y") && !isVowel(word[len(word)-2]) {
		return word[:len(word)-1] + "ies"
	}

	// 默认情况，在单词末尾加上 "s"
	return word + "s"
}

func isVowel(c uint8) bool {
	vowels := "aeiou"

	return strings.ContainsRune(vowels, rune(c))
}

// Ternary 三元运算符
func Ternary[T any](cond bool, ifTrue, ifFalse T) T {
	if cond {
		return ifTrue
	}
	return ifFalse
}

// SliceFilter 从数组spans中过滤出符合条件的元素，返回一个新的切片
//
//	@param	spans	[]T					原始切片
//	@param	filter	func(span T) bool	如果为true则包含在结果集中
func SliceFilter[T any](spans []T, filter func(span T) bool) []T {
	newSpans := make([]T, 0)
	for _, span := range spans {
		if filter(span) {
			newSpans = append(newSpans, span)
		}
	}

	return newSpans
}
