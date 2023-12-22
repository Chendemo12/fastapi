package test

import (
	"errors"
	"fmt"
	"github.com/Chendemo12/fastapi"
	"github.com/Chendemo12/fastapi/middleware/fiberWrapper"
	"github.com/Chendemo12/fastapi/middleware/routers"
	"testing"
	"time"
)

// ============================================================================

type BaseTypeRouter struct {
	fastapi.BaseRouter
	Counter int
}

func (r *BaseTypeRouter) Prefix() string { return "/api/base-type" }

func (r *BaseTypeRouter) ReturnStringGet(c *fastapi.Context) (string, error) {
	return "hello", nil
}

func (r *BaseTypeRouter) ReturnBoolGet(c *fastapi.Context) (bool, error) {
	return true, nil
}

func (r *BaseTypeRouter) ReturnIntGet(c *fastapi.Context) (int, error) {
	r.Counter++
	return r.Counter, nil
}

func (r *BaseTypeRouter) ReturnUint16Get(c *fastapi.Context) (uint16, error) {
	return 65535, nil
}

func (r *BaseTypeRouter) ReturnFloatGet(c *fastapi.Context) (float32, error) {
	return 3.14, nil
}

func (r *BaseTypeRouter) ReturnErrorGet(c *fastapi.Context) (string, error) {
	return "", errors.New("return error, default StatusCode: 500")
}

func (r *BaseTypeRouter) ReturnErrorBadGet(c *fastapi.Context) (string, error) {
	c.Status(400)
	return "", errors.New("return error, custom StatusCode: 400")
}

// ============================================================================

type QueryParamRouter struct {
	fastapi.BaseRouter
}

func (r *QueryParamRouter) Prefix() string { return "/api/query-param" }

func (r *QueryParamRouter) Path() map[string]string {
	return map[string]string{
		"IntQueryParamGet": "int-query/:param",
	}
}

func (r *QueryParamRouter) InParamsName() map[string]map[int]string {
	return map[string]map[int]string{
		"ManyGet": {
			2: "age",
			3: "name",
			4: "graduate",
			5: "source",
		},
		"TimeGet": {
			2: "day",
		},
	}
}

func (r *QueryParamRouter) IntGet(c *fastapi.Context, age int) (int, error) {
	return age, nil
}

func (r *QueryParamRouter) FloatGet(c *fastapi.Context, source float64) (float64, error) {
	return source, nil
}

func (r *QueryParamRouter) ManyGet(c *fastapi.Context, age int, name string, graduate bool, source float64) (float64, error) {
	return source, nil
}

func (r *QueryParamRouter) StructDelete(c *fastapi.Context, param *Name) (string, error) {
	return param.Father + " " + param.Name, nil
}

func (r *QueryParamRouter) TimeGet(c *fastapi.Context, day time.Time, param *DateTime) (*DateTime, error) {
	return &DateTime{
		Name: &Name{
			Father: "father",
			Name:   "name",
		},
		Birthday: day,
	}, nil
}

func (r *QueryParamRouter) TimePost(c *fastapi.Context, day time.Time) (time.Time, error) {
	return day, nil
}

type Name struct {
	Father string `query:"father" json:"father" validate:"required" description:"姓氏"` // 必选查询参数
	Name   string `query:"name" json:"name,omitempty" description:"姓名"`               // 可选查询参数
}

type DateTime struct {
	Name         *Name     `json:"name" query:"name"`
	Birthday     time.Time `json:"birthday" query:"birthday" description:"生日"` // 日期时间类型
	ImportantDay *struct {
		LoveDay          time.Time   `json:"love_day"`
		NameDay          time.Time   `json:"name_day"`
		ChildrenBirthday []time.Time `json:"children_birthday"`
	} `json:"important_day,omitempty" description:"纪念日"`
}

// ============================================================================

