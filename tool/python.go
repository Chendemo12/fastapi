// Package tool 常用的python标准库方法实现
package tool

import (
	"github.com/Chendemo12/fastapi/types"
	"reflect"
	"strconv"
)

// Any 任意一个参数为true时为true
func Any(args ...bool) bool {
	for i := 0; i < len(args); i++ {
		if args[i] {
			return true
		}
	}
	return false
}

// All 参数全部为true时为true
func All(args ...bool) bool {
	if len(args) == 0 {
		return false
	} else {
		for i := 0; i < len(args); i++ {
			if !args[i] {
				return false
			}
		}
		return true
	}
}

// Repr 格式化显示对象
// 将任意对象输出为字符串格式，内部对struct,map,array,string,error实现了string的格式化转换
//
//	number -> number
//	bool -> "true" or "false"
//	array -> "0d 56 f6";
//	map -> json;
//	struct -> StructName({json});
//	error -> error.Error();
//
//	@param	object		any		需要格式化显示的对象
//	@param	excludeName	bool	不显示对象名称,	针对struct有效
//	@return	string ObjectName(...)
func Repr(object any, excludeName ...bool) (message string) {
	if object == nil {
		message = ""
	}
	switch object.(type) {
	// 数值类型
	case int8:
		return strconv.FormatInt(int64(object.(int8)), 10)
	case int16:
		return strconv.FormatInt(int64(object.(int16)), 10)
	case int32:
		return strconv.FormatInt(int64(object.(int32)), 10)
	case int:
		return strconv.FormatInt(int64(object.(int)), 10)
	case int64:
		return strconv.FormatInt(object.(int64), 10)

	case uint8:
		return strconv.FormatUint(uint64(object.(uint8)), 10)
	case uint16:
		return strconv.FormatUint(uint64(object.(uint16)), 10)
	case uint32:
		return strconv.FormatUint(uint64(object.(uint32)), 10)
	case uint:
		return strconv.FormatUint(uint64(object.(uint)), 10)
	case uint64:
		return strconv.FormatUint(object.(uint64), 10)

	case float32:
		return strconv.FormatFloat(float64(object.(float32)), 'f', -1, 64)
	case float64:
		return strconv.FormatFloat(object.(float64), 'f', -1, 64)

	// 字符串类型
	case string:
		message = object.(string)
	case bool:
		if object.(bool) {
			return "true"
		} else {
			return "false"
		}

	// 数组类型
	case []string:
		message = CombineStrings(object.([]string)...)
	case []byte:
		message = HexBeautify(object.([]byte))

	// 其他类型
	case error:
		message = object.(error).Error()
	case uintptr:
		return "uintptr"

	default: // 复杂的自定义类型
		ex := Any(excludeName...) // 是否排除对象名
		if !ex {                  // 不排除对象名
			name := ""
			at := reflect.TypeOf(object)
			switch at.Kind() {
			case reflect.Pointer: // 指针类型,结构体类型
				name = at.Elem().Name()
			case reflect.Struct:
				name = at.Name()
			}

			bs, err := Marshal(&object)
			if err != nil {
				message = name
			} else {
				message = CombineStrings(name, "(", string(bs), ")")
			}
		} else {
			bytes, err := Marshal(object)
			if err != nil {
				message = ""
			}
			message = string(bytes)
		}
	}
	return
}

// Max 计算列表内部的最大元素, 需确保目标列表不为空
func Max[T types.Ordered](p ...T) T {
	r := p[0]
	for i := 0; i < len(p); i++ {
		if r > p[i] {
			continue
		}
		r = p[i]
	}
	return r
}

// Min 计算列表内部的最小元素, 需确保目标列表不为空
func Min[T types.Ordered](p ...T) T {
	r := p[0]
	for i := 0; i < len(p); i++ {
		if r < p[i] {
			continue
		}
		r = p[i]
	}
	return r
}

// Index 获取指定元素在列表中的下标,若不存在则返回-1
func Index[T comparable](s []T, x T) int {
	for i := 0; i < len(s); i++ {
		if s[i] == x {
			return i
		}
	}

	return -1
}

// In 查找序列s内是否存在元素x
//
//	@param	s	[]T	查找序列
//	@param	x	T	特定元素
//	@return	bool true if s contains x, false otherwise
func In[T comparable](s []T, x T) bool {
	for i := 0; i < len(s); i++ {
		if s[i] == x {
			return true
		}
	}
	return false
}
