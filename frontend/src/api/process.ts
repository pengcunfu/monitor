import { get, post } from './client'
import type { MetricPoint, ProcessSample } from '../types'

export const listProcesses = (top = 20, sort = 'cpu') =>
  get<ProcessSample[]>(`/process/current?top=${top}&sort=${sort}`)

export const processHistory = (name: string, from: number, to: number) => {
  const q = new URLSearchParams({ name, from: String(from), to: String(to) })
  return get<{ cpu: MetricPoint[]; mem: MetricPoint[] }>(`/process/history?${q}`)
}

export const processNames = () => get<string[]>('/process/names')

// —— 进程管理（仅管理员，后端 adminOnly 校验）——
export interface ProcessOpResult {
  pid: number
  name: string
  exe?: string
  new_pid?: number
  action: string
}

export const killProcess = (pid: number) => post<ProcessOpResult>(`/process/${pid}/kill`)
export const restartProcess = (pid: number) => post<ProcessOpResult>(`/process/${pid}/restart`)
