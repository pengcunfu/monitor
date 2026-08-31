import { useMemo, useState } from 'react'
import { App, Button, Card, Col, Row, Select, Space } from 'antd'
import useSWR from 'swr'
import { getHistory } from '../api/metrics'
import { getLatest } from '../api/metrics'
import LineChart from '../components/charts/LineChart'
import type { MetricPoint } from '../types'
import { formatSpeed } from '../utils/format'

const METRIC_OPTIONS = [
  { value: 'cpu_usage', label: 'CPU 使用率', color: '#fa541c', unit: '%' },
  { value: 'mem_usage', label: '内存使用率', color: '#1677ff', unit: '%' },
  { value: 'load1', label: '系统负载 1m', color: '#722ed1', unit: '' },
  { value: 'net_rx_bps', label: '网络入带宽', color: '#13c2c2', unit: 'B/s' },
  { value: 'net_tx_bps', label: '网络出带宽', color: '#52c41a', unit: 'B/s' },
  { value: 'disk_used_percent', label: '磁盘使用率', color: '#fa8c16', unit: '%' },
]

const RANGES = [
  { label: '1 小时', ms: 3600_000 },
  { label: '6 小时', ms: 6 * 3600_000 },
  { label: '24 小时', ms: 24 * 3600_000 },
  { label: '7 天', ms: 7 * 24 * 3600_000 },
]

// History 历史查询页：按时间范围 + 指标多选展示趋势曲线。
export default function History() {
  const [rangeMs, setRangeMs] = useState(3600_000)
  const [metrics, setMetrics] = useState<string[]>(['cpu_usage', 'mem_usage'])
  const [diskMount, setDiskMount] = useState<string>()
  const { message } = App.useApp()

  // 磁盘分区列表（用于 disk_used_percent 的 target）
  const { data: latest } = useSWR('/metrics/latest', getLatest)

  const range = useMemo(() => {
    const to = Date.now()
    return { from: to - rangeMs, to }
  }, [rangeMs])

  const key = useMemo(
    () => ['/history', metrics.join(','), diskMount ?? '', range.from, range.to] as const,
    [metrics, diskMount, range],
  )

  const { data, isLoading } = useSWR(key, async () => {
    const series: { name: string; points: MetricPoint[]; color?: string }[] = []
    for (const m of metrics) {
      const meta = METRIC_OPTIONS.find((x) => x.value === m)
      if (!meta) continue
      const target = m === 'disk_used_percent' ? diskMount : undefined
      const points = await getHistory(m, range.from, range.to, target)
      series.push({ name: meta.label, points, color: meta.color })
    }
    return series
  })

  const formatter = (v: number) => {
    // 按当前指标集合推断单位
    if (metrics.some((m) => m === 'net_rx_bps' || m === 'net_tx_bps')) return formatSpeed(v)
    return v.toFixed(1)
  }

  return (
    <div>
      <Card size="small" style={{ marginBottom: 16 }}>
        <Space wrap>
          <span>时间范围：</span>
          {RANGES.map((r) => (
            <Button
              key={r.ms}
              size="small"
              type={rangeMs === r.ms ? 'primary' : 'default'}
              onClick={() => setRangeMs(r.ms)}
            >
              {r.label}
            </Button>
          ))}
          <span style={{ marginLeft: 16 }}>指标：</span>
          <Select
            mode="multiple"
            style={{ minWidth: 320 }}
            placeholder="选择指标"
            value={metrics}
            onChange={(v: string[]) => {
              if (v.length === 0) {
                message.warning('请至少选择一个指标')
                return
              }
              setMetrics(v)
            }}
            options={METRIC_OPTIONS.map((m) => ({ value: m.value, label: m.label }))}
          />
          {metrics.includes('disk_used_percent') && (
            <Select
              style={{ minWidth: 140 }}
              placeholder="选择分区（默认全部）"
              allowClear
              value={diskMount}
              onChange={setDiskMount}
              options={(latest?.disk_usage ?? []).map((d) => ({
                value: d.mount,
                label: `${d.mount} (${d.used_percent.toFixed(1)}%)`,
              }))}
            />
          )}
        </Space>
      </Card>
      <Card size="small" loading={isLoading}>
        <LineChart series={data ?? []} yName="值" formatter={formatter} height={420} />
        {(data ?? []).length === 0 && !isLoading && (
          <div style={{ textAlign: 'center', color: '#999', padding: 40 }}>当前时间范围暂无数据</div>
        )}
      </Card>
      <Row style={{ marginTop: 8 }}>
        <Col span={24}>
          <span style={{ color: '#999', fontSize: 12 }}>
            提示：时间范围超过 2 小时时服务端自动降采样；图表支持滚轮缩放。
          </span>
        </Col>
      </Row>
    </div>
  )
}
