import { useAuthStore } from '../store/auth'

const BASE = '/api/v1'

export class ApiError extends Error {
  status: number
  code: number
  constructor(status: number, code: number, msg: string) {
    super(msg)
    this.status = status
    this.code = code
  }
}

interface ResponseBody<T> {
  code: number
  msg: string
  data: T
}

// request 统一的 fetch 封装：注入 token、统一 JSON 解析与错误处理。
export async function request<T = any>(path: string, options: RequestInit = {}): Promise<T> {
  const token = useAuthStore.getState().token
  const headers: Record<string, string> = {
    'Content-Type': 'application/json',
    ...((options.headers as Record<string, string>) || {}),
  }
  if (token) headers.Authorization = `Bearer ${token}`

  let resp: Response
  try {
    resp = await fetch(BASE + path, { ...options, headers })
  } catch {
    throw new ApiError(0, 0, '网络请求失败，请确认后端服务已启动')
  }

  let body: ResponseBody<T> | null = null
  try {
    body = await resp.json()
  } catch {
    // 非 JSON 响应
  }

  if (resp.status === 401) {
    useAuthStore.getState().logout()
    if (window.location.pathname !== '/login') {
      window.location.href = '/login'
    }
    throw new ApiError(401, 401, '登录已过期，请重新登录')
  }

  if (!resp.ok || !body || body.code !== 0) {
    throw new ApiError(resp.status, body?.code ?? resp.status, body?.msg || `请求失败 (${resp.status})`)
  }
  return body.data as T
}

export const get = <T = any>(path: string) => request<T>(path)
export const post = <T = any>(path: string, data?: any) =>
  request<T>(path, { method: 'POST', body: data === undefined ? undefined : JSON.stringify(data) })
export const put = <T = any>(path: string, data?: any) =>
  request<T>(path, { method: 'PUT', body: data === undefined ? undefined : JSON.stringify(data) })
export const del = <T = any>(path: string) => request<T>(path, { method: 'DELETE' })
