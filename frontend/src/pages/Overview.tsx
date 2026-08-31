import { useCallback, useMemo, useState } from 'react'
import { Card, Col, Progress, Row } from 'antd'
import useSWR from 'swr'
import { getHistory, getLatest } from '../api/metrics'
import { useRealtime } from '../hooks/useRealtime'
import StatCard from '../components/StatCard'
import LineChart from '../components/charts/LineChart'
import { formatBytes, formatSpeed, formatUptime } from '../utils/format'
import type { MetricPoint, MetricSnapshot } from '../types'

// mergeLive 把实时快照作为曲线尾点追加到历史序列后（避免重复点）。
function mergeLive(points: MetricPoint[] | undefined, liveValue: number | undefined, liveTs: number | undefined) {
  if (!points || points.length === 0) {
    return liveValue === undefined ? [] : [{ ts: liveTs ?? Date.now(), value: liveValue }]
  }
  if (liveValue !== undefined && liveTs !== undefined && liveTs > points[points.length - 1].ts) {
    return [...points, { ts: liveTs, value: liveValue }]
  }
  return points
}

// Overview 总览页：当前指标（WS 实时）+ 近 1 小时趋势。
export default function Overview() {
  const { data: snap, isLoading } = useSWR('/metrics/latest', getLatest, {
    refreshInterval: 10000,
  })
  const [liveSnap, setLiveSnap] = useState<MetricSnapshot | null>(null)
  const onMetric = useCallback((data: any) => setLiveSnap(data as MetricSnapshot), [])
  useRealtime('metric', onMetric)
  const ov = liveSnap ?? snap ?? null

  const { data: cpuHis } = useSWR('/history/cpu', () => getHistory('cpu_usage', Date.now() - 3600_000, Date.now()), {
    refreshInterval: 60000,
  })
  const { data: memHis } = useSWR('/history/mem', () => getHistory('mem_usage', Date.now() - 3600_000, Date.now()), {
    refreshInterval: 60000,
  })
  const { data: rxHis } = useSWR('/history/rx', () => getHistory('net_rx_bps', Date.now() - 3600_000, Date.now()), {
    refreshInterval: 60000,
  })
  const { data: txHis } = useSWR('/history/tx', () => getHistory('net_tx_bps', Date.now() - 3600_000, Date.now()), {
    refreshInterval: 60000,
  })

  const live = useMemo(
    () => ({
      cpu: mergeLive(cpuHis, liveSnap?.cpu_usage, liveSnap?.ts),
      mem: mergeLive(memHis, liveSnap ? (liveSnap.mem_used / liveSnap.mem_total) * 100 : undefined, liveSnap?.ts),
      rx: mergeLive(rxHis, liveSnap?.net_rx_bps, liveSnap?.ts),
      tx: mergeLive(txHis, liveSnap?.net_tx_bps, liveSnap?.ts),
    }),
    [cpuHis, memHis, rxHis, txHis, liveSnap],
  )

  const topDisk = ov?.disk_usage?.reduce((max, d) => (d.used_percent > max.used_percent ? d : max), ov?.disk_usage?.[0])
  const memUsedPct = ov ? (ov.mem_used / ov.mem_total) * 100 : 0

  return (
    <div>
      <Row gutter={[16, 16]}>
        <Col xs={12} sm={8} md={6} lg={4}>
          <StatCard title="CPU 使用率" value={ov?.cpu_usage ?? 0} unit="%" precision={1} color="#fa541c" />
        </Col>
        <Col xs={12} sm={8} md={6} lg={4}>
          <StatCard title="内存使用率" value={memUsedPct} unit="%" precision={1} color="#1677ff" />
        </Col>
        <Col xs={12} sm={8} md={6} lg={4}>
          <StatCard title="系统负载 (1m)" value={ov?.load1 ?? 0} precision={2} color="#722ed1" />
        </Col>
        <Col xs={12} sm={8} md={6} lg={4}>
          <StatCard title="磁盘使用率" value={topDisk?.used_percent ?? 0} unit="%" precision={1} color="#fa8c16" />
        </Col>
        <Col xs={12} sm={8} md={6} lg={4}>
          <StatCard title="网络入带宽" value={ov?.net_rx_bps ?? 0} unit="B/s" color="#13c2c2" />
        </Col>
        <Col xs={12} sm={8} md={6} lg={4}>
          <StatCard title="网络出带宽" value={ov?.net_tx_bps ?? 0} unit="B/s" color="#52c41a" />
        </Col>
      </Row>

      <Row gutter={[16, 16]} style={{ marginTop: 16 }}>
        <Col xs={24} lg={12}>
          <Card size="small" title="CPU 使用率（近 1 小时）" loading={isLoading}>
            <LineChart
              series={[{ name: 'CPU %', points: live.cpu, color: '#fa541c' }]}
              yName="%"
              formatter={(v) => v.toFixed(1) + '%'}
            />
          </Card>
        </Col>
        <Col xs={24} lg={12}>
          <Card size="small" title="内存使用率（近 1 小时）" loading={isLoading}>
            <LineChart
              series={[{ name: '内存 %', points: live.mem, color: '#1677ff' }]}
              yName="%"
              formatter={(v) => v.toFixed(1) + '%'}
            />
          </Card>
        </Col>
        <Col xs={24} lg={12}>
          <Card size="small" title="网络带宽（近 1 小时）" loading={isLoading}>
            <LineChart
              series={[
                { name: '入带宽', points: live.rx, color: '#13c2c2' },
                { name: '出带宽', points: live.tx, color: '#52c41a' },
              ]}
              yName="B/s"
              formatter={(v) => formatSpeed(v)}
            />
          </Card>
        </Col>
        <Col xs={24} lg={12}>
          <Card size="small" title="磁盘分区使用情况" loading={isLoading}>
            {(ov?.disk_usage ?? []).map((d) => (
              <div key={d.mount} style={{ marginBottom: 12 }}>
                <div style={{ display: 'flex', justifyContent: 'space-between', marginBottom: 4 }}>
                  <span>
                    {d.mount}
                    <span style={{ color: '#999', marginLeft: 8, fontSize: 12 }}>{d.fs}</span>
                  </span>
                  <span>
                    {d.used_percent.toFixed(1)}% （已用 {formatBytes(d.used)} / {formatBytes(d.total)}）
                  </span>
                </div>
                <Progress
                  percent={Math.round(d.used_percent)}
                  status={d.used_percent > 90 ? 'exception' : d.used_percent > 80 ? 'active' : 'normal'}
                />
              </div>
            ))}
          </Card>
        </Col>
      </Row>

      <Row gutter={[16, 16]} style={{ marginTop: 16 }}>
        <Col span={24}>
          <Card size="small">
            <span style={{ color: '#666' }}>
              主机：<b>{ov?.host_name ?? '-'}</b> ｜ 运行时长：{formatUptime(ov?.uptime_sec ?? 0)} ｜ 逻辑核数：
              {ov?.cpu_cores ?? '-'}
            </span>
          </Card>
        </Col>
      </Row>
    </div>
  )
}
