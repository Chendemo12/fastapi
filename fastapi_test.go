package fastapi

import (
	"testing"
	"time"
)

// ============================================================================

type BaseTypeRouter struct {
	BaseRouter
}

func (r *BaseTypeRouter) Prefix() string { return "/api/base-type" }

func (r *BaseTypeRouter) ReturnStringGet(c *Context) (string, error) {
	return "hello", nil
}

func (r *BaseTypeRouter) ReturnBoolGet(c *Context) (bool, error) {
	return true, nil
}

func (r *BaseTypeRouter) ReturnIntGet(c *Context) (int, error) {
	return 1, nil
}

func (r *BaseTypeRouter) ReturnUint16Get(c *Context) (uint16, error) {
	return 65535, nil
}

func (r *BaseTypeRouter) ReturnFloatGet(c *Context) (float32, error) {
	return 3.14, nil
}

// ============================================================================

type QueryParamRouter struct {
	BaseRouter
}

func (r *QueryParamRouter) Prefix() string { return "/api/query-param" }

func (r *QueryParamRouter) IntQueryParamGet(c *Context, age int) (int, error) {
	return age, nil
}

func (r *QueryParamRouter) FloatQueryParamGet(c *Context, source float64) (float64, error) {
	return source, nil
}

func (r *QueryParamRouter) ManyQueryParamGet(c *Context, age int, name string, graduate bool, source float64) (float64, error) {
	return source, nil
}

type Name struct {
	Father string `query:"father" validate:"required" description:"姓氏"` // 必选查询参数
	Name   string `query:"name" description:"姓名"`                       // 可选查询参数
}

func (r *QueryParamRouter) StructQueryParamDelete(c *Context, param *Name) (string, error) {
	return param.Father + " " + param.Name, nil
}

// ============================================================================

type RequestBodyRouter struct {
	BaseRouter
}

func (r *RequestBodyRouter) Prefix() string { return "/api/request" }

type RegisterForm struct {
	Email    string `json:"email" validate:"required" description:"邮箱"`
	Username string `json:"username" validate:"required" description:"用户名"`
	Password string `json:"password" validate:"required" description:"密码"`
	Age      int8   `json:"age" default:"18"`
	Male     bool   `json:"male" validate:"required" description:"是否是男性"`
}

func (r *RegisterForm) SchemaDesc() string { return "注册表单" }

func (r *RequestBodyRouter) RegisterPost(c *Context, location string, form *RegisterForm) (string, error) {
	return "123456789", nil
}

func (r *RequestBodyRouter) RegisterWithParamPost(c *Context, name *Name, form *RegisterForm) (string, error) {
	return "123456789", nil
}

func (r *RequestBodyRouter) StringQueryParamPatch(c *Context, name string) (string, error) {
	return name, nil
}

func (r *RequestBodyRouter) Path() map[string]string {
	return map[string]string{
		"RegisterWithParamPost": "register-with/:location",
	}
}

// ============================================================================

type ResponseModelRouter struct {
	BaseRouter
}

func (r *ResponseModelRouter) Prefix() string { return "/api/response" }

// ServerValidateErrorModel 服务器内部模型校验错误示例
type ServerValidateErrorModel struct {
	ServerName string `json:"server_name" validate:"required" description:"服务名称"`
	Version    string `json:"version" validate:"required" description:"服务版本号"`
	Links      int    `json:"links,omitempty" description:"连接数"`
}

func (s *ServerValidateErrorModel) SchemaDesc() string { return "服务器内部模型示例" }

func (r *ResponseModelRouter) ReturnSimpleStructGet(c *Context) (*ServerValidateErrorModel, error) {
	return &ServerValidateErrorModel{
		ServerName: "FastApi",
		Version:    "v0.2.0",
	}, nil
}

type Tunnel struct {
	No      int `json:"no" binding:"required"`
	BoardId int `json:"board_id" binding:"required"`
}

func (t *Tunnel) SchemaDesc() string { return "通道信息" }

type CPU struct {
	Core   int ` json:"core" description:"核心数量"`
	Thread int `json:"thread" description:"线程数量"`
}

type BoardCard struct {
	Serial    string    `json:"serial" validate:"required" description:"序列号"`
	PcieSlots int       `json:"pcie_slots"`
	Cpu       *CPU      `json:"cpu"`
	Tunnels   []*Tunnel `json:"tunnels" description:"通道信息"`
}

func (r *ResponseModelRouter) ReturnNormalStructGet(c *Context) (*BoardCard, error) {
	return &BoardCard{
		Serial:    "0x987654321",
		PcieSlots: 2,
		Tunnels: []*Tunnel{
			{
				No:      10,
				BoardId: 0x4321,
			},
			{
				No:      12,
				BoardId: 0x4323,
			},
		},
	}, nil
}

// ============================================================================

type FastApiRouter struct {
	BaseRouter
}

func (f *FastApiRouter) Prefix() string { return "/api/example" }

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

type LogonForm struct {
	Name   string `json:"name" description:"姓名" validate:"required"`
	Age    string `json:"age" description:"年龄" validate:"required,gte=50"`
	Father string `json:"father"`
	Family string `json:"family"`
}

func (s *LogonForm) SchemaDesc() string { return "简单的登录表单" }

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
		Version:     "v0.2.0",
		Description: "",
		Title:       "FastApi Example",
		Debug:       true,
	})

	//app.IncludeRouter(&FastApiRouter{})
	app.IncludeRouter(&BaseTypeRouter{}).
		IncludeRouter(&QueryParamRouter{}).
		IncludeRouter(&RequestBodyRouter{}).
		IncludeRouter(&ResponseModelRouter{})

	app.Run(svc.Conf.HTTP.Host, svc.Conf.HTTP.Port) // 阻塞运行
}
