package fastapi

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"time"

	"github.com/Chendemo12/fastapi/openapi"
)

var one = &sync.Once{}
var wrapper *Wrapper = nil // 默认实例

// EventKind 事件类型
type EventKind string

const (
	StartupEvent  EventKind = "startup"
	ShutdownEvent EventKind = "shutdown"
)

// Event 事件
type Event struct {
	Fc   func()
	Type EventKind // 事件类型：startup 或 shutdown
}

// Wrapper 服务对象，本质是一个包装器
//
//	# usage
//	./test/group_router_test.go
type Wrapper struct {
	conf                *Config             `description:"配置项"`
	openApi             *openapi.OpenApi    `description:"模型文档"`
	pool                *sync.Pool          `description:"Wrapper.Context资源池"`
	ctx                 context.Context     `description:"根Context"`
	cancel              context.CancelFunc  `description:"取消函数"`
	mux                 MuxWrapper          `description:"后端路由器"`
	isStarted           chan struct{}       `description:"标记程序是否完成启动"`
	groupRouters        []*GroupRouterMeta  `description:"路由组对象"`
	events              []*Event            `description:"启动和关闭事件"`
	finder              Finder[RouteIface]  `description:"路由对象查找器"`
	previousDeps        []DependenceHandle  `description:"在接口参数校验前执行的依赖函数"`
	afterDeps           []DependenceHandle  `description:"在接口参数校验成功后执行的依赖函数(相当于路由函数前钩子)"`
	beforeWrite         func(c *Context)    `description:"在数据写入响应流之前执行的钩子方法"`
	routeErrorFormatter RouteErrorFormatter `description:"handle返回错误时的格式化方法"`
}

type FastApi = Wrapper

// Default 创建应用，多次调用获取同一个对象
func Default(c ...Config) *Wrapper {
	one.Do(func() {
		conf := cleanConfig(c...)
		wrapper = New(conf)
	})

	return wrapper
}

type Config struct {
	Title                 string `json:"title,omitempty" description:"程序名,同时作为日志文件名"`
	Description           string `json:"description,omitempty" description:"APP描述"`
	Version               string `json:"version,omitempty" description:"APP版本号"`
	ShutdownTimeout       int    `json:"shutdown_timeout,omitempty" description:"平滑关机,单位秒"`
	DisableSwagAutoCreate bool   `json:"disable_swag_auto_create,omitempty" description:"禁用OpenApi文档，但是不禁用参数校验"`
	// 默认情况下当请求校验过程遇到错误字段时，仍会继续向下校验其他字段，并最终将所有的错误消息一次性返回给调用方-
	// 当此设置被开启后，在遇到一个错误的参数时，会立刻停止终止流程，直接返回错误消息
	StopImmediatelyWhenErrorOccurs     bool `json:"stopImmediatelyWhenErrorOccurs" description:"是否在遇到错误字段时立刻停止校验"`
	ContextAutomaticDerivationDisabled bool `json:"contextAutomaticDerivationDisabled,omitempty" description:"禁止为每一个请求创建单独的Context"`
	DisableResponseValidate            bool `json:"disableResponseValidate" description:"是否禁用响应参数校验，仅JSON类型有效"`

	host string
	port string
}

func (c *Config) Copy() *Config {
	return &Config{
		Title:                              c.Title,
		Description:                        c.Description,
		Version:                            c.Version,
		ShutdownTimeout:                    c.ShutdownTimeout,
		DisableSwagAutoCreate:              c.DisableSwagAutoCreate,
		ContextAutomaticDerivationDisabled: c.ContextAutomaticDerivationDisabled,
		StopImmediatelyWhenErrorOccurs:     c.StopImmediatelyWhenErrorOccurs,
		DisableResponseValidate:            c.DisableResponseValidate,
		host:                               c.host,
		port:                               c.port,
	}
}

func (c *Config) ListenAddr() string { return net.JoinHostPort(c.host, c.port) }

