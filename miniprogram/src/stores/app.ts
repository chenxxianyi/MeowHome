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
    // 小程序无浏览器在线状态 API，且 uni.getNetworkType 是异步的，
    // 因此乐观初值设为在线，由 App.vue onLaunch 调 syncNetworkStatus() 校准。
    isOnline: true,
    isOffline: false,
    completedReminders: [],
    aiInput: '',
    aiSession: null,
    aiLoading: false
  }),
  actions: {
    setCurrentCat(catId: string) {
      this.currentCat = catId
    },
    setCurrentPage(page: string) {
      this.currentPage = page
    },
    setOnline(online: boolean) {
      this.isOnline = online
      this.isOffline = !online
    },
    /** 主动查询一次网络状态（异步）。 */
    syncNetworkStatus() {
      uni.getNetworkType({
        success: (res) => this.setOnline(res.networkType !== 'none'),
        fail: () => this.setOnline(true) // 查询失败时不误报离线
      })
    },
    /** 注册网络状态监听。应在 App.vue 的 onLaunch 中调用一次。 */
    listenNetworkStatus() {
      uni.onNetworkStatusChange((res) => this.setOnline(res.isConnected))
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
