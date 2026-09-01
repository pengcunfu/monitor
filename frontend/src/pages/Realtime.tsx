import { useCallback, useRef, useState } from 'react'
import { ArrowDownOutlined, ArrowUpOutlined } from '@ant-design/icons'
import { Card, Col, Row } from 'antd'
import { useRealtime } from '../hooks/useRealtime'
import BaseChart from '../components/charts/BaseChart'
import type { EChartsOption } from 'echarts'
import type { MetricPoint, MetricSnapshot } from '../types'
import { formatBytes, formatSpeed } from '../utils/format'

const MAX = 300 // 环形缓冲大小

function useRingChart(label: string, color: string, formatter?: (v: number) => string) {
  const buf = useRef<MetricPoint[]>([])
  const [points, setPoints] = useState<MetricPoint[]>([])

  const push = useCallback((ts: number, value: number) => {
    const arr = buf.current
    arr.push({ ts, value })
    if (arr.length > MAX) arr.shift()
    setPoints([...arr])
  }, [])

  const option: EChartsOption = {
    title: { text: label, left: 8, top: 6, textStyle: { fontSize: 14 } },
    grid: { left: 12, right: 16, top: 40, bottom: 8, containLabel: true },
    tooltip: {
      trigger: 'axis',
      valueFormatter: formatter ? (v) => formatter(Number(v)) : undefined,
    },
    xAxis: { type: 'time' },
    yAxis: { type: 'value', axisLabel: { formatter: (v: number) => (formatter ? formatter(v) : String(v)) } },
    series: [
      {
        name: label,
        type: 'line',
        showSymbol: false,
        animation: false,
        lineStyle: { width: 1.5, color },
        itemStyle: { color },
        areaStyle: { color, opacity: 0.08 },
        data: points.map((p) => [p.ts, p.value]),
      },
    ],
  }
  return { option, push }
}

// Realtime 实时大屏：纯 WebSocket 推送 + 环形缓冲曲线。
export default function Realtime() {
  const [snap, setSnap] = useState<MetricSnapshot | null>(null)
  const cpu = useRingChart('CPU 使用率 %', '#fa541c', (v) => v.toFixed(1) + '%')
  const mem = useRingChart('内存使用率 %', '#1677ff', (v) => v.toFixed(1) + '%')
  const rx = useRingChart('网络入带宽', '#13c2c2', formatSpeed)
  const tx = useRingChart('网络出带宽', '#52c41a', formatSpeed)

  useRealtime(
    'metric',
    useCallback(
      (data: MetricSnapshot) => {
        setSnap(data)
        cpu.push(data.ts, data.cpu_usage)
        mem.push(data.ts, data.mem_usage)
        rx.push(data.ts, data.net_rx_bps)
        tx.push(data.ts, data.net_tx_bps)
      },
      [cpu, mem, rx, tx],
    ),
  )

  const totalCpu = snap?.cpu_cores ?? 0

  return (
    <div>
      <Row gutter={[16, 16]}>
        <Col xs={24} sm={12} xl={6}>
          <Card size="small" styles={{ body: { padding: 12 } }}>
            <div style={{ fontSize: 13, color: '#666' }}>主机：{snap?.host_name ?? '—'}</div>
            <div style={{ fontSize: 13, color: '#666' }}>逻辑核数：{totalCpu}</div>
            <div style={{ fontSize: 13, color: '#666' }}>
              内存：{snap ? `${formatBytes(snap.mem_used)} / ${formatBytes(snap.mem_total)}` : '—'}
            </div>
            <div style={{ fontSize: 13, color: '#666' }}>
              负载 1/5/15：{snap?.load1 ?? 0} / {snap?.load5 ?? 0} / {snap?.load15 ?? 0}
            </div>
          </Card>
        </Col>
        <Col xs={24} sm={12} xl={6}>
          <Card size="small" styles={{ body: { padding: 12 } }}>
            <div style={{ fontSize: 13, color: '#666' }}>
              磁盘分区（使用率）：
            </div>
            {(snap?.disk_usage ?? []).map((d) => (
              <div key={d.mount} style={{ fontSize: 13, color: '#666' }}>
                {d.mount}：{d.used_percent.toFixed(1)}%
              </div>
            ))}
          </Card>
        </Col>
        <Col xs={24} sm={12} xl={6}>
          <Card size="small" styles={{ body: { padding: 12 } }}>
            <div style={{ fontSize: 13, color: '#666' }}>网卡速率：</div>
            {(snap?.net ?? [])
              .filter((n) => n.rx_bps > 0 || n.tx_bps > 0)
              .map((n) => (
                <div key={n.name} style={{ fontSize: 13, color: '#666' }}>
                  {n.name}：
                  <ArrowDownOutlined style={{ color: '#13c2c2', fontSize: 10 }} />
                  {formatSpeed(n.rx_bps)}{' '}
                  <ArrowUpOutlined style={{ color: '#52c41a', fontSize: 10 }} />
                  {formatSpeed(n.tx_bps)}
                </div>
              ))}
          </Card>
        </Col>
        <Col xs={24} sm={12} xl={6}>
          <Card size="small" styles={{ body: { padding: 12 } }}>
            <div style={{ fontSize: 13, color: '#666' }}>磁盘 IO：</div>
            {(snap?.disk_io_rates ?? []).map((d) => (
              <div key={d.device} style={{ fontSize: 13, color: '#666' }}>
                {d.device}：读 {formatSpeed(d.read_bps)} 写 {formatSpeed(d.write_bps)}
              </div>
            ))}
          </Card>
        </Col>
      </Row>

      <Row gutter={[16, 16]} style={{ marginTop: 16 }}>
        <Col xs={24} lg={12}>
          <BaseChart option={cpu.option} height={280} />
        </Col>
        <Col xs={24} lg={12}>
          <BaseChart option={mem.option} height={280} />
        </Col>
        <Col xs={24} lg={12}>
          <BaseChart option={rx.option} height={280} />
        </Col>
        <Col xs={24} lg={12}>
          <BaseChart option={tx.option} height={280} />
        </Col>
      </Row>
    </div>
  )
}
