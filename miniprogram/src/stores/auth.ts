import { defineStore } from 'pinia'

import { clearAuthStorage, getRefreshToken, getStoredFamilyId, getToken, saveFamilyId, saveTokens } from '../api/client'
import { authApi, toUser } from '../api/endpoints'
// 小程序编译器会将业务模块的动态 import 转为路径字符串。
// 保留静态导入，仅在 action 中调用 store，避免初始化时相互读取。
import { useAgentStore } from './agent'
import type { User } from '../types'

const PENDING_REGISTRATION_KEY = 'meowhome.pendingRegistration'
const REMEMBERED_CREDENTIALS_KEY = 'meowhome.rememberedCredentials'
const AUTO_LOGIN_KEY = 'meowhome.autoLogin'

interface PendingRegistration {
  email: string
  familyName: string
}

interface RememberedCredentials {
  email: string
  password: string
}

function normalizeEmail(email: string) {
  return email.trim().toLowerCase()
}

export function getRememberedCredentials(): RememberedCredentials | null {
  const value = uni.getStorageSync(REMEMBERED_CREDENTIALS_KEY) as Partial<RememberedCredentials> | undefined
  if (typeof value?.email !== 'string' || typeof value.password !== 'string') return null
  return { email: value.email, password: value.password }
}

export function saveRememberedCredentials(email: string, password: string) {
  uni.setStorageSync(REMEMBERED_CREDENTIALS_KEY, { email: email.trim(), password } satisfies RememberedCredentials)
}

export function clearRememberedCredentials() {
  uni.removeStorageSync(REMEMBERED_CREDENTIALS_KEY)
}

export function getAutoLoginPreference() {
  const value = uni.getStorageSync(AUTO_LOGIN_KEY)
  // 保持升级前已有令牌用户的自动恢复行为。
  return typeof value === 'boolean' ? value : true
}

/** 手动登录成功后取出本次注册填写的家庭名，供首次使用引导页预填。 */
export function takePendingRegistrationFamilyName(email: string): string | null {
  const pending = uni.getStorageSync(PENDING_REGISTRATION_KEY) as Partial<PendingRegistration> | undefined
  if (!pending || typeof pending.email !== 'string' || normalizeEmail(pending.email) !== normalizeEmail(email)) {
    return null
  }

  uni.removeStorageSync(PENDING_REGISTRATION_KEY)
  return typeof pending.familyName === 'string' && pending.familyName.trim() ? pending.familyName : null
}

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
      console.info('[auth] bootstrap.start')
      if (!getAutoLoginPreference() && !this.isAuthenticated) {
        clearAuthStorage()
        useAgentStore().reset()
        this.user = null
        this.isAuthenticated = false
        this.familyId = null
        this.ready = true
        console.info('[auth] bootstrap.auto-login-disabled')
        return
      }
      if (!getToken()) {
        clearAuthStorage()
        useAgentStore().reset()
        this.user = null
        this.isAuthenticated = false
        this.familyId = null
        this.ready = true
        console.info('[auth] bootstrap.no-session')
        return
      }
      try {
        const me = await authApi.me()
        this.user = toUser(me)
        this.isAuthenticated = true
        if (me.family_id) {
          if (this.familyId && this.familyId !== me.family_id) useAgentStore().reset()
          this.familyId = me.family_id
          saveFamilyId(me.family_id)
        } else {
          // 尚未加入任何家庭
          this.familyId = null
          await this.resolveFamily()
        }
      } catch {
        console.info('[auth] bootstrap.failed')
        useAgentStore().reset()
        clearAuthStorage()
        this.isAuthenticated = false
        this.user = null
        this.familyId = null
      } finally {
        this.ready = true
        console.info('[auth] bootstrap.done')
      }
    },

    /** 从 /families 兜底解析 familyId（/me 未返回时使用）。 */
    async resolveFamily() {
      try {
        const list = await authApi.myFamilies()
        if (list.length > 0) {
          if (this.familyId && this.familyId !== list[0].id) useAgentStore().reset()
          this.familyId = list[0].id
          saveFamilyId(list[0].id)
        }
        return this.familyId
      } catch {
        return null
      }
    },

    async login(email: string, password: string, autoLogin = getAutoLoginPreference()) {
      console.info('[auth] login.start')
      const res = await authApi.login(email, password)
      console.info('[auth] login.response')
      uni.setStorageSync(AUTO_LOGIN_KEY, autoLogin)
      saveTokens(res.access_token, res.refresh_token)
      this.isAuthenticated = true
      this.ready = true
      await this.bootstrap()
      console.info('[auth] login.done')
      return this.familyId
    },

    async register(email: string, password: string, userName: string, familyName?: string) {
      console.info('[auth] register.start')
      await authApi.register(email, password, userName)
      console.info('[auth] register.response')
      uni.setStorageSync(PENDING_REGISTRATION_KEY, {
        email: normalizeEmail(email),
        familyName: familyName?.trim() || `${userName.trim()}的猫宅`
      } satisfies PendingRegistration)
      console.info('[auth] register.done')
    },

    async createFamily(name: string) {
      console.info('[auth] family.create.start')
      const f = await authApi.createFamily(name)
      console.info('[auth] family.create.response')
      useAgentStore().reset()
      this.familyId = f.id
      saveFamilyId(f.id)
      await this.bootstrap()
      console.info('[auth] family.create.done')
      return f.id
    },

    async logout() {
      useAgentStore().reset()
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
