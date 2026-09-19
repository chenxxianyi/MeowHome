import { onLoad } from '@dcloudio/uni-app'
import { reactive } from 'vue'

import { TAB_PAGES } from './guard'

type RouteValues = Record<string, string>

/**
 * 把 Web 端业务路径转换成已注册的小程序页面路径。
 *
 * 业务组件仍可使用 `router.push('/cats/xxx')` 这种清晰的领域路径，
 * 平台差异统一收敛在这里，避免 17 个页面各自维护一份映射。
 */
export function toMiniProgramUrl(path: string): string {
  if (path === '/' || path === '/today') return '/pages/today/index'
  if (path === '/login') return '/pages/auth/index'
  if (path === '/onboarding') return '/pages/onboarding/index'
  if (path === '/records') return '/pages/records/index'
  if (path === '/records/ai') return '/pages/ai-input/index'
  if (path === '/records/ai/confirm') return '/pages/ai-confirm/index'
  if (path === '/cats') return '/pages/cats/index'
  if (path === '/moments') return '/pages/moments/index'
  if (path === '/family') return '/pages/family/index'
  if (path === '/family/inventory') return '/pages/inventory/index'
  if (path === '/family/expenses') return '/pages/expenses/index'
  if (path === '/medical/upload') return '/pages/medical-upload/index'
  if (path === '/reminders') return '/pages/reminders/index'
  if (path === '/settings') return '/pages/settings/index'

  const quickRecord = path.match(/^\/records\/quick\/([^/?#]+)(?:\?(.+))?$/)
  if (quickRecord) {
    const suffix = quickRecord[2] ? `&${quickRecord[2]}` : ''
    return `/pages/quick-record/index?type=${encodeURIComponent(quickRecord[1])}${suffix}`
  }

  const trends = path.match(/^\/cats\/([^/?#]+)\/trends$/)
  if (trends) return `/pages/trends/index?id=${encodeURIComponent(trends[1])}`

  const catDetail = path.match(/^\/cats\/([^/?#]+)$/)
  if (catDetail) return `/pages/cat-detail/index?id=${encodeURIComponent(catDetail[1])}`

  // 已经是小程序路径时原样放行，方便守卫和登录回跳共用。
  if (path.startsWith('/pages/')) return path
  return '/pages/today/index'
}

function open(path: string, replace = false): Promise<void> {
  const url = toMiniProgramUrl(path)
  const pagePath = url.split('?')[0]
  return new Promise((resolve, reject) => {
    const callbacks = { url, success: () => resolve(), fail: reject }
    if (TAB_PAGES.includes(pagePath)) {
      setTimeout(() => uni.switchTab(callbacks), 10)
    } else if (replace) {
      uni.redirectTo(callbacks)
    } else {
      uni.navigateTo(callbacks)
    }
  })
}

/** Web 端 `useRouter` 的小程序兼容门面。 */
export function useRouter() {
  return {
    push: (path: string) => open(path),
    replace: (path: string) => open(path, true),
    back: () => uni.navigateBack()
  }
}

/**
 * Web 端 `useRoute` 的小程序兼容门面。
 * `id` 同时映射为 `params.catId`，`type` 映射为 `params.type`。
 */
export function useRoute() {
  const params = reactive<RouteValues>({})
  const query = reactive<RouteValues>({})

  onLoad((options = {}) => {
    Object.assign(query, options)
    Object.assign(params, options)
    if (options.id) params.catId = options.id
    if (options.type) params.type = options.type
  })

  return { params, query }
}
