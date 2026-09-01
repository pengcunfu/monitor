package api

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"monitor/internal/config"
	"monitor/internal/manager"
	"monitor/internal/notifier"
	"monitor/internal/store"
	"monitor/internal/ws"
)

// API 持有各业务 handler 的依赖。
type API struct {
	store *store.Store
	cfg   *config.Config
	hub   *ws.Hub
	mgr   *notifier.Manager
	ops   *manager.Manager

	alertEngine AlertEngine
}

// AlertEngine 告警引擎接口（由 alert 包实现，供路由触发 reload 等）。
type AlertEngine interface {
	Reload()
	FiringCount() int
	FiringEvents(limit int) []interface{}
}

// New 构建 gin 引擎并注册全部路由。
func New(st *store.Store, cfg *config.Config, hub *ws.Hub, engine AlertEngine, mgr *notifier.Manager, ops *manager.Manager) *gin.Engine {
	a := &API{store: st, cfg: cfg, hub: hub, alertEngine: engine, mgr: mgr, ops: ops}

	r := gin.New()
	r.Use(gin.Logger(), gin.Recovery(), a.CORS())

	v1 := r.Group("/api/v1")
	{
		v1.GET("/health", a.health)
		v1.POST("/auth/login", a.login)
		v1.GET("/ws", hub.Handler(cfg.Server.JWTSecret))

		// 以下路由需要登录（Bearer JWT）
		auth := v1.Group("", a.authRequired())
		{
			auth.GET("/auth/me", a.me)
			auth.PUT("/auth/password", a.changePassword)

			// P2 指标
			auth.GET("/overview", a.overview)
			auth.GET("/metrics/latest", a.metricsLatest)
			auth.GET("/metrics/disks", a.metricsDisks)
			auth.GET("/metrics/nics", a.metricsNics)
			auth.GET("/metrics/history", a.metricsHistory)

			// P5 规则与告警
			auth.GET("/rules", a.ruleList)
			auth.POST("/rules", a.ruleCreate)
			auth.GET("/rules/:id", a.ruleGet)
			auth.PUT("/rules/:id", a.ruleUpdate)
			auth.DELETE("/rules/:id", a.ruleDelete)
			auth.PUT("/rules/:id/toggle", a.ruleToggle)
			auth.POST("/rules/reload", a.ruleReload)

			auth.GET("/alerts", a.alertList)
			auth.GET("/alerts/stats", a.alertStats)
			auth.GET("/alerts/firing", a.alertFiringList)
			auth.GET("/alerts/:id", a.alertGet)
			auth.POST("/alerts/:id/ack", a.alertAck)

			// P6 通知渠道
			auth.GET("/channels", a.channelList)
			auth.GET("/channels/types", a.channelTypes)
			auth.POST("/channels", a.channelCreate)
			auth.GET("/channels/:id", a.channelGet)
			auth.PUT("/channels/:id", a.channelUpdate)
			auth.DELETE("/channels/:id", a.channelDelete)
			auth.POST("/channels/:id/test", a.channelTest)

			// P7 进程/服务/设置
			auth.GET("/process/current", a.processCurrent)
			auth.GET("/process/history", a.processHistory)
			auth.GET("/process/names", a.processNames)
			auth.GET("/services", a.serviceList)
			auth.GET("/services/history", a.serviceHistory)
			auth.GET("/services/names", a.serviceNames)
			auth.GET("/settings", a.settingGet)
			auth.PUT("/settings", a.settingUpdate)
			auth.POST("/settings/smtp/test", a.settingSMTPTest)

			// P8 进程/服务管理（仅管理员）
			auth.GET("/capabilities", a.capabilities)
			admin := auth.Group("", a.adminOnly())
			{
				admin.POST("/process/:pid/kill", a.processKill)
				admin.POST("/process/:pid/restart", a.processRestart)
				admin.POST("/services/:name/start", a.serviceStart)
				admin.POST("/services/:name/stop", a.serviceStop)
				admin.POST("/services/:name/restart", a.serviceRestart)
				admin.POST("/services/:name/enable", a.serviceEnable)
				admin.POST("/services/:name/disable", a.serviceDisable)
			}
		}
	}

	// 生产环境内嵌前端（NoRoute 兜底必须在所有路由之后注册）
	a.mountStatic(r)

	return r
}

// CORS 开发期跨域中间件（前端 Vite dev server 直连时使用；生产 embed 同源无需）。
func (a *API) CORS() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Content-Type, Authorization")
		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}
		c.Next()
	}
}

// health 健康检查。
func (a *API) health(c *gin.Context) {
	ok(c, gin.H{"status": "ok", "time": time.Now().UnixMilli()})
}
