import { useMemo } from 'react'
import type { EChartsOption } from 'echarts'
import BaseChart from './BaseChart'
import type { MetricPoint } from '../../types'

interface Series {
  name: string
  points: MetricPoint[]
  color?: string
}

interface Props {
  series: Series[]
  yName?: string
  formatter?: (v: number) => string
  height?: number
  loading?: boolean
}

// LineChart 时间序列折线图（多系列）。
export default function LineChart({ series, yName, formatter, height, loading }: Props) {
  const option = useMemo<EChartsOption>(() => {
    return {
      legend: { top: 0, type: 'scroll' },
      xAxis: { type: 'time' },
      yAxis: {
        type: 'value',
        name: yName,
        axisLabel: { formatter: (v: number) => (formatter ? formatter(v) : String(v)) },
      },
      series: series
        .filter((s) => s.points.length > 0)
        .map((s) => ({
          name: s.name,
          type: 'line',
          showSymbol: false,
          smooth: true,
          sampling: 'lttb',
          lineStyle: { width: 1.5, color: s.color },
          itemStyle: { color: s.color },
          data: s.points.map((p) => [p.ts, p.value]),
        })),
    }
  }, [series, yName, formatter])

  return <BaseChart option={option} height={height} loading={loading} />
}
