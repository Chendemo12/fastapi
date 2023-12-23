package fastapi

import "io"

// UploadFile 上传文件
// 通过声明此类型,以接收来自用户的上传文件
type UploadFile struct {
	Filename string
	File     io.Reader
	Headers  map[string]string
}
