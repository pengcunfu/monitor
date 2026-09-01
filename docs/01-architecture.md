# 架构设计

## 总体架构

单机自监控架构：一个二进制部署在被监控的 Linux 机器上，采集本机指标、评估告警、推送通知、提供 Web UI。

```
┌──────────────────────────────────────────────────────────────┐
│                        monitor 二进制                         │
│                                                              │
│  ┌────────────┐   ┌────────────┐   ┌──────────────┐          │
│  │  Collector │──▶│ SQLite     │   │  Alert Engine│          │
│  │ 采集循环     │   │ (WAL)      │◀──│  状态机/规则   │          │
│  └─────┬──────┘   └─────┬──────┘   └──────┬───────┘          │
│        │ 实时广播       │ 查询            │ 触发/恢复          │
│        ▼                ▼                ▼                    │
│  ┌──────────┐   ┌──────────────┐   ┌───────────┐             │
│  │ WS Hub   │   │  Gin API     │   │ Notifier  │──▶ 邮件/     │
│  │ (实时推送) │   │ (REST+embed) │   │ Manager   │    Webhook/ │
│  └────┬─────┘   └──────┬───────┘   └─────┬─────┘    飞书/企微/ │
│       │                │                 │          钉钉/Server酱│
│       ▼                ▼                 ▼                     │
│   浏览器 UI  ◀──(WS)──  浏览器 UI  ◀─(HTTP/静态资源)──          │
└──────────────────────────────────────────────────────────────┘
```

## 数据流

1. **采集**：Collector 启动三条独立循环（主指标 10s / 进程 30s / 服务 30s，间隔可由设置动态调整）。
   - 网络带宽与磁盘 IO 通过「网卡/设备计数器差值 ÷ ΔT」计算，首轮仅建立基线不落库。
   - 进程采集使用 8 个 worker 并发，按 CPU% 截断 top N（gopsutil，跨平台）。
   - 服务采集：Linux 走 systemd D-Bus、Windows 走 SCM（`x/sys/windows/svc/mgr`），状态归一化为统一模型（build tag 隔离）。
2. **落库**：主指标快照（含磁盘/网卡 JSON 明细）、进程采样、服务状态写入 SQLite（WAL 模式）。
3. **评估**：每条快照评估启用中的告警规则 → 状态机（NORMAL → FIRING → RESOLVED）→ 落库告警事件。
4. **通知**：触发/恢复时按规则绑定渠道经 Notifier 分发，带冷却去重；每次发送记录日志。
5. **展示**：WebSocket 广播实时指标/告警帧；REST 提供历史查询；前端内嵌（go:embed）。

## 进程/服务管理（internal/manager）

`internal/manager` 提供跨平台的管理能力，与只读采集分离，全部经 `adminOnly` 中间件（role=admin）保护并留痕日志：

| 能力 | Linux | Windows | macOS |
|---|---|---|---|
| 进程 kill / restart | unix（SIGTERM→SIGKILL + Setsid 重启） | windows（TerminateProcess + DETACHED_PROCESS） | unix |
| 服务 start/stop/restart/enable/disable | systemd D-Bus | SCM（`svc/mgr`） | 暂不支持（预留 launchd） |

- 平台拆分：`manager_{linux,windows,other}.go` 工厂 + `process_{unix,windows}.go` + `service_{linux,windows,other}.go`，沿用 build tag 惯例。
- 进程重启解析原进程可执行路径与参数重新拉起（`exec.Command(Exe, CmdlineSlice...)`），输出重定向到临时目录日志，避免命令行注入。
- 服务状态模型 `ServiceState` 新增 `Enabled`（开机自启），Windows 状态归一化映射（running→active、stopped→inactive 等），告警引擎只读 `IsActive` 保持兼容。
- 邮件 SMTP 在「系统设置」页配置，自动同步为内置「邮件告警」通知渠道。

## 关键设计决策

| 决策 | 说明 |
|---|---|
| 纯 Go SQLite（glebarez/sqlite） | 无 CGO，支持交叉编译；DSN 开启 WAL + busy_timeout(5000) 避免读写互锁 |
| 时间统一 epoch 毫秒 | SQLite 无原生时区，全部存 int64 毫秒，前端 dayjs 本地化 |
| 多实例明细用 JSON 列 | 磁盘分区/网卡明细存 JSON 文本，单分区历史用 sqlite json_extract 可查 |
| 单写协程 WebSocket | gorilla/websocket 并发写会 panic，每连接一个 send channel + 唯一 writer |
| 采集写入集中于采集器 | 批量 insert 单事务，避免多写竞争 |
| 告警去重 | 同一「规则×目标实例」未恢复只保留一条 firing 事件；冷却期内不重复通知 |
| 保留策略 | 快照/进程/服务/告警/通知日志按设置天数分批清理（启动 + 每 6h） |
| 渠道敏感字段 | password/secret/sendkey 返回脱敏为 `***`，更新留空保留原值 |
| 管理操作权限 | 进程/服务管理仅 role=admin 可执行（后端 `adminOnly` 中间件 + 前端按钮隐藏），操作留痕日志 |
| 服务状态归一化 | systemd/Windows 状态统一映射为 active/inactive/activating/paused/failed + IsActive 布尔，告警引擎兼容 |
| SMTP 邮件设置 | 系统设置页配置，自动 upsert 内置「邮件告警」渠道；密码留空保留原值 |

## 数据表

- `users` — 登录用户
- `metric_snapshots` — 主指标快照（CPU/内存/负载/磁盘/网络）
- `process_samples` — 进程采样（top N）
- `service_states` — systemd/Windows 服务状态（含 `enabled` 开机自启）
- `alert_rules` — 告警规则
- `alert_events` — 告警事件（firing/resolved）
- `notification_channels` — 通知渠道配置（含内置「邮件告警」SMTP 渠道）
- `notification_logs` — 通知发送日志
- `settings` — 全局设置（采集间隔/保留天数/邮件 SMTP）
