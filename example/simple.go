package main

import (
	"bytes"
	"context"
	"fmt"
	"time"

	"github.com/Chendemo12/fastapi"
	"github.com/Chendemo12/fastapi-tool/cronjob"
	"github.com/Chendemo12/fastapi-tool/logger"
)

// Configuration 配置文件类
type Configuration struct {
	HTTP struct {
		Host string `json:"host" yaml:"host"` // API host
		Port string `json:"port" yaml:"port"` // API port
	}
}

// ServiceContext 全局服务依赖
type ServiceContext struct {
	Conf   *Configuration
	Logger logger.Iface
}

func (c *ServiceContext) Config() any { return c.Conf }

// -------------------------------- 模型路由 --------------------------------

// ServerValidateErrorModel 服务器内部模型校验错误示例
type ServerValidateErrorModel struct {
	fastapi.BaseModel
	ServerName string `json:"server_name" description:"服务名称"`
	Version    string `json:"version" description:"服务版本号"`
}

func (s *ServerValidateErrorModel) SchemaDesc() string { return "服务器内部模型示例" }

func serverValidateErrorExample(c *fastapi.Context) *fastapi.Response {
	//app := c.App()
	//return c.OKResponse(ServerValidateErrorModel{
	//	ServerName: app.Title(),
	//	Version:    app.Version(),
	//})
	return c.OKResponse(12)
}

type ComplexModel struct {
	fastapi.BaseModel
	Name    string    `json:"name" validate:"required,oneof=lee wang fan"`
	Steps   []Step    `json:"steps"`
	Actions []*Action `json:"actions"`
}

func (s *ComplexModel) SchemaDesc() string { return "一个复杂的模型" }

type Step struct {
	Click string `json:"click"`
}

type Action struct {
	OneStep  Step     `json:"one_step"`
	TwoSteps []Step   `json:"two_steps"`
	Next     []string `json:"next"`
}

func getComplexModelExample(s *fastapi.Context) *fastapi.Response {
	return s.OKResponse(&ComplexModel{
		Name:    "lee",
		Steps:   nil,
		Actions: nil,
	})
}

type LogonForm struct {
	fastapi.BaseModel
	Name   string `json:"name" description:"姓名" validate:"required"`
	Age    string `json:"age" description:"年龄" validate:"required,gte=50"`
	Father string `json:"father"`
	Family string `json:"family"`
}

func (s *LogonForm) SchemaDesc() string { return "简单的登录表单" }

type QueryFieldExample struct {
	fastapi.QueryModel
	Father string `json:"father"`
	Family string `json:"family"`
	Size   int    `json:"size"`
}

func getLogonForm(s *fastapi.Context) *fastapi.Response {
	form := LogonForm{
		Name:   s.PathFields["name"],
		Age:    s.PathFields["age"],
		Father: s.QueryFields["father"],
		Family: s.QueryFields["family"],
	}

	return s.OKResponse(form)
}

// ReturnLinkInfo 反向链路参数，仅当网管代理配置的参数与此匹配时才转发小站消息到NCC
type ReturnLinkInfo struct {
	fastapi.BaseModel
	ModType     string            `json:"mod_type"`
	FecRate     string            `json:"fec_rate"`
	ForwardLink []ForwardLinkInfo `json:"forward_link" description:"前向链路"`
	IfFrequency int               `json:"if_frequency" description:"中频频点"`
	SymbolRate  int               `json:"symbol_rate" description:"符号速率"`
}

func (m ReturnLinkInfo) SchemaDesc() string {
	return "反向链路参数，仅当网管代理配置的参数与此匹配时才转发小站消息到NCC"
}

type ForwardLinkInfo struct {
	fastapi.BaseModel
	ModType     string      `json:"mod_type"`
	FecRate     string      `json:"fec_rate"`
	FecType     string      `json:"fec_type"`
	PositionRcu PositionGeo `json:"position_rcu"`
	PositionCu  PositionGeo `json:"position_cu"`
	PositionSat PositionGeo `json:"position_sat"`
	IfFrequency int         `json:"if_frequency" description:"中频频点"`
	SymbolRate  int         `json:"symbol_rate" description:"符号速率"`
	FreqOffset  int         `json:"freq_offset"`
	TunnelNo    int         `json:"tunnel_no" validate:"required, oneof=1 0"`
	Power       float32     `json:"power" validate:"required, gte=-100, lte=70"`
	Reset       bool        `json:"reset"`
}

type PositionGeo struct {
	fastapi.BaseModel
	Longi float32 `json:"longi" binding:"required"`
	Lati  float32 `json:"lati" binding:"required"`
}

func (m ForwardLinkInfo) SchemaDesc() string { return "前向链路参数" }

func setReturnLinks(s *fastapi.Context) *fastapi.Response {
	p := make([]ReturnLinkInfo, 0)
	if err := s.ShouldBindJSON(&p); err != nil {
		return err
	}
	time.Sleep(time.Millisecond * 200) // 休眠200ms,模拟设置硬件时长
	return s.OKResponse(p[0].ForwardLink)
}

type Tunnel struct {
	fastapi.BaseModel
	No      int `json:"no" binding:"required"`
	BoardId int `json:"board_id" binding:"required"`
}

func (t *Tunnel) SchemaDesc() string { return "通道信息" }

