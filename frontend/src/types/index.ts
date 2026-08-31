// 与后端 internal/model 对齐的类型定义

export interface User {
  id: number
  username: string
  role: string
}

export interface DiskUsage {
  mount: string
  fs: string
  total: number
  used: number
  used_percent: number
}

export interface DiskIORate {
  device: string
  read_bps: number
  write_bps: number
  iops: number
}

export interface NetRate {
  name: string
  rx_bytes: number
  tx_bytes: number
  rx_bps: number
  tx_bps: number
}

export interface MetricSnapshot {
  id: number
  ts: number
  host_name: string
  cpu_usage: number
  cpu_cores: number
  load1: number
  load5: number
  load15: number
  mem_total: number
  mem_used: number
  mem_avail: number
  mem_usage: number
  swap_total: number
  swap_used: number
  disk_usage: DiskUsage[]
  disk_io_rates: DiskIORate[]
  net: NetRate[]
  net_rx_bps: number
  net_tx_bps: number
  uptime_sec: number
}

export interface OverviewData {
  ts: number
  host_name: string
  uptime_sec: number
  cpu_usage: number
  cpu_cores: number
  load1: number
  load5: number
  load15: number
  mem_total: number
  mem_used: number
  mem_avail: number
  mem_usage: number
  swap_total: number
  swap_used: number
  net_rx_bps: number
  net_tx_bps: number
  disk_usage: DiskUsage[]
  net: NetRate[]
}

export interface MetricPoint {
  ts: number
  value: number
}

export interface ProcessSample {
  id: number
  ts: number
  pid: number
  name: string
  user: string
  cpu_percent: number
  mem_percent: number
  mem_rss: number
  state: string
  cmdline: string
}

export interface ServiceState {
  id: number
  ts: number
  name: string
  description: string
  load_state: string
  active_state: string
  sub_state: string
  is_active: boolean
  main_pid: number
  exit_code: number
}

export interface AlertRule {
  id: number
  name: string
  metric: string
  target: string
  operator: string
  threshold: number
  duration_ticks: number
  severity: 'critical' | 'warning'
  channel_ids: number[]
  cooldown_sec: number
  notify_on_resolve: boolean
  enabled: boolean
  description: string
  created_at: number
}

export interface AlertEvent {
  id: number
  rule_id: number
  rule_name: string
  metric: string
  target: string
  severity: 'critical' | 'warning'
  status: 'firing' | 'resolved'
  message: string
  value: number
  threshold: number
  fired_at: number
  resolved_at: number
  duration_sec: number
  notified: boolean
  notify_at: number
  acked: boolean
  ack_by: string
  ack_at: number
}

export interface Channel {
  id: number
  name: string
  type: string
  config: Record<string, any>
  enabled: boolean
  created_at: number
}

export interface PageData<T> {
  list: T[]
  total: number
  page: number
  size: number
}