// 初始化路由, 必须在路由添加完成，swagger注册之前调用
func (f *Wrapper) initRoutes() *Wrapper {
	var err error
	// 解析路由组路由
	for _, group := range f.groupRouters {
		// 必须先设置参数，再 Init 初始化
		group.errorFormatter = f.routeErrorFormatter

		err = group.Init()
		if err != nil {
			panic(fmt.Errorf("group-router: '%s' created failld, %v", group.String(), err))
		}
	}

	return f
}

// 记录全部的路由对象, 包含路由组路由和泛型路由
func (f *Wrapper) initFinder() *Wrapper {
	routes := make([]RouteIface, 0)
	for _, group := range f.groupRouters {
		for _, r := range group.Routes() {
			routes = append(routes, r)
		}
	}

	// 初始化finder
	f.finder = DefaultFinder()
	f.finder.Init(routes)

	return f
}

// 创建 OpenApi Swagger 文档, 必须等上层注册完路由之后才能调用
func (f *Wrapper) initSwagger() *Wrapper {
	f.openApi = openapi.NewOpenApi(f.Config().Title, f.Config().Version, f.Config().Description)
	if !f.conf.DisableSwagAutoCreate {
		f.registerRouteDoc()
		f.registerRouteHandle()
	}

	return f
}

// 绑定数据到路由器上
func (f *Wrapper) initMux() *Wrapper {
	var err error
	// 挂载路由到路由器上
	for _, group := range f.groupRouters {
		for _, route := range group.Routes() {
			err = f.mux.BindRoute(route.Swagger().Method, route.Swagger().Url, f.Handler)
			if err != nil {
				// 此时日志已初始化完毕
				Errorf(
					"route: '%s:%s' bind failed, %v", route.Swagger().Method, route.Swagger().Url, err,
				)
			}
		}
	}

	return f
}

// 初始化Wrapper,并完成服务依赖的建立，启动前，必须显式的初始化Wrapper的基本配置，若初始化中发生异常则panic
func (f *Wrapper) initialize() *Wrapper {
	if f.mux == nil {
		panic("mux is not initialized")
	}

	if f.conf.Version == "" {
		f.conf.Version = "1.0.0"
	}

	f.pool = &sync.Pool{New: func() interface{} {
		c := new(Context)
		return c
	}}

	LazyInit()

	f.initRoutes()
	f.initFinder()
	f.initMux()
	f.initSwagger() // === 必须最后调用

	return f
}

// ================================ Api ================================

func (f *Wrapper) Config() *Config { return f.conf.Copy() }

// Done 监听程序是否退出或正在关闭，仅当程序关闭时解除阻塞
func (f *Wrapper) Done() <-chan struct{} { return f.ctx.Done() }

// Context Wrapper根 context
func (f *Wrapper) Context() context.Context { return f.ctx }

// Mux 获取路由器
func (f *Wrapper) Mux() MuxWrapper { return f.mux }

// SetMux 设置路由器，必须在启动之前设置
func (f *Wrapper) SetMux(mux MuxWrapper) *Wrapper {
	f.mux = mux
	return f
}

// OnEvent 添加事件
func (f *Wrapper) OnEvent(kind EventKind, fc func()) *Wrapper {
	switch kind {
	case StartupEvent:
		f.events = append(f.events, &Event{
			Type: StartupEvent,
			Fc:   fc,
		})
	case ShutdownEvent:
		f.events = append(f.events, &Event{
			Type: ShutdownEvent,
			Fc:   fc,
		})
	default:
	}
	return f
}

// SetDescription 设置APP的详细描述信息
func (f *Wrapper) SetDescription(description string) *Wrapper {
	f.conf.Description = description
	return f
}

// IncludeRouter 注册一个路由组
func (f *Wrapper) IncludeRouter(router GroupRouter) *Wrapper {
	f.groupRouters = append(f.groupRouters, NewGroupRouteMeta(router, f.routeErrorFormatter))
	return f
}

