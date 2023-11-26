package fastapi

import (
	"context"
	"errors"
	"fmt"
	"github.com/Chendemo12/fastapi-tool/helper"
	jsoniter "github.com/json-iterator/go"
	"log"
	"net"
	"os"
	"os/signal"
	"runtime"
	"strconv"
	"sync"
	"syscall"
	"time"

	"github.com/Chendemo12/fastapi-tool/cronjob"
	"github.com/Chendemo12/fastapi-tool/logger"
	"github.com/Chendemo12/fastapi/internal/core"
	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
)

const (
	startupEvent  EventKind = "startup"
	shutdownEvent EventKind = "shutdown"
)

var (
	once               = sync.Once{}
	appEngine *FastApi = nil // 单例模式
)

type EventKind string

type Event struct {
	Fc   func()
	Type EventKind // 事件类型：startup 或 shutdown
}

type FastApi struct {
	service       *Service           `description:"全局服务依赖"`
	engine        *fiber.App         `description:"fiber.App"`
	pool          *sync.Pool         `description:"FastApi.Context资源池"`
	isStarted     chan struct{}      `description:"标记程序是否完成启动"`
	host          string             `description:"运行地址"`
	description   string             `description:"程序描述"`
	title         string             `description:"程序名,同时作为日志文件名"`
	port          string             `description:"运行端口"`
	version       string             `description:"程序版本号"`
	groupRouters  []*GroupRouterMeta `description:"路由组"`
	genericRoutes []RouteIface       `description:"泛型路由对象"`
	groupRouter   []*GroupRouterMeta `description:"路由组对象"`
	events        []*Event           `description:"启动和关闭事件"`
	finder        Finder             `description:"路由对象查找器"`
	previousDeps  []any              `description:"在接口参数校验前执行的中间件"` // TODO Future-231126.4: 路由前后中间件
	afterDeps     []any              `description:"在接口参数校验成功后执行的中间件"`
}

func (f *FastApi) isFieldsOk() *FastApi {
	f.service.addr = net.JoinHostPort(f.host, f.port)

	if f.version == "" {
		f.version = "1.0.0"
	}

	// 初始化日志logger logger.NewLogger
	if f.service.logger == nil {
		f.service.logger = logger.NewLogger(os.Stdout, "", log.LstdFlags|log.Lshortfile)
	}

	f.pool = &sync.Pool{
		New: func() interface{} {
			c := new(Context)
			c.svc = f.service
			return c
		},
	}
	f.service.setLogger(f.service.Logger())

	return f
}

// mountUserRoutes 挂载并记录自定义路由
func (f *FastApi) mountUserRoutes() {

}

// 初始化FastApi,并完成服务依赖的建立
// FastApi启动前，必须显式的初始化FastApi的基本配置，若初始化中发生异常则panic
//  1. 记录工作地址： host:Port
//  2. 创建fiber.App createFiberApp
//  3. 挂载中间件
//  4. 按需挂载基础路由 mountBaseRoutes
//  5. 挂载自定义路由 mountUserRoutes
//  6. 安装创建swagger文档 makeSwaggerDocs
func (f *FastApi) initialize() *FastApi {
	f.service.Logger().Debug("Run at: " + core.GetMode(true))

	// 创建 fiber.App
	f.engine = createFiberApp(f.title, f.version)
	// 注册中间件
	for _, middleware := range f.afterDeps {
		f.engine.Use(middleware)
	}

	// 挂载基础路由
	if !core.BaseRoutesDisabled {
		f.IncludeRouter(&BaseGroupRouter{
			Title:   f.Title(),
			Version: f.Version(),
			Desc:    f.Description(),
			Debug:   f.IsDebug(),
		})
	}

	// 挂载自定义路由
	f.mountUserRoutes()

	// 反射路由数据，必须在路由添加完成，swagger注册之前调用
	var err error
	for _, group := range f.groupRouters {
		err = group.Init()
		if err != nil {
			panic(fmt.Errorf("swagger created failld, %v", err))
		}
	}

	// TODO：初始化finder

	// 创建 OpenApi Swagger 文档, 必须等上层注册完路由之后才能调用
	if !core.SwaggerDisabled || core.IsDebug() {
		f.createOpenApiDoc()
	}

	return f
}

// Title 应用程序名和日志文件名
func (f *FastApi) Title() string   { return f.title }
func (f *FastApi) Host() string    { return f.host }
func (f *FastApi) Port() string    { return f.port }
func (f *FastApi) Version() string { return f.version }
func (f *FastApi) IsDebug() bool   { return core.IsDebug() }
func (f *FastApi) PID() int        { return os.Getpid() }

// Description 描述信息，同时会显示在Swagger文档上
func (f *FastApi) Description() string { return f.description }

