package api

import (
	"strconv"
	"time"

	"github.com/gin-gonic/gin"

	"monitor/internal/model"
)

// overview 返回最新快照摘要：CPU/内存/负载/磁盘/网络当前值 + uptime + 主机名。
func (a *API) overview(c *gin.Context) {
	snap, err := a.store.LatestSnapshot()
	if err != nil {
		internalError(c, "查询最新快照失败")
		return
	}
	if snap == nil {
		ok(c, gin.H{})
		return
	}
	var disks []model.DiskUsage
	var nets []model.NetRate
	_ = snap.DiskUsageJSON.Unmarshal(&disks)
	_ = snap.NetJSON.Unmarshal(&nets)

	ok(c, gin.H{
		"ts":           snap.Ts,
		"host_name":    snap.HostName,
		"uptime_sec":   snap.UptimeSec,
		"cpu_usage":    snap.CPUUsage,
		"cpu_cores":    snap.CPUCores,
		"load1":        snap.Load1,
		"load5":        snap.Load5,
		"load15":       snap.Load15,
		"mem_total":    snap.MemTotal,
		"mem_used":     snap.MemUsed,
		"mem_avail":    snap.MemAvail,
		"mem_usage":    snap.MemUsage,
		"swap_total":   snap.SwapTotal,
		"swap_used":    snap.SwapUsed,
		"net_rx_bps":   snap.NetRXBps,
		"net_tx_bps":   snap.NetTXBps,
		"disk_usage":   disks,
		"net":          nets,
	})
}

// metricsLatest 返回最新快照全量。
func (a *API) metricsLatest(c *gin.Context) {
	snap, err := a.store.LatestSnapshot()
	if err != nil {
		internalError(c, "查询最新快照失败")
		return
	}
	if snap == nil {
		ok(c, nil)
		return
	}
	ok(c, snap)
}

// metricsDisks 返回最新磁盘分区明细。
func (a *API) metricsDisks(c *gin.Context) {
	snap, err := a.store.LatestSnapshot()
	if err != nil {
		internalError(c, "查询最新快照失败")
		return
	}
	if snap == nil {
		ok(c, []model.DiskUsage{})
		return
	}
	var disks []model.DiskUsage
	_ = snap.DiskUsageJSON.Unmarshal(&disks)
	ok(c, disks)
}

// metricsNics 返回最新网卡明细（含速率）。
func (a *API) metricsNics(c *gin.Context) {
	snap, err := a.store.LatestSnapshot()
	if err != nil {
		internalError(c, "查询最新快照失败")
		return
	}
	if snap == nil {
		ok(c, []model.NetRate{})
		return
	}
	var nets []model.NetRate
	_ = snap.NetJSON.Unmarshal(&nets)
	ok(c, nets)
}

// metricPoint 时间序列点。
type metricPoint struct {
	Ts    int64   `json:"ts"`
	Value float64 `json:"value"`
}

// metricsHistory 返回指定指标在 [from, to] 区间的时间序列（服务端下采样，最多 ~1000 点）。
// 支持 metric: cpu_usage/mem_usage/load1/net_rx_bps/net_tx_bps/disk_used_percent。
// disk_used_percent 需带 target 挂载点参数。
func (a *API) metricsHistory(c *gin.Context) {
	metric := c.Query("metric")
	target := c.Query("target")

	to, _ := strconv.ParseInt(c.Query("to"), 10, 64)
	from, _ := strconv.ParseInt(c.Query("from"), 10, 64)
	if to == 0 {
		to = time.Now().UnixMilli()
	}
	if from == 0 || from >= to {
		from = to - 3600*1000
	}

	rows, err := a.store.SnapshotHistory(from, to)
	if err != nil {
		internalError(c, "查询历史数据失败")
		return
	}

	var points []metricPoint
	for _, r := range rows {
		if v, ok := extractMetricValue(&r, metric, target); ok {
			points = append(points, metricPoint{Ts: r.Ts, Value: v})
		}
	}

	const maxPoints = 1000
	if len(points) > maxPoints {
		points = downsample(points, maxPoints)
	}
	ok(c, points)
}

// extractMetricValue 从快照中提取指定指标值。
func extractMetricValue(snap *model.MetricSnapshot, metric, target string) (float64, bool) {
	switch metric {
	case "cpu_usage":
		return snap.CPUUsage, true
	case "mem_usage":
		return snap.MemUsage, true
	case "load1":
		return snap.Load1, true
	case "net_rx_bps":
		return float64(snap.NetRXBps), true
	case "net_tx_bps":
		return float64(snap.NetTXBps), true
	case "disk_used_percent":
		var disks []model.DiskUsage
		if err := snap.DiskUsageJSON.Unmarshal(&disks); err != nil {
			return 0, false
		}
		for _, d := range disks {
			if target == "" || d.Mount == target {
				return d.UsedPct, true
			}
		}
	}
	return 0, false
}

// downsample 把点序列按时间均匀抽稀为最多 target 个点（取平均）。
func downsample(points []metricPoint, target int) []metricPoint {
	if len(points) <= target {
		return points
	}
	out := make([]metricPoint, 0, target)
	step := float64(len(points)) / float64(target)
	for i := 0; i < target; i++ {
		start := int(float64(i) * step)
		end := int(float64(i+1) * step)
		if end > len(points) {
			end = len(points)
		}
		if start >= end {
			continue
		}
		var sum float64
		for j := start; j < end; j++ {
			sum += points[j].Value
		}
		out = append(out, metricPoint{Ts: points[start].Ts, Value: sum / float64(end-start)})
	}
	return out
}
