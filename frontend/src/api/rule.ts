import { del, get, post, put } from './client'
import type { AlertRule, PageData } from '../types'

export const listRules = (page = 1, size = 20) =>
  get<PageData<AlertRule>>(`/rules?page=${page}&size=${size}`)

export const createRule = (data: Partial<AlertRule>) => post<AlertRule>('/rules', data)

export const updateRule = (id: number, data: Partial<AlertRule>) => put<AlertRule>(`/rules/${id}`, data)

export const deleteRule = (id: number) => del(`/rules/${id}`)

export const toggleRule = (id: number, enabled: boolean) => put(`/rules/${id}/toggle?enabled=${enabled}`)

export const reloadRules = () => post('/rules/reload')
