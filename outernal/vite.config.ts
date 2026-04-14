import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'

export default defineConfig({
  plugins: [react()],
  server: {
    port: 5175,
    strictPort: true,
    proxy: {
      '/api': {
        target: 'https://balooai.ru/hack',
        changeOrigin: true,
        secure: false,
      },
      '/ws': {
        target: 'wss://balooai.ru/hack',
        changeOrigin: true,
        ws: true,
        rewriteWsOrigin: true,
        secure: false,
      },
    },
  },
})
