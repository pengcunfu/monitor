package api

import (
	"strconv"

	"github.com/gin-gonic/gin"
)

// serviceList 最新一轮服务状态，可按状态过滤。
func (a *API) serviceList(c *gin.Context) {
	state := c.Query("state")
	list, err := a.store.LatestServiceStates()
	if err != nil {
		internalError(c, "查询服务状态失败")
		return
	}
	if state != "" {
		filtered := list[:0]
		for _, s := range list {
			if s.ActiveState == state {
				filtered = append(filtered, s)
			}
		}
		list = filtered
	}
	ok(c, list)
}

// serviceHistory 指定服务的状态变化历史。
func (a *API) serviceHistory(c *gin.Context) {
	name := c.Query("name")
	if name == "" {
		badRequest(c, "缺少 name 参数")
		return
	}
	from, _ := strconv.ParseInt(c.Query("from"), 10, 64)
	to, _ := strconv.ParseInt(c.Query("to"), 10, 64)
	if to == 0 {
		to = nowMs()
	}
	if from == 0 || from >= to {
		from = to - 24*3600*1000
	}
	list, err := a.store.ServiceHistory(name, from, to)
	if err != nil {
		internalError(c, "查询服务历史失败")
		return
	}
	ok(c, list)
}

// serviceNames 服务名列表。
func (a *API) serviceNames(c *gin.Context) {
	names, err := a.store.ServiceNames()
	if err != nil {
		internalError(c, "查询服务名失败")
		return
	}
	ok(c, names)
}