// Service 获取FastApi全局服务依赖
func (f *FastApi) Service() *Service { return f.service }

// Engine 获取fiber引擎
//
//	@return	*fiber.App fiber引擎
func (f *FastApi) Engine() *fiber.App { return f.engine }

// OnEvent 添加事件
//
//	@param	Type	事件类型，取值需为	"startup"	/	"shutdown"
//	@param	fs		func()		事件
func (f *FastApi) OnEvent(kind EventKind, fc func()) *FastApi {
	switch kind {
	case startupEvent:
		f.events = append(f.events, &Event{
			Type: startupEvent,
			Fc:   fc,
		})
	case shutdownEvent:
		f.events = append(f.events, &Event{
			Type: shutdownEvent,
			Fc:   fc,
		})
	default:
	}
	return f
}

// OnStartup  添加启动事件
//
//	@param	fs	func()	事件
func (f *FastApi) OnStartup(fc func()) *FastApi {
	f.events = append(f.events, &Event{
		Type: startupEvent,
		Fc:   fc,
	})

	return f
}

// OnShutdown 添加关闭事件
//
//	@param	fs	func()	事件
func (f *FastApi) OnShutdown(fc func()) *FastApi {
	f.events = append(f.events, &Event{
		Type: shutdownEvent,
		Fc:   fc,
	})

	return f
}

// SetUserSVC 设置一个自定义服务依赖
//
//	@param	service	UserService	服务依赖
func (f *FastApi) SetUserSVC(svc UserService) *FastApi {
	f.service.setUserSVC(svc)
	return f
}

// SetLogger 替换日志句柄，此操作必须在run之前进行
//
//	@param	logger	logger.Iface	日志句柄
func (f *FastApi) SetLogger(logger logger.Iface) *FastApi {
	f.service.setLogger(logger)
	return f
}

// SetDescription 设置APP的详细描述信息
//
//	@param	Description	string	详细描述信息
func (f *FastApi) SetDescription(description string) *FastApi {
	f.description = description
	return f
}

// IncludeRouter 注册一个路由组
//
//	@param	router	*Router	路由组
func (f *FastApi) IncludeRouter(router GroupRouter) *FastApi {
	f.groupRouters = append(f.groupRouters, NewGroupRouteMeta(router))
	return f
}

// Use 添加中间件
func (f *FastApi) Use(middleware ...any) *FastApi {
	f.afterDeps = append(f.afterDeps, middleware...)
	return f
}

// AddCronjob 添加定时任务(循环调度任务)
// 此任务会在各种初始化及启动事件全部执行完成之后触发
func (f *FastApi) AddCronjob(jobs ...cronjob.CronJob) *FastApi {
	f.service.scheduler.Add(jobs...)
	return f
}

// ActivateHotSwitch 创建一个热开关，监听信号量30，用来改变程序调试开关状态
func (f *FastApi) ActivateHotSwitch() *FastApi {
	swt := make(chan os.Signal, 1)
	signal.Notify(swt, syscall.Signal(core.HotSwitchSigint))

	go func() {
		for range swt {
			if f.IsDebug() {
				resetRunMode(false)
			} else {
				resetRunMode(true)
			}
			f.service.Logger().Debug("Hot-switch received, convert to:", core.GetMode())
		}
	}()

	return f
}

// AcquireCtx 申请一个 Context 并初始化
func (f *FastApi) AcquireCtx(fctx *fiber.Ctx) *Context {
	c := f.pool.Get().(*Context)
	// 初始化各种参数
	c.ec = fctx
	c.routeCtx, c.routeCancel = context.WithCancel(f.service.ctx) // 为每一个路由创建一个独立的ctx
	c.RequestBody = int64(1)                                      // 初始化为1，避免访问错误
	c.PathFields = map[string]string{}
	c.QueryFields = map[string]string{}

	return c
}

// ReleaseCtx 释放并归还 Context
func (f *FastApi) ReleaseCtx(ctx *Context) {
	ctx.ec = nil
	ctx.route = nil
	ctx.routeCtx = nil
	ctx.routeCancel = nil
	ctx.response = nil // 释放内存

	ctx.RequestBody = int64(1)
	ctx.PathFields = nil
	ctx.QueryFields = nil

	f.pool.Put(ctx)
}

// ReplaceErrorHandler 替换fiber错误处理方法，即 请求错误处理方法
func (f *FastApi) ReplaceErrorHandler(fc fiber.ErrorHandler) *FastApi {
	fiberErrorHandler = fc
	return f
}

// ReplaceStackTraceHandler 替换错误堆栈处理函数，即 recover 方法
func (f *FastApi) ReplaceStackTraceHandler(fc StackTraceHandlerFunc) *FastApi {
	recoverHandler = fc
	return f
}

