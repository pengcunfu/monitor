package collector

import (
	"context"
	"log"
	"time"

	"github.com/shirou/gopsutil/v3/disk"
	"github.com/shirou/gopsutil/v3/host"
	"github.com/shirou/gopsutil/v3/net"

	"monitor/internal/config"
	"monitor/internal/model"
	"monitor/internal/store"
)

// AlertEvaluator 告警引擎接口（评估主指标快照）。
type AlertEvaluator interface {
	Evaluate(snap *model.MetricSnapshot)
}

// SampleUpdater 进程/服务采样上报接口（供引擎评估进程类与服务类规则）。
type SampleUpdater interface {
	UpdateProcessSamples(samples []model.ProcessSample)
	UpdateServiceStates(states []model.ServiceState)
}

// Broadcaster 实时广播接口。
type Broadcaster interface {
	Broadcast(topic string, data interface{})
}

// Collector 定时采集本机系统指标并落库、评估告警、广播实时数据。
type Collector struct {
	st  *store.Store
	cfg *config.Config

	engine  AlertEvaluator
	updater SampleUpdater
	hub     Broadcaster

	firstMain bool
	netPrev   map[string]net.IOCountersStat
	netPrevT  time.Time
	ioPrev    map[string]disk.IOCountersStat
	ioPrevT   time.Time
}

// New 创建采集器。
func New(st *store.Store, cfg *config.Config) *Collector {
	return &Collector{
		st:        st,
		cfg:       cfg,
		firstMain: true,
		netPrev:   map[string]net.IOCountersStat{},
		ioPrev:    map[string]disk.IOCountersStat{},
	}
}

// SetAlertEngine 注入告警引擎（可选）。
func (c *Collector) SetAlertEngine(e AlertEvaluator) { c.engine = e }

// SetSampleUpdater 注入进程/服务采样上报（可选）。
func (c *Collector) SetSampleUpdater(u SampleUpdater) { c.updater = u }

// SetHub 注入实时广播（可选）。
func (c *Collector) SetHub(h Broadcaster) { c.hub = h }

// Run 启动三条独立采集循环（主指标/进程/服务），间隔可被数据库 settings 动态调整。
func (c *Collector) Run(ctx context.Context) {
	go c.loop(ctx, store.SettingCollectIntervalSec, c.cfg.Collector.IntervalSec, c.collectMain)
	go c.loop(ctx, store.SettingProcessIntervalSec, c.cfg.Collector.ProcessIntervalSec, c.collectProcesses)
	go c.loop(ctx, store.SettingServiceIntervalSec, c.cfg.Collector.ServiceIntervalSec, c.collectServices)
	log.Println("[collector] 采集器已启动（主指标/进程/服务）")
}

// loop 立即执行一次 fn，随后按设置间隔循环；panic 不中断循环。
func (c *Collector) loop(ctx context.Context, settingKey string, def int, fn func()) {
	run := func() {
		defer func() {
			if r := recover(); r != nil {
				log.Printf("[collector] 采集任务 panic 已恢复: %v", r)
			}
		}()
		fn()
	}
	run()
	for {
		iv := time.Duration(c.st.GetSettingInt(settingKey, def)) * time.Second
		if iv < time.Second {
			iv = time.Second
		}
		timer := time.NewTimer(iv)
		select {
		case <-ctx.Done():
			timer.Stop()
			return
		case <-timer.C:
			run()
		}
	}
}

// collectMain 采集 CPU/内存/负载/磁盘/网络并落库；首轮仅建立差值基线。
func (c *Collector) collectMain() {
	now := time.Now()

	cpuUsage, cores := collectCPU()
	memTotal, memUsed, memAvail, swapTotal, swapUsed, memUsage := collectMem()
	load1, load5, load15 := collectLoad()
	hostName, uptime := collectHost()
	diskUsages := collectDiskUsage()
	ioRates := c.collectDiskIORates(now)
	netRates, rxSum, txSum := c.collectNetRates(now)

	// 首轮仅记录差值基线，不落库
	if c.firstMain {
		c.firstMain = false
		log.Printf("[collector] 首轮基线采集完成（CPU/磁盘/网络差值已就绪）")
		return
	}

	var diskJSON, ioJSON, netJSON model.JSON
	_ = diskJSON.Set(diskUsages)
	_ = ioJSON.Set(ioRates)
	_ = netJSON.Set(netRates)

	snap := &model.MetricSnapshot{
		Ts:              now.UnixMilli(),
		HostName:        hostName,
		CPUUsage:        cpuUsage,
		CPUCores:        cores,
		Load1:           load1,
		Load5:           load5,
		Load15:          load15,
		MemTotal:        memTotal,
		MemUsed:         memUsed,
		MemAvail:        memAvail,
		MemUsage:        memUsage,
		SwapTotal:       swapTotal,
		SwapUsed:        swapUsed,
		DiskUsageJSON:   diskJSON,
		DiskIORatesJSON: ioJSON,
		NetJSON:         netJSON,
		NetRXBps:        rxSum,
		NetTXBps:        txSum,
		UptimeSec:       uptime,
	}

	if err := c.st.InsertSnapshot(snap); err != nil {
		log.Printf("[collector] 写入快照失败: %v", err)
		return
	}

	if c.engine != nil {
		c.engine.Evaluate(snap)
	}
	if c.hub != nil {
		c.hub.Broadcast("metric", snap)
	}
}

// hostInfo 缓存 host.Info() 结果，避免每次采集都调用（该调用在部分平台较慢）。
func collectHost() (string, uint64) {
	hi, err := host.Info()
	if err != nil {
		return "", 0
	}
	return hi.Hostname, hi.Uptime
}
