package fastapi

// routeKey 作为 map 的复合键：两个 string 字段的 struct。
// 查找时在栈上构造、无堆分配，一次哈希即可定位 —— 这是热路径零分配的关键。
type routeKey struct {
	method string
	path   string
}

// Finder 固定元素的查找器
// 用于从一个固定的路由集合内，依据 (Method, StaticPath) 快速查找元素。
//
// 并发约定：Init 在初始化阶段阻塞调用一次，完成后集合不再变更；
// 此后 Get / Range 均为只读操作，可安全并发调用。
type Finder[T RouteIface] interface {
	Init(items []T) // 通过固定元素初始化查找器
	Get(method, path string) (T, bool)
	Range(fn func(item T) bool)
}

// IndexFinder 基于哈希表的路由查找器，Get 为 O(1)。
//
// Init 完成后 index/cache 只读，因此 Get/Range 无需加锁即可并发调用
// （Go 的 map 在无并发写入时支持任意并发读）。这里不使用 sync.Map，
// 因为 sync.Map 是为“并发读写”优化的，而本场景初始化后只读，普通 map 更快。
//
// 契约：查找是对已解析的静态路由模板做精确相等匹配，调用方须保证两侧
// 规范化一致（method 统一大写、path 用同一套注册模板形式）。同一
// (method, path) 重复注册时后者覆盖前者（视为注册期编程错误）。
type IndexFinder[T RouteIface] struct {
	prototype T              // 查找失败时返回的零值
	index     map[routeKey]T // (method, path) -> 元素，负责 O(1) 查找
	cache     []T            // 保留注册顺序，供 Range 使用
}

func (f *IndexFinder[T]) Init(items []T) {
	f.index = make(map[routeKey]T, len(items))
	f.cache = make([]T, len(items))
	for i, item := range items {
		f.cache[i] = item
		f.index[routeKey{item.Swagger().Method, item.Swagger().Url}] = item
	}
}

func (f *IndexFinder[T]) Get(method, path string) (T, bool) {
	item, ok := f.index[routeKey{method, path}]
	if !ok {
		return f.prototype, false
	}
	return item, true
}

// Range if false returned, for-loop will stop
func (f *IndexFinder[T]) Range(fn func(item T) bool) {
	for _, item := range f.cache {
		if !fn(item) {
			return
		}
	}
}

// DefaultFinder 返回通用的高性能路由查找器（O(1) 查找）。
func DefaultFinder() Finder[RouteIface] {
	return &IndexFinder[RouteIface]{}
}
