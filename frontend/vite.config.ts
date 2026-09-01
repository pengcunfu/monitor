import react from '@vitejs/plugin-react'
import { defineConfig } from 'vite'

// https://vite.dev/config/
export default defineConfig({
  plugins: [react()],
  server: {
    port: 5173,
    proxy: {
      // 开发期代理到 Go 后端。前端实时大屏通过 /api/v1/ws 建立 WebSocket，
      // 必须开启 ws:true 才能代理 upgrade 握手（否则实时数据收不到）。
      '/api': { target: 'http://localhost:8080', changeOrigin: true, ws: true },
    },
  },
  build: {
    // 构建产物输出到 backend/web/dist（供 Go embed 打包为单二进制）
    outDir: '../backend/web/dist',
    emptyOutDir: true,
    chunkSizeWarningLimit: 1500,
  },
})
