import { useEffect } from 'react'
import { realtimeSocket } from '../ws/socket'

// useRealtime 订阅 WebSocket 实时流，组件挂载时自动连接。
// topic 默认 'metric'（实时指标快照），可选 'alert'（新告警事件）。
export function useRealtime(topic: string, handler: (data: any) => void) {
  useEffect(() => {
    realtimeSocket.connect()
    return realtimeSocket.on(topic, handler)
  }, [topic, handler])
}

// useAlertRealtime 订阅告警事件流。
export function useAlertRealtime(handler: (data: any) => void) {
  return useRealtime('alert', handler)
}
