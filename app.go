package fastapi

import (
	"context"
	"fmt"
	"github.com/Chendemo12/fastapi-tool/helper"
	"github.com/Chendemo12/fastapi/openapi"
	"github.com/Chendemo12/fastapi/utils"
	jsoniter "github.com/json-iterator/go"
	"log"
	"net"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/Chendemo12/fastapi-tool/cronjob"
	"github.com/Chendemo12/fastapi-tool/logger"
)

var one = &sync.Once{}
var wrapper *Wrapper = nil // 默认实例

type FastApi = Wrapper

// Wrapper 服务对象，本质是一个包装器
type Wrapper struct {
	conf          *Profile           `description:"配置项"`
	service       *Service           `description:"全局服务依赖"`
	pool          *sync.Pool         `description:"Wrapper.Context资源池"`
	mux           MuxWrapper         `description:"后端路由器"`
	isStarted     chan struct{}      `description:"标记程序是否完成启动"`
	groupRouters  []*GroupRouterMeta `description:"路由组对象"`
	genericRoutes []RouteIface       `description:"泛型路由对象"`
	events        []*Event           `description:"启动和关闭事件"`
	finder        Finder[RouteIface] `description:"路由对象查找器"`
	previousDeps  []MiddlewareHandle `description:"在接口参数校验前执行的中间件"` // TODO Future-231126.4: 路由前后中间件
	afterDeps     []MiddlewareHandle `description:"在接口参数校验成功后执行的中间件"`
}

type Profile struct {
	Host                               string        `json:"host,omitempty" description:"运行地址"`
	Port                               string        `json:"port,omitempty" description:"运行端口"`
	Title                              string        `json:"title,omitempty" description:"程序名,同时作为日志文件名"`
	Version                            string        `json:"version,omitempty" description:"程序版本号"`
	Description                        string        `json:"description,omitempty" description:"程序描述"`
	ShutdownTimeout                    time.Duration `json:"shutdownTimeout,omitempty" description:"平滑关机,单位秒"`
	Debug                              bool          `json:"debug,omitempty" description:"调试开关"`
	SwaggerDisabled                    bool          `json:"swaggerDisabled,omitempty" description:"禁用自动文档"`
	ContextAutomaticDerivationDisabled bool          `json:"contextAutomaticDerivationDisabled,omitempty" description:"禁用context自动派生"`
	// 默认情况下当请求校验过程遇到错误字段时，仍会继续向下校验其他字段，并最终将所有的错误消息一次性返回给调用方-
	// 当此设置被开启后，在遇到一个错误的参数时，会立刻停止终止流程，直接返回错误消息
	StopImmediatelyWhenErrorOccurs bool `json:"stopImmediatelyWhenErrorOccurs" description:"是否在遇到错误字段时立刻停止校验"`
}

func (f *Wrapper) initService() *Wrapper {
	f.service.addr = net.JoinHostPort(f.conf.Host, f.conf.Port)

	if f.conf.Version == "" {
		f.conf.Version = "1.0.0"
	}

	f.pool = &sync.Pool{
		New: func() interface{} {
			c := new(Context)
			c.svc = f.service
			return c
		},
	}

	return f
}

