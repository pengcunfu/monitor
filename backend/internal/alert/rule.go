package alert

import (
	"monitor/internal/model"
)

// MetricLabel 指标中文名（用于描述与前端下拉）。
func MetricLabel(m string) string {
	switch m {
	case model.MetricCPUUsage:
		return "CPU 使用率"
	case model.MetricMemUsage:
		return "内存使用率"
	case model.MetricLoad1:
		return "系统负载 (1m)"
	case model.MetricDiskUsedPct:
		return "磁盘使用率"
	case model.MetricNetRXBps:
		return "网络入带宽"
	case model.MetricNetTXBps:
		return "网络出带宽"
	case model.MetricServiceActive:
		return "服务状态"
	case model.MetricProcessCPU:
		return "进程 CPU"
	default:
		return m
	}
}

// extractValues 按规则指标从快照提取取值列表。
// 标量指标返回单元素；多实例指标（磁盘）返回每个匹配实例。
func (e *Engine) extractValues(r *model.AlertRule, snap *model.MetricSnapshot) []metricValue {
	switch r.Metric {
	case model.MetricCPUUsage:
		return []metricValue{{value: snap.CPUUsage}}
	case model.MetricMemUsage:
		return []metricValue{{value: snap.MemUsage}}
	case model.MetricLoad1:
		return []metricValue{{value: snap.Load1}}
	case model.MetricNetRXBps:
		return []metricValue{{value: float64(snap.NetRXBps)}}
	case model.MetricNetTXBps:
		return []metricValue{{value: float64(snap.NetTXBps)}}
	case model.MetricDiskUsedPct:
		var disks []model.DiskUsage
		if err := snap.DiskUsageJSON.Unmarshal(&disks); err != nil {
			return nil
		}
		var out []metricValue
		for _, d := range disks {
			if r.Target == "" || d.Mount == r.Target {
				out = append(out, metricValue{target: d.Mount, value: d.UsedPct})
			}
		}
		return out
	case model.MetricProcessCPU:
		seen := map[string]bool{}
		var out []metricValue
		for _, p := range e.procs {
			if r.Target != "" && p.Name != r.Target {
				continue
			}
			if seen[p.Name] {
				continue
			}
			seen[p.Name] = true
			out = append(out, metricValue{target: p.Name, value: p.CPUPercent})
		}
		return out
	case model.MetricServiceActive:
		// 服务类规则在 UpdateServiceStates 中评估
		return nil
	}
	return nil
}