func getTunnels(c *fastapi.Context) *fastapi.Response {
	return c.OKResponse([]Tunnel{
		{
			No:      10,
			BoardId: 1,
		},
		{
			No:      11,
			BoardId: 1,
		},
		{
			No:      20,
			BoardId: 2,
		},
	})
}

func intToInts(s *fastapi.Context) *fastapi.Response {
	i := 0
	err := s.BodyParser(&i)
	if err != nil {
		return err
	}
	return s.OKResponse([]int{i})
}

func routeCtxCancel(s *fastapi.Context) *fastapi.Response {
	cl := s.Logger() // 当路由执行完毕退出时, ctx将被释放
	ctx := s.DisposableCtx()

	go func() {
		for {
			select {
			case <-ctx.Done():
				cl.Info("route canceled.")
				return
			case <-time.Tick(time.Millisecond * 400):
				cl.Info("route not cancel.")
			}
		}
	}()
	time.Sleep(time.Second * 2)
	return s.OKResponse(12)
}

func makeRouter() *fastapi.Router {
	exampleRouter := fastapi.APIRouter("/example", []string{"Example"})
	{ // 基本示例
		exampleRouter.GET(
			"/complex-model", &ComplexModel{}, "复杂模型的文档生成示例", getComplexModelExample,
		)

		exampleRouter.GET(
			"/logon/:name/:age?", &LogonForm{}, "带路径参数和查询参数的示例", getLogonForm,
		).SetQ(QueryFieldExample{})

		exampleRouter.POST(
			"/tunnel/work", &ReturnLinkInfo{}, &ForwardLinkInfo{}, "带请求体和响应体验证的POST请求", setReturnLinks,
		)

		exampleRouter.DELETE(
			"/tunnel/work", nil, "缺省返回值示例",
			func(s *fastapi.Context) *fastapi.Response {
				return s.OKResponse("anything is allowed")
			})

		exampleRouter.POST(
			"/base-model", fastapi.Int, fastapi.Bytes, "基本数据类型的示例",
			func(s *fastapi.Context) *fastapi.Response {
				return s.StreamResponse(bytes.NewReader([]byte("hello world")))
			})

		exampleRouter.Websocket("/ws", nil)
	}

	validateRouter := fastapi.APIRouter("/example", []string{"Validator"})
	{ // 校验示例
		validateRouter.GET(
			"/server-error", &ServerValidateErrorModel{}, "服务器内部模型校验错误示例", serverValidateErrorExample,
		)
		validateRouter.GET(
			"/bool-error", fastapi.Bool, "接口应返回Bool却返回了字符串的示例", func(s *fastapi.Context) *fastapi.Response {
				return s.StringResponse("无法通过返回值校验")
			},
		)
		validateRouter.GET(
			"/logon-error", &LogonForm{}, "接口应返回LogonForm却返回了Bool的示例", func(s *fastapi.Context) *fastapi.Response {
				return s.OKResponse(true)
			},
		)
	}

	arrayRouter := fastapi.APIRouter("/example", []string{"Array"})
	{
		arrayRouter.POST("/base/int", fastapi.Uint8, fastapi.Ints, "将提交的数字转换成数组并返回示例", intToInts)
		arrayRouter.POST("/tunnels", fastapi.Ints, fastapi.List(&Tunnel{}), "返回值是数组类型的示例", getTunnels)
	}

	textRouter := fastapi.APIRouter("/text", []string{"Text"})
	{
		textRouter.GET("/cancel", fastapi.Int, "当路由执行完毕退出时关闭context", routeCtxCancel)
	}

	router := fastapi.APIRouter("/api", []string{})
	router.IncludeRouter(exampleRouter).
		IncludeRouter(arrayRouter).
		IncludeRouter(validateRouter).
		IncludeRouter(textRouter)

	return router
}

// Clock 定时任务
type Clock struct {
	cronjob.Job
}

func (c *Clock) String() string          { return "Clock" }
func (c *Clock) Interval() time.Duration { return time.Second * 5 }

func (c *Clock) Do(ctx context.Context) error {
	fmt.Println("current second:", time.Now().Second())
	return nil
}

func ExampleFastApi_App() {
	conf := &Configuration{}
	conf.HTTP.Host = "0.0.0.0"
	conf.HTTP.Port = "8088"
	svc := &ServiceContext{Conf: conf, Logger: logger.NewDefaultLogger()}

	app := fastapi.New("FastApi Example", "1.0.0", true, svc)
	app.DisableMultipleProcess().
		EnableDumpPID().
		DisableRequestValidate().
		SetLogger(svc.Logger).
		SetDescription("一个简单的FastApi应用程序,在启动app之前首先需要创建并替换ServiceContext,最后调用Run来运行程序").
		SetShutdownTimeout(5).
		IncludeRouter(makeRouter()).
		AddCronjob(&Clock{})

	app.OnEvent("startup", func() { app.Service().Logger().Info("current pid: ", app.PID()) })
	app.OnEvent("startup", func() { app.Service().Logger().Info("startup event: 1") })
	app.OnEvent("startup", func() { app.Service().Logger().Info("startup event: 2") })
	app.OnEvent("shutdown", func() { app.Service().Logger().Info("shutdown event: 1") })
	app.OnEvent("shutdown", func() { app.Service().Logger().Info("shutdown event: 2") })

	app.Run(conf.HTTP.Host, conf.HTTP.Port) // 阻塞运行
}

// -----------------------------------------------------------------

func main() {
	ExampleFastApi_App()
}
