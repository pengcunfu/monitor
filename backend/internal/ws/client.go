package ws

import (
	"time"

	"github.com/gorilla/websocket"
)

const (
	// 写入超时
	writeWait = 10 * time.Second
	// 客户端 60s 无任何消息视为失联
	pongWait = 60 * time.Second
	// 服务端心跳周期
	pingPeriod = 30 * time.Second
	// 单帧消息上限
	maxMessageSize = 8192
	// 发送缓冲
	sendBuffer = 64
)

// Client 单个 WebSocket 连接：单写协程模型（并发写会 panic，必须统一由 writePump 写）。
type Client struct {
	hub  *Hub
	conn *websocket.Conn
	send chan []byte
}

// newClient 创建客户端并注册到 Hub。
func newClient(hub *Hub, conn *websocket.Conn) *Client {
	return &Client{hub: hub, conn: conn, send: make(chan []byte, sendBuffer)}
}

// run 启动读写协程（阻塞直到连接关闭）。
func (c *Client) run() {
	go c.writePump()
	c.readPump()
}

// readPump 处理读消息与心跳。
func (c *Client) readPump() {
	defer func() {
		c.hub.Unregister(c)
		_ = c.conn.Close()
	}()
	c.conn.SetReadLimit(maxMessageSize)
	_ = c.conn.SetReadDeadline(time.Now().Add(pongWait))
	c.conn.SetPongHandler(func(string) error {
		return c.conn.SetReadDeadline(time.Now().Add(pongWait))
	})
	for {
		if _, _, err := c.conn.ReadMessage(); err != nil {
			return
		}
		// 当前仅支持心跳；订阅协议预留
	}
}

// writePump 唯一的写协程：发送缓冲消息 + 周期性 Ping。
func (c *Client) writePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		_ = c.conn.Close()
	}()
	for {
		select {
		case msg, ok := <-c.send:
			_ = c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				_ = c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}
			if err := c.conn.WriteMessage(websocket.TextMessage, msg); err != nil {
				return
			}
		case <-ticker.C:
			_ = c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}
