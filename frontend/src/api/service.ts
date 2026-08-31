import { get } from './client'
import type { ServiceState } from '../types'

export const listServices = (state?: string) =>
  get<ServiceState[]>(`/services${state ? `?state=${state}` : ''}`)

export const serviceNames = () => get<string[]>('/services/names')

export const serviceHistory = (name: string, from: number, to: number) => {
  const q = new URLSearchParams({ name, from: String(from), to: String(to) })
  return get<ServiceState[]>(`/services/history?${q}`)
}