// 初始化路由, 必须在路由添加完成，swagger注册之前调用
func (f *Wrapper) initRoutes() *Wrapper {
	var err error
	// 解析路由组路由
	for _, group := range f.groupRouters {
		err = group.Init()
		if err != nil {
			panic(fmt.Errorf("group-router: '%s' created failld, %v", group.String(), err))
		}
	}

	// 处理并记录泛型路由
	for _, route := range f.genericRoutes {
		_ = route
	}

	for _, route := range f.genericRoutes {
		_ = route
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

	// 处理并记录泛型路由
	for _, r := range f.genericRoutes {
		routes = append(routes, r)
	}

	// 初始化finder
	f.finder = DefaultFinder()
	f.finder.Init(routes)

	return f
}

// 创建 OpenApi Swagger 文档, 必须等上层注册完路由之后才能调用
func (f *Wrapper) initSwagger() *Wrapper {
	f.service.openApi = openapi.NewOpenApi(f.Config().Title, f.Config().Version, f.Config().Description)
	if !f.conf.SwaggerDisabled || f.conf.Debug {
		f.registerRouteDoc()
		f.registerRouteHandle()
	}

	return f
}

func (f *Wrapper) initMux() *Wrapper {
	if f.mux == nil {
		panic("mux is not initialized")
	}

	f.wrap(f.mux)
	return f
}

// 初始化Wrapper,并完成服务依赖的建立
// 启动前，必须显式的初始化Wrapper的基本配置，若初始化中发生异常则panic
func (f *Wrapper) initialize() *Wrapper {
	helper.SetJsonEngine(jsoniter.ConfigCompatibleWithStandardLibrary)
	LazyInit()

	f.initService()
	f.initRoutes()
	f.initFinder()
	f.initMux()
	f.initSwagger() // === 必须最后调用

	f.service.Logger().Debug(
		"Run at: " + utils.Ternary[string](f.conf.Debug, "Development", "Production"),
	)
	return f
}

// ResetRunMode 重设运行时环境
func (f *Wrapper) resetRunMode(md bool) {
	f.conf.Debug = md
}

// 绑定数据到路由器上
func (f *Wrapper) wrap(mux MuxWrapper) *Wrapper {
	var err error
	// 挂载路由到路由器上
	for _, group := range f.groupRouters {
		for _, route := range group.Routes() {
			err = mux.BindRoute(route.Swagger().Method, route.Swagger().Url, f.Handler)
			if err != nil {
				// 此时日志已初始化完毕
				f.Service().Logger().Error(fmt.Sprintf(
					"route: '%s:%s' bind failed, %v", route.Swagger().Method, route.Swagger().Url, err,
				))
			}
		}
	}

	for _, route := range f.genericRoutes {
		err = mux.BindRoute(route.Swagger().Method, route.Swagger().Url, f.Handler)
		if err != nil {
			// 此时日志已初始化完毕
			f.Service().Logger().Error(fmt.Sprintf(
				"route: '%s:%s' bind failed, %v", route.Swagger().Method, route.Swagger().Url, err,
			))
		}
	}

	return f
}

// ================================ Api ================================

func (f *Wrapper) Config() Config {
	return Config{
		Logger:                         f.service.Logger(),
		Version:                        f.conf.Version,
		Description:                    f.conf.Description,
		Title:                          f.conf.Title,
		ShutdownTimeout:                int(f.conf.ShutdownTimeout.Seconds()),
		DisableSwagAutoCreate:          f.conf.SwaggerDisabled,
		StopImmediatelyWhenErrorOccurs: f.conf.StopImmediatelyWhenErrorOccurs,
		Debug:                          f.conf.Debug,
	}
}

// Service 获取Wrapper全局服务依赖
func (f *Wrapper) Service() *Service { return f.service }

// Mux 获取路由器
func (f *Wrapper) Mux() MuxWrapper { return f.mux }

// SetMux 设置路由器，必须在启动之前设置
func (f *Wrapper) SetMux(mux MuxWrapper) *Wrapper {
	f.mux = mux
	return f
}

// OnEvent 添加事件
//
//	@param	kind	事件类型，取值需为	"startup"/"shutdown"
//	@param	fs		func()		事件
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

// SetLogger 替换日志句柄，此操作必须在run之前进行
//
//	@param	logger	logger.Iface	日志句柄
func (f *Wrapper) SetLogger(logger logger.Iface) *Wrapper {
	f.service.setLogger(logger)
	return f
}

// SetDescription 设置APP的详细描述信息
//
//	@param	Description	string	详细描述信息
func (f *Wrapper) SetDescription(description string) *Wrapper {
	f.conf.Description = description
	return f
}

// IncludeRouter 注册一个路由组
//
//	@param	router	*Router	路由组
func (f *Wrapper) IncludeRouter(router GroupRouter) *Wrapper {
	f.groupRouters = append(f.groupRouters, NewGroupRouteMeta(router))
	return f
}

// UsePrevious 添加一个校验前中间件，此中间件会在：请求参数校验前调用
func (f *Wrapper) UsePrevious(middleware ...MiddlewareHandle) *Wrapper {
	f.previousDeps = append(f.previousDeps, middleware...)
	return f
}

// UseAfter 添加一个校验后中间件, 此中间件会在：请求参数校验后-路由函数调用前执行
func (f *Wrapper) UseAfter(middleware ...MiddlewareHandle) *Wrapper {
	f.afterDeps = append(f.afterDeps, middleware...)
	return f
}

// Use 添加一个中间件, 数据校验后中间件
func (f *Wrapper) Use(middleware ...MiddlewareHandle) *Wrapper {
	return f.UseAfter(middleware...)
}

// AddCronjob 添加定时任务(循环调度任务)
// 此任务会在各种初始化及启动事件全部执行完成之后触发
func (f *Wrapper) AddCronjob(jobs ...cronjob.CronJob) *Wrapper {
	f.service.scheduler.Add(jobs...)
	return f
}

// ActivateHotSwitch 创建一个热开关，监听信号量(默认值：30)，用来改变程序调试开关状态
func (f *Wrapper) ActivateHotSwitch(s ...int) *Wrapper {
	var st = HotSwitchSigint
	if len(s) > 0 {
		st = s[0]
	}

	swt := make(chan os.Signal, 1)
	signal.Notify(swt, syscall.Signal(st))

	go func() {
		for {
			select {
			case <-f.Service().Done():
				return
			case <-swt:
				if f.conf.Debug {
					f.resetRunMode(false)
				} else {
					f.resetRunMode(true)
				}
				f.service.Logger().Debug(
					"Hot-switch received, convert to: ", utils.Ternary[string](f.conf.Debug, "Development", "Production"),
				)
			}
		}
	}()

	return f
}

// SetShutdownTimeout 修改关机前最大等待时间
//
//	@param	timeout	in	修改关机前最大等待时间,	单位秒
func (f *Wrapper) SetShutdownTimeout(timeout int) *Wrapper {
	f.conf.ShutdownTimeout = time.Duration(timeout) * time.Second
	return f
}

// DisableSwagAutoCreate 禁用文档自动生成
func (f *Wrapper) DisableSwagAutoCreate() *Wrapper {
	f.conf.SwaggerDisabled = true
	return f
}

// Shutdown 平滑关闭
func (f *Wrapper) Shutdown() {
	f.service.cancel() // 标记结束

	// 执行关机前事件
	for _, event := range f.events {
		if event.Type == ShutdownEvent {
			event.Fc()
		}
	}

	go func() {
		err := f.mux.ShutdownWithTimeout(f.conf.ShutdownTimeout)
		if err != nil {
			fmt.Println(err.Error())
		}
	}()
	// Engine().Shutdown() 执行成功后将会直接退出进程，以下代码段仅当超时未关闭时执行到。
	// Shutdown() 不会关闭设置了 keepalive 的连接，除非设置了 ReadTimeout ，因此设置以下内容以确保关闭.
	<-time.After(f.conf.ShutdownTimeout * time.Second)
	// 此处避免因logger关闭引发错误
	fmt.Println("Forced shutdown.") // 仅当超时时会到达此行
}

// Run 启动服务, 此方法会阻塞运行，因此必须放在main函数结尾
// 此方法已设置关闭事件和平滑关闭.
// 当 Interrupt 信号被触发时，首先会关闭 根Context，然后逐步执行“关机事件”，最后调用平滑关闭方法，关闭服务
// 启动前通过 SetShutdownTimeout 设置"平滑关闭异常时"的最大超时时间
func (f *Wrapper) Run(host, port string) {
	f.conf.Host = host
	f.conf.Port = port

	f.initialize()
	f.ActivateHotSwitch()

	// 执行启动前事件
	for _, event := range f.events {
		if event.Type == StartupEvent {
			event.Fc()
		}
	}

	f.isStarted <- struct{}{} // 解除阻塞上层的任务
	f.service.Logger().Debug("HTTP server listening on: " + f.service.Addr())

	// 在各种初始化及启动事件执行完成之后触发
	f.service.scheduler.Run()
	close(f.isStarted)

	go func() {
		log.Fatal(f.mux.Listen(f.service.Addr()))
	}()

	// 关闭开关, buffered
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt)

	<-quit // 阻塞进程，直到接收到停止信号,准备关闭程序
	f.Shutdown()
}

