package api

import (
	"strconv"

	"github.com/gin-gonic/gin"
)

// processCurrent 最新一轮进程 top N。
func (a *API) processCurrent(c *gin.Context) {
	top := atoiDefault(c.Query("top"), 20)
	sortBy := c.Query("sort")
	if sortBy != "mem" {
		sortBy = "cpu"
	}
	list, err := a.store.LatestProcessSamples(top, sortBy)
	if err != nil {
		internalError(c, "查询进程失败")
		return
	}
	ok(c, list)
}

// processHistory 指定进程的历史曲线。
func (a *API) processHistory(c *gin.Context) {
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
	list, err := a.store.ProcessHistory(name, from, to)
	if err != nil {
		internalError(c, "查询进程历史失败")
		return
	}
	// 转换为时间序列（cpu/mem 双序列）
	cpu := make([]metricPoint, 0, len(list))
	mem := make([]metricPoint, 0, len(list))
	for _, p := range list {
		cpu = append(cpu, metricPoint{Ts: p.Ts, Value: p.CPUPercent})
		mem = append(mem, metricPoint{Ts: p.Ts, Value: p.MemPercent})
	}
	if len(cpu) > 1000 {
		cpu = downsample(cpu, 1000)
		mem = downsample(mem, 1000)
	}
	ok(c, gin.H{"cpu": cpu, "mem": mem})
}

// processNames 进程名列表（下拉选择）。
func (a *API) processNames(c *gin.Context) {
	names, err := a.store.ProcessNames()
	if err != nil {
		internalError(c, "查询进程名失败")
		return
	}
	ok(c, names)
}
