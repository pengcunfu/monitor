import { useMemo } from 'react'
import ReactECharts from 'echarts-for-react'
import type { EChartsOption } from 'echarts'

interface Props {
  option: EChartsOption
  height?: number
  loading?: boolean
}

// BaseChart ECharts 统一封装：响应式尺寸、加载态。
export default function BaseChart({ option, height = 260, loading = false }: Props) {
  const merged = useMemo<EChartsOption>(
    () => ({
      grid: { left: 12, right: 16, top: 32, bottom: 8, containLabel: true },
      tooltip: { trigger: 'axis' },
      animation: false,
      ...option,
    }),
    [option],
  )
  return (
    <ReactECharts
      option={merged}
      notMerge
      style={{ height }}
      showLoading={loading}
      loadingOption={{ text: '加载中…' }}
      opts={{ renderer: 'canvas' }}
    />
  )
}