type Config struct {
	Logger                         logger.Iface `json:"-" description:"日志"`
	Version                        string       `json:"version,omitempty" description:"APP版本号"`
	Description                    string       `json:"description,omitempty" description:"APP描述"`
	Title                          string       `json:"title,omitempty" description:"APP标题,也是日志文件名"`
	ShutdownTimeout                int          `json:"shutdown_timeout,omitempty" description:"平滑关机,单位秒"`
	DisableSwagAutoCreate          bool         `json:"disable_swag_auto_create,omitempty" description:"禁用自动文档"`
	StopImmediatelyWhenErrorOccurs bool         `json:"stopImmediatelyWhenErrorOccurs" description:"是否在遇到错误字段时立刻停止校验"`
	Debug                          bool         `json:"debug,omitempty" description:"调试模式"`
}

func cleanConfig(confs ...Config) Config {
	conf := Config{
		Title:                 "FastAPI",
		Version:               "1.0.0",
		Debug:                 false,
		Description:           "FastAPI Application",
		Logger:                nil,
		ShutdownTimeout:       5,
		DisableSwagAutoCreate: false,
	}
	if len(confs) > 0 {
		if confs[0].Title != "" {
			conf.Title = confs[0].Title
		}
		if confs[0].Version != "" {
			conf.Version = confs[0].Version
		}
		if confs[0].Description != "" {
			conf.Description = confs[0].Description
		}
		conf.Debug = confs[0].Debug
		conf.Logger = confs[0].Logger
		conf.ShutdownTimeout = confs[0].ShutdownTimeout
		conf.DisableSwagAutoCreate = confs[0].DisableSwagAutoCreate
		conf.StopImmediatelyWhenErrorOccurs = confs[0].StopImmediatelyWhenErrorOccurs
	}

	return conf
}

