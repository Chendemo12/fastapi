package test

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/Chendemo12/fastapi"
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

func (r *ExampleRouter) GetSse(c *fastapi.Context) (*fastapi.SSE, error) {
	for i := 1; i < 11; i++ {
		time.Sleep(time.Millisecond * 500)
		sse := &fastapi.SSE{
			Id: fmt.Sprintf("%d", i),
			Data: []string{
				fmt.Sprintf("第%d次发送", i),
			},
			Comment: "这是注释",
		}
		if i == 5 {
			sse.Retry = 5000
		}
		if i%2 == 0 {
			sse.Event = "ding"
			sse.Data = append(sse.Data, "2的整数倍")
		}
		err := c.SSE(sse)
		if err != nil {
			return nil, err
		}
	}

	return &fastapi.SSE{}, nil
}

func (r *ExampleRouter) GetSseKeepAlive(c *fastapi.Context) (*fastapi.SSE, error) {
	ctx, cancel := context.WithTimeout(c.Context(), time.Second*60)
	defer cancel()
	err := c.SSEKeepAlive(ctx, time.Second*1)
	if err != nil {
		return nil, err
	}
	return &fastapi.SSE{}, nil
}

type SendTimes struct {
	Num int `json:"num" validate:"required,lte=10,gte=1"`
}

func (r *ExampleRouter) PostSse(c *fastapi.Context, param *SendTimes) (*fastapi.SSE, error) {
	for i := 1; i < param.Num+1; i++ {
		time.Sleep(time.Millisecond * 500)
		sse := &fastapi.SSE{
			Id: fmt.Sprintf("%d", i),
			Data: []string{
				fmt.Sprintf("第%d次发送", i),
			},
		}
		if i == 5 {
			sse.Retry = 5000
		}
		if i%2 == 0 {
			sse.Event = "ding"
			sse.Data = append(sse.Data, "2的整数倍")
		}
		err := c.SSE(sse)
		if err != nil {
			return nil, err
		}
	}

	return &fastapi.SSE{}, nil
}