type RequestBodyRouter struct {
	fastapi.BaseRouter
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

func (r *RequestBodyRouter) RegisterPost(c *fastapi.Context, location string, form *RegisterForm) (*RegisterForm, error) {
	return form, nil
}

func (r *RequestBodyRouter) RegisterWithParamPost(c *fastapi.Context, name *Name, form *RegisterForm) (*Name, error) {
	return &Name{
		Father: name.Father,
		Name:   name.Name,
	}, nil
}

func (r *RequestBodyRouter) ArrayRequestBodyPost(c *fastapi.Context, names []*Name) ([]*Name, error) {
	return nil, errors.New(fmt.Sprintf("(array request) always return error, length: %d", len(names)))
}

func (r *RequestBodyRouter) StringQueryParamPatch(c *fastapi.Context, name string) (string, error) {
	return name, nil
}

func (r *RequestBodyRouter) Path() map[string]string {
	return map[string]string{
		"RegisterWithParamPost": "register-with/:location",
	}
}

// ============================================================================

type ResponseModelRouter struct {
	fastapi.BaseRouter
}

func (r *ResponseModelRouter) Prefix() string { return "/api/response" }

// ServerValidateErrorModel 服务器内部模型校验错误示例
type ServerValidateErrorModel struct {
	ServerName string `json:"server_name" validate:"required" description:"服务名称"`
	Version    string `json:"version" validate:"required" description:"服务版本号"`
	Links      int    `json:"links,omitempty" description:"连接数"`
}

func (s *ServerValidateErrorModel) SchemaDesc() string { return "服务器内部模型示例" }

func (r *ResponseModelRouter) ReturnSimpleStructGet(c *fastapi.Context) (*ServerValidateErrorModel, error) {
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
	Cpu       *CPU      `json:"cpu"`
	Serial    string    `json:"serial" validate:"required" description:"序列号"`
	Tunnels   []*Tunnel `json:"tunnels" description:"通道信息"`
	PcieSlots int       `json:"pcie_slots"`
}

func (r *ResponseModelRouter) ReturnNormalStructGet(c *fastapi.Context) (*BoardCard, error) {
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

func (r *ResponseModelRouter) GetTunnels(c *fastapi.Context) ([]*Tunnel, error) {
	return []*Tunnel{
		{
			No:      10,
			BoardId: 0x4321,
		},
		{
			No:      12,
			BoardId: 0x4323,
		},
	}, nil
}

type Child struct {
	Name string
	Age  int
}

func (r *ResponseModelRouter) GetChildren(c *fastapi.Context) ([]*Child, error) {
	return []*Child{
		{
			Age:  10,
			Name: "li",
		},
	}, nil
}

func (r *ResponseModelRouter) PostReportMessage(c *fastapi.Context, form []*Child) ([]*Child, error) {
	return form, nil
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
	Data   *EnosDataItem `json:"data"`
	Kind   string        `json:"kind"`
	Msg    string        `json:"msg,omitempty"`
	Submsg string        `json:"submsg,omitempty"`
	Code   int           `json:"code"`
}

func (r *ResponseModelRouter) GetComplexModel(c *fastapi.Context) (*EnosData, error) {
	return &EnosData{
		Data:   &EnosDataItem{},
		Kind:   "",
		Msg:    "",
		Submsg: "",
		Code:   0,
	}, nil
}

func routeCtxCancel(s *fastapi.Context) *fastapi.Response {
	cl := s.Logger() // 当路由执行完毕退出时, ctx将被释放
	ctx := s.Context()

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
	IP     string `json:"ip" description:"IPv4地址"`
	Detail struct {
		IPv4     string `json:"IPv4" description:"IPv4地址"`
		IPv4Full string `json:"IPv4_full" description:"带端口的IPv4地址"`
		Ipv6     string `json:"IPv6" description:"IPv6地址"`
	} `json:"detail" description:"详细信息"`
}

func (m IPModel) SchemaDesc() string { return "IP信息" }

type DomainRecord struct {
	IP struct {
		Record *IPModel `json:"record" description:"解析记录"`
	} `json:"ip"`
	Addresses []struct {
		Host string `json:"host"`
		Port string `json:"port"`
	} `json:"addresses" description:"主机地址"`
	Timestamp int64 `json:"timestamp" description:"时间戳"`
}

func (r *ResponseModelRouter) GetMoreComplexModel(c *fastapi.Context) (*DomainRecord, error) {
	m := &DomainRecord{
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
	m.IP.Record = &IPModel{
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
	return m, nil
}

func TestNew(t *testing.T) {
	app := fastapi.New(fastapi.Config{
		Version:     "v0.2.0",
		Description: "",
		Title:       "FastApi Example",
		Debug:       true,
	})

	// 底层采用fiber
	app.SetMux(fiberWrapper.Default())

	// 创建路由
	app.IncludeRouter(&BaseTypeRouter{}).
		IncludeRouter(&QueryParamRouter{}).
		IncludeRouter(&RequestBodyRouter{}).
		IncludeRouter(&ResponseModelRouter{}).
		IncludeRouter(routers.NewInfoRouter(app.Config())) // 开启默认基础路由

	app.Run("0.0.0.0", "8099") // 阻塞运行
}
