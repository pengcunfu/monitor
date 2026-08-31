import { get, put } from './client'

export const getSettings = () => get<Record<string, any>>('/settings')

export const updateSettings = (data: Record<string, any>) => put('/settings', data)
