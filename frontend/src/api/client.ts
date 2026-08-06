// API 客户端 — 统一 Axios 实例（真实后端联调时只需替换 timeout/baseURL/Mock 出口）
import axios from 'axios'

export const baseURL: string = import.meta.env.VITE_API_BASE_URL || '/api'

// 普通业务 API 客户端
export const http = axios.create({
  baseURL,
  timeout: Number(import.meta.env.VITE_API_TIMEOUT || 15000)
})

// AI 专用客户端：独立超时，不影响普通业务 API
export const aiHttp = axios.create({
  baseURL,
  timeout: Number(import.meta.env.VITE_AI_TIMEOUT || 30000)
})

// 请求拦截：携带认证 + 注入 request_id
function authHeader(config: import('axios').InternalAxiosRequestConfig) {
  const token = localStorage.getItem('meowhome.token')
  if (token) config.headers.set('Authorization', `Bearer ${token}`)
  config.headers.set('X-Request-Id', crypto?.randomUUID?.() || `req-${Date.now()}`)
  return config
}

http.interceptors.request.use(authHeader)
aiHttp.interceptors.request.use(authHeader)

// 统一错误处理（真实后端联调时启用）
function errorHandler(error: unknown) {
  if (axios.isCancel(error)) return Promise.reject(error)
  return Promise.reject(error)
}

http.interceptors.response.use((res) => res, errorHandler)
aiHttp.interceptors.response.use((res) => res, errorHandler)
