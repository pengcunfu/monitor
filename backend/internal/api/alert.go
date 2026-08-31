package api

import (
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
)

// alertList 分页查询告警事件。
func (a *API) alertList(c *gin.Context) {
	status := c.Query("status")
	from, _ := strconv.ParseInt(c.Query("from"), 10, 64)
	to, _ := strconv.ParseInt(c.Query("to"), 10, 64)
	page := atoiDefault(c.Query("page"), 1)
	size := atoiDefault(c.Query("size"), 20)
	if size > 100 {
		size = 100
	}
	list, total, err := a.store.ListAlertEvents(status, from, to, page, size)
	if err != nil {
		internalError(c, "查询告警记录失败")
		return
	}
	okPage(c, list, total, page, size)
}

// alertGet 告警详情。
func (a *API) alertGet(c *gin.Context) {
	ev, err := a.store.GetAlertEvent(idParam(c))
	if err != nil || ev == nil {
		notFound(c, "告警记录不存在")
		return
	}
	ok(c, ev)
}

// alertAck 确认告警。
func (a *API) alertAck(c *gin.Context) {
	ackBy := c.GetString("username")
	if ackBy == "" {
		ackBy = "admin"
	}
	if err := a.store.AckAlert(idParam(c), ackBy); err != nil {
		internalError(c, "确认告警失败")
		return
	}
	ok(c, nil)
}

// alertStats 触发/恢复数量统计（大屏用）。
func (a *API) alertStats(c *gin.Context) {
	to, _ := strconv.ParseInt(c.Query("to"), 10, 64)
	from, _ := strconv.ParseInt(c.Query("from"), 10, 64)
	if to == 0 {
		to = nowMs()
	}
	if from == 0 || from >= to {
		from = to - 24*3600*1000
	}
	fired, resolved, err := a.store.AlertStats(from, to)
	if err != nil {
		internalError(c, "查询告警统计失败")
		return
	}
	ok(c, gin.H{"fired": fired, "resolved": resolved})
}

// nowMs 当前毫秒时间戳。
func nowMs() int64 {
	return time.Now().UnixMilli()
}
