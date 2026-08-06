import { createRouter, createWebHashHistory } from 'vue-router'

// Lazy-loaded views
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
