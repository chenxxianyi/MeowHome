import { defineStore } from 'pinia'

interface AppState {
  currentCat: string
  currentPage: string
  isOnline: boolean
  isOffline: boolean
  completedReminders: string[]
  aiInput: string
  aiSession: unknown
  aiLoading: boolean
}

export const useAppStore = defineStore('app', {
  state: (): AppState => ({
    currentCat: 'cat-whit',
    currentPage: 'today',
    isOnline: typeof navigator !== 'undefined' ? navigator.onLine !== false : true,
    isOffline: typeof navigator !== 'undefined' ? !navigator.onLine : false,
    completedReminders: [],
    aiInput: '',
    aiSession: null,
    aiLoading: false
  }),
  actions: {
    setCurrentCat(catId: string) { this.currentCat = catId },
    setCurrentPage(page: string) { this.currentPage = page },
    setOnline(online: boolean) {
      this.isOnline = online
      this.isOffline = !online
    },
    setCompletedReminders(ids: string[]) {
      this.completedReminders = ids
    },
    setAIState(input: string, session: unknown, loading: boolean) {
      this.aiInput = input
      this.aiSession = session
      this.aiLoading = loading
    },
    clearAIState() {
      this.aiInput = ''
      this.aiSession = null
      this.aiLoading = false
    }
  }
})
