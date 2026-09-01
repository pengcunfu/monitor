package model

// 时间约定：所有时间字段统一存 UTC epoch 毫秒（int64），避免 SQLite 无原生时区导致混乱。

// Metric 枚举常量：告警规则支持的指标类型。
const (
	MetricCPUUsage       = "cpu_usage"         // CPU 使用率 %
	MetricMemUsage       = "mem_usage"         // 内存使用率 %
	MetricLoad1          = "load1"             // 1 分钟负载
	MetricDiskUsedPct    = "disk_used_percent" // 磁盘使用率 %（target=挂载点）
	MetricNetRXBps       = "net_rx_bps"        // 网络入带宽 B/s（target=网卡名）
	MetricNetTXBps       = "net_tx_bps"        // 网络出带宽 B/s（target=网卡名）
	MetricServiceActive  = "service_active"    // systemd 服务是否 active（0/1，target=unit 名）
	MetricProcessCPU     = "process_cpu"       // 进程 CPU %（target=进程名）
)

// AlertRule 支持的比较操作符。
const (
	OpGT = "gt" // >
	OpGE = "ge" // >=
	OpLT = "lt" // <
	OpLE = "le" // <=
)

// 告警严重级别。
const (
	SeverityCritical = "critical"
	SeverityWarning  = "warning"
)

// 告警事件状态。
const (
	EventFiring   = "firing"
	EventResolved = "resolved"
)

// ===================== 基础模型 =====================

// Model 通用字段：自增主键 + 毫秒级创建/更新时间。
type Model struct {
	ID        uint  `gorm:"primarykey" json:"id"`
	CreatedAt int64 `gorm:"autoCreateTime:milli" json:"created_at"`
	UpdatedAt int64 `gorm:"autoUpdateTime:milli" json:"updated_at"`
}

// ===================== 用户 =====================

// User 登录用户。
type User struct {
	ID           uint   `gorm:"primarykey" json:"id"`
	Username     string `gorm:"size:64;uniqueIndex:uniq_users_username;not null" json:"username"`
	PasswordHash string `gorm:"size:128;not null" json:"-"`
	Role         string `gorm:"size:32;default:admin" json:"role"`
	CreatedAt    int64  `gorm:"autoCreateTime:milli" json:"created_at"`
	UpdatedAt    int64  `gorm:"autoUpdateTime:milli" json:"updated_at"`
}

// ===================== 指标快照 =====================

// DiskUsage 磁盘分区使用情况。
type DiskUsage struct {
	Mount   string  `json:"mount"`
	FS      string  `json:"fs"`
	Total   uint64  `json:"total"`
	Used    uint64  `json:"used"`
	UsedPct float64 `json:"used_percent"`
}

// DiskIORate 磁盘 IO 速率。
type DiskIORate struct {
	Device   string `json:"device"`
	ReadBps  uint64 `json:"read_bps"`
	WriteBps uint64 `json:"write_bps"`
	IOPS     uint64 `json:"iops"`
}

// NetRate 网卡累计计数与速率。
type NetRate struct {
	Name    string `json:"name"`
	RXBytes uint64 `json:"rx_bytes"`
	TXBytes uint64 `json:"tx_bytes"`
	RXBps   uint64 `json:"rx_bps"`
	TXBps   uint64 `json:"tx_bps"`
}

// MetricSnapshot 每采集周期一行的系统指标快照。
type MetricSnapshot struct {
	ID        uint  `gorm:"primarykey" json:"id"`
	Ts        int64 `gorm:"index:idx_snap_ts" json:"ts"`
	HostName  string `gorm:"size:128" json:"host_name"`
	CPUUsage  float64 `json:"cpu_usage"`
	CPUCores  int     `json:"cpu_cores"`
	Load1     float64 `json:"load1"`
	Load5     float64 `json:"load5"`
	Load15    float64 `json:"load15"`
	MemTotal  uint64  `json:"mem_total"`
	MemUsed   uint64  `json:"mem_used"`
	MemAvail  uint64  `json:"mem_avail"`
	MemUsage  float64 `json:"mem_usage"`
	SwapTotal uint64  `json:"swap_total"`
	SwapUsed  uint64  `json:"swap_used"`

	DiskUsageJSON   JSON `gorm:"type:text" json:"disk_usage"`
	DiskIORatesJSON JSON `gorm:"type:text" json:"disk_io_rates"`
	NetJSON         JSON `gorm:"type:text" json:"net"`

	NetRXBps  uint64 `json:"net_rx_bps"`
	NetTXBps  uint64 `json:"net_tx_bps"`
	UptimeSec uint64 `json:"uptime_sec"`
}

// ===================== 进程采样 =====================

// ProcessSample 单进程的一次采样（只保留 top N）。
type ProcessSample struct {
	ID         uint    `gorm:"primarykey" json:"id"`
	Ts         int64   `gorm:"index:idx_proc_ts;index:idx_proc_ts_cpu,priority:1;index:idx_proc_name_ts,priority:2" json:"ts"`
	PID        int32   `gorm:"column:pid" json:"pid"`
	Name       string  `gorm:"size:128;index:idx_proc_name_ts,priority:1" json:"name"`
	User       string  `gorm:"size:64" json:"user"`
	CPUPercent float64 `gorm:"index:idx_proc_ts_cpu,priority:2" json:"cpu_percent"`
	MemPercent float64 `json:"mem_percent"`
	MemRSS     uint64  `json:"mem_rss"`
	State      string  `gorm:"size:16" json:"state"`
	CmdLine    string  `gorm:"size:256" json:"cmdline"`
}

// ===================== 服务状态 =====================

