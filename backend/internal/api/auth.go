package api

import (
	"strings"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"

	"monitor/internal/auth"
)

type loginReq struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

// login 用户登录，返回 JWT 与用户信息。
func (a *API) login(c *gin.Context) {
	var req loginReq
	if err := c.ShouldBindJSON(&req); err != nil {
		badRequest(c, "请输入用户名和密码")
		return
	}
	user, err := a.store.FindUserByUsername(req.Username)
	if err != nil {
		internalError(c, "查询用户失败")
		return
	}
	if user == nil || bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)) != nil {
		unauthorized(c, "用户名或密码错误")
		return
	}
	token, err := auth.Sign(a.cfg.Server.JWTSecret, a.cfg.Server.JWTExpireH, user)
	if err != nil {
		internalError(c, "生成令牌失败")
		return
	}
	ok(c, gin.H{"token": token, "user": user})
}

// me 返回当前登录用户。
func (a *API) me(c *gin.Context) {
	user, err := a.store.FindUserByID(c.GetUint("uid"))
	if err != nil || user == nil {
		unauthorized(c, "用户不存在")
		return
	}
	ok(c, user)
}

type changePwdReq struct {
	OldPassword string `json:"old_password" binding:"required"`
	NewPassword string `json:"new_password" binding:"required,min=6"`
}

// changePassword 修改当前用户密码。
func (a *API) changePassword(c *gin.Context) {
	var req changePwdReq
	if err := c.ShouldBindJSON(&req); err != nil {
		badRequest(c, "参数错误（新密码至少 6 位）")
		return
	}
	user, err := a.store.FindUserByID(c.GetUint("uid"))
	if err != nil || user == nil {
		unauthorized(c, "用户不存在")
		return
	}
	if bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.OldPassword)) != nil {
		badRequest(c, "原密码错误")
		return
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(req.NewPassword), bcrypt.DefaultCost)
	if err != nil {
		internalError(c, "生成密码哈希失败")
		return
	}
	if err := a.store.UpdateUserPassword(user.ID, string(hash)); err != nil {
		internalError(c, "更新密码失败")
		return
	}
	ok(c, nil)
}

// authRequired JWT 鉴权中间件，通过后注入 uid/username。
func (a *API) authRequired() gin.HandlerFunc {
	return func(c *gin.Context) {
		const prefix = "Bearer "
		h := c.GetHeader("Authorization")
		if !strings.HasPrefix(h, prefix) {
			unauthorized(c, "未登录或缺少令牌")
			c.Abort()
			return
		}
		claims, err := auth.Parse(a.cfg.Server.JWTSecret, strings.TrimPrefix(h, prefix))
		if err != nil {
			unauthorized(c, "令牌无效或已过期")
			c.Abort()
			return
		}
		c.Set("uid", claims.UserID)
		c.Set("username", claims.Username)
		c.Next()
	}
}
