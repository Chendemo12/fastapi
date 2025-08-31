package fastapi

type SSE struct {
	Event   string   `json:"event,omitempty" description:"事件类型（可选），缺省则为message事件"`
	Data    []string `json:"data" validate:"required" description:"消息内容（必需），无需以换行符结尾，程序会自行处理，注意；在SSE中，data字段只能包含一条纯文本，但是可以有多个data字段，多行数据会被连接成一个消息。"`
	Id      string   `json:"id,omitempty"  description:"消息ID（可选）"`
	Retry   int      `json:"retry,omitempty" description:"重试间隔，毫秒（可选）"`
	Comment string   `json:":,omitempty" description:"注释（可选），注释行可以用来防止连接超时，服务器可以定期发送一条消息注释行，以保持连接不断。"`
}

func (m *SSE) SchemaDesc() string {
	return `SSE消息格式说明`
}
