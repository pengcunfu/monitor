import { get, post } from './client'
import type { ServiceState } from '../types'

export const listServices = (state?: string) =>
  get<ServiceState[]>(`/services${state ? `?state=${state}` : ''}`)

export const serviceNames = () => get<string[]>('/services/names')

export const serviceHistory = (name: string, from: number, to: number) => {
  const q = new URLSearchParams({ name, from: String(from), to: String(to) })
  return get<ServiceState[]>(`/services/history?${q}`)
}

// —— 服务管理（仅管理员，后端 adminOnly 校验）——
export const serviceAction = (name: string, action: string) =>
  post<{ name: string; action: string }>(`/services/${encodeURIComponent(name)}/${action}`)
export const startService = (name: string) => serviceAction(name, 'start')
export const stopService = (name: string) => serviceAction(name, 'stop')
export const restartService = (name: string) => serviceAction(name, 'restart')
export const enableService = (name: string) => serviceAction(name, 'enable')
export const disableService = (name: string) => serviceAction(name, 'disable')
