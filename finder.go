package fastapi

type FinderItem interface {
	Id() string
}

// Finder 固定元素的查找器
// 用于从一个固定的元素集合内依据唯一标识来快速查找元素
type Finder interface {
	Init(items []FinderItem) // 通过固定元素初始化查找器
	Get(id string) (FinderItem, bool)
}

type IndexFinder[T FinderItem] struct {
	prototype  T
	elementNum int
	cache      []T
	calc       func(id string) int // 输入索引获得元素所在集合的下标
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
