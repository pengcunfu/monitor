import { get, post } from './client'
import type { AlertEvent, PageData } from '../types'

export interface AlertQuery {
  status?: string
  page?: number
  size?: number
}

export const listAlerts = (params: AlertQuery) => {
  const q = new URLSearchParams()
  if (params.status) q.set('status', params.status)
  q.set('page', String(params.page ?? 1))
  q.set('size', String(params.size ?? 20))
  return get<PageData<AlertEvent>>(`/alerts?${q}`)
}

export const ackAlert = (id: number) => post(`/alerts/${id}/ack`)

export const alertStats = () => get<{ fired: number; resolved: number }>('/alerts/stats')

export const firingAlerts = (limit = 10) => get<AlertEvent[]>(`/alerts/firing?limit=${limit}`)
