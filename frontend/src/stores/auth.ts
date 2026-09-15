import { defineStore } from 'pinia'

import { clearAuthStorage, getRefreshToken, getStoredFamilyId, getToken, saveFamilyId, saveTokens } from '../api/client'
import { authApi, toUser } from '../api/endpoints'
import type { User } from '../types'

interface AuthState {
  isAuthenticated: boolean
  user: User | null
  familyId: string | null
  /** 是否已完成首次会话恢复，避免路由守卫在恢复前误判。 */
  ready: boolean
}

export const useAuthStore = defineStore('auth', {
  state: (): AuthState => ({
    isAuthenticated: false,
    user: null,
    familyId: getStoredFamilyId(),
    ready: false
  }),

  getters: {
    displayName: (s) => s.user?.name ?? '',
    isOwner: (s) => s.user?.role === 'owner'
  },

  actions: {
    /** 应用启动时恢复会话：有 token 就拉一次 /me 校准用户与家庭。 */
    async bootstrap() {
      if (!getToken()) {
        this.ready = true
        return
      }
      try {
        const me = await authApi.me()
        this.user = toUser(me)
        this.isAuthenticated = true
        if (me.family_id) {
          this.familyId = me.family_id
          saveFamilyId(me.family_id)
        } else {
          // 尚未加入任何家庭
          this.familyId = null
          await this.resolveFamily()
        }
      } catch {
        clearAuthStorage()
        this.isAuthenticated = false
        this.user = null
        this.familyId = null
      } finally {
        this.ready = true
      }
    },

    /** 从 /families 兜底解析 familyId（/me 未返回时使用）。 */
    async resolveFamily() {
      try {
        const list = await authApi.myFamilies()
        if (list.length > 0) {
          this.familyId = list[0].id
          saveFamilyId(list[0].id)
        }
        return this.familyId
      } catch {
        return null
      }
    },

    async login(email: string, password: string) {
      const res = await authApi.login(email, password)
      saveTokens(res.access_token, res.refresh_token)
      this.isAuthenticated = true
      this.ready = true
      await this.bootstrap()
      return this.familyId
    },

    async register(email: string, password: string, userName: string, familyName?: string) {
      const res = await authApi.register(email, password, userName)
      saveTokens(res.access_token, res.refresh_token)
      this.isAuthenticated = true
      this.ready = true
      this.user = { id: res.user.id, name: res.user.name || userName, role: 'owner' }

      if (familyName) {
        await this.createFamily(familyName)
      } else {
        await this.resolveFamily()
      }
      return this.familyId
    },

    async createFamily(name: string) {
      const f = await authApi.createFamily(name)
      this.familyId = f.id
      saveFamilyId(f.id)
      await this.bootstrap()
      return f.id
    },

    async logout() {
      const refresh = getRefreshToken()
      if (refresh) {
        try {
          await authApi.logout(refresh)
        } catch {
          // 登出失败不应阻塞本地清理
        }
      }
      clearAuthStorage()
      this.isAuthenticated = false
      this.user = null
      this.familyId = null
    }
  }
})
