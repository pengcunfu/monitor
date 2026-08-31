import { get, post, put } from './client'
import type { User } from '../types'

export interface LoginResult {
  token: string
  user: User
}

export const login = (username: string, password: string) =>
  post<LoginResult>('/auth/login', { username, password })

export const me = () => get<User>('/auth/me')

export const changePassword = (old_password: string, new_password: string) =>
  put('/auth/password', { old_password, new_password })
