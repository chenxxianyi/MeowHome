// API 客户端 — 统一 Axios 实例，负责鉴权头、request_id 与 401 自动刷新。
import axios from 'axios'
import type { AxiosError, AxiosInstance, InternalAxiosRequestConfig } from 'axios'

import { ApiError, toApiError, unwrap } from './adapter'
import type { Envelope } from './adapter'

export const baseURL: string = import.meta.env.VITE_API_BASE_URL || '/api'

/** 业务接口前缀（后端为 /api/v1）。 */
export const apiBase = `${baseURL}/v1`

// localStorage 键名（client.ts 原本就读 meowhome.token，保持一致）
export const TOKEN_KEY = 'meowhome.token'
export const REFRESH_KEY = 'meowhome.refresh'
export const FAMILY_KEY = 'meowhome.familyId'

export function getToken(): string | null {
  return localStorage.getItem(TOKEN_KEY)
}
export function getRefreshToken(): string | null {
  return localStorage.getItem(REFRESH_KEY)
}
export function getStoredFamilyId(): string | null {
  return localStorage.getItem(FAMILY_KEY)
}
export function saveTokens(access: string, refresh?: string) {
  localStorage.setItem(TOKEN_KEY, access)
  if (refresh) localStorage.setItem(REFRESH_KEY, refresh)
}
export function saveFamilyId(id: string) {
  localStorage.setItem(FAMILY_KEY, id)
}
export function clearAuthStorage() {
  localStorage.removeItem(TOKEN_KEY)
  localStorage.removeItem(REFRESH_KEY)
  localStorage.removeItem(FAMILY_KEY)
}

// 普通业务 API 客户端
export const http = axios.create({
  baseURL: apiBase,
  timeout: Number(import.meta.env.VITE_API_TIMEOUT || 15000)
})

// AI 专用客户端：独立超时，不影响普通业务 API
export const aiHttp = axios.create({
  baseURL: apiBase,
  timeout: Number(import.meta.env.VITE_AI_TIMEOUT || 30000)
})

// 请求拦截：携带认证 + 注入 request_id
function authHeader(config: InternalAxiosRequestConfig) {
  const token = getToken()
  if (token) config.headers.set('Authorization', `Bearer ${token}`)
  const rid =
    typeof crypto !== 'undefined' && crypto.randomUUID
      ? crypto.randomUUID()
      : `req-${Date.now()}-${Math.random().toString(16).slice(2)}`
  config.headers.set('X-Request-Id', rid)
  return config
}

http.interceptors.request.use(authHeader)
aiHttp.interceptors.request.use(authHeader)

// 401 自动刷新：并发请求共享同一次刷新，避免令牌被反复轮换。
let refreshing: Promise<string | null> | null = null

async function refreshAccessToken(): Promise<string | null> {
  const refresh = getRefreshToken()
  if (!refresh) return null
  try {
    // 用裸 axios，绕开拦截器，避免刷新失败时递归
    const res = await axios.post<Envelope<{ access_token: string; refresh_token: string }>>(
      `${apiBase}/auth/refresh`,
      { refresh_token: refresh },
      { timeout: 10000 }
    )
    const data = res.data?.data
    if (data?.access_token) {
      saveTokens(data.access_token, data.refresh_token)
      return data.access_token
    }
    return null
  } catch {
    return null
  }
}

function attachRefresh(instance: AxiosInstance) {
  instance.interceptors.response.use(
    (res) => res,
    async (error: AxiosError) => {
      const original = error.config as InternalAxiosRequestConfig & { _retried?: boolean }
      const status = error.response?.status

      if (status === 401 && original && !original._retried) {
        original._retried = true
        if (!refreshing) {
          refreshing = refreshAccessToken().finally(() => {
            refreshing = null
          })
        }
        const token = await refreshing
        if (token) {
          original.headers.set('Authorization', `Bearer ${token}`)
          return instance.request(original)
        }
        // 刷新失败：清空登录态，交由路由守卫跳转登录页
        clearAuthStorage()
        if (!location.hash.startsWith('#/login')) {
          location.hash = '#/login'
        }
      }
      return Promise.reject(error)
    }
  )
}

attachRefresh(http)
attachRefresh(aiHttp)

/** 发起请求并解包 Envelope；失败时抛出 ApiError。 */
export async function request<T>(
  instance: AxiosInstance,
  config: Parameters<AxiosInstance['request']>[0]
): Promise<T> {
  try {
    const res = await instance.request<Envelope<T>>(config)
    // 204 No Content 没有 body
    if (res.status === 204 || res.data == null) {
      return undefined as T
    }
    const env = res.data
    if (env.code && env.code !== 'SUCCESS') {
      throw new ApiError(env.code, env.message || '请求失败', res.status, env.request_id)
    }
    return (env.data ?? (env as unknown as T)) as T
  } catch (error) {
    throw toApiError(error)
  }
}

/** 同 request，但返回完整 APIResponse（供 service 层透传 success 标志）。 */
export async function requestEnvelope<T>(
  instance: AxiosInstance,
  config: Parameters<AxiosInstance['request']>[0]
): Promise<{ success: true; data: T; requestId?: string }> {
  try {
    const res = await instance.request<Envelope<T>>(config)
    if (res.status === 204 || res.data == null) {
      return { success: true, data: undefined as T }
    }
    return unwrap(res.data) as { success: true; data: T; requestId?: string }
  } catch (error) {
    throw toApiError(error)
  }
}
