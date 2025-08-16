package fastapi

import (
	"errors"
	"io"
	"mime/multipart"
)

// File 上传文件
// 通过声明此类型, 以接收来自用户的上传文件
// 上传文件类型, 必须与 FileModelPkg 保持一致
type File struct {
	files []*multipart.FileHeader `description:"文件"`
}

func (f *File) Files() []*multipart.FileHeader {
	return f.files
}

func (f *File) RangeFile(fc func(file *multipart.FileHeader) (next bool)) {
	for _, file := range f.files {
		if !fc(file) {
			break
		}
	}
}

func (f *File) Open(index int) (multipart.File, error) {
	if index >= len(f.files) {
		return nil, errors.New("index out of range")
	}

	return f.files[index].Open()
}

// First 获取第一个文件, 需自行确认文件数量，否则会panic
func (f *File) First() *multipart.FileHeader {
	return f.files[0]
}

func (f *File) OpenFirst() (multipart.File, error) {
	return f.files[0].Open()
}

// FileResponse 返回文件或消息流
type FileResponse struct {
	mode     FileResponseMode
	filename string
	filepath string
	reader   io.Reader
}

type FileResponseMode int

const (
	FileResponseModeSendFile       FileResponseMode = iota // 直接发送本地文件
	FileResponseModeFileAttachment                         // 触发浏览器下载文件
	FileResponseModeReaderFile                             // 从reader中直接返回文件
	FileResponseModeStream                                 // 返回任意的数据流
)

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
