package ws

import (
	"encoding/json"
	"sync"
)

// Hub 管理所有 WebSocket 连接并支持按主题广播。
type Hub struct {
	mu      sync.Mutex
	clients map[*Client]struct{}
}

// NewHub 创建 Hub。
func NewHub() *Hub {
	return &Hub{clients: make(map[*Client]struct{})}
}

// Register 注册连接。
func (h *Hub) Register(c *Client) {
	h.mu.Lock()
	h.clients[c] = struct{}{}
	h.mu.Unlock()
}

// Unregister 注销连接。
func (h *Hub) Unregister(c *Client) {
	h.mu.Lock()
	delete(h.clients, c)
	h.mu.Unlock()
}

// Broadcast 向所有连接广播一帧 {type, ts, data}。
// 慢客户端（send 缓冲已满）直接丢弃该帧，避免阻塞与并发写 panic。
func (h *Hub) Broadcast(topic string, data interface{}) {
	frame, err := json.Marshal(map[string]interface{}{
		"type": topic,
		"data": data,
	})
	if err != nil {
		return
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	for c := range h.clients {
		select {
		case c.send <- frame:
		default:
			// 丢弃；若持续跟不上可在此断开慢客户端
		}
	}
}
