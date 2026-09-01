import { get, post, put } from './client'

export interface SMTPSetting {
  host: string
  port: number
  user: string
  password: string
  from: string
  to: string[] // 接口返回数组，表单中转为逗号分隔串
  tls: boolean
  insecure_skip_verify: boolean
  enabled: boolean
}

export const getSettings = () => get<Record<string, any>>('/settings')

export const updateSettings = (data: Record<string, any>) => put('/settings', data)

// 用当前表单配置发送测试邮件（未保存也可测试）
export const testSMTPSetting = (smtp: SMTPSetting) => post('/settings/smtp/test', { smtp })
