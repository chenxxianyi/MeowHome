// 响应适配层：把后端的统一 Envelope 转成前端既有的 APIResponse 形状。
//
// 后端返回 { code, message, data, request_id }（成功时 code === 'SUCCESS'），
// 前端视图与 store 一直按 mock 时代的 { success, data, requestId } 消费。
// 在这里做一次收敛，视图层无需改动。
import type { APIResponse } from '../types'

/** 后端统一响应包裹。 */
export interface Envelope<T> {
  code: string
  message?: string
  data?: T
  request_id?: string
}

/** 后端业务错误码 → 用户可读文案。 */
const CODE_MESSAGES: Record<string, string> = {
  AUTH_REQUIRED: '登录已失效，请重新登录',
  TOKEN_EXPIRED: '登录已过期，请重新登录',
  FAMILY_FORBIDDEN: '无权访问该家庭的数据',
  FORBIDDEN: '没有操作权限',
  NOT_FOUND: '资源不存在',
  CAT_NOT_FOUND: '猫咪不存在',
  CONFLICT: '数据冲突，请刷新后重试',
  VALIDATION_FAILED: '提交的数据不合法',
  INVALID_JSON: '请求格式错误',
  INVALID_REQUEST: '请求参数错误',
  UPLOAD_TOO_LARGE: '文件超出大小限制',
  AI_UNAVAILABLE: 'AI 服务暂不可用',
  INTERNAL_ERROR: '服务异常，请稍后重试'
}

/** 带业务码的 API 错误。 */
export class ApiError extends Error {
  readonly code: string
  readonly status: number
  readonly requestId?: string

  constructor(code: string, message: string, status: number, requestId?: string) {
    super(message)
    this.name = 'ApiError'
    this.code = code
    this.status = status
    this.requestId = requestId
  }

  /** 是否需要重新登录。 */
  get isAuthError(): boolean {
    return this.code === 'AUTH_REQUIRED' || this.code === 'TOKEN_EXPIRED' || this.status === 401
  }
}

/** 把 Envelope 解包为前端 APIResponse。 */
export function unwrap<T>(env: Envelope<T>): APIResponse<T> {
  return {
    success: env.code === 'SUCCESS',
    data: env.data as T,
    requestId: env.request_id
  }
}

/** 由任意错误构造 ApiError，便于统一处理。 */
export function toApiError(error: unknown): ApiError {
  if (error instanceof ApiError) return error

  const err = error as {
    response?: { status?: number; data?: Partial<Envelope<unknown>> }
    message?: string
    code?: string
  }
  const status = err?.response?.status ?? 0
  const body = err?.response?.data
  const code = body?.code || err?.code || (status === 0 ? 'NETWORK_ERROR' : 'INTERNAL_ERROR')
  const message =
    (body?.message && CODE_MESSAGES[body.code ?? '']) ||
    body?.message ||
    CODE_MESSAGES[code] ||
    err?.message ||
    '请求失败'

  return new ApiError(code, message, status, body?.request_id)
}
