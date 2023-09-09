package fastapi

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"testing"
	"time"

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

func NEWCtx() *ServiceContext {
	conf := &Configuration{}
	conf.HTTP.Host = "0.0.0.0"
	conf.HTTP.Port = "8099"
	return &ServiceContext{Conf: conf, Logger: logger.NewDefaultLogger()}
}

func TestContext_UserSVC(t *testing.T) {
	ctx := &Context{svc: &Service{}}
	ctx.svc.setUserSVC(NEWCtx())

	conf, ok := ctx.UserSVC().Config().(*Configuration)
	if ok {
		t.Logf("http host: %s", conf.HTTP.Host)
		t.Logf("http port: %s", conf.HTTP.Port)
	} else {
		t.Errorf("conv to config failed")
	}
}

func TestNEW(t *testing.T) {
	svc := NEWCtx()
	app := NEW("FastApi Example", "1.0.0", true, svc)
	app.SetDescription("一个简单的FastApi应用程序,在启动app之前首先需要创建并替换ServiceContext,最后调用Run来运行程序").
		SetShutdownTimeout(5)

	t.Logf("FastApi app created.")
}

// Clock 定时任务
type Clock struct {
	cronjob.Job
}

func (c *Clock) String() string          { return "Clock" }
func (c *Clock) Interval() time.Duration { return time.Second * 1 }
func (c *Clock) Do(_ context.Context) error {
	fmt.Println("current second:", time.Now().Second())
	return nil
}

func TestFastApi_AddCronjob(t *testing.T) {
	svc := NEWCtx()
	app := NEW("FastApi Example", "1.0.0", true, svc)
	app.AddCronjob(&Clock{})

	go func() {
		time.Sleep(10 * time.Second)
		app.Shutdown()
	}()

	app.Run(svc.Conf.HTTP.Host, svc.Conf.HTTP.Port) // 阻塞运行
}

func TestFastApi_DumpPID(t *testing.T) {
	svc := NEWCtx()
	svc.Conf.HTTP.Port = "8089"
	app := NEW("FastApi Example", "1.0.0", true, svc)
	app.EnableDumpPID()

	go func() {
		time.Sleep(2 * time.Second)

		_bytes, err := os.ReadFile("pid.txt")
		if errors.Is(err, os.ErrNotExist) {
			t.Errorf("dump pid failed: %s", err.Error())
		} else {
			t.Logf("current pid is: %s", string(_bytes))
		}
		app.Shutdown()
	}()

	app.Run(svc.Conf.HTTP.Host, svc.Conf.HTTP.Port) // 阻塞运行
}

func TestFastApi_Description(t *testing.T) {
	svc := NEWCtx()
	app := NEW("FastApi Example", "1.0.0", true, svc)
	app.SetDescription("一个简单的FastApi应用程序,在启动app之前首先需要创建并替换ServiceContext,最后调用Run来运行程序")

	s := app.Description()
	if s == "" {
		t.Errorf("get description failed: %s", s)
	} else {
		t.Logf("get description: %s", s)
	}
}

// ReturnLinkInfo 反向链路参数
type ReturnLinkInfo struct {
	BaseModel
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
	BaseModel
	ModType     string  `json:"mod_type"`
	FecRate     string  `json:"fec_rate"`
	FecType     string  `json:"fec_type"`
	IfFrequency int     `json:"if_frequency" description:"中频频点"`
	SymbolRate  int     `json:"symbol_rate" description:"符号速率"`
	FreqOffset  int     `json:"freq_offset"`
	TunnelNo    int     `json:"tunnel_no" validate:"required, oneof=1 0"`
	Power       float32 `json:"power" validate:"required, gte=-100, lte=70"`
	Reset       bool    `json:"reset"`
}

func (m ForwardLinkInfo) SchemaDesc() string { return "前向链路参数" }

func setReturnLinks(s *Context) *Response {
	p := make([]ReturnLinkInfo, 0)
	if err := s.ShouldBindJSON(&p); err != nil {
		return err
	}
	time.Sleep(time.Millisecond * 200) // 休眠200ms,模拟设置硬件时长
	return s.OKResponse(p[0].ForwardLink)
}

