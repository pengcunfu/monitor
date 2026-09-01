package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// adminOnly 管理操作权限中间件：反查数据库取最新 Role（与 me handler 同模式），
// 仅 role=admin 可执行管理操作。必须在 authRequired 之后注册（依赖 uid/username）。
func (a *API) adminOnly() gin.HandlerFunc {
	return func(c *gin.Context) {
		username := c.GetString("username")
		user, err := a.store.FindUserByUsername(username)
		if err != nil || user == nil {
			unauthorized(c, "用户不存在")
			c.Abort()
			return
		}
		if user.Role != "admin" {
			fail(c, http.StatusForbidden, 403, "仅管理员可执行该操作")
			c.Abort()
			return
		}
		c.Next()
	}
}
