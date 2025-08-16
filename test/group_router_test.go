package test

import (
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/Chendemo12/fastapi"
	"github.com/Chendemo12/fastapi/middleware/fiberWrapper"
	"github.com/Chendemo12/fastapi/middleware/routers"
)

// ============================================================================

type BaseTypeRouter struct {
	fastapi.BaseGroupRouter
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
	fastapi.BaseGroupRouter
}

func (r *QueryParamRouter) Prefix() string { return "/api/query-param" }

func (r *QueryParamRouter) Path() map[string]string {
	return map[string]string{
		"IntQueryParamGet":     "int-query/:param",
		"GetPathParam":         "path-param/:day",
		"GetPathAndQueryParam": "path-query-param/:day",
	}
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

// PageReq 分页请求参数
type PageReq struct {
	PageNum  int `json:"pageNum,omitempty" default:"1" query:"pageNum" description:"当前页数"`
	PageSize int `json:"pageSize,omitempty" default:"10" query:"pageSize"  description:"当前页长度"`
}

func (r *QueryParamRouter) GetPathAndQueryParam(c *fastapi.Context, page *PageReq) (*PageReq, error) {
	pf := c.PathField("day", time.Now().String())

	fastapi.Info("path params: ", pf)
	return page, nil
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
	fastapi.BaseGroupRouter
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

func (r *RequestBodyRouter) RegisterPost(c *fastapi.Context, form *RegisterForm) (*RegisterForm, error) {
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

func (r *RequestBodyRouter) NoRequestBodyParamPatch(c *fastapi.Context, null *fastapi.None) (string, error) {
	return "no-request-body-param", nil
}

func (r *RequestBodyRouter) Path() map[string]string {
	return map[string]string{
		"RegisterWithParamPost": "register-with/:location",
	}
}

// ============================================================================

type ResponseModelRouter struct {
	fastapi.BaseGroupRouter
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
		Record *IPModel `json:"record" validate:"required" description:"解析记录"`
	} `json:"ip" validate:"required"`
	Addresses []struct {
		Host string `json:"host"`
		Port string `json:"port"`
	} `json:"addresses" validate:"required,gte=1" description:"主机地址"`
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

type PageResp[T any] struct {
	PageNum     int   `json:"pageNum" description:"当前页数"`
	PageSize    int   `json:"pageSize" description:"当前页长度"`
	Total       int64 `json:"total" description:"总数据条数"`
	Pages       int64 `json:"pages" description:"总页数"`
	Data        T     `json:"data" description:"返回数据列表"`
	IsLastPage  bool  `json:"isLastPage" description:"是否是最后一页"`
	IsFirstPage bool  `json:"isFirstPage" description:"是否是第一页"`
}

type MemoryNote struct {
	No      int64  `json:"no" validate:"required"`
	Title   string `json:"title,omitempty"`
	Content string `json:"content,omitempty"`
}

type GenericRouter struct {
	fastapi.BaseGroupRouter
}

func (r *GenericRouter) Prefix() string { return "/api/generic" }

func (r *GenericRouter) GetGenericBaseModel(c *fastapi.Context) (*PageResp[int], error) {
	resp := &PageResp[int]{
		PageNum:     1,
		PageSize:    20,
		Total:       40,
		Pages:       2,
		Data:        12345,
		IsLastPage:  false,
		IsFirstPage: true,
	}
	return resp, nil
}

func (r *GenericRouter) GetGenericModel(c *fastapi.Context) (*PageResp[[]*MemoryNote], error) {
	resp := &PageResp[[]*MemoryNote]{
		PageNum:  1,
		PageSize: 20,
		Total:    40,
		Pages:    2,
		Data: []*MemoryNote{
			{
				No:      1,
				Title:   "hello",
				Content: "hello generic model",
			},
			{
				No:      2,
				Title:   "name",
				Content: "generic model",
			},
		},
		IsLastPage:  false,
		IsFirstPage: true,
	}
	return resp, nil
}

func (r *GenericRouter) GetArrayGenericModel(c *fastapi.Context) ([]*PageResp[[]*MemoryNote], error) {
	resp := &PageResp[[]*MemoryNote]{
		PageNum:  1,
		PageSize: 20,
		Total:    40,
		Pages:    2,
		Data: []*MemoryNote{
			{
				No:      1,
				Title:   "hello",
				Content: "hello generic model",
			},
			{
				No:      2,
				Title:   "name",
				Content: "generic model",
			},
		},
		IsLastPage:  false,
		IsFirstPage: true,
	}

	return []*PageResp[[]*MemoryNote]{resp}, nil
}

func (r *GenericRouter) PostGenericModel(c *fastapi.Context, form *PageResp[[]*MemoryNote]) (*MemoryNote, error) {
	return &MemoryNote{}, nil
}

func (r *GenericRouter) PostArrayGenericModel(c *fastapi.Context, form []*PageResp[[]*MemoryNote]) (*MemoryNote, error) {
	return &MemoryNote{}, nil
}

type MemberScore struct {
	PageReq
	UserId   string `json:"userId" query:"userId"`
	UserName string `json:"userName" query:"userName"`
}

type AnonymousQueryModel struct {
	MemberScore
	Email string `json:"email" query:"email"`
}

type AnonymousRouter struct {
	fastapi.BaseGroupRouter
}

func (r *AnonymousRouter) Prefix() string { return "/api/anonymous" }

func (r *AnonymousRouter) GetAnonymousModel(c *fastapi.Context, params *AnonymousQueryModel) (*MemberScore, error) {
	return &MemberScore{
		PageReq: PageReq{
			PageNum:  1,
			PageSize: 1,
		},
		UserId:   params.UserId,
		UserName: params.UserName,
	}, nil
}

type ErrorRouter struct {
	fastapi.BaseGroupRouter
}

func (r *ErrorRouter) Prefix() string { return "/api/error" }

func (r *ErrorRouter) GetReturnString(c *fastapi.Context) (string, error) {
	return "", errors.New("string error")
}

func BeforeValidate(c *fastapi.Context) error {
	c.Set("before-validate", time.Now())

	return nil
}

func PrintRequestLog(c *fastapi.Context) {
	fastapi.Info("请求耗时: ", time.Since(c.GetTime("before-validate")))
	fastapi.Info("响应状态码: ", c.Response().StatusCode)
}

type ErrorMessage struct {
	Code      string `json:"code,omitempty" description:"错误码"`
	Message   string `json:"message,omitempty" description:"错误信息"`
	Timestamp int64  `json:"timestamp,omitempty"`
}

// 格式化路由函数错误消息
func FormatErrorMessage(c *fastapi.Context, err error) (statusCode int, message any) {
	return 400, &ErrorMessage{
		Code:      "0x1234",
		Message:   err.Error(),
		Timestamp: time.Now().Unix(),
	}
}

func TestNew(t *testing.T) {
	app := fastapi.New(fastapi.Config{
		Version:     "v1.0.0",
		Description: "这是一段Http服务描述信息，会显示在openApi文档的顶部",
		Title:       "FastApi Example",
	})

	// 底层采用fiber
	app.SetMux(fiberWrapper.Default())

	app.UsePrevious(BeforeValidate)
	app.UseBeforeWrite(PrintRequestLog)

	// 创建路由
	app.IncludeRouter(&BaseTypeRouter{}).
		IncludeRouter(&QueryParamRouter{}).
		IncludeRouter(&RequestBodyRouter{}).
		IncludeRouter(&ResponseModelRouter{}).
		IncludeRouter(&GenericRouter{}).
		IncludeRouter(&AnonymousRouter{}).
		IncludeRouter(&ErrorRouter{}).
		IncludeRouter(routers.NewInfoRouter(app.Config())) // 开启默认基础路由

	// 自定义错误格式
	app.SetRouteErrorFormatter(FormatErrorMessage, fastapi.RouteErrorOpt{
		StatusCode:   500,
		ResponseMode: &ErrorMessage{},
		Description:  "服务器内部发生错误，请稍后重试",
	})

	app.Run("0.0.0.0", "8099") // 阻塞运行
}
