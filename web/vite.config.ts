import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'
import { fileURLToPath, URL } from 'node:url'

// 构建产物直接进 Go 的 embed，所以 outDir 指向后端要嵌入的位置。
export default defineConfig({
  plugins: [vue()],
  resolve: {
    alias: { '@': fileURLToPath(new URL('./src', import.meta.url)) },
  },
  build: {
    // 直接产到 Go 的 embed 目录，省掉一步拷贝：改完前端 pnpm build 再 go build 即可。
    outDir: '../internal/web/dist',
    emptyOutDir: true,
  },
  server: {
    port: 5173,
    // 开发期前端独立跑，接口与 WebSocket 都代理到后端。
    proxy: {
      '/api': {
        target: 'http://127.0.0.1:8777',
        changeOrigin: true,
        ws: true,
      },
    },
  },
})
