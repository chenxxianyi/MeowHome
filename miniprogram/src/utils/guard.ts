/**
 * 路由守卫（Web 端 `router/index.ts` 的 `beforeEach` 的小程序等价实现）。
 *
 * ## 为什么需要单独一个模块
 *
 * 小程序**没有全局路由守卫**。Web 端在 `router.beforeEach` 里统一做了四件事：
 *   1. 首次会话恢复（`auth.bootstrap()`）
 *   2. 未登录 → 跳登录页（带 redirect）
 *   3. 已登录但无家庭 → 跳引导页
 *   4. 预载猫咪列表
 *
 * 小程序只能落到「页面生命周期」上。这里把它封装成两个函数，页面在
 * `onShow` 里调用即可，避免每个页面重复实现。
 *
 * ## 用法
 *
 * ```ts
 * import { onShow } from '@dcloudio/uni-app'
 * import { guardOnShow } from '../../utils/guard'
 *
 * onShow(async () => {
 *   if (!(await guardOnShow())) return   // 守卫已发起跳转，本页应停止后续渲染逻辑
 *   await load()
 * })
 * ```
 *
 * ## 为什么用 onShow 而不是 onLoad
 *
 * 从登录页 `reLaunch` 回来后页面会重新走 `onShow`，用 `onShow` 能保证每次都校验；
 * 而 `onLoad` 只在页面创建时触发一次。
 */
import { useAuthStore } from '../stores/auth'
import { useCatStore } from '../stores/cat'
import { catApi, toCat } from '../api/endpoints'

/** 无需登录即可访问的页面 */
const PUBLIC_PAGES = ['/pages/auth/index']

/** tab 页集合：跳转这些页必须用 switchTab（reLaunch 会静默失败） */
export const TAB_PAGES = [
  '/pages/today/index',
  '/pages/records/index',
  '/pages/cats/index',
  '/pages/moments/index',
  '/pages/family/index'
]

/** 会话恢复只需一次；与 Web 端 `let bootstrapped = false` 对应 */
let bootstrapped = false
let catsForFamily = ''

/** 恢复登录态（只真正执行一次）。返回是否已登录。 */
export async function ensureSession(): Promise<boolean> {
  const auth = useAuthStore()
  if (!bootstrapped) {
    bootstrapped = true
    await auth.bootstrap()
  }
  return auth.isAuthenticated
}

/** 当前页面路径，形如 `/pages/cats/index`。 */
export function currentPagePath(): string {
  const pages = getCurrentPages()
  if (!pages.length) return ''
  const route = pages[pages.length - 1].route || ''
  return route.startsWith('/') ? route : `/${route}`
}

/**
 * 跳转到指定小程序页面。
 * tab 页必须用 `switchTab`（用 `reLaunch`/`navigateTo` 会静默失败），普通页用 `reLaunch`。
 */
export function navigate(url: string) {
  if (TAB_PAGES.includes(url)) {
    uni.switchTab({ url })
  } else {
    uni.reLaunch({ url })
  }
}

function go(url: string) {
  if (url.startsWith('/pages/auth/index?')) {
    // 登录页带 query，不能走 switchTab；且它是普通页
    uni.reLaunch({ url })
    return
  }
  navigate(url)
}

/**
 * 执行守卫。
 *
 * @returns `true` 表示可继续渲染；`false` 表示已发起跳转，调用方应放弃后续逻辑。
 */
export function guardPage(path: string): boolean {
  const auth = useAuthStore()
  const target = path.startsWith('/') ? path : `/${path}`

  // 未登录 → 登录页（携带原目标，登录后可回跳）
  if (!auth.isAuthenticated) {
    if (!PUBLIC_PAGES.includes(target)) {
      go(`/pages/auth/index?redirect=${encodeURIComponent(target)}`)
    }
    return false
  }

  // 已登录却停在登录页 → 回首页
  if (PUBLIC_PAGES.includes(target)) {
    go('/pages/today/index')
    return false
  }

  // 已登录但没有家庭 → 引导页（否则所有家庭维度接口都会 403/404）
  if (!auth.familyId && target !== '/pages/onboarding/index') {
    go('/pages/onboarding/index')
    return false
  }

  return true
}

/** 页面 `onShow` 里的组合调用：先恢复会话，再执行守卫。 */
export async function guardOnShow(): Promise<boolean> {
  await ensureSession()
  if (!guardPage(currentPagePath())) return false

  // 与 Web 端全局守卫一致：进入受保护页面前，按家庭预载一次猫咪列表。
  // 加载失败不阻塞页面，下一次 onShow 会自动重试。
  const auth = useAuthStore()
  if (auth.familyId && catsForFamily !== auth.familyId) {
    try {
      useCatStore().setCats((await catApi.list(auth.familyId)).map(toCat))
      catsForFamily = auth.familyId
    } catch {
      catsForFamily = ''
    }
  }
  return true
}
