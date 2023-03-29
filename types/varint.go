package types

import (
	"encoding/binary"
	"reflect"
	"sync"
)

// *****************************************************************************************

type VarIntIface interface {
	GetBytes() []byte // 获取字节串
	Build() []byte
}

// PyVarInt Python-construct VarInt类型的Golang实现
type PyVarInt struct {
	bytes []byte // 不定长的字节串
	Value int64  // 解码后的值
}

// GetBytes 获取字节串
func (pv *PyVarInt) GetBytes() []byte {
	if pv == nil {
		return nil
	}
	return pv.bytes
}

// Build 将一个int64类型的数字编码成不定长的字节串
//	@return	编码后的字节串
func (pv *PyVarInt) Build() []byte {
	acc := make([]byte, 8)
	wri := binary.PutVarint(acc, pv.Value)
	pv.bytes = acc[:wri]
	return acc[:wri]
}

// Parse 解析一个不定长的字节串为int64数字
//	@return	解析后的数字、字节串的有效长度
func (pv *PyVarInt) Parse(data []byte) (int64, int) {
	pv.bytes = data
	return binary.Varint(data)
}

type PyUVarInt struct {
	bytes []byte // 不定长的字节串
	Value uint64 // 解码后的值
}

// GetBytes 获取字节串
func (pv *PyUVarInt) GetBytes() []byte {
	if pv == nil {
		return nil
	}
	return pv.bytes
}

// Build 将一个uint64类型的数字编码成不定长的字节串
//	@return	编码后的字节串
func (pv *PyUVarInt) Build() []byte {
	acc := make([]byte, 8)
	wri := binary.PutUvarint(acc, pv.Value)
	pv.bytes = acc[:wri]
	return acc[:wri]
}

// Parse 解析一个不定长的字节串为uint64数字
//	@return	解析后的数字、字节串的有效长度
func (pv *PyUVarInt) Parse(data []byte) (uint64, int) {
	pv.bytes = data
	return binary.Uvarint(data)
}

//	________  ____  ____ __________ _/ /_____  _____
//	/ ___/ _ \/ __ \/ __ `/ ___/ __ `/ __/ __ \/ ___/
//	(__  )  __/ /_/ / /_/ / /  / /_/ / /_/ /_/ / /
//	/____/\___/ .___/\__,_/_/   \__,_/\__/\____/_/
//	/_/
//

// VarIntArray 可变长数组
type VarIntArray struct {
	array []int
	len   int
	cap   int
	lock  sync.Mutex
}

// MakeVarIntArray 新建一个可变长数组
func MakeVarIntArray(len, cap int) *VarIntArray {
	s := new(VarIntArray)
	if len > cap {
		panic("len large than cap")
	}
	// 把切片当数组用
	array := make([]int, cap, cap)
	// 元数据
	s.array = array
	s.cap = cap
	s.len = 0
	return s
}

// Append 增加一个元素
func (a *VarIntArray) Append(element int) {
	// 并发锁
	a.lock.Lock()
	defer a.lock.Unlock()
	// 大小等于容量，表示没多余位置了
	if a.len == a.cap {
		// 没容量，数组要扩容，扩容到两倍
		newCap := 2 * a.len
		// 如果之前的容量为0，那么新容量为1
		if a.cap == 0 {
			newCap = 1
		}
		newArray := make([]int, newCap, newCap)
		// 把老数组的数据移动到新数组
		for k, v := range a.array {
			newArray[k] = v
		}
		// 替换数组
		a.array = newArray
		a.cap = newCap
	}
	// 把元素放在数组里
	a.array[a.len] = element
	// 真实长度+1
	a.len = a.len + 1
}

// AppendMany 增加多个元素
func (a *VarIntArray) AppendMany(element ...int) {
	for _, v := range element {
		a.Append(v)
	}
}

// Get 获取某个下标的元素
func (a *VarIntArray) Get(index int) int {
	// 越界了
	if a.len == 0 || index >= a.len {
		panic("index over len")
	}
	return a.array[index]
}

// Len 返回真实长度
func (a *VarIntArray) Len() int {
	return a.len
}

// Cap 返回容量
func (a *VarIntArray) Cap() int {
	return a.cap
}

// Reversed 反转数组
func Reversed(s interface{}) {
	size := reflect.ValueOf(s).Len()
	swap := reflect.Swapper(s)
	for i, j := 0, size-1; i < j; i, j = i+1, j-1 {
		swap(i, j)
	}
}
