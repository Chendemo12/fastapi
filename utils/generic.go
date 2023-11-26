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
