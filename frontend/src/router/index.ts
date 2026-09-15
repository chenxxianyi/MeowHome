import { createRouter, createWebHashHistory } from 'vue-router'

import { catApi, toCat } from '../api/endpoints'
import { useAuthStore } from '../stores/auth'
import { useCatStore } from '../stores/cat'

// Lazy-loaded views
const AuthView = () => import('../views/auth/AuthView.vue')
const TodayView = () => import('../views/today/TodayView.vue')
const OnboardingView = () => import('../views/onboarding/OnboardingView.vue')
const RecordsView = () => import('../views/records/RecordsView.vue')
const QuickRecordView = () => import('../views/records/QuickRecordView.vue')
const AIInputView = () => import('../views/records/AIInputView.vue')
const AIConfirmView = () => import('../views/records/AIConfirmView.vue')
const CatsView = () => import('../views/cats/CatsView.vue')
const CatDetailView = () => import('../views/cats/CatDetailView.vue')
const TrendsView = () => import('../views/cats/TrendsView.vue')
const MedicalUploadView = () => import('../views/medical/MedicalUploadView.vue')
const RemindersView = () => import('../views/reminders/RemindersView.vue')
const MomentsView = () => import('../views/moments/MomentsView.vue')
const FamilyView = () => import('../views/family/FamilyView.vue')
const InventoryView = () => import('../views/family/InventoryView.vue')
const ExpensesView = () => import('../views/family/ExpensesView.vue')
const SettingsView = () => import('../views/settings/SettingsView.vue')

const routes = [
  { path: '/login', name: 'Auth', component: AuthView, meta: { public: true } },
  { path: '/', redirect: '/today' },
  { path: '/today', name: 'Today', component: TodayView },
  { path: '/onboarding', name: 'Onboarding', component: OnboardingView },
  { path: '/records', name: 'Records', component: RecordsView },
  { path: '/records/quick/:type', name: 'QuickRecord', component: QuickRecordView, props: true },
  { path: '/records/ai', name: 'AIInput', component: AIInputView },
  { path: '/records/ai/confirm', name: 'AIConfirm', component: AIConfirmView },
  { path: '/cats', name: 'Cats', component: CatsView },
  { path: '/cats/:catId', name: 'CatDetail', component: CatDetailView, props: true },
  { path: '/cats/:catId/trends', name: 'Trends', component: TrendsView, props: true },
  { path: '/medical/upload', name: 'MedicalUpload', component: MedicalUploadView },
  { path: '/reminders', name: 'Reminders', component: RemindersView },
  { path: '/moments', name: 'Moments', component: MomentsView },
  { path: '/family', name: 'Family', component: FamilyView },
  { path: '/family/inventory', name: 'Inventory', component: InventoryView },
  { path: '/family/expenses', name: 'Expenses', component: ExpensesView },
  { path: '/settings', name: 'Settings', component: SettingsView }
]

export const router = createRouter({
  history: createWebHashHistory(),
  routes,
  scrollBehavior(to, from, savedPosition) {
    if (savedPosition) return savedPosition
    return { top: 0 }
  }
})

// 会话恢复与猫咪列表只需各加载一次；按 familyId 记住已加载的家庭
let bootstrapped = false
let catsForFamily = ''

router.beforeEach(async (to) => {
  const auth = useAuthStore()

  if (!bootstrapped) {
    bootstrapped = true
    await auth.bootstrap()
  }

  if (to.meta.public) {
    return auth.isAuthenticated ? { path: '/today' } : true
  }

  if (!auth.isAuthenticated) {
    return { path: '/login', query: to.fullPath === '/' ? {} : { redirect: to.fullPath } }
  }

  // 已登录但还没有家庭：先去创建，否则所有家庭维度接口都会 403/404
  if (!auth.familyId && to.path !== '/onboarding') {
    return { path: '/onboarding' }
  }

  // 多处视图依赖猫咪列表（切换器、快捷记录默认猫咪），进应用后统一预载一次
  if (auth.familyId && catsForFamily !== auth.familyId) {
    catsForFamily = auth.familyId
    try {
      useCatStore().setCats((await catApi.list(auth.familyId)).map(toCat))
    } catch {
      catsForFamily = '' // 失败则下次导航重试
    }
  }

  return true
})
