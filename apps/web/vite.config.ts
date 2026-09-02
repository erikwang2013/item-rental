import { defineConfig } from 'vite'
import path from 'path'

export default defineConfig({
  resolve: {
    alias: { '@': path.resolve(__dirname, 'src') },
  },
  server: {
    proxy: {
      '/api/v1': { target: 'http://127.0.0.1:8080', changeOrigin: true },
    },
  },
})
