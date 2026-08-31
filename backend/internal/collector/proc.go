package collector

import (
	"log"
	"sort"
	"sync"
	"time"

	"github.com/shirou/gopsutil/v3/process"

	"monitor/internal/model"
	"monitor/internal/store"
)

// collectProcesses 采集进程 top N 并落库、广播。
func (c *Collector) collectProcesses() {
	topN := c.st.GetSettingInt(store.SettingProcessTopN, c.cfg.Collector.ProcessTopN)
	if topN <= 0 {
		topN = 20
	}
	samples := sampleTopProcesses(topN)
	if len(samples) == 0 {
		return
	}
	if err := c.st.InsertProcessSamples(samples); err != nil {
		log.Printf("[collector] 写入进程采样失败: %v", err)
		return
	}
	if c.updater != nil {
		c.updater.UpdateProcessSamples(samples)
	}
	if c.hub != nil {
		c.hub.Broadcast("process", samples)
	}
}

// procStat 单进程采集结果。
type procStat struct {
	pid        int32
	name       string
	user       string
	cpu        float64
	mem        float64
	rss        uint64
	state      string
	cmd        string
}

// sampleTopProcesses 用 worker pool 并发采集全部进程，按 CPU% 排序截断 topN。
func sampleTopProcesses(topN int) []model.ProcessSample {
	pids, err := process.Pids()
	if err != nil || len(pids) == 0 {
		return nil
	}

	const workers = 8
	pidCh := make(chan int32, len(pids))
	resCh := make(chan *procStat, len(pids))
	var wg sync.WaitGroup
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for pid := range pidCh {
				resCh <- readProcess(pid)
			}
		}()
	}
	for _, p := range pids {
		pidCh <- p
	}
	close(pidCh)
	wg.Wait()
	close(resCh)

	var stats []procStat
	for s := range resCh {
		if s != nil {
			stats = append(stats, *s)
		}
	}
	sort.Slice(stats, func(i, j int) bool { return stats[i].cpu > stats[j].cpu })
	if len(stats) > topN {
		stats = stats[:topN]
	}

	now := time.Now().UnixMilli()
	out := make([]model.ProcessSample, 0, len(stats))
	for _, s := range stats {
		out = append(out, model.ProcessSample{
			Ts:         now,
			PID:        s.pid,
			Name:       s.name,
			User:       s.user,
			CPUPercent: s.cpu,
			MemPercent: s.mem,
			MemRSS:     s.rss,
			State:      s.state,
			CmdLine:    truncate(s.cmd, 256),
		})
	}
	return out
}

// readProcess 读取单个进程信息；无有效信息（权限不足/已退出）返回 nil。
func readProcess(pid int32) *procStat {
	p, err := process.NewProcess(pid)
	if err != nil {
		return nil
	}
	st := &procStat{pid: pid}
	if v, err := p.Name(); err == nil {
		st.name = v
	}
	if v, err := p.Username(); err == nil {
		st.user = v
	}
	if v, err := p.CPUPercent(); err == nil {
		st.cpu = v
	}
	if v, err := p.MemoryPercent(); err == nil {
		st.mem = float64(v)
	}
	if mi, err := p.MemoryInfo(); err == nil && mi != nil {
		st.rss = mi.RSS
	}
	if v, err := p.Status(); err == nil && len(v) > 0 {
		st.state = string(v[0])
	}
	if v, err := p.Cmdline(); err == nil {
		st.cmd = v
	}
	if st.name == "" && st.cmd == "" && st.user == "" {
		return nil
	}
	return st
}

// truncate 截断字符串到指定长度（按字节，避免 SQLite 超长列）。
func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n]
}
