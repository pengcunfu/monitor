import { get } from './client'
import type { MetricPoint, ProcessSample } from '../types'

export const listProcesses = (top = 20, sort = 'cpu') =>
  get<ProcessSample[]>(`/process/current?top=${top}&sort=${sort}`)

export const processHistory = (name: string, from: number, to: number) => {
  const q = new URLSearchParams({ name, from: String(from), to: String(to) })
  return get<{ cpu: MetricPoint[]; mem: MetricPoint[] }>(`/process/history?${q}`)
}

export const processNames = () => get<string[]>('/process/names')
