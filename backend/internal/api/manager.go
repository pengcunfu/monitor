package api

import (
	"errors"
	"log"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"monitor/internal/manager"
)

// capabilities 返回当前平台的管理能力（前端据此隐藏不支持的按钮）。
func (a *API) capabilities(c *gin.Context) {
	ok(c, a.ops.Capabilities())
}

// processKill 结束指定进程。
func (a *API) processKill(c *gin.Context) {
	pid, err := strconv.ParseInt(c.Param("pid"), 10, 32)
	if err != nil || pid <= 0 {
		badRequest(c, "非法 PID")
		return
	}
	res, err := a.ops.Process().Kill(c.Request.Context(), int32(pid))
	if err != nil {
		a.handleManagerErr(c, err)
		return
	}
	log.Printf("[manager] %s 结束进程 pid=%d (%s)", opUser(c), pid, res.Name)
	ok(c, res)
}

// processRestart 重启指定进程。
func (a *API) processRestart(c *gin.Context) {
	pid, err := strconv.ParseInt(c.Param("pid"), 10, 32)
	if err != nil || pid <= 0 {
		badRequest(c, "非法 PID")
		return
	}
	res, err := a.ops.Process().Restart(c.Request.Context(), int32(pid))
	if err != nil {
		a.handleManagerErr(c, err)
		return
	}
	log.Printf("[manager] %s 重启进程 pid=%d (%s) → 新 pid=%d", opUser(c), pid, res.Name, res.NewPID)
	ok(c, res)
}

// serviceStart 启动服务。
func (a *API) serviceStart(c *gin.Context) {
	a.serviceOp(c, "start")
}

// serviceStop 停止服务。
func (a *API) serviceStop(c *gin.Context) {
	a.serviceOp(c, "stop")
}

// serviceRestart 重启服务。
func (a *API) serviceRestart(c *gin.Context) {
	a.serviceOp(c, "restart")
}

// serviceEnable 设置服务开机自启。
func (a *API) serviceEnable(c *gin.Context) {
	a.serviceOp(c, "enable")
}

// serviceDisable 取消服务开机自启。
func (a *API) serviceDisable(c *gin.Context) {
	a.serviceOp(c, "disable")
}

// serviceOp 按操作名分发服务管理请求。
func (a *API) serviceOp(c *gin.Context, action string) {
	name := c.Param("name")
	if name == "" {
		badRequest(c, "缺少服务名")
		return
	}
	svc := a.ops.Service()
	ctx := c.Request.Context()
	var err error
	switch action {
	case "start":
		err = svc.Start(ctx, name)
	case "stop":
		err = svc.Stop(ctx, name)
	case "restart":
		err = svc.Restart(ctx, name)
	case "enable":
		err = svc.Enable(ctx, name)
	case "disable":
		err = svc.Disable(ctx, name)
	}
	if err != nil {
		a.handleManagerErr(c, err)
		return
	}
	log.Printf("[manager] %s 服务操作 %s=%s", opUser(c), action, name)
	ok(c, gin.H{"name": name, "action": action + "ed"})
}

// opUser 返回当前操作者用户名（审计留痕用）。
func opUser(c *gin.Context) string {
	if v := c.GetString("username"); v != "" {
		return v
	}
	return "unknown"
}

// handleManagerErr 把 manager 包的哨兵错误归类为 HTTP 状态码。
func (a *API) handleManagerErr(c *gin.Context, err error) {
	switch {
	case errors.Is(err, manager.ErrPermission):
		fail(c, http.StatusForbidden, 403, err.Error())
	case errors.Is(err, manager.ErrNotFound):
		notFound(c, err.Error())
	case errors.Is(err, manager.ErrUnsupported):
		fail(c, http.StatusNotImplemented, 501, err.Error())
	case errors.Is(err, manager.ErrConflict):
		fail(c, http.StatusConflict, 409, err.Error())
	default:
		internalError(c, err.Error())
	}
}
