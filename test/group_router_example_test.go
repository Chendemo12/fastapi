package test

import (
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/Chendemo12/fastapi"
	"github.com/Chendemo12/fastapi/middleware/fiberWrapper"
)

// ExampleRouter 创建一个结构体实现fastapi.GroupRouter接口
type ExampleRouter struct {
	fastapi.BaseGroupRouter
}

func (r *ExampleRouter) Prefix() string { return "/api/example" }

func (r *ExampleRouter) Summary() map[string]string {
	return map[string]string{
		"PostUploadFile":         "上传文件",
		"GetDownloadFile":        "请求文件",
		"PostUploadFileWithForm": "修改用户信息",
	}
}

func (r *ExampleRouter) GetAppTitle(c *fastapi.Context) (string, error) {
	return "FastApi Example", nil
}

type UpdateAppTitleReq struct {
	Title string `json:"title" validate:"required" description:"App标题"`
}

func (r *ExampleRouter) PatchUpdateAppTitle(c *fastapi.Context, form *UpdateAppTitleReq) (*UpdateAppTitleReq, error) {
	return form, nil
}

func (r *ExampleRouter) GetError(c *fastapi.Context) (string, error) {
	return "", errors.New("内部错误")
}

func (r *ExampleRouter) GetDomainRecord(c *fastapi.Context) (*DomainRecord, error) {
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

func (r *ExampleRouter) DeleteMyDate(c *fastapi.Context, day time.Time, param *DateTime) (*DateTime, error) {
	return &DateTime{
		Name: &Name{
			Father: "father",
			Name:   "name",
		},
		Birthday: day,
	}, nil
}

func (r *ExampleRouter) PostUploadFile(c *fastapi.Context, file *fastapi.File) (int64, error) {
	fr := file.First()
	fmt.Println("文件名：", fr.Filename)

	return fr.Size, nil
}

type UpdateUserInfoReq struct {
	Name  string `json:"name" validate:"required"`
	Email string `json:"email" validate:"required"`
}

func (r *ExampleRouter) PostUploadFileWithForm(c *fastapi.Context, file *fastapi.File, param *UpdateUserInfoReq) (int64, error) {
	fr := file.First()
	fmt.Println("文件名：", fr.Filename)

	return fr.Size, nil
}

type DownloadFileReq struct {
	FileName string `json:"fileName" validate:"required"`
}

func (r *ExampleRouter) GetFileAttachment(c *fastapi.Context, param *DownloadFileReq) (*fastapi.FileResponse, error) {
	return fastapi.FileAttachment("../README.md", "README.md"), nil
}

func (r *ExampleRouter) GetSendFile(c *fastapi.Context) (*fastapi.FileResponse, error) {
	return fastapi.SendFile("../README.md"), nil
}

func TestExampleRouter(t *testing.T) {
	// 可选的 fastapi.Config 参数
	app := fastapi.New(fastapi.Config{
		Version:     "v1.0.0",
		Description: "这是一段Http服务描述信息，会显示在openApi文档的顶部",
		Title:       "FastApi Example",
	})

	// 此处采用默认的内置Fiber实现, 必须在启动之前设置
	mux := fiberWrapper.Default()
	app.SetMux(mux)

	// 注册路由
	app.IncludeRouter(&ExampleRouter{})
	// 自定义错误格式
	app.SetRouteErrorFormatter(FormatErrorMessage, fastapi.RouteErrorOpt{
		StatusCode:   400,
		ResponseMode: &ErrorMessage{},
		Description:  "此状态代表服务器处理中遇上了错误",
	})
	app.UsePrevious(BeforeValidate)
	//app.Use(returnErrorDeps)
	app.UseBeforeWrite(PrintRequestLog)

	app.Run("0.0.0.0", "8090") // 阻塞运行
}