// ReplaceRecover 重写全局 recover 方法
func (f *FastApi) ReplaceRecover(fc StackTraceHandlerFunc) *FastApi {
	return f.ReplaceStackTraceHandler(fc)
}

// AddResponseHeader 添加一个响应头
//
//	@param	key		string	键
//	@param	value	string	值
func (f *FastApi) AddResponseHeader(key, value string) *FastApi {
	// 首先判定是否已经存在
	for i := 0; i < len(responseHeaders); i++ {
		if responseHeaders[i].Key == key {
			responseHeaders[i].Value = value
			return f
		}
	}
	// 不存在，新建
	responseHeaders = append(responseHeaders, &ResponseHeader{
		Key:   key,
		Value: value,
	})
	return f
}

// DeleteResponseHeader 删除一个响应头
//
//	@param	key	string	键
func (f *FastApi) DeleteResponseHeader(key string) *FastApi {
	for i := 0; i < len(responseHeaders); i++ {
		if responseHeaders[i].Key == key {
			responseHeaders[i].Value = ""
		}
	}
	return f
}

// SetShutdownTimeout 修改关机前最大等待时间
//
//	@param	timeout	in	修改关机前最大等待时间,	单位秒
func (f *FastApi) SetShutdownTimeout(timeout int) *FastApi {
	core.ShutdownWithTimeout = time.Duration(timeout) * time.Second
	return f
}

// DisableBaseRoutes 禁用基础路由
func (f *FastApi) DisableBaseRoutes() *FastApi {
	core.BaseRoutesDisabled = true
	return f
}

// DisableSwagAutoCreate 禁用文档自动生成
func (f *FastApi) DisableSwagAutoCreate() *FastApi {
	core.SwaggerDisabled = true
	return f
}

// EnableMultipleProcess 开启多进程
func (f *FastApi) EnableMultipleProcess() *FastApi {
	core.MultipleProcessEnabled = true
	return f
}

// ShutdownWithTimeout 关机前最大等待时间
func (f *FastApi) ShutdownWithTimeout() time.Duration {
	return core.ShutdownWithTimeout * time.Second
}

// EnableDumpPID 启用PID存储
func (f *FastApi) EnableDumpPID() *FastApi {
	core.DumpPIDEnabled = true
	return f
}

// DumpPID 存储PID, 文件权限为0775
// 对于 Windows 其文件为当前运行目录下的pid.txt;
// 对于 类unix系统，其文件为/run/{Name}.pid, 若无读写权限则改为当前运行目录下的pid.txt;
func (f *FastApi) DumpPID() {
	var path string
	switch runtime.GOOS {
	case "darwin", "linux":
		path = fmt.Sprintf("/run/%s.pid", f.title)
	case "windows":
		path = "pid.txt"
	}

	pid := []byte(strconv.Itoa(f.PID()))
	err := os.WriteFile(path, pid, 755)
	if errors.Is(err, os.ErrPermission) {
		_ = os.WriteFile("pid.txt", pid, 0775)
	}
}

// Shutdown 平滑关闭
func (f *FastApi) Shutdown() {
	f.service.cancel() // 标记结束

	// 执行关机前事件
	for _, event := range f.events {
		if event.Type == shutdownEvent {
			event.Fc()
		}
	}

	go func() {
		err := f.Engine().Shutdown()
		if err != nil {
			fmt.Println(err.Error())
		}
	}()
	// Engine().Shutdown() 执行成功后将会直接退出进程，以下代码段仅当超时未关闭时执行到。
	// Shutdown() 不会关闭设置了 keepalive 的连接，除非设置了 ReadTimeout ，因此设置以下内容以确保关闭.
	<-time.After(core.ShutdownWithTimeout * time.Second)
	// 此处避免因logger关闭引发错误
	fmt.Println("Forced shutdown.") // 仅当超时时会到达此行
}

// Run 启动服务, 此方法会阻塞运行，因此必须放在main函数结尾
// 此方法已设置关闭事件和平滑关闭.
// 当 Interrupt 信号被触发时，首先会关闭 根Context，然后逐步执行“关机事件”，最后调用平滑关闭方法，关闭服务
// 启动前通过 SetShutdownTimeout 设置"平滑关闭异常时"的最大超时时间
func (f *FastApi) Run(host, port string) {
	helper.SetJsonEngine(jsoniter.ConfigCompatibleWithStandardLibrary)
	if !fiber.IsChild() {
		f.host = host
		f.port = port
		f.isFieldsOk().initialize().ActivateHotSwitch()

		// 执行启动前事件
		for _, event := range f.events {
			if event.Type == startupEvent {
				event.Fc()
			}
		}

		f.isStarted <- struct{}{} // 解除阻塞上层的任务
		f.service.Logger().Debug("HTTP server listening on: " + f.service.Addr())

		// 在各种初始化及启动事件执行完成之后触发
		f.service.scheduler.Run()
		close(f.isStarted)
	}

	// TODO:
	go func() {
		log.Fatal(f.engine.Listen(f.service.Addr()))
	}()

	// 关闭开关, buffered
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt)

	if core.DumpPIDEnabled {
		f.DumpPID()
	}

	<-quit // 阻塞进程，直到接收到停止信号,准备关闭程序
	f.Shutdown()
}

