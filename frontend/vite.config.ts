import { fileURLToPath, URL } from 'node:url'
import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'
import { VitePWA } from 'vite-plugin-pwa'

// 后端地址：dev 时代理 /api 与 /health，避免跨域与 CORS 配置耦合
const BACKEND = process.env.VITE_BACKEND_ORIGIN || 'http://127.0.0.1:8080'

export default defineConfig({
  plugins: [
    vue(),
    VitePWA({
      registerType: 'autoUpdate',
      includeAssets: ['favicon.svg'],
      manifest: {
        name: '猫宅 MeowHome',
        short_name: '猫宅',
        description: '多猫家庭的生活与健康管理系统',
        theme_color: '#FFFCF7',
        background_color: '#F7F4EE',
        display: 'standalone',
        start_url: '.',
        icons: [
          { src: '/favicon.svg', sizes: 'any', type: 'image/svg+xml' }
        ]
      },
      workbox: {
        globPatterns: ['**/*.{js,css,html,svg,png,jpg,gif,woff2}'],
        // 业务接口一律走网络，禁止被 Service Worker 缓存，否则数据会“卡住”
        navigateFallbackDenylist: [/^\/api/, /^\/health/],
        runtimeCaching: []
      }
    })
  ],
  resolve: {
    alias: { '@': fileURLToPath(new URL('./src', import.meta.url)) }
  },
  server: {
    port: 5173,
    host: true,
    proxy: {
      '/api': {
        target: BACKEND,
        changeOrigin: true
      },
      '/health': {
        target: BACKEND,
        changeOrigin: true
      }
    }
  },
  test: {
    globals: true,
    environment: 'jsdom'
  }
})