// New 实例化一个默认 Wrapper, 此方法与 Create 不能同时使用
// 与 Create 区别在于：Create 每次都会创建一个新的实例，NewNotStruct 多次调用获得的是同一个实例
// 语义相当于 __new__ 实现的单例
func New(c ...Config) *Wrapper {
	one.Do(func() {
		conf := cleanConfig(c...)
		wrapper = Create(conf)
	})

	return wrapper
}

// Create 创建一个新的 Wrapper 服务
// 其存在目的在于在同一个应用里创建多个 Wrapper 实例，并允许每个实例绑定不同的服务器实现
//
// getWrapper
func Create(c Config) *Wrapper {
	conf := cleanConfig(c)

	sc := &Service{}
	sc.ctx, sc.cancel = context.WithCancel(context.Background())
	sc.scheduler = cronjob.NewScheduler(sc.ctx, nil)

	app := &Wrapper{
		conf: &Profile{
			Title:           conf.Title,
			Version:         conf.Version,
			Description:     conf.Description,
			Debug:           conf.Debug,
			SwaggerDisabled: conf.DisableSwagAutoCreate,
			ShutdownTimeout: time.Duration(conf.ShutdownTimeout) * time.Second,
		},
		service:       sc,
		genericRoutes: make([]RouteIface, 0),
		groupRouters:  make([]*GroupRouterMeta, 0),
		isStarted:     make(chan struct{}, 1),
		previousDeps:  make([]MiddlewareHandle, 0),
		afterDeps:     make([]MiddlewareHandle, 0),
		events:        make([]*Event, 0),
	}

	if conf.Description != "" {
		app.SetDescription(conf.Description)
	}

	if conf.Logger != nil {
		app.SetLogger(conf.Logger)
	}

	if conf.ShutdownTimeout != 0 {
		app.SetShutdownTimeout(conf.ShutdownTimeout)
	}

	if conf.DisableSwagAutoCreate {
		app.DisableSwagAutoCreate()
	}

	return app
}
