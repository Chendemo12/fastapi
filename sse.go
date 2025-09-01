package fastapi

import (
	"strconv"
	"strings"
)

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

func (m *SSE) ToBuilder() *strings.Builder {
	var builder = &strings.Builder{}

	// 1. 处理注释（如果有）
	if m.Comment != "" {
		// 注释行以冒号开头
		builder.WriteString(": ")
		builder.WriteString(m.Comment)
		builder.WriteString("\n")
	}

	// 2. 处理事件类型（可选）
	if m.Event != "" {
		builder.WriteString("event: ")
		builder.WriteString(m.Event)
		builder.WriteString("\n")
	}

	// 3. 处理数据（必需），每个 data 字段单独一行
	for _, dataLine := range m.Data {
		builder.WriteString("data: ")
		builder.WriteString(dataLine)
		builder.WriteString("\n")
	}

	// 4. 处理消息ID（可选）
	if m.Id != "" {
		builder.WriteString("id: ")
		builder.WriteString(m.Id)
		builder.WriteString("\n")
	}

	// 5. 处理重试间隔（可选）
	if m.Retry > 0 {
		builder.WriteString("retry: ")
		builder.WriteString(strconv.Itoa(m.Retry))
		builder.WriteString("\n")
	}

	// 6. 必须以空行结束消息（SSE 规范要求）
	builder.WriteString("\n")

	return builder
}
