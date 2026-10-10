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
  console.info('[auth] session.save.start')
  uni.setStorageSync(TOKEN_KEY, access)
  if (refresh) uni.setStorageSync(REFRESH_KEY, refresh)
  console.info('[auth] session.save.done')
}

export function saveFamilyId(id: string) {
  console.info('[auth] family.save.start')
  uni.setStorageSync(FAMILY_KEY, id)
  console.info('[auth] family.save.done')
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

/** 原生 timeout 之外独立结束 Promise，避免调试环境回调缺失时一直等待。 */
function requestWithDeadline(options: UniApp.RequestOptions, timeout: number): Promise<RawResponse> {
  const path = options.url.replace(/^https?:\/\/[^/]+/, '').split('?')[0]
  const traceEnabled = /\/(?:auth\/(?:register|login|refresh)|me|families)$/.test(path)
  const startedAt = Date.now()
  // 只记录认证步骤与请求 ID；不得输出请求体、响应体或鉴权头。
  function trace(phase: string, status?: number) {
    if (!traceEnabled) return
    console.info(`[api] ${phase}`, {
      path,
      requestId: options.header?.['X-Request-Id'],
      elapsedMs: Date.now() - startedAt,
      ...(status === undefined ? {} : { status })
    })
  }
  return new Promise((resolve, reject) => {
    let settled = false
    let task: UniApp.RequestTask | undefined
    const timer = setTimeout(() => {
      if (settled) return
      trace('request.timeout')
      finish(() => reject(new ApiError('REQUEST_TIMEOUT', '连接服务超时，请检查网络后重试', 0)))
      try {
        task?.abort()
      } catch {
        // Promise 已超时结束；平台中止异常不能再次阻塞界面。
      }
    }, timeout)

    function finish(callback: () => void) {
      if (settled) return
      settled = true
      clearTimeout(timer)
      callback()
    }

    try {
      trace('request.start')
      task = uni.request({
        ...options,
        timeout,
        success: (res) =>
          finish(() => {
            trace('response.success', res.statusCode)
            resolve({ statusCode: res.statusCode, data: res.data })
          }),
        fail: (err) =>
          finish(() => {
            const timedOut = /timeout/i.test(err?.errMsg || '')
            trace(timedOut ? 'response.timeout' : 'response.failed')
            reject(
              new ApiError(
                timedOut ? 'REQUEST_TIMEOUT' : 'NETWORK_ERROR',
                timedOut ? '连接服务超时，请检查网络后重试' : '暂时无法连接服务，请检查网络后重试',
                0
              )
            )
          })
      })
      trace('request.dispatched')
    } catch (error) {
      trace('request.threw')
      finish(() => reject(error))
    }
  })
}

function rawRequest(client: ApiClientOptions, config: RequestConfig, token: string | null): Promise<RawResponse> {
  const header: Record<string, string> = {
    'Content-Type': 'application/json',
    'X-Request-Id': genRequestId()
  }
  if (token) header.Authorization = `Bearer ${token}`

  // 微信 wx.request 不支持 PATCH，后端 MethodOverride 将此 POST 按 PATCH 路由。
  const isPatch = config.method === 'PATCH'
  const method: UniMethod = isPatch ? 'POST' : (config.method as UniMethod)
  if (isPatch) header['X-HTTP-Method-Override'] = 'PATCH'

  return requestWithDeadline(
    {
      url: apiBase + config.url + buildQuery(config.params),
      method,
      data: config.data as string | AnyObject | ArrayBuffer | undefined,
      header
    },
    client.timeout
  )
}

// 401 自动刷新：并发请求共享同一次刷新，避免令牌被反复轮换。
let refreshing: Promise<string | null> | null = null

async function refreshAccessToken(): Promise<string | null> {
  const refresh = getRefreshToken()
  if (!refresh) return null

  try {
    // 绕开 send，避免刷新失败时递归；刷新也必须独立超时结束。
    const res = await requestWithDeadline(
      {
        url: `${apiBase}/auth/refresh`,
        method: 'POST',
        data: { refresh_token: refresh },
        header: { 'Content-Type': 'application/json', 'X-Request-Id': genRequestId() }
      },
      10000
    )
    const body = res.data as Envelope<{ access_token: string; refresh_token: string }> | undefined
    const data = body && body.data
    if (res.statusCode === 200 && data && data.access_token) {
      saveTokens(data.access_token, data.refresh_token)
      return data.access_token
    }
    return null
  } catch {
    return null
  }
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
