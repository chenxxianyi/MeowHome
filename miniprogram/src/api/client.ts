// API 客户端（小程序）— 基于 uni.request，负责鉴权头、request_id 与 401 自动刷新。
//
// 设计约束：**对外接口与 Web 端 `frontend/src/api/client.ts` 完全一致**
// （导出 http / aiHttp / request / requestEnvelope / apiBase，签名相同），
// 因此 `api/endpoints.ts` 可以原样复用，无需任何改动。
//
// 平台差异处理：
//   axios.create / 拦截器  → uni.request + 手写重试
//   localStorage           → uni.*StorageSync
//   location.hash 跳转     → uni.reLaunch
//   crypto.randomUUID      → 时间戳 + 随机串
import { ApiError, toApiError, unwrap } from './adapter'
import type { Envelope } from './adapter'
import { API_BASE } from './config'

/** 业务接口前缀（与 Web 端同名导出，endpoints.ts 会 re-export）。 */
export const apiBase = API_BASE

// 存储键名（与 Web 端保持一致，便于两端对照排查）
export const TOKEN_KEY = 'meowhome.token'
export const REFRESH_KEY = 'meowhome.refresh'
export const FAMILY_KEY = 'meowhome.familyId'

// ---------------------------------------------------------------- 存储

export function getToken(): string | null {
  const v = uni.getStorageSync(TOKEN_KEY)
  return typeof v === 'string' && v.length > 0 ? v : null
}

export function getRefreshToken(): string | null {
  const v = uni.getStorageSync(REFRESH_KEY)
  return typeof v === 'string' && v.length > 0 ? v : null
}

export function getStoredFamilyId(): string | null {
  const v = uni.getStorageSync(FAMILY_KEY)
  return typeof v === 'string' && v.length > 0 ? v : null
}

export function saveTokens(access: string, refresh?: string) {
  uni.setStorageSync(TOKEN_KEY, access)
  if (refresh) uni.setStorageSync(REFRESH_KEY, refresh)
}

export function saveFamilyId(id: string) {
  uni.setStorageSync(FAMILY_KEY, id)
}

export function clearAuthStorage() {
  uni.removeStorageSync(TOKEN_KEY)
  uni.removeStorageSync(REFRESH_KEY)
  uni.removeStorageSync(FAMILY_KEY)
}

// ---------------------------------------------------------------- 客户端

/** 对外暴露的请求方法（与 Web 端 endpoints.ts 的用法保持一致）。 */
export type HttpMethod = 'GET' | 'POST' | 'PUT' | 'PATCH' | 'DELETE'

/** wx.request 实际支持的方法 —— **不含 PATCH**。 */
type UniMethod = 'GET' | 'POST' | 'PUT' | 'DELETE' | 'OPTIONS' | 'HEAD' | 'TRACE' | 'CONNECT'

export interface RequestConfig {
  method: HttpMethod
  url: string
  /** 请求体（POST/PATCH/PUT） */
  data?: unknown
  /** 查询参数（GET），值为 undefined 的键会被跳过 */
  params?: Record<string, unknown>
}

/** 客户端选项。Web 端这里是 axios 实例，小程序端只需承载超时配置。 */
export interface ApiClientOptions {
  timeout: number
}

export const http: ApiClientOptions = { timeout: 15000 }

/** AI 专用：独立超时，不影响普通业务 API */
export const aiHttp: ApiClientOptions = { timeout: 30000 }

interface RawResponse {
  statusCode: number
  data: unknown
}

function genRequestId(): string {
  return `req-${Date.now()}-${Math.random().toString(16).slice(2, 10)}`
}

/** 把 params 拼成查询串；值为 undefined / null / '' 的键跳过。 */
function buildQuery(params?: Record<string, unknown>): string {
  if (!params) return ''
  const parts: string[] = []
  for (const key of Object.keys(params)) {
    const v = params[key]
    if (v === undefined || v === null || v === '') continue
    parts.push(`${encodeURIComponent(key)}=${encodeURIComponent(String(v))}`)
  }
  return parts.length ? `?${parts.join('&')}` : ''
}

function rawRequest(
  client: ApiClientOptions,
  config: RequestConfig,
  token: string | null
): Promise<RawResponse> {
  return new Promise((resolve, reject) => {
    const header: Record<string, string> = {
      'Content-Type': 'application/json',
      'X-Request-Id': genRequestId()
    }
    if (token) header.Authorization = `Bearer ${token}`

    // ⚠️ 微信 wx.request 不支持 PATCH（合法值仅 GET/POST/PUT/DELETE/OPTIONS/HEAD/TRACE/CONNECT）。
    // 降级为 POST + X-HTTP-Method-Override，需要后端识别该头并路由到 PATCH 处理器。
    // 后端未实现前，涉及 PATCH 的接口（更新家庭 / 更新猫咪 / 完成提醒）会失败。
    const isPatch = config.method === 'PATCH'
    const method: UniMethod = isPatch ? 'POST' : (config.method as UniMethod)
    if (isPatch) header['X-HTTP-Method-Override'] = 'PATCH'

    uni.request({
      url: apiBase + config.url + buildQuery(config.params),
      method,
      data: config.data as string | AnyObject | ArrayBuffer | undefined,
      header,
      timeout: client.timeout,
      success: (res) => resolve({ statusCode: res.statusCode, data: res.data }),
      // uni.request 的失败对象是 { errMsg }，规范成 { message } 以便 toApiError 提取文案
      fail: (err) => reject({ message: (err && err.errMsg) || '网络请求失败' })
    })
  })
}

