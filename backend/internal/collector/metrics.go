package collector

import (
	"runtime"
	"time"

	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/disk"
	"github.com/shirou/gopsutil/v3/load"
	"github.com/shirou/gopsutil/v3/mem"
	"github.com/shirou/gopsutil/v3/net"

	"monitor/internal/model"
)

// collectCPU 返回整体 CPU 使用率（%）与逻辑核数。
// cpu.Percent(0) 使用差值语义：本次与上次调用之间的平均占用。
func collectCPU() (float64, int) {
	pct := float64(0)
	if p, err := cpu.Percent(0, false); err == nil && len(p) > 0 {
		pct = p[0]
	}
	cores, err := cpu.Counts(true)
	if err != nil || cores <= 0 {
		cores = 1
	}
	return pct, cores
}

// collectMem 返回内存/交换分区使用情况。usage 为内存使用率（%）。
func collectMem() (total, used, avail, swapTotal, swapUsed uint64, usage float64) {
	vm, err := mem.VirtualMemory()
	if err == nil {
		total, used, avail, usage = vm.Total, vm.Used, vm.Available, vm.UsedPercent
	}
	sm, err := mem.SwapMemory()
	if err == nil {
		swapTotal, swapUsed = sm.Total, sm.Used
	}
	return
}

// collectLoad 返回 1/5/15 分钟负载；非 Linux 平台返回 0。
func collectLoad() (l1, l5, l15 float64) {
	if runtime.GOOS != "linux" {
		return 0, 0, 0
	}
	la, err := load.Avg()
	if err != nil {
		return 0, 0, 0
	}
	return la.Load1, la.Load5, la.Load15
}

// collectDiskUsage 遍历物理分区，返回各挂载点使用情况。
func collectDiskUsage() []model.DiskUsage {
	parts, err := disk.Partitions(false)
	if err != nil {
		return nil
	}
	var out []model.DiskUsage
	seen := make(map[string]bool, len(parts))
	for _, p := range parts {
		if seen[p.Mountpoint] {
			continue
		}
		seen[p.Mountpoint] = true
		u, err := disk.Usage(p.Mountpoint)
		if err != nil || u == nil {
			continue
		}
		out = append(out, model.DiskUsage{
			Mount:   p.Mountpoint,
			FS:      p.Fstype,
			Total:   u.Total,
			Used:    u.Used,
			UsedPct: u.UsedPercent,
		})
	}
	return out
}

// collectDiskIORates 计算各磁盘设备的读写速率（计数器差值 / ΔT）。
func (c *Collector) collectDiskIORates(now time.Time) []model.DiskIORate {
	cur, err := disk.IOCounters()
	if err != nil {
		return nil
	}
	var out []model.DiskIORate
	for dev, s := range cur {
		rate := model.DiskIORate{Device: dev}
		if prev, ok := c.ioPrev[dev]; ok && !c.ioPrevT.IsZero() {
			dt := now.Sub(c.ioPrevT).Seconds()
			if dt > 0 && s.ReadBytes >= prev.ReadBytes && s.WriteBytes >= prev.WriteBytes {
				rate.ReadBps = uint64(float64(s.ReadBytes-prev.ReadBytes) / dt)
				rate.WriteBps = uint64(float64(s.WriteBytes-prev.WriteBytes) / dt)
			}
		}
		if s.IopsInProgress > 0 {
			rate.IOPS = s.IopsInProgress
		}
		out = append(out, rate)
	}
	c.ioPrev = cur
	c.ioPrevT = now
	return out
}

// collectNetRates 计算各网卡累计计数与实时速率，返回明细、入/出合计速率（B/s）。
func (c *Collector) collectNetRates(now time.Time) ([]model.NetRate, uint64, uint64) {
	cur, err := net.IOCounters(true)
	if err != nil {
		return nil, 0, 0
	}
	var out []model.NetRate
	var rxSum, txSum uint64
	for _, s := range cur {
		if s.Name == "lo" { // 环回口不计入合计
			continue
		}
		nr := model.NetRate{Name: s.Name, RXBytes: s.BytesRecv, TXBytes: s.BytesSent}
		if prev, ok := c.netPrev[s.Name]; ok && !c.netPrevT.IsZero() {
			dt := now.Sub(c.netPrevT).Seconds()
			if dt > 0 && s.BytesRecv >= prev.BytesRecv && s.BytesSent >= prev.BytesSent {
				nr.RXBps = uint64(float64(s.BytesRecv-prev.BytesRecv) / dt)
				nr.TXBps = uint64(float64(s.BytesSent-prev.BytesSent) / dt)
			}
		}
		out = append(out, nr)
		rxSum += nr.RXBps
		txSum += nr.TXBps
	}
	next := make(map[string]net.IOCountersStat, len(cur))
	for _, s := range cur {
		next[s.Name] = s
	}
	c.netPrev = next
	c.netPrevT = now
	return out, rxSum, txSum
}
