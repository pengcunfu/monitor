import { Card, Statistic } from 'antd'
import type { ReactNode } from 'react'

interface Props {
  title: string
  value: number | string
  unit?: string
  precision?: number
  color?: string
  suffix?: ReactNode
  extra?: ReactNode
}

// StatCard 概览数值卡。
export default function StatCard({ title, value, unit, precision, color, suffix, extra }: Props) {
  return (
    <Card size="small" title={title} extra={extra}>
      <Statistic
        value={value as number}
        precision={precision}
        suffix={unit}
        valueStyle={{ color, fontSize: 22 }}
      />
      {suffix}
    </Card>
  )
}