// 401 自动刷新：并发请求共享同一次刷新，避免令牌被反复轮换。
let refreshing: Promise<string | null> | null = null

function refreshAccessToken(): Promise<string | null> {
  const refresh = getRefreshToken()
  if (!refresh) return Promise.resolve(null)

  return new Promise((resolve) => {
    // 直接调 uni.request，绕开本方封装，避免刷新失败时递归
    uni.request({
      url: `${apiBase}/auth/refresh`,
      method: 'POST',
      data: { refresh_token: refresh },
      header: { 'Content-Type': 'application/json' },
      timeout: 10000,
      success: (res) => {
        const body = res.data as Envelope<{ access_token: string; refresh_token: string }> | undefined
        const data = body && body.data
        if (res.statusCode === 200 && data && data.access_token) {
          saveTokens(data.access_token, data.refresh_token)
          resolve(data.access_token)
        } else {
          resolve(null)
        }
      },
      fail: () => resolve(null)
    })
  })
}

/** 跳转登录页（等价于 Web 端的 location.hash 判断，避免重复跳转）。 */
function redirectToLogin() {
  const pages = getCurrentPages()
  const current = pages.length > 0 ? pages[pages.length - 1].route : ''
  if (current !== 'pages/auth/index') {
    uni.reLaunch({ url: '/pages/auth/index' })
  }
}

/** 发送请求；遇 401 则尝试刷新令牌并重试一次。 */
async function send(client: ApiClientOptions, config: RequestConfig): Promise<RawResponse> {
  const first = await rawRequest(client, config, getToken())
  if (first.statusCode !== 401) return first

  if (!refreshing) {
    refreshing = refreshAccessToken().finally(() => {
      refreshing = null
    })
  }
  const token = await refreshing
  if (token) {
    return rawRequest(client, config, token)
  }

  // 刷新失败：清空登录态并回登录页
  clearAuthStorage()
  redirectToLogin()
  return first
}

/**
 * 校验响应体确实是本项目的 Envelope，否则抛错。
 *
 * ⚠️ 这个校验不能省。后端契约是「任何响应都带 code」，但**未注册的路径**返回的是
 * Gin 的 `404 page not found` 纯文本，网关故障返回 HTML，这些都不是 Envelope。
 * 旧写法 `if (env.code && env.code !== 'SUCCESS')` 在 `code` 为 undefined 时条件为假，
 * 于是继续 `return env.data ?? env`，把 `"404 page not found"` 这串字符串当成
 * **成功结果**返回给调用方 —— 失败被静默吞掉，只表现为界面数据莫名不对。
 *
 * 这个坑在 1.5 联调时被真实踩到：小程序把 PATCH 降级为 POST + 覆盖头，
 * 后端不认该头而返回 404，调用方却毫无察觉。
 */
function assertEnvelope<T>(res: RawResponse): Envelope<T> {
  const body = res.data as Partial<Envelope<T>> | null
  if (!body || typeof body !== 'object' || typeof body.code !== 'string') {
    throw new ApiError(
      `HTTP_${res.statusCode}`,
      res.statusCode >= 400
        ? `服务返回异常状态（HTTP ${res.statusCode}）`
        : `响应格式不符合约定（HTTP ${res.statusCode}）`,
      res.statusCode
    )
  }
  return body as Envelope<T>
}

/** 发起请求并解包 Envelope；失败时抛出 ApiError。 */
export async function request<T>(client: ApiClientOptions, config: RequestConfig): Promise<T> {
  try {
    const res = await send(client, config)
    // 204 No Content 没有 body
    if (res.statusCode === 204 || res.data == null) {
      return undefined as T
    }
    const env = assertEnvelope<T>(res)
    if (env.code !== 'SUCCESS') {
      throw new ApiError(env.code, env.message || '请求失败', res.statusCode, env.request_id)
    }
    return env.data as T
  } catch (error) {
    throw toApiError(error)
  }
}

/** 同 request，但返回完整 APIResponse（供 service 层透传 success 标志）。 */
export async function requestEnvelope<T>(
  client: ApiClientOptions,
  config: RequestConfig
): Promise<{ success: true; data: T; requestId?: string }> {
  try {
    const res = await send(client, config)
    if (res.statusCode === 204 || res.data == null) {
      return { success: true, data: undefined as T }
    }
    // 非 Envelope 属协议故障，不是业务失败，因此抛错而不是返回 success:false
    return unwrap(assertEnvelope<T>(res)) as { success: true; data: T; requestId?: string }
  } catch (error) {
    throw toApiError(error)
  }
}
