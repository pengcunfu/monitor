import { useAuthStore } from '../store/auth'

type Handler = (data: any) => void

// RealtimeSocket 单例 WebSocket 客户端：自动重连、心跳、按主题分发。
class RealtimeSocket {
  private ws: WebSocket | null = null
  private handlers = new Map<string, Handler>()
  private reconnectTimer: ReturnType<typeof setTimeout> | null = null
  private heartbeatTimer: ReturnType<typeof setInterval> | null = null
  private manualClose = false

  // on 注册主题处理器，返回取消函数。
  on(topic: string, handler: Handler): () => void {
    this.handlers.set(topic, handler)
    return () => {
      if (this.handlers.get(topic) === handler) this.handlers.delete(topic)
    }
  }

  connect() {
    const token = useAuthStore.getState().token
    if (!token || this.ws) return
    this.manualClose = false

    const proto = window.location.protocol === 'https:' ? 'wss' : 'ws'
    this.ws = new WebSocket(`${proto}://${window.location.host}/api/v1/ws?token=${encodeURIComponent(token)}`)

    this.ws.onopen = () => {
      this.heartbeatTimer = setInterval(() => {
        this.ws?.send(JSON.stringify({ type: 'ping' }))
      }, 30000)
    }

    this.ws.onmessage = (e) => {
      try {
        const frame = JSON.parse(e.data)
        const h = this.handlers.get(frame.type)
        if (h) h(frame.data)
      } catch {
        // 忽略解析失败帧
      }
    }

    this.ws.onclose = () => {
      this.cleanup()
      if (!this.manualClose) this.scheduleReconnect()
    }
    this.ws.onerror = () => this.ws?.close()
  }

  close() {
    this.manualClose = true
    if (this.ws) {
      this.ws.onclose = null
      this.ws.close()
      this.ws = null
    }
    this.cleanup()
  }

  private cleanup() {
    if (this.heartbeatTimer) {
      clearInterval(this.heartbeatTimer)
      this.heartbeatTimer = null
    }
  }

  private scheduleReconnect() {
    if (this.reconnectTimer) return
    this.reconnectTimer = setTimeout(() => {
      this.reconnectTimer = null
      this.ws = null
      this.connect()
    }, 3000)
  }
}

export const realtimeSocket = new RealtimeSocket()