// UsePrevious 添加一个校验前依赖函数，此依赖函数会在：请求参数校验前调用
func (f *Wrapper) UsePrevious(hooks ...DependenceHandle) *Wrapper {
	f.previousDeps = append(f.previousDeps, hooks...)
	return f
}

// UseAfter 添加一个校验后依赖函数(也即路由前), 此依赖函数会在：请求参数校验后-路由函数调用前执行
func (f *Wrapper) UseAfter(hooks ...DependenceHandle) *Wrapper {
	f.afterDeps = append(f.afterDeps, hooks...)
	return f
}

// UseBeforeWrite 在数据写入响应流之前执行的钩子方法; 可用于日志记录, 所有请求无论何时终止都会执行此方法
func (f *Wrapper) UseBeforeWrite(fc func(c *Context)) *Wrapper {
	f.beforeWrite = fc
	return f
}

// Use 添加一个依赖函数(锚点), 数据校验后依赖函数
//
// 由于 Wrapper 的核心实现类似于装饰器, 而非常规的中间件,因此无法通过 MuxWrapper 的中间件来影响到 Wrapper 的执行过程;
// 因此 Wrapper 在关键环节均定义了相关的依赖函数，类似于hook，以此来控制执行流程;
//
//	与python-FastApi的Depends不同的地方在于：
//		python-FastApi.Depends接收Request作为入参，并将其返回值作为路由函数Handler的入参;
//		而此处的hook不返回值，而是通过 Context.Set 和 Context.Get 来进行上下文数据的传递，并通过返回 error 来终止后续的流程;
//		同时，由于 Context.Set 和 Context.Get 是线程安全的，因此可以放心的在依赖函数中操作 Context;
//	   	依赖函数的执行始终是顺序进行的，其执行顺序是固定的：
//	   	始终由 UsePrevious -> (请求参数)Validate -> UseAfter -> (路由函数)RouteHandler -> (响应参数)Validate -> UseBeforeWrite -> exit;
//
// 此处的依赖函数有校验前依赖函数和校验后依赖函数,分别通过 Wrapper.UsePrevious 和 Wrapper.UseAfter 注册;
// 当请求参数校验失败时不会执行 Wrapper.UseAfter 依赖函数, 请求参数会在 Wrapper.UsePrevious 执行完成之后被触发;
// 如果依赖函数要终止后续的流程,应返回 error, 错误消息会作为消息体返回给客户端, 响应数据格式默认为500+string,可通过 Wrapper.SetRouteErrorFormatter 进行修改;
func (f *Wrapper) Use(hooks ...DependenceHandle) *Wrapper {
	return f.UseAfter(hooks...)
}

// Deprecated:UseDepends 【别名】添加一个数据校验后依赖函数
func (f *Wrapper) UseDepends(hooks ...DependenceHandle) *Wrapper {
	return f.UseAfter(hooks...)
}

// SetRouteErrorFormatter 设置路由错误信息格式化函数
// @param	handle	RouteErrorFormatter	路由错误信息格式化函数
// @param	opt		RouteErrorOpt		路由错误信息的配置项, 仅用于设置文档显示
func (f *Wrapper) SetRouteErrorFormatter(handle RouteErrorFormatter, opt ...RouteErrorOpt) *Wrapper {
	if handle == nil {
		Warn("route error formatter is nil, ignore")
	} else {
		f.routeErrorFormatter = handle
	}
	if len(opt) > 0 {
		openapi.SetRouteErrorResponse(opt[0].ResponseMode)
		openapi.SetRouteErrorStatusCode(opt[0].StatusCode)
		openapi.SetRouteErrorDescription(opt[0].Description)
	}
	return f
}

// SetShutdownTimeout 修改关机前最大等待时间
//
//	@param	timeout	in	修改关机前最大等待时间,	单位秒
func (f *Wrapper) SetShutdownTimeout(timeout int) *Wrapper {
	f.conf.ShutdownTimeout = timeout
	return f
}

// DisableSwagAutoCreate 禁用文档自动生成
func (f *Wrapper) DisableSwagAutoCreate() *Wrapper {
	f.conf.DisableSwagAutoCreate = true
	return f
}

