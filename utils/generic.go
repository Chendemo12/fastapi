package utils

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

// Reverse 数组倒序, 就地修改
//
//	@param	s	*[]T	需要倒序的序列
func Reverse[T any](s *[]T) {
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
