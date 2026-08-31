# 熔岩网络安全事件应急处置系统

基于 **Go + React + Vite + SQLite** 的全平台网络安全事件应急处置系统，部署在被监控的 Linux 机器上，采集并监控系统资源、进程、systemd 服务与网络，支持阈值告警与多渠道通知。

> 本项目为全新实现，历史 Django 代码已移除。

## 功能特性

- **系统监控**：CPU 使用率、内存/交换分区、系统负载、磁盘分区使用率、磁盘 IO 速率、网络带宽（网卡级）
- **进程监控**：top N 进程（CPU/内存排序）、进程历史曲线
- **服务监控**：systemd 服务运行状态与状态历史（仅 Linux）
- **实时推送**：WebSocket 实时指标（概览页无需刷新自动更新）
- **告警引擎**：阈值规则（`>` `>=` `<` `<=`）、持续周期判定、冷却去重、触发/恢复状态机
- **通知渠道**：邮件 SMTP、通用 Webhook、飞书/企业微信/钉钉机器人、Server酱
- **历史查询**：任意时间范围趋势曲线（服务端降采样）
- **数据保留**：按设置的天数自动清理过期数据
- **单二进制部署**：前端资源内嵌进 Go 二进制，附带 systemd 单元

## 技术栈

| 端 | 技术 |
|---|---|
| 后端 | Go + gin + gopsutil/v3 + GORM + glebarez/sqlite（纯 Go SQLite，无 CGO）+ gorilla/websocket + JWT |
| 前端 | Vite + React 19 + TypeScript + Ant Design 6 + ECharts + SWR + zustand |
| 存储 | SQLite（WAL 模式） |

## 目录结构

```
├── backend/            # Go 后端
│   ├── cmd/server/     # 入口
│   ├── internal/       # config/model/store/collector/alert/notifier/api/ws/server
│   └── web/dist/       # 前端构建产物（go:embed 内嵌）
├── frontend/           # React 前端（Vite）
├── deploy/             # systemd 单元 + 生产配置示例
├── scripts/            # 开发/构建脚本
└── docs/               # 架构与 API 文档
```

## 快速开始

### 本地开发（Windows / Linux）

```bash
# 后端（默认 :8080）
cd backend && go run ./cmd/server

# 前端（默认 :5173，代理 /api /ws 到 :8080）
cd frontend && npm install && npm run dev
```

Windows 下可直接运行 `scripts/dev.bat`。

### 构建 Linux 单二进制

```bash
bash scripts/build.sh
# 产出 bin/monitor-linux-amd64 与 monitor-linux-amd64-<日期>.tar.gz
```

### Linux 部署

```bash
# 1. 安装
sudo useradd -r -s /usr/sbin/nologin monitor
sudo mkdir -p /opt/monitor/data /etc/monitor
sudo cp bin/monitor-linux-amd64 /opt/monitor/monitor
sudo cp deploy/monitor.yaml /etc/monitor/monitor.yaml   # 修改 jwt_secret、db_path
sudo cp deploy/monitor.service /etc/systemd/system/
sudo chown -R monitor:monitor /opt/monitor

# 2. 启动
sudo systemctl daemon-reload
sudo systemctl enable --now monitor
```

访问 `http://<机器IP>:8080`，默认账号 **admin / admin123**（首次登录后请修改密码）。

## 告警与通知

1. 在「通知渠道」配置渠道并「测试发送」
2. 在「告警规则」新建规则（如：CPU 使用率 > 90% 持续 2 个周期），绑定渠道
3. 触发后「告警中心」实时显示，并通过渠道推送；恢复后自动标记 resolved

## 文档

- [架构设计](docs/01-architecture.md)
- [API 接口](docs/02-api.md)
