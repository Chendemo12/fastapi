package fastapi

import (
	"github.com/Chendemo12/fastapi/websocket"
	"net/http"
)

type WSHandler interface {
	OnConnected() error
	OnClosed() error
	OnReceived() error
	OnError()
}

type Websocket struct {
	conn    *websocket.Conn
	handler WSHandler
	point   string
}

func (w *Websocket) Upgrade(resp http.ResponseWriter, req *http.Request, responseHeader http.Header) error {
	upgrader := websocket.Upgrader{}
	conn, err := upgrader.Upgrade(resp, req, responseHeader) // 升级连接为 WebSocket 连接
	if err != nil {
		return err
	}
	w.conn = conn

	return nil
}

func NewWebsocket(point string, handler WSHandler) *Websocket {

	return &Websocket{
		conn:    nil,
		handler: handler,
		point:   point,
	}
}

//func WSFiberHandler(c *fiber.Ctx) error {
//	upgrader := websocket.Upgrader{}
//	conn, err := upgrader.Upgrade(c.Request().Header, req, responseHeader) // 升级连接为 WebSocket 连接
//}
