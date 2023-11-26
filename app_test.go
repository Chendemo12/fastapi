package fastapi

import (
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

func NewCtx() *ServiceContext {
	conf := &Configuration{}
	conf.HTTP.Host = "0.0.0.0"
	conf.HTTP.Port = "8099"

	return &ServiceContext{Conf: conf, Logger: logger.NewDefaultLogger()}
}

func TestContext_UserSVC(t *testing.T) {
	ctx := &Context{svc: &Service{}}
	ctx.svc.setUserSVC(NewCtx())

	conf, ok := ctx.UserSVC().Config().(*Configuration)
	if ok {
		t.Logf("http host: %s", conf.HTTP.Host)
		t.Logf("http port: %s", conf.HTTP.Port)
	} else {
		t.Errorf("conv to config failed")
	}
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
	svc := NewCtx()
	app := New(Config{
		UserSvc:     svc,
		Version:     "1.0.0",
		Description: "",
		Title:       "FastApi Example",
		Debug:       true,
	})
	app.AddCronjob(&Clock{})

	go func() {
		time.Sleep(10 * time.Second)
		app.Shutdown()
	}()

	app.Run(svc.Conf.HTTP.Host, svc.Conf.HTTP.Port) // 阻塞运行
}

func TestFastApi_DumpPID(t *testing.T) {
	svc := NewCtx()
	svc.Conf.HTTP.Port = "8089"
	app := New(Config{
		UserSvc:     svc,
		Version:     "1.0.0",
		Description: "",
		Title:       "FastApi Example",
		Debug:       true,
	})
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
	svc := NewCtx()
	app := New(Config{
		UserSvc:     svc,
		Version:     "1.0.0",
		Description: "",
		Title:       "FastApi Example",
		Debug:       true,
	})
	app.SetDescription("一个简单的FastApi应用程序,在启动app之前首先需要创建并替换ServiceContext,最后调用Run来运行程序")

	s := app.Description()
	if s == "" {
		t.Errorf("get description failed: %s", s)
	} else {
		t.Logf("get description: %s", s)
	}
}

func TestFastApi_OnEvent(t *testing.T) {
	svc := NewCtx()
	app := New(Config{
		UserSvc:     svc,
		Version:     "1.0.0",
		Description: "",
		Title:       "FastApi Example",
		Debug:       true,
	})

	app.OnEvent("startup", func() { app.Service().Logger().Info("current pid: ", app.PID()) })
	app.OnEvent("startup", func() { app.Service().Logger().Info("startup event: 1") })
	app.OnEvent("shutdown", func() { app.Service().Logger().Info("shutdown event: 1") })
	app.OnEvent("shutdown", func() { app.Service().Logger().Info("shutdown event: 2") })

	app.Run(svc.Conf.HTTP.Host, svc.Conf.HTTP.Port) // 阻塞运行
}

type FastApiRouter struct {
	BaseRouter
}

// ReturnLinkInfo 反向链路参数
type ReturnLinkInfo struct {
	BaseModel
	ModType     string             `json:"mod_type"`
	FecRate     string             `json:"fec_rate"`
	ForwardLink []*ForwardLinkInfo `json:"forward_link" description:"前向链路"`
	IfFrequency int                `json:"if_frequency" description:"中频频点"`
	SymbolRate  int                `json:"symbol_rate" description:"符号速率"`
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

func (m *ForwardLinkInfo) SchemaDesc() string { return "前向链路参数" }

func (f *FastApiRouter) SetReturnLinkPost(c *Context, req []*ReturnLinkInfo) (*ForwardLinkInfo, error) {
	time.Sleep(time.Millisecond * 200) // 休眠200ms,模拟设置硬件时长

	return req[0].ForwardLink[0], nil
}

// ServerValidateErrorModel 服务器内部模型校验错误示例
type ServerValidateErrorModel struct {
	BaseModel
	ServerName string `json:"server_name" description:"服务名称"`
	Version    string `json:"version" description:"服务版本号"`
}

func (s *ServerValidateErrorModel) SchemaDesc() string { return "服务器内部模型示例" }

func (f *FastApiRouter) GetServerValidateError(c *Context) (*ServerValidateErrorModel, error) {

	return &ServerValidateErrorModel{
		ServerName: "FastApi",
		Version:    "0.2.0",
	}, nil
}

type LogonForm struct {
	BaseModel
	Name   string `json:"name" description:"姓名" validate:"required"`
	Age    string `json:"age" description:"年龄" validate:"required,gte=50"`
	Father string `json:"father"`
	Family string `json:"family"`
}

func (s *LogonForm) SchemaDesc() string { return "简单的登录表单" }

func (f *FastApiRouter) GetLogon(c *Context, father string, family string) (*LogonForm, error) {
	form := &LogonForm{
		Name:   c.Query("name", "undefined"),
		Age:    c.PathField("age", "18"),
		Father: c.QueryFields["father"],
		Family: c.QueryFields["family"],
	}

	return form, nil
}

type Tunnel struct {
	BaseModel
	No      int `json:"no" binding:"required"`
	BoardId int `json:"board_id" binding:"required"`
}

func (t *Tunnel) SchemaDesc() string { return "通道信息" }

func (f *FastApiRouter) GetTunnels(c *Context, boardId int) ([]Tunnel, error) {
	return []Tunnel{
		{
			No:      boardId + 10,
			BoardId: boardId,
		},
		{
			No:      boardId + 12,
			BoardId: boardId,
		},
		{
			No:      boardId + 14,
			BoardId: boardId,
		},
	}, nil
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

func (f *FastApiRouter) GetAddress(c *Context) (*IPModel, error) {
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

	return info, nil
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

func (f *FastApiRouter) PostPushEnOSData(c *Context, req *EnosData) (int, error) {
	c.Logger().Info("receive enos data: ", req.Kind)

	return 200, nil
}

func (f *FastApiRouter) GetDomainRecord(c *Context) (*DomainRecord, error) {
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
	return r, nil
}

func TestNew(t *testing.T) {
	svc := NewCtx()
	app := New(Config{
		UserSvc:     svc,
		Version:     "1.0.0",
		Description: "",
		Title:       "FastApi Example",
		Debug:       true,
	})

	app.IncludeRouter(&FastApiRouter{})

	app.Run(svc.Conf.HTTP.Host, svc.Conf.HTTP.Port) // 阻塞运行
}
