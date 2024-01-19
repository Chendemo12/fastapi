package fastapi

// Finder 固定元素的查找器
// 用于从一个固定的元素集合内依据唯一标识来快速查找元素
type Finder[T RouteIface] interface {
	Init(items []T) // 通过固定元素初始化查找器
	Get(id string) (T, bool)
	Range(fn func(item T) bool)
}

type IndexFinder[T RouteIface] struct {
	prototype  T                   `description:"todo"`
	calc       func(id string) int `description:"输入索引获得元素所在集合的下标"`
	cache      []T
	elementNum int
}

func (f *IndexFinder[T]) Init(items []T) {
	f.elementNum = len(items) // 记录元素总数量
	f.cache = make([]T, f.elementNum)

	const limit = 1 << 16
	// TODO: 存在问题
	if f.elementNum < limit {
		f.calc = func(id string) int {
			return f.checksum([]byte(id)) % f.elementNum
		}
	} else {
		f.calc = func(id string) int {
			return f.crc([]byte(id)) % f.elementNum
		}
	}

	for _, item := range items {
		index := f.calc(item.Id())
		f.cache[index] = item
	}
}

func (f *IndexFinder[T]) Get(id string) (T, bool) {
	index := f.calc(id)
	item := f.cache[index]
	// TODO:
	return item, true
}

// Range if false returned, for-loop will stop
func (f *IndexFinder[T]) Range(fn func(item T) bool) {
	for i := 0; i < f.elementNum; i++ {
		b := fn(f.cache[i])
		if !b {
			return
		}
	}
}

// 经典校验和算法, 适用于长度小于65535个元素
func (f *IndexFinder[T]) checksum(data []byte) int {
	sum := 0
	for i := 0; i < len(data); i += 2 {
		if i+1 == len(data) {
			sum += int(data[i])
		} else {
			sum += int(data[i])<<8 + int(data[i+1])
		}
	}
	sum = (sum >> 16) + (sum & 0xffff)
	sum += sum >> 16

	return ^sum
}

func (f *IndexFinder[T]) crc(data []byte) int {
	return f.checksum(data)
}

type SimpleFinder[T RouteIface] struct {
	prototype  T
	cache      []T
	elementNum int
}

func (s *SimpleFinder[T]) Init(items []T) {
	s.elementNum = len(items)
	s.cache = make([]T, s.elementNum)
	for i := 0; i < s.elementNum; i++ {
		s.cache[i] = items[i]
	}
}

func (s *SimpleFinder[T]) Get(id string) (T, bool) {
	for i := 0; i < s.elementNum; i++ {
		if s.cache[i].Id() == id {
			return s.cache[i], true
		}
	}
	return s.prototype, false
}

// Range if false returned, for-loop will stop
func (s *SimpleFinder[T]) Range(fn func(item T) bool) {
	for i := 0; i < s.elementNum; i++ {
		b := fn(s.cache[i])
		if !b {
			return
		}
	}
}

func DefaultFinder() Finder[RouteIface] {
	return &SimpleFinder[RouteIface]{}
}
