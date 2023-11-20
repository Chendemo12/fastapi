package fastapi

type RoutePathSchema string

const (
	LowerCamelCase RoutePathSchema = "LowerCamelCase"
	UpperCamelCase RoutePathSchema = "UpperCamelCase"
	Dash           RoutePathSchema = "Dash"      // 短横线
	UnixDash       RoutePathSchema = "Dash"      // 短横线
	Original       RoutePathSchema = "Original"  // 原始不变，保持结构体名
	Backslash      RoutePathSchema = "Backslash" // 反斜杠
	DoNothing      RoutePathSchema = "DoNothing" // 不做任何解析
)

type RouterGroup interface {
	// Prefix 路由组前缀，如果为空则设置为结构体名称转小写并去除可能包含的Http.Method
	// type Auth struct{}  ==> auth
	// type ReadHistory{}  ==> readHistory(方案), / read/history(方案), / read-history(方案), ReadHistory(方案)
	Prefix() string
	// Tags 标签组，如果为空则设置为结构体名称
	Tags() []string
	// RoutePathSchema 路由解析规则，对路由前缀和路由地址都有效
	RoutePathSchema() RoutePathSchema
}

type DRouterGroup struct{}

func (g *DRouterGroup) Prefix() string { return "" }

func (g *DRouterGroup) Tags() []string { return []string{"DEFAULT"} }

func (g *DRouterGroup) RoutePathSchema() RoutePathSchema { return UnixDash }

type ExampleRouterGroup struct {
	DRouterGroup
}