// DisableResponseValidate 禁用响应参数校验, 仅JSON类型有效
func (f *Wrapper) DisableResponseValidate() *Wrapper {
	f.conf.DisableResponseValidate = true
	return f
}

// Shutdown 平滑关闭
func (f *Wrapper) Shutdown() {
	Debug("ready to shutdown...")

	// 执行关机前事件
	for _, event := range f.events {
		if event.Type == ShutdownEvent {
			event.Fc()
		}
	}

	err := f.mux.ShutdownWithTimeout(time.Duration(f.conf.ShutdownTimeout) * time.Second)
	if err != nil {
		Warnf("graceful shutdown failed, err: %s", err)
	}

	f.cancel() // 停止所有请求，需最后关闭
}

// Run 启动服务, 此方法会阻塞运行，因此必须放在main函数结尾
// 此方法已设置关闭事件和平滑关闭.
// 当 Interrupt 信号被触发时，首先会关闭 根Context，然后逐步执行“关机事件”，最后调用平滑关闭方法，关闭服务
// 启动前通过 SetShutdownTimeout 设置"平滑关闭异常时"的最大超时时间
func (f *Wrapper) Run(host, port string) {
	f.conf.host = host
	f.conf.port = port
	Debugf("%s starting...", f.Config().Title)

	f.initialize()

	// 执行启动前事件
	for _, event := range f.events {
		if event.Type == StartupEvent {
			event.Fc()
		}
	}

	f.isStarted <- struct{}{} // 解除阻塞上层的任务
	addr := net.JoinHostPort(f.conf.host, f.conf.port)
	Debug("http server listening on: " + addr)

	close(f.isStarted)

	go func() {
		err := f.mux.Listen(addr)
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			panic(err)
		}
	}()

	// 关闭开关, buffered
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt)

	<-quit // 阻塞进程，直到接收到停止信号,准备关闭程序
	f.Shutdown()
}

func cleanConfig(cs ...Config) Config {
	conf := Config{
		Title:                              "FastAPI",
		Description:                        "FastAPI Application",
		Version:                            "1.0.0",
		ShutdownTimeout:                    5,
		DisableSwagAutoCreate:              false,
		StopImmediatelyWhenErrorOccurs:     false,
		ContextAutomaticDerivationDisabled: false,
		DisableResponseValidate:            false,
		host:                               "",
		port:                               "",
	}
	if len(cs) > 0 {
		if cs[0].Title != "" {
			conf.Title = cs[0].Title
		}
		if cs[0].Version != "" {
			conf.Version = cs[0].Version
		}
		if cs[0].Description != "" {
			conf.Description = cs[0].Description
		}
		conf.ShutdownTimeout = cs[0].ShutdownTimeout
		conf.DisableSwagAutoCreate = cs[0].DisableSwagAutoCreate
		conf.StopImmediatelyWhenErrorOccurs = cs[0].StopImmediatelyWhenErrorOccurs
		conf.ContextAutomaticDerivationDisabled = cs[0].ContextAutomaticDerivationDisabled
		conf.DisableResponseValidate = cs[0].DisableResponseValidate
	}

	return conf
}

// New 创建一个新的 Wrapper 服务
// 其存在目的在于在同一个应用里创建多个 Wrapper 实例，并允许每个实例绑定不同的服务器实现
func New(c Config) *Wrapper {
	conf := cleanConfig(c)

	app := &Wrapper{
		conf:                conf.Copy(),
		groupRouters:        make([]*GroupRouterMeta, 0),
		isStarted:           make(chan struct{}, 1),
		previousDeps:        make([]DependenceHandle, 0),
		afterDeps:           make([]DependenceHandle, 0),
		events:              make([]*Event, 0),
		routeErrorFormatter: defaultRouteErrorFormatter,
	}
	app.ctx, app.cancel = context.WithCancel(context.Background())
	app.beforeWrite = func(c *Context) {}

	return app
}
