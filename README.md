# 熔岩网络安全事件应急处置系统

基于 **Go + React + Vite + SQLite** 的全平台网络安全事件应急处置系统，部署在被监控的机器上（Linux / Windows，后续兼容 macOS），采集并监控系统资源、进程、服务与网络，支持阈值告警、多渠道通知，以及进程/服务管理（应急处置）。

> 本项目为全新实现，历史 Django 代码已移除。

## 功能特性

- **系统监控**：CPU 使用率、内存/交换分区、系统负载、磁盘分区使用率、磁盘 IO 速率、网络带宽（网卡级）
- **进程监控**：top N 进程（CPU/内存排序）、进程历史曲线
- **进程管理**：结束进程、重启进程（按 PID，跨平台；需管理员权限）
- **服务监控**：systemd（Linux）/ Windows 服务（SCM）运行状态、自启状态与状态历史
- **服务管理**：start / stop / restart / enable（开机自启）/ disable（跨平台；需管理员权限）
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

### 构建单二进制（Linux + Windows）

```bash
bash scripts/build.sh
# 产出 bin/monitor-linux-amd64、monitor-windows-amd64.exe 与 monitor-linux-amd64-<日期>.tar.gz
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

> **权限说明**：默认 systemd 单元以低权限 `User=monitor` 运行，**只支持只读监控**。
> 如需启用「进程管理（结束/重启）+ 服务管理（启停/自启）」，
> 进程管理要求与目标进程同权限、systemd StartUnit/EnableUnitFiles 需要更高 D-Bus 策略权限，
> 请将 `deploy/monitor.service` 的 `User/Group` 改为 `root`（或为 monitor 配置 polkit/systemd 授权）后重新部署。
> 管理操作默认仅 `role=admin` 用户可执行。

### Windows 部署

1. 将 `bin/monitor-windows-amd64.exe` 与 `deploy/monitor.yaml` 放到目标机器（如 `C:\monitor\`）
2. **以管理员身份运行**（管理操作需要管理员权限，否则 kill 进程/启停服务返回 403 权限不足）
3. 可选：注册为 Windows 服务自启：
   ```bat
   sc.exe create monitor binPath= "C:\monitor\monitor-windows-amd64.exe -c C:\monitor\monitor.yaml" start= auto
   sc.exe start monitor
   ```
   或直接双击 exe 前台运行（调试用）。

访问 `http://<机器IP>:8080`，默认账号 **admin / admin123**（首次登录后请修改密码）。

## 进程与服务管理（应急处置）

登录后在「进程监控」页可对 top N 进程执行**结束 / 重启**（Popconfirm 二次确认）；
在「服务监控」页可按状态对服务执行**启动 / 停止 / 重启 / 开启自启 / 关闭自启**（按钮随状态自动切换）。
仅 `role=admin` 用户可见并允许操作；每次操作均在后端日志留痕（`[manager]`）。

> 进程「重启」会解析原进程的可执行路径与参数重新拉起（Linux 下脱离会话、Windows 下脱离进程组），
> 新进程属主为 monitor 运行用户，输出写入系统临时目录 `monitor-restart-*.log`。
> 内核线程、系统进程因无法获取可执行路径将返回「无法重启」。

## 告警与通知

### 邮件告警（快捷配置）

在「系统设置」→「邮件 SMTP」填入发件服务器/账号/授权码/收件人，点「保存 SMTP」并「测试发送」，
系统会自动生成名为「邮件告警」的 SMTP 通知渠道；之后在「告警规则」中绑定该渠道即可通过邮件接收告警。
密码留空表示不修改（首次配置必填）。

### 多渠道通知

1. 在「通知渠道」配置渠道（SMTP / Webhook / 飞书 / 企微 / 钉钉 / Server酱）并「测试发送」
2. 在「告警规则」新建规则（如：CPU 使用率 > 90% 持续 2 个周期），绑定渠道
3. 触发后「告警中心」实时显示，并通过渠道推送；恢复后自动标记 resolved

## 文档

- [架构设计](docs/01-architecture.md)
- [API 接口](docs/02-api.md)
