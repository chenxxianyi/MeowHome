import { defineStore } from 'pinia'
import { agentApi } from '../api/endpoints'
import { useAuthStore } from './auth'
import { useReminderStore } from './reminder'
import type { AgentConfirmResponse, AgentMessage, AgentMessageListParams, AgentReminderEditRequest, AgentReminderDraft } from '../types/agent'

interface AgentState {
  messages: AgentMessage[]
  nextCursor?: string
  loading: boolean
  error: string
  familyId: string | null
  requestSeq: number
}

export const useAgentStore = defineStore('agent', {
  state: (): AgentState => ({ messages: [], nextCursor: undefined, loading: false, error: '', familyId: null, requestSeq: 0 }),
  getters: {
    pending: (s) => s.messages.filter((m) => m.action_status === 'pending'),
    important: (s) => s.messages.filter((m) => m.display_status !== 'dismissed').sort((a, b) => severityRank(b.severity) - severityRank(a.severity))
  },
  actions: {
    reset() {
      this.requestSeq++
      this.messages = []
      this.nextCursor = undefined
      this.loading = false
      this.error = ''
      this.familyId = null
    },
    async load(params: AgentMessageListParams = {}) {
      const familyId = useAuthStore().familyId
      if (!familyId) { this.reset(); return }
      if (this.familyId !== familyId) this.reset()
      this.familyId = familyId
      const seq = ++this.requestSeq
      this.loading = true
      this.error = ''
      try {
        const result = await agentApi.listMessages(familyId, params)
        if (seq !== this.requestSeq || useAuthStore().familyId !== familyId) return
        this.messages = params.before ? [...this.messages, ...result.messages] : result.messages
        this.nextCursor = result.next_cursor
      } catch (error) {
        if (seq === this.requestSeq && useAuthStore().familyId === familyId) this.error = error instanceof Error ? error.message : '猫管家暂时不可用'
      } finally {
        if (seq === this.requestSeq) this.loading = false
      }
    },
    async getMessage(id: string): Promise<AgentMessage | null> {
      const familyId = useAuthStore().familyId
      if (!familyId) return null
      const message = await agentApi.getMessage(familyId, id)
      if (useAuthStore().familyId !== familyId) return null
      return message
    },
    async editDraft(id: string, data: AgentReminderEditRequest): Promise<AgentReminderDraft | null> {
      const familyId = useAuthStore().familyId
      if (!familyId) return null
      const draft = await agentApi.editDraft(familyId, id, data)
      if (useAuthStore().familyId !== familyId) return null
      const item = this.messages.find((m) => m.id === id)
      if (item) {
        item.type = 'reminder_draft'
        item.action_status = 'pending'
        item.draft_version = draft.version
        item.draft_reminder = draft.reminder
        item.draft_expires_at = draft.expires_at
      }
      return draft
    },
    async dismiss(id: string) {
      const familyId = useAuthStore().familyId
      if (!familyId) return
      await agentApi.dismiss(familyId, id)
      if (useAuthStore().familyId !== familyId) return
      const item = this.messages.find((m) => m.id === id)
      if (item) {
        item.display_status = 'dismissed'
        if (item.action_status === 'pending') item.action_status = 'dismissed'
      }
    },
    async confirm(id: string, version: number): Promise<AgentConfirmResponse | null> {
      const familyId = useAuthStore().familyId
      if (!familyId) return null
      const result = await agentApi.confirm(familyId, id, version)
      if (useAuthStore().familyId !== familyId) return null
      const item = this.messages.find((m) => m.id === id)
      if (item) {
        item.action_status = 'confirmed'
        item.draft_version = version
      }
      const reminderStore = useReminderStore()
      const reminder = result.reminder
      if (!reminderStore.items.some((entry) => entry.id === reminder.id)) {
        reminderStore.items.unshift({
          id: reminder.id, catId: reminder.cat_id, type: reminder.type as typeof reminderStore.items[number]['type'],
          title: reminder.title, subtitle: reminder.subtitle || '', time: reminder.time || '', state: 'todo',
          icon: reminder.icon, rule: reminder.rule, scheduled_at: reminder.scheduled_at, timezone: reminder.timezone
        })
      }
      return result
    }
  }
})

function severityRank(value: AgentMessage['severity']): number {
  return value === 'danger' ? 3 : value === 'warning' ? 2 : 1
}
