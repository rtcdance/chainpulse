import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'

export default defineConfig({
  plugins: [react()],
  server: {
    port: 3000,
    proxy: {
      '/events': {
        target: 'http://localhost:8080',
        changeOrigin: true,
        ws: true,
      },
      '/api': {
        target: 'http://localhost:8080',
        changeOrigin: true
      },
      '/health': {
        target: 'http://localhost:8080',
        changeOrigin: true,
      },
      '/metrics': {
        target: 'http://localhost:8080',
        changeOrigin: true,
      },
      '/runtime': {
        target: 'http://localhost:8080',
        changeOrigin: true,
      },
      '/graphql': {
        target: 'http://localhost:8080',
        changeOrigin: true,
      },
      '/models': {
        target: 'http://localhost:8080',
        changeOrigin: true,
      },
      '/ws': {
        target: 'ws://localhost:8080',
        changeOrigin: true,
        ws: true,
      },
      '/__proxy/api-gateway': {
        target: 'http://localhost:8080',
        changeOrigin: true,
        rewrite: (path) => path.replace(/^\/__proxy\/api-gateway/, ''),
      },
      '/__proxy/api-service': {
        target: 'http://localhost:8081',
        changeOrigin: true,
        rewrite: (path) => path.replace(/^\/__proxy\/api-service/, ''),
      },
      '/__proxy/event-processor': {
        target: 'http://localhost:8082',
        changeOrigin: true,
        rewrite: (path) => path.replace(/^\/__proxy\/event-processor/, ''),
      },
      '/__proxy/puller': {
        target: 'http://localhost:8083',
        changeOrigin: true,
        rewrite: (path) => path.replace(/^\/__proxy\/puller/, ''),
      },
      '/__ws/api-gateway': {
        target: 'ws://localhost:8080',
        changeOrigin: true,
        ws: true,
        rewrite: (path) => path.replace(/^\/__ws\/api-gateway/, ''),
      },
      '/__ws/api-service': {
        target: 'ws://localhost:8081',
        changeOrigin: true,
        ws: true,
        rewrite: (path) => path.replace(/^\/__ws\/api-service/, ''),
      },
    }
  }
})
