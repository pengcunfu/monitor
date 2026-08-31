import { get } from './client'
import type { MetricPoint, MetricSnapshot, OverviewData } from '../types'

export const getOverview = () => get<OverviewData>('/overview')

export const getLatest = () => get<MetricSnapshot>('/metrics/latest')

export const getHistory = (metric: string, from: number, to: number, target?: string) => {
  const q = new URLSearchParams({ metric, from: String(from), to: String(to) })
  if (target) q.set('target', target)
  return get<MetricPoint[]>(`/metrics/history?${q}`)
}
