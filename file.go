package fastapi

// 上传文件类型, 必须与 FileModelPkg 保持一致

import (
	"io"
)

// File 上传文件
// 通过声明此类型,以接收来自用户的上传文件
type File struct {
	Filename string
	File     io.Reader         `json:"-"`
	Headers  map[string]string `json:"-"`
}

type FileResponseMode int

const (
	FileResponseModeSendFile       FileResponseMode = iota // 直接发送本地文件
	FileResponseModeFileAttachment                         // 触发浏览器下载文件
	FileResponseModeReaderFile                             // 从reader中直接返回文件
	FileResponseModeStream                                 // 返回任意的数据流
)

// FileResponse 返回文件或消息流
type FileResponse struct {
	mode     FileResponseMode
	filename string
	filepath string
	reader   io.Reader
}

func (m *FileResponse) SchemaDesc() string {
	return "文件响应"
}

// SendFile 向客户端发送本地文件，此时会读取文件内容，并将文件内容作为响应体返回给客户端
func SendFile(filepath string) *FileResponse {
	return &FileResponse{
		mode:     FileResponseModeSendFile,
		filepath: filepath,
	}
}

// FileAttachment 以附件形式返回本地文件，自动设置"Content-Disposition"，浏览器会触发自动下载
func FileAttachment(filepath, filename string) *FileResponse {
	return &FileResponse{
		mode:     FileResponseModeFileAttachment,
		filename: filename,
		filepath: filepath,
	}
}

// FileFromReader 从io.Reader中读取文件并返回给客户端
func FileFromReader(reader io.Reader, filename string) *FileResponse {
	return &FileResponse{
		mode:   FileResponseModeReaderFile,
		reader: reader,
	}
}

// Stream 发送字节流到客户端，Content-Type为application/octet-stream
func Stream(reader io.Reader) *FileResponse {
	return &FileResponse{
		mode:   FileResponseModeStream,
		reader: reader,
	}
}
