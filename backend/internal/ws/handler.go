package ws

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"

	"monitor/internal/auth"
)

// Handler 创建 WS 升级处理器：query token 鉴权 + 升级 + 注册。
// 浏览器 WebSocket 无法自定义请求头，故 token 放 query 参数。
func (h *Hub) Handler(jwtSecret string) gin.HandlerFunc {
	upgrader := websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
		CheckOrigin:     func(r *http.Request) bool { return true },
	}

	return func(c *gin.Context) {
		token := c.Query("token")
		if _, err := auth.Parse(jwtSecret, token); err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"code": 401, "msg": "令牌无效或已过期"})
			return
		}
		conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
		if err != nil {
			return
		}
		client := newClient(h, conn)
		h.Register(client)
		client.run()
	}
}
