package tool

import (
	"encoding/base64"
	jsoniter "github.com/json-iterator/go"
	"reflect"
	"strings"
	"unsafe"
)

const hexTable = "0123456789abcdef"

//goland:noinspection GoUnusedGlobalVariable
var ( // 替换json标准库，提供更好的性能
	// 与标准库 100%兼容的配置
	json              = jsoniter.ConfigCompatibleWithStandardLibrary
	MarshalJSON       = json.Marshal
	UnmarshalJSON     = json.Unmarshal
	MarshalJSONIndent = json.MarshalIndent
	NewJSONDecoder    = json.NewDecoder
	NewJSONEncoder    = json.NewEncoder
	// JSONFast 更快的配置，浮点数仅能保留6位小数, 且不能序列化HTML
	JSONFast          = jsoniter.ConfigFastest
	FastMarshalJSON   = JSONFast.Marshal
	FastUnmarshalJSON = JSONFast.Unmarshal
	// JSON 默认配置
	JSON      = JSONFast
	Marshal   = JSON.Marshal
	Unmarshal = JSON.Unmarshal
)

//goland:noinspection GoUnusedGlobalVariable
var (
	F           = CombineStrings
	StringsJoin = CombineStrings
)

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
	if bytes, err := Marshal(&object); err != nil {
		return ""
	} else {
		return string(bytes)
	}
}

// Has 查找序列s内是否存在元素x
//
//	@param	s	[]T	查找序列
//	@param	x	T	特定元素
//	@return	bool true if s contains x, false otherwise
func Has[T comparable](s []T, x T) bool {
	for i := 0; i < len(s); i++ {
		if s[i] == x {
			return true
		}
	}
	return false
}

// Reversed 数组倒序, 就地修改
//
//	@param	s	*[]T	需要倒序的序列
func Reversed[T any](s *[]T) {
	length := len(*s)
	var temp T
	for i := 0; i < length/2; i++ {
		temp = (*s)[i]
		(*s)[i] = (*s)[length-1-i]
		(*s)[length-1-i] = temp
	}
}

// IsEqual 判断2个切片是否相等
//
//	@return	true if is equal
func IsEqual[T comparable](a, b []T) bool {
	if len(a) != len(b) {
		return false
	}
	for i := 0; i < len(a); i++ {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

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
