import { del, get, post, put } from './client'
import type { Channel } from '../types'

export const listChannels = () => get<Channel[]>('/channels')

export const channelTypes = () => get<{ type: string; name: string }[]>('/channels/types')

export const createChannel = (data: any) => post<Channel>('/channels', data)

export const updateChannel = (id: number, data: any) => put<Channel>(`/channels/${id}`, data)

export const deleteChannel = (id: number) => del(`/channels/${id}`)

export const testChannel = (id: number) => post(`/channels/${id}/test`)