type Config struct {
	UserSvc               UserService           `json:"-" description:"自定义服务依赖"`
	Logger                logger.Iface          `json:"-" description:"日志"`
	ErrorHandler          fiber.ErrorHandler    `json:"-" description:"请求错误处理方法"`
	RecoverHandler        StackTraceHandlerFunc `json:"-" description:"异常处理方法"`
	Version               string                `json:"version,omitempty" description:"APP版本号"`
	Description           string                `json:"description,omitempty" description:"APP描述"`
	Title                 string                `json:"title,omitempty" description:"APP标题,也是日志文件名"`
	ShutdownTimeout       int                   `json:"shutdown_timeout,omitempty" description:"平滑关机,单位秒"`
	DisableSwagAutoCreate bool                  `json:"disable_swag_auto_create,omitempty" description:"禁用自动文档"`
	EnableMultipleProcess bool                  `json:"enable_multiple_process,omitempty" description:"开启多进程"`
	EnableDumpPID         bool                  `json:"enable_dump_pid,omitempty" description:"输出PID文件"`
	DisableBaseRoutes     bool                  `json:"disable_base_routes,omitempty" description:"禁用基础路由"`
	Debug                 bool                  `json:"debug,omitempty" description:"调试模式"`
}

// New 创建一个 FastApi 服务
func New(confs ...Config) *FastApi {
	conf := Config{
		Title:                 "FastAPI",
		Version:               "1.0.0",
		Debug:                 false,
		UserSvc:               nil,
		Description:           "FastAPI Application",
		Logger:                nil,
		ShutdownTimeout:       5,
		DisableBaseRoutes:     false,
		DisableSwagAutoCreate: false,
		EnableMultipleProcess: false,
		EnableDumpPID:         false,
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
		conf.UserSvc = confs[0].UserSvc
		conf.Logger = confs[0].Logger
		conf.ShutdownTimeout = confs[0].ShutdownTimeout
		conf.DisableBaseRoutes = confs[0].DisableBaseRoutes
		conf.DisableSwagAutoCreate = confs[0].DisableSwagAutoCreate
		conf.EnableMultipleProcess = confs[0].EnableMultipleProcess
		conf.EnableDumpPID = confs[0].EnableDumpPID
		conf.ErrorHandler = confs[0].ErrorHandler
		conf.RecoverHandler = confs[0].RecoverHandler
	}
	core.SetMode(conf.Debug)
	once.Do(func() {
		sc := &Service{userSVC: conf.UserSvc, validate: validator.New()}
		sc.ctx, sc.cancel = context.WithCancel(context.Background())
		sc.scheduler = cronjob.NewScheduler(sc.ctx, nil)

		appEngine = &FastApi{
			title:         conf.Title,
			version:       conf.Version,
			description:   conf.Title + " Micro Context",
			service:       sc,
			genericRoutes: make([]RouteIface, 1),
			isStarted:     make(chan struct{}, 1),
			afterDeps:     make([]any, 0),
			events:        make([]*Event, 0),
		}
	})

	app := appEngine
	if conf.Description != "" {
		app.SetDescription(conf.Description)
	}
	if conf.UserSvc != nil {
		app.SetUserSVC(conf.UserSvc)
	}
	if conf.Logger != nil {
		app.SetLogger(conf.Logger)
	}
	if conf.ErrorHandler != nil {
		app.ReplaceErrorHandler(conf.ErrorHandler)
	}
	if conf.RecoverHandler != nil {
		app.ReplaceRecover(conf.RecoverHandler)
	}

	if conf.ShutdownTimeout != 0 {
		app.SetShutdownTimeout(conf.ShutdownTimeout)
	}
	if conf.DisableBaseRoutes {
		app.DisableBaseRoutes()
	}

	if conf.DisableSwagAutoCreate {
		app.DisableSwagAutoCreate()
	}
	if conf.EnableMultipleProcess {
		//app.EnableMultipleProcess()
	}
	if conf.EnableDumpPID {
		app.EnableDumpPID()
	}
	return app
}

// resetRunMode 重设运行时环境
//
//	@param	md	string	开发环境
func resetRunMode(md bool) {
	core.SetMode(md)
}
