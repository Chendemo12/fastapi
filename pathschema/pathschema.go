package pathschema

import (
	"strings"
	"sync"
)

const (
	PathParamPrefix          = ":" // 路径参数起始字符
	PathSeparator            = "/" // 路径分隔符
	OptionalQueryParamPrefix = "?" // 查询参数起始字符,也是路径参数结束字符
)

// RoutePathSchema 路由格式化方案
type RoutePathSchema interface {
	Name() string                       // 路由名称
	Connector() string                  // 路径连接符
	Split(relativePath string) []string // 将相对路由分组
}

// LowerCamelCase 小驼峰
type LowerCamelCase struct{}

func (s LowerCamelCase) Name() string { return "LowerCamelCase" }

func (s LowerCamelCase) Connector() string { return "" }

func (s LowerCamelCase) Split(relativePath string) []string {
	return []string{}
}

// UpperCamelCase 大驼峰
type UpperCamelCase struct{}

func (s UpperCamelCase) Name() string { return "UpperCamelCase" }

func (s UpperCamelCase) Connector() string { return "" }

func (s UpperCamelCase) Split(relativePath string) []string {
	return []string{}
}

// Dash 短横线
type Dash UnixDash

// UnixDash 短横线
type UnixDash struct{}

func (s UnixDash) Name() string { return "UnixDash" }

func (s UnixDash) Connector() string { return "" }

func (s UnixDash) Split(relativePath string) []string {
	return []string{}
}

// Backslash 反斜杠
type Backslash struct{}

func (s Backslash) Name() string { return "Backslash" }

func (s Backslash) Connector() string { return "" }
func (s Backslash) Split(relativePath string) []string {
	return []string{}
}

// Original 原始不变，保持结构体方法名,只拼接成合法的路由
type Original struct{}

func (s Original) Name() string { return "Original" }

func (s Original) Connector() string { return "" }

func (s Original) Split(relativePath string) []string {
	return []string{relativePath}
}

// Composition 组合式路由格式化方案, 通过按顺序执行多个 RoutePathSchema 获得最终路由
// 此方案会将多个 RoutePathSchema.Connector 拼接成一个唯一的 Connector
// 在执行 Split 时, 具体步骤为:
//
//  1. 将 relativePath 代入第一个 RoutePathSchema,得到 Split 后的字符串数组 spans,
//
//  2. 将 spans 的元素作为相对路由代入之后的 RoutePathSchema.Split, 得到临时字符串数组 ss,
//     将 ss 内部的元素拼接起来(直接+连接)代入下一个 RoutePathSchema.Split, 直到执行完全部的 RoutePathSchema.Split,
//     把最后的字符串数组拼接起来得到 S1
//
//  3. 继续迭代 spans 的元素, 重复步骤2得到 Sx
//
//  4. Format 方法会将 S1 ~ Sx 以 Connector 为连接符组合起来得到最终的路由
type Composition struct {
	schemas []RoutePathSchema
	linker  string
}

func (s *Composition) Name() string { return "Composition" }

func (s *Composition) Connector() string {
	if s.linker == "" {
		links := make([]string, len(s.schemas))
		for index, schema := range s.schemas {
			links[index] = schema.Connector()
		}
		s.linker = strings.Join(links, "")
	}

	return s.linker
}

func (s *Composition) Split(relativePath string) []string {
	if len(s.schemas) == 0 {
		return []string{relativePath}
	}

	spans := s.schemas[0].Split(relativePath)
	for i := 0; i <= len(spans); i++ {
		span := spans[i]
		spans[i] = s.iter(span)
	}

	return spans
}

func (s *Composition) next(relativePath string, index int) string {
	ss := s.schemas[index+1].Split(relativePath)
	return strings.Join(ss, "")
}

func (s *Composition) iter(relativePath string) string {
	span := ""

	// TODO: wrong
	var ss []string
	// 必须>1才有意义
	for i := 1; i < len(s.schemas)-1; i++ {
		ss = s.schemas[i].Split(relativePath)
		span = s.next(strings.Join(ss, ""), i)
	}

	return span
}

func NewComposition(schemas ...RoutePathSchema) *Composition {
	schema := &Composition{schemas: schemas}
	return schema
}

// Format 按照方案格式化并组合路由
func Format(prefix string, relativePath string, schema RoutePathSchema) string {
	if len(relativePath) == 0 {
		return prefix
	}

	var full string
	// 首先判断是否有尾随符号
	if !strings.HasSuffix(prefix, PathSeparator) {
		full = prefix + PathSeparator
	} else {
		full = prefix
	}

	spans := schema.Split(relativePath)
	for _, span := range spans {
		full += span
		full += schema.Connector()
	}

	return full
}

var schema RoutePathSchema
var one = &sync.Once{}

// Default 默认路由解析方法, 全小写-短横线
func Default() RoutePathSchema {
	one.Do(func() {
		schema = &Composition{schemas: []RoutePathSchema{&LowerCamelCase{}, &UnixDash{}}}
		// TODO:
		schema = &Original{}
	})

	return schema
}