// ServiceState systemd 服务的单次状态快照。
type ServiceState struct {
	ID          uint   `gorm:"primarykey" json:"id"`
	Ts          int64  `gorm:"index:idx_svc_ts;index:idx_svc_name_ts,priority:2" json:"ts"`
	Name        string `gorm:"size:128;index:idx_svc_name_ts,priority:1" json:"name"`
	Description string `gorm:"size:256" json:"description"`
	LoadState   string `gorm:"size:32" json:"load_state"`
	ActiveState string `gorm:"size:32" json:"active_state"`
	SubState    string `gorm:"size:32" json:"sub_state"`
	IsActive    bool   `json:"is_active"`
	Enabled     bool   `gorm:"default:false" json:"enabled"` // 开机自启（systemd enabled / Windows AUTO_START）
	MainPID     int32  `gorm:"column:main_pid" json:"main_pid"`
	ExitCode    int32  `json:"exit_code"`
}

// ===================== 告警规则 =====================

// AlertRule 阈值告警规则。
type AlertRule struct {
	Model
	Name           string  `gorm:"size:128;not null" json:"name"`
	Metric         string  `gorm:"size:32;index:idx_rule_enabled;not null" json:"metric"`
	Target         string  `gorm:"size:128" json:"target"` // 空=全部；磁盘填挂载点、网络填网卡名、服务/进程填名称
	Operator       string  `gorm:"size:8;not null" json:"operator"`
	Threshold      float64 `json:"threshold"`
	DurationTicks  int     `gorm:"default:1" json:"duration_ticks"` // 持续 N 个采集周期才触发
	Severity       string  `gorm:"size:16;default:warning" json:"severity"`
	ChannelIDsJSON JSON    `gorm:"type:text" json:"channel_ids"`
	CooldownSec    int     `gorm:"default:900" json:"cooldown_sec"`
	NotifyOnResolve bool   `gorm:"default:false" json:"notify_on_resolve"`
	Enabled        bool    `gorm:"index:idx_rule_enabled;default:true" json:"enabled"`
	Description    string  `gorm:"size:512" json:"description"`
}

// ChannelIDs 解析规则绑定的渠道 id 列表。
func (r *AlertRule) ChannelIDs() []uint {
	var ids []uint
	_ = r.ChannelIDsJSON.Unmarshal(&ids)
	return ids
}

// ===================== 告警事件 =====================

// AlertEvent 告警事件（触发/恢复记录）。
type AlertEvent struct {
	ID         uint    `gorm:"primarykey" json:"id"`
	RuleID     uint    `gorm:"index:idx_event_rule_status,priority:1" json:"rule_id"`
	RuleName   string  `gorm:"size:128" json:"rule_name"`
	Metric     string  `gorm:"size:32" json:"metric"`
	Target     string  `gorm:"size:128" json:"target"`
	Severity   string  `gorm:"size:16" json:"severity"`
	Status     string  `gorm:"size:16;index:idx_event_status;index:idx_event_rule_status,priority:2" json:"status"`
	Message    string  `gorm:"size:1024" json:"message"`
	Value      float64 `json:"value"`
	Threshold  float64 `json:"threshold"`
	FiredAt    int64   `gorm:"index:idx_event_fired_at" json:"fired_at"`
	ResolvedAt int64   `json:"resolved_at"`
	DurationSec int64  `json:"duration_sec"`
	Notified   bool    `gorm:"default:false" json:"notified"`
	NotifyAt   int64   `json:"notify_at"`
	Acked      bool    `gorm:"default:false" json:"acked"`
	AckBy      string  `gorm:"size:64" json:"ack_by"`
	AckAt      int64   `json:"ack_at"`
}

// ===================== 通知渠道 =====================

// 渠道类型常量。
const (
	ChannelSMTP       = "smtp"
	ChannelWebhook    = "webhook"
	ChannelFeishu     = "feishu"
	ChannelWecom      = "wecom"
	ChannelDingTalk   = "dingtalk"
	ChannelServerChan = "serverchan"
)

// NotificationChannel 通知渠道配置。
type NotificationChannel struct {
	Model
	Name       string `gorm:"size:64;not null" json:"name"`
	Type       string `gorm:"size:32;index:idx_chan_type" json:"type"`
	ConfigJSON JSON   `gorm:"type:text" json:"config"`
	Enabled    bool   `gorm:"index:idx_chan_enabled;default:true" json:"enabled"`
}

// ===================== 设置 =====================

// Setting 全局设置（key-value，value 为任意 JSON）。
type Setting struct {
	ID        uint   `gorm:"primarykey" json:"id"`
	Key       string `gorm:"size:64;uniqueIndex:uniq_setting_key;not null" json:"key"`
	ValueJSON JSON   `gorm:"type:text" json:"value"`
	UpdatedAt int64  `gorm:"autoUpdateTime:milli" json:"updated_at"`
}

// ===================== 通知日志 =====================

// NotificationLog 通知发送日志。
type NotificationLog struct {
	ID        uint   `gorm:"primarykey" json:"id"`
	ChannelID uint   `json:"channel_id"`
	EventID   uint   `json:"event_id"`
	Type      string `gorm:"size:32" json:"type"`
	Target    string `gorm:"size:128" json:"target"` // 收件人/url（脱敏）
	Title     string `gorm:"size:256" json:"title"`
	Success   bool   `json:"success"`
	Response  string `gorm:"size:500" json:"response"`
	CreatedAt int64  `gorm:"autoCreateTime:milli;index:idx_nlog_created" json:"created_at"`
}

// AllModels 返回需要 AutoMigrate 的全部模型。
func AllModels() []interface{} {
	return []interface{}{
		&User{},
		&MetricSnapshot{},
		&ProcessSample{},
		&ServiceState{},
		&AlertRule{},
		&AlertEvent{},
		&NotificationChannel{},
		&Setting{},
		&NotificationLog{},
	}
}
