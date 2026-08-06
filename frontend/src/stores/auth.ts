import { defineStore } from 'pinia'
import type { User } from '../types'

interface AuthState {
  isAuthenticated: boolean
  user: User | null
  token: string | null
}

export const useAuthStore = defineStore('auth', {
  state: (): AuthState => ({
    isAuthenticated: true,
    user: { id: 'mem-1', name: '小明', role: 'owner' },
    token: 'mock-token'
  }),
  actions: {
    setAuth(user: User, token: string) {
      this.isAuthenticated = true
      this.user = user
      this.token = token
    },
    logout() {
      this.isAuthenticated = false
      this.user = null
      this.token = null
    }
  }
})