type ComplexModel struct {
	BaseModel
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

func getComplexModel(s *Context) *Response {
	return s.OKResponse(&ComplexModel{
		Name:    "lee",
		Steps:   nil,
		Actions: nil,
	})
}

// ServerValidateErrorModel 服务器内部模型校验错误示例
type ServerValidateErrorModel struct {
	BaseModel
	ServerName string `json:"server_name" description:"服务名称"`
	Version    string `json:"version" description:"服务版本号"`
}

func (s *ServerValidateErrorModel) SchemaDesc() string { return "服务器内部模型示例" }

func returnServerValidateError(c *Context) *Response {
	//app := c.App()
	//return c.OKResponse(ServerValidateErrorModel{
	//	ServerName: app.Title(),
	//	Version:    app.Version(),
	//})
	return c.OKResponse(12)
}

type LogonForm struct {
	BaseModel
	Name   string `json:"name" description:"姓名" validate:"required"`
	Age    string `json:"age" description:"年龄" validate:"required,gte=50"`
	Father string `json:"father"`
	Family string `json:"family"`
}

func (s *LogonForm) SchemaDesc() string { return "简单的登录表单" }

func getLogonForm(c *Context) *Response {
	form := LogonForm{
		Name:   c.PathFields["name"],
		Age:    c.PathFields["age"],
		Father: c.QueryFields["father"],
		Family: c.QueryFields["family"],
	}

	return c.OKResponse(form)
}

type QueryFieldExample struct {
	QueryModel
	Father string `json:"father"`
	Family string `json:"family"`
	Size   int    `json:"size"`
}

func TestFastApi_IncludeRouter(t *testing.T) {
	svc := NEWCtx()
	app := NEW("FastApi Example", "1.0.0", true, svc)

	r := APIRouter("/example", []string{"Example"})
	{ // 基本示例
		r.GET("/complex-model", &ComplexModel{}, "复杂模型的文档生成示例", getComplexModel)

		r.GET("/logon/:name/:age?", &LogonForm{}, "带路径参数和查询参数的示例", getLogonForm).
			SetQ(QueryFieldExample{})

		r.POST("/tunnel/work", &ReturnLinkInfo{}, &ForwardLinkInfo{}, "带请求体和响应体验证的POST请求", setReturnLinks)

		r.DELETE("/tunnel/work", nil, "缺省返回值示例", func(s *Context) *Response {
			return s.OKResponse("anything is allowed")
		})

		r.POST("/base-model", Int, Bytes, "基本数据类型的示例", func(s *Context) *Response {
			return s.StreamResponse(bytes.NewReader([]byte("hello world")))
		})
	}

	app.IncludeRouter(r)
	app.Run(svc.Conf.HTTP.Host, svc.Conf.HTTP.Port) // 阻塞运行
}

func TestFastApi_IncludeRouter_Validate(t *testing.T) {
	svc := NEWCtx()
	app := NEW("FastApi Example", "1.0.0", true, svc)

	r := APIRouter("/example", []string{"Validator"})
	{ // 校验示例
		r.GET(
			"/server-error", &ServerValidateErrorModel{}, "服务器内部模型校验错误示例", returnServerValidateError,
		)
		r.GET(
			"/bool-error", Bool, "接口应返回Bool却返回了字符串的示例", func(s *Context) *Response {
				return s.StringResponse("无法通过返回值校验")
			},
		)
		r.GET(
			"/logon-error", &LogonForm{}, "接口应返回LogonForm却返回了Bool的示例", func(s *Context) *Response {
				return s.OKResponse(true)
			},
		)
	}

	app.IncludeRouter(r)
	app.Run(svc.Conf.HTTP.Host, svc.Conf.HTTP.Port) // 阻塞运行
}

type Tunnel struct {
	BaseModel
	No      int `json:"no" binding:"required"`
	BoardId int `json:"board_id" binding:"required"`
}

func (t *Tunnel) SchemaDesc() string { return "通道信息" }

func getTunnels(c *Context) *Response {
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

func intToInts(s *Context) *Response {
	i := 0
	err := s.BodyParser(&i)
	if err != nil {
		return err
	}
	return s.OKResponse([]int{i})
}

func TestFastApi_IncludeRouter_Array(t *testing.T) {
	svc := NEWCtx()
	app := NEW("FastApi Example", "1.0.0", true, svc)

	r := APIRouter("/example", []string{"Array"})
	{
		r.POST("/base/int", Uint8, Ints, "将提交的数字转换成数组并返回示例", intToInts)
		r.POST("/tunnels", Ints, List(&Tunnel{}), "返回值是数组类型的示例", getTunnels)
	}

	app.IncludeRouter(r)
	app.Run(svc.Conf.HTTP.Host, svc.Conf.HTTP.Port) // 阻塞运行
}

func routeCtxCancel(s *Context) *Response {
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

func TestFastApi_IncludeRouter_Context(t *testing.T) {
	svc := NEWCtx()
	app := NEW("FastApi Example", "1.0.0", true, svc)

	r := APIRouter("/text", []string{"Text"})
	{
		r.GET("/cancel", Int, "当路由执行完毕退出时关闭context", routeCtxCancel)
	}

	app.IncludeRouter(r)
	app.Run(svc.Conf.HTTP.Host, svc.Conf.HTTP.Port) // 阻塞运行
}

func TestFastApi_OnEvent(t *testing.T) {
	svc := NEWCtx()
	app := NEW("FastApi Example", "1.0.0", true, svc)

	app.OnEvent("startup", func() { app.Service().Logger().Info("current pid: ", app.PID()) })
	app.OnEvent("startup", func() { app.Service().Logger().Info("startup event: 1") })
	app.OnEvent("shutdown", func() { app.Service().Logger().Info("shutdown event: 1") })
	app.OnEvent("shutdown", func() { app.Service().Logger().Info("shutdown event: 2") })

	app.Run(svc.Conf.HTTP.Host, svc.Conf.HTTP.Port) // 阻塞运行
}

type IPModel struct {
	BaseModel
	IP     string `json:"ip" description:"IPv4地址"`
	Detail struct {
		IPv4     string `json:"IPv4" description:"IPv4地址"`
		IPv4Full string `json:"IPv4_full" description:"带端口的IPv4地址"`
		Ipv6     string `json:"IPv6" description:"IPv6地址"`
	} `json:"detail" description:"详细信息"`
}

func (m IPModel) SchemaDesc() string { return "IP信息" }

func getAddress(c *Context) *Response {
	info := &IPModel{}
	info.Detail.IPv4Full = c.EngineCtx().Context().RemoteAddr().String()

	fiberIP := c.EngineCtx().IP()
	headerIP := c.EngineCtx().Get("X-Forwarded-For")

	if fiberIP == headerIP || headerIP == "" {
		info.IP = fiberIP
		info.Detail.IPv4 = fiberIP
	} else {
		info.IP = headerIP
		info.Detail.IPv4 = headerIP
	}

	c.Logger().Debug("fiber think: ", fiberIP, " X-Forwarded-For: ", headerIP)

	return c.OKResponse(info)
}

type EnosDataItem struct {
	Items []struct {
		AssetId   string  `json:"assetId"`
		Localtime string  `json:"localtime,omitempty"`
		PointId   int     `json:"pointId"`
		Timestamp float64 `json:"timestamp"`
		Quality   int     `json:"quality,omitempty"`
	} `json:"items"`
}

type EnosData struct {
	BaseModel
	Data   *EnosDataItem `json:"data"`
	Kind   string        `json:"kind"`
	Msg    string        `json:"msg,omitempty"`
	Submsg string        `json:"submsg,omitempty"`
	Code   int           `json:"code"`
}

type DomainRecord struct {
	BaseModel
	Timestamp int64 `json:"timestamp" description:"时间戳"`
	IP        struct {
		Record *IPModel `json:"record" description:"解析记录"`
	} `json:"ip"`
	Addresses []struct {
		Host string `json:"host"`
		Port string `json:"port"`
	} `json:"addresses" description:"主机地址"`
}

func pushEnOSData(c *Context) *Response {
	data := &EnosData{}
	err := c.ShouldBindJSON(data)
	if err != nil {
		return err
	}
	c.Logger().Info("receive enos data: ", data.Kind)

	return c.OKResponse(data)
}

func getDomainRecord(c *Context) *Response {
	r := &DomainRecord{
		Timestamp: 0,
		Addresses: []struct {
			Host string `json:"host"`
			Port string `json:"port"`
		}{
			{
				"127.0.0.1",
				"8090",
			},
		},
	}
	r.IP.Record = &IPModel{
		IP: "",
		Detail: struct {
			IPv4     string `json:"IPv4" description:"IPv4地址"`
			IPv4Full string `json:"IPv4_full" description:"带端口的IPv4地址"`
			Ipv6     string `json:"IPv6" description:"IPv6地址"`
		}(struct {
			IPv4     string
			IPv4Full string
			Ipv6     string
		}{
			"10.64.73.25",
			"10.64.73.25:8000",
			"0:0:0:0:0",
		}),
	}
	return c.OKResponse(r)
}

func TestFastApi_Run(t *testing.T) {
	svc := NEWCtx()
	app := New(Config{
		Title:             "FastAPI Example",
		Version:           "v1.2.0",
		Debug:             true,
		UserSvc:           svc,
		Description:       "一个简单的FastApi应用程序,在启动app之前首先需要创建并替换ServiceContext,最后调用Run来运行程序",
		Logger:            svc.Logger,
		ShutdownTimeout:   5,
		DisableBaseRoutes: true,
	})

	app.Get("/example", getAddress)

	app.Get("/example/ip", getAddress, Opt{
		Summary:       "返回当前请求的来源IP地址",
		ResponseModel: &IPModel{},
	})

	app.Get("/example/domain", getDomainRecord, Opt{
		Summary:       "获取地址解析记录",
		ResponseModel: &DomainRecord{},
	})

	app.Post("/example/pusher", pushEnOSData, Opt{
		RequestModel: &EnosData{}, ResponseModel: List(&EnosData{}),
	})

	app.Run(svc.Conf.HTTP.Host, svc.Conf.HTTP.Port) // 阻塞运行
}
