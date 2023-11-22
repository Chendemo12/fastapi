package fastapi

import "path"

// RoutePathSchema 路由格式化方案
type RoutePathSchema interface {
	Name() string                                     // 路由名称
	Format(prefix string, relativePath string) string // 路由组合方法
}

// LowerCamelCase 小驼峰
type LowerCamelCase struct{}

func (s LowerCamelCase) Name() string { return "LowerCamelCase" }

func (s LowerCamelCase) Format(prefix string, relativePath string) string {
	return path.Join(prefix, relativePath)
}

// UpperCamelCase 大驼峰
type UpperCamelCase struct{}

func (s UpperCamelCase) Name() string { return "UpperCamelCase" }

func (s UpperCamelCase) Format(prefix string, relativePath string) string {
	return path.Join(prefix, relativePath)
}

// Dash 短横线
type Dash struct{}

func (s Dash) Name() string { return "Dash" }

func (s Dash) Format(prefix string, relativePath string) string {
	return path.Join(prefix, relativePath)
}

// UnixDash 短横线
type UnixDash struct{}

func (s UnixDash) Name() string { return "UnixDash" }

func (s UnixDash) Format(prefix string, relativePath string) string {
	return path.Join(prefix, relativePath)
}

// Original 原始不变，保持结构体名
type Original struct{}

func (s Original) Name() string { return "Original" }

func (s Original) Format(prefix string, relativePath string) string {
	return path.Join(prefix, relativePath)
}

// Backslash 反斜杠
type Backslash struct{}

func (s Backslash) Name() string { return "Backslash" }

func (s Backslash) Format(prefix string, relativePath string) string {
	return path.Join(prefix, relativePath)
}

// DoNothing 不做任何解析
type DoNothing struct{}

func (s DoNothing) Name() string { return "DoNothing" }

func (s DoNothing) Format(prefix string, relativePath string) string {
	return path.Join(prefix, relativePath)
}
