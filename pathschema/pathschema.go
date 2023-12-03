package pathschema

import (
	"regexp"
	"strings"
	"unicode"
)

const (
	PathParamPrefix          = ":" // 路径参数起始字符
	PathSeparator            = "/" // 路径分隔符
	OptionalQueryParamPrefix = "?" // 查询参数起始字符,也是路径参数结束字符
)

var rule = regexp.MustCompile(`[A-Z][a-z]*`)

// Default 默认路由解析方法, 全小写-短横线
func Default() RoutePathSchema { return LowerCaseDash }

// LowerCaseDash 全小写-短横线
var LowerCaseDash = NewComposition(&LowerCase{}, &UnixDash{})

// LowerCaseBackslash 全小写段路由
var LowerCaseBackslash = NewComposition(&LowerCase{}, &Backslash{})

// RoutePathSchema 路由格式化方案
type RoutePathSchema interface {
	Name() string                       // 路由名称
	Connector() string                  // 路径连接符，仅对分组后的相对路由有效，用于将分段后的相对路由组合为一个完成的路由
	Split(relativePath string) []string // 将相对路由分段，relativePath 为去除 Http.Method 后的方法名
}

// Dash 短横线
type Dash UnixDash

// LowerCamelCase 小驼峰
// 将结构体名称按单词分割后转换为小驼峰的形式后作为相对路由
//
//	# example
//
//	GetClipboardContent()	=> /clipboardContent
//	ClipSettingsPost()		=> /clipSettings
//	QueryTextHistoryGet()	=> /queryTextHistory
type LowerCamelCase struct{}

func (s LowerCamelCase) Name() string { return "LowerCamelCase" }

func (s LowerCamelCase) Connector() string { return "" }

func (s LowerCamelCase) Split(relativePath string) []string {
	return []string{LowercaseFirstLetter(relativePath)}
}

// LowerCase 全小写字符
// 将方法名按单词分割后全部转换为小写字符再直接拼接起来
//
//	# example
//
//	GetClipboardContent()	=> /clipboardcontent
//	ClipSettingsPost()		=> /clipsettings
//	QueryTextHistoryGet()	=> /querytexthistory
type LowerCase struct{}

func (s LowerCase) Name() string { return "LowerCase" }

func (s LowerCase) Connector() string { return "" }

func (s LowerCase) Split(relativePath string) []string {
	spans := SplitWords(relativePath)
	for i := 0; i < len(spans); i++ {
		spans[i] = LowercaseFirstLetter(spans[i])
	}
	return spans
}

// UnixDash 短横线
// 将方法名按单词分割后用"-"相连接
//
//	# example
//
//	GetClipboardContent()	=> /Clipboard-Content
//	ClipSettingsPost()		=> /Clip-Settings
//	QueryTextHistoryGet()	=> /Query-Text-History
type UnixDash struct{}

func (s UnixDash) Name() string { return "UnixDash" }

func (s UnixDash) Connector() string { return "-" }

func (s UnixDash) Split(relativePath string) []string {
	return SplitWords(relativePath)
}

// Underline 下划线
// 将方法名按单词分割后用"_"相连接
//
//	# example
//
//	GetClipboardContent()	=> /Clipboard-Content
//	ClipSettingsPost()		=> /Clip-Settings
//	QueryTextHistoryGet()	=> /Query-Text-History
type Underline struct{}

func (s Underline) Name() string { return "Underline" }

func (s Underline) Connector() string { return "_" }

func (s Underline) Split(relativePath string) []string {
	return SplitWords(relativePath)
}

// Backslash 反斜杠
// 按单词分段，每一个单词都作为一个路由段
//
//	# example
//
//	GetClipboardContent()	=> /Clipboard/Content
//	ClipSettingsPost()		=> /Clip/Settings
//	QueryTextHistoryGet()	=> /Query/Text/History
type Backslash struct{}

func (s Backslash) Name() string { return "Backslash" }

func (s Backslash) Connector() string { return PathSeparator }

func (s Backslash) Split(relativePath string) []string {
	return SplitWords(relativePath)
}

// Original 原始不变，保持结构体方法名(不含HTTP方法名),只拼接成合法的路由
// 由于结构体非导出方法不会作为路由函数处理，因此此方案等同于大驼峰形式
//
//	# example
//
//	GetClipboardContent()	=> /ClipboardContent
//	ClipSettingsPost()		=> /ClipSettings
//	QueryTextHistoryGet()	=> /QueryTextHistory
type Original struct{}

func (s Original) Name() string { return "Original" }

func (s Original) Connector() string { return "" }

func (s Original) Split(relativePath string) []string {
	return []string{relativePath}
}

// AddPrefix 用于在分段路由基础上添加一个前缀字符，作用于每一段路径，通常与其他方案组合使用
type AddPrefix struct {
	Prefix string
}

func (s AddPrefix) Name() string { return "AddPrefix" }

func (s AddPrefix) Connector() string { return "" }

func (s AddPrefix) Split(relativePath string) []string {
	spans := SplitWords(relativePath)
	for i := 0; i < len(spans); i++ {
		spans[i] = s.Prefix + spans[i]
	}

	return spans
}

// AddSuffix 用于在分段路由基础上添加一个后缀字符，作用于每一段路径，通常与其他方案组合使用
type AddSuffix struct {
	Suffix string
}

func (s AddSuffix) Name() string { return "AddSuffix" }

func (s AddSuffix) Connector() string { return "" }

func (s AddSuffix) Split(relativePath string) []string {
	spans := SplitWords(relativePath)
	for i := 0; i < len(spans); i++ {
		spans[i] = spans[i] + s.Suffix
	}

	return spans
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
//
// 如果格式化方案为空，则效果等同于 Original
type Composition struct {
	schemas []RoutePathSchema
	linker  string
}

func (s *Composition) Name() string { return "Composition" }

func (s *Composition) Connector() string {
	return s.linker
}

func (s *Composition) Split(relativePath string) []string {
	if len(s.schemas) == 0 {
		return []string{relativePath}
	}

	spans := s.schemas[0].Split(relativePath)
	for i := 0; i < len(spans); i++ {
		for _, schema := range s.schemas[0:] {
			spans[i] = strings.Join(schema.Split(spans[i]), "")
		}
	}

	return spans
}

func NewComposition(schemas ...RoutePathSchema) *Composition {
	schema_ := &Composition{schemas: schemas}
	links := make([]string, len(schemas))
	for index, s := range schemas {
		links[index] = s.Connector()
	}
	schema_.linker = strings.Join(links, "")

	return schema_
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

	return full + strings.Join(spans, schema.Connector())
}

// SplitWords 将字符串s按照单词进行切分, 判断单词的依据为：是否首字母大写
// 如果输入s无法切分，则返回只有s构成的一个元素的数组
func SplitWords(s string) []string {
	// TODO: 处理数字，和下划线等
	spans := rule.FindAllString(s, -1)
	if spans == nil {
		spans = []string{s}
	}

	return spans
}

// LowercaseFirstLetter 将一个字符串的首字母转换为小写形式
func LowercaseFirstLetter(s string) string {
	if len(s) == 0 {
		return s
	}

	r := []rune(s)
	r[0] = unicode.ToLower(r[0])

	return string(r)
}

// UppercaseFirstLetter 将一个字符串的首字母转换为大写形式
func UppercaseFirstLetter(s string) string {
	if len(s) == 0 {
		return s
	}

	r := []rune(s)
	r[0] = unicode.ToTitle(r[0])

	return string(r)
}
