# API 接口文档

- 基础路径：`/api/v1`
- 响应结构：`{"code": 0, "msg": "ok", "data": {...}}`，`code=0` 表示成功
- 鉴权：除 `POST /auth/login` 与 `/ws`（query token）外，均需请求头 `Authorization: Bearer <jwt>`

## 认证

| 方法 | 路径 | 说明 |
|---|---|---|
| POST | `/auth/login` | 登录，body `{username, password}` → `{token, user}` |
| GET | `/auth/me` | 当前用户 |
| PUT | `/auth/password` | 修改密码，body `{old_password, new_password}` |

## 总览与指标

| 方法 | 路径 | 说明 |
|---|---|---|
| GET | `/overview` | 最新快照摘要（CPU/内存/负载/磁盘/网络 + 主机信息） |
| GET | `/metrics/latest` | 最新快照全量（含磁盘/网卡 JSON 明细） |
| GET | `/metrics/disks` | 最新磁盘分区明细 |
| GET | `/metrics/nics` | 最新网卡明细（含速率） |
| GET | `/metrics/history` | 时间序列，参数 `metric, from, to, target`；服务端降采样至 ≤1000 点。metric 支持 `cpu_usage/mem_usage/load1/net_rx_bps/net_tx_bps/disk_used_percent` |

## 进程

| 方法 | 路径 | 说明 |
|---|---|---|
| GET | `/process/current?top=&sort=cpu\|mem` | 最新一轮 top N 进程 |
| GET | `/process/history?name=&from=&to=` | 指定进程历史 → `{cpu:[], mem:[]}` |
| GET | `/process/names` | 进程名列表 |
| POST | `/process/:pid/kill` | 结束进程（仅 admin）→ `{pid, name, action}` |
| POST | `/process/:pid/restart` | 重启进程（仅 admin）→ `{pid, name, exe, new_pid, action}` |

## 服务（Linux / Windows）

| 方法 | 路径 | 说明 |
|---|---|---|
| GET | `/services?state=` | 最新服务状态，可按 `active/inactive/failed` 过滤 |
| GET | `/services/history?name=&from=&to=` | 服务状态变化历史 |
| GET | `/services/names` | 服务名列表 |
| POST | `/services/:name/start` | 启动服务（仅 admin） |
| POST | `/services/:name/stop` | 停止服务（仅 admin） |
| POST | `/services/:name/restart` | 重启服务（仅 admin） |
| POST | `/services/:name/enable` | 开启开机自启（仅 admin） |
| POST | `/services/:name/disable` | 关闭开机自启（仅 admin） |

服务状态 `active_state` 归一化：`active/inactive/activating/deactivating/paused/failed`；`enabled` 表示开机自启。

## 能力声明

| 方法 | 路径 | 说明 |
|---|---|---|
| GET | `/capabilities` | 当前平台管理能力 → `{platform, process_manage, service_manage}`，前端据此隐藏不支持的按钮 |

管理操作错误码：`403` 权限不足/非管理员、`404` 目标不存在、`409` 状态冲突（如启动被禁用服务、单元无 [Install] 段）、`501` 当前平台不支持。

## 告警规则

| 方法 | 路径 | 说明 |
|---|---|---|
| GET | `/rules?page=&size=` | 分页列表 |
| POST | `/rules` | 新建，body 见下 |
| GET/PUT/DELETE | `/rules/:id` | 详情 / 更新 / 删除 |
| PUT | `/rules/:id/toggle?enabled=` | 启用/停用 |
| POST | `/rules/reload` | 立即重载规则缓存（引擎每 30s 自动刷新） |

规则 body：

```json
{
  "name": "CPU 使用率过高",
  "metric": "cpu_usage",
  "target": "",            // 空=全部；磁盘填挂载点、网络填网卡名、服务/进程填名称
  "operator": "gt",        // gt/ge/lt/le
  "threshold": 90,
  "duration_ticks": 2,     // 持续 N 个采集周期
  "severity": "critical",  // critical/warning
  "channel_ids": [1],
  "cooldown_sec": 900,     // 通知冷却（秒）
  "notify_on_resolve": false,
  "enabled": true,
  "description": ""
}
```

指标枚举：`cpu_usage` `mem_usage` `load1` `disk_used_percent` `net_rx_bps` `net_tx_bps` `service_active` `process_cpu`

## 告警事件

| 方法 | 路径 | 说明 |
|---|---|---|
| GET | `/alerts?status=&page=&size=&from=&to=` | 分页过滤（firing/resolved） |
| GET | `/alerts/stats?from=&to=` | 触发/恢复数量统计 |
| GET | `/alerts/firing?limit=` | 当前触发中的告警 |
| GET | `/alerts/:id` | 详情 |
| POST | `/alerts/:id/ack` | 确认告警 |

## 通知渠道

| 方法 | 路径 | 说明 |
|---|---|---|
| GET | `/channels` | 渠道列表（敏感字段脱敏） |
| GET | `/channels/types` | 支持的渠道类型 |
| POST | `/channels` | 新建，body `{name, type, config, enabled}` |
| GET/PUT/DELETE | `/channels/:id` | 详情 / 更新 / 删除（敏感字段留空或 `***` 保留原值） |
| POST | `/channels/:id/test` | 测试发送 |

渠道类型 `type` 与 `config` 字段：

| type | config 关键字段 |
|---|---|
| `smtp` | `host, port, user, password, from, to[], tls, insecure_skip_verify` |
| `webhook` | `url, method, headers, body_template`（Go text/template：`.Title .Content .Severity .Time`） |
| `feishu` | `webhook_url, secret` |
| `wecom` | `webhook_url` |
| `dingtalk` | `webhook_url, secret` |
| `serverchan` | `sendkey` |

## 设置

| 方法 | 路径 | 说明 |
|---|---|---|
| GET | `/settings` | 全部设置（含 `smtp` 邮件配置，密码脱敏为 `***`） |
| PUT | `/settings` | 批量更新，body `{key: value}`；含 `smtp` 时自动同步内置「邮件告警」通知渠道 |
| POST | `/settings/smtp/test` | 用提交的 SMTP 配置发送测试邮件，body `{smtp: {...}}`（未保存也可测试） |

设置 key：`collect_interval_sec` `process_interval_sec` `service_interval_sec` `process_top_n` `snapshot_retain_days` `process_retain_days` `service_retain_days` `alert_retain_days` `notify_log_retain_days`

邮件 SMTP 配置（`smtp`）：

```json
{
  "smtp": {
    "host": "smtp.qq.com",
    "port": 465,
    "user": "发件邮箱账号",
    "password": "授权码/密码（留空或 *** 表示不修改）",
    "from": "发件人邮箱",
    "to": ["收件人1", "收件人2"],
    "tls": true,
    "insecure_skip_verify": false,
    "enabled": true
  }
}
```

## WebSocket

`GET /ws?token=<jwt>`（开发期经 Vite 代理 `/ws`）。

服务端推送帧：

```json
{"type": "metric", "data": { 最新快照 }}
{"type": "alert",  "data": { 触发/恢复事件 }}
```

心跳：服务端每 30s 发 Ping；客户端 60s 无消息则判死。前端连接自动重连。
