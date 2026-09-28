import { defineStore } from 'pinia'
import { agentApi } from '../api/endpoints'
import { useAuthStore } from './auth'
import type { AgentConfirmResponse, AgentMessage, AgentMessageListParams } from '../types/agent'

interface AgentState {
  messages: AgentMessage[]
  nextCursor?: string
  loading: boolean
  error: string
}

export const useAgentStore = defineStore('agent', {
  state: (): AgentState => ({ messages: [], nextCursor: undefined, loading: false, error: '' }),
  getters: {
    pending: (s) => s.messages.filter((m) => m.action_status === 'pending'),
    important: (s) => [...s.messages].sort((a, b) => severityRank(b.severity) - severityRank(a.severity))
  },
  actions: {
    async load(params: AgentMessageListParams = {}) {
      const familyId = useAuthStore().familyId
      if (!familyId) return
      this.loading = true
      this.error = ''
      try {
        const result = await agentApi.listMessages(familyId, params)
        this.messages = params.before ? [...this.messages, ...result.messages] : result.messages
        this.nextCursor = result.next_cursor
      } catch (error) {
        this.error = error instanceof Error ? error.message : '猫管家暂时不可用'
      } finally {
        this.loading = false
      }
    },
    async dismiss(id: string) {
      const familyId = useAuthStore().familyId
      if (!familyId) return
      await agentApi.dismiss(familyId, id)
      const item = this.messages.find((m) => m.id === id)
      if (item) item.action_status = 'dismissed'
    },
    async confirm(id: string, version: number): Promise<AgentConfirmResponse | null> {
      const familyId = useAuthStore().familyId
      if (!familyId) return null
      const result = await agentApi.confirm(familyId, id, version)
      const item = this.messages.find((m) => m.id === id)
      if (item) {
        item.action_status = 'confirmed'
        item.draft_version = version
      }
      return result
    }
  }
})

function severityRank(value: AgentMessage['severity']): number {
  return value === 'danger' ? 3 : value === 'warning' ? 2 : 1
}
