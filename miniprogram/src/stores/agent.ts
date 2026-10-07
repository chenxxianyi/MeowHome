import { defineStore } from 'pinia'
import { agentApi } from '../api/endpoints'
import { useAuthStore } from './auth'
import { useReminderStore } from './reminder'
import { completedChatReply, mergeAgentMessages, newAgentClientId } from '../utils/agent-chat'
import type { AgentChatRequest, AgentSessionSummary, AgentConfirmResponse, AgentMessage, AgentMessageListParams, AgentReminderEditRequest, AgentReminderDraft } from '../types/agent'

interface AgentState {
  messages: AgentMessage[]
  nextCursor?: string
  loading: boolean
  error: string
  familyId: string | null
  requestSeq: number
  chatSeq: number
  sessions: AgentSessionSummary[]
  sessionsCursor: string
  sessionId: string
  chatMessages: AgentMessage[]
  chatCursor: string
  chatSending: boolean
  chatLoading: boolean
  chatError: string
  pendingChat: AgentChatRequest | null
}

export const useAgentStore = defineStore('agent', {
  state: (): AgentState => ({ messages: [], nextCursor: undefined, loading: false, error: '', familyId: null, requestSeq: 0, chatSeq: 0, sessions: [], sessionsCursor: '', sessionId: '', chatMessages: [], chatCursor: '', chatSending: false, chatLoading: false, chatError: '', pendingChat: null }),
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
      this.newChat()
      this.sessions = []
      this.sessionsCursor = ''
    },
    newChat() {
      this.chatSeq++
      this.sessionId = ''
      this.chatMessages = []
      this.chatCursor = ''
      this.chatSending = false
      this.chatLoading = false
      this.chatError = ''
      this.pendingChat = null
    },
    async loadSessions(before = '') {
      const familyId = useAuthStore().familyId
      if (!familyId) return
      const seq = this.chatSeq
      const result = await agentApi.sessions(familyId, before ? { before } : {})
      if (seq !== this.chatSeq || useAuthStore().familyId !== familyId) return
      this.sessions = before ? [...this.sessions, ...result.sessions] : result.sessions
      this.sessionsCursor = result.next_cursor
    },
    async resumeChat(requestedSessionId = '') {
      const familyId = useAuthStore().familyId
      if (!familyId) return
      const seq = this.chatSeq
      this.chatError = ''
      try {
        await this.loadSessions()
        if (seq !== this.chatSeq || useAuthStore().familyId !== familyId) return
        const target = requestedSessionId || this.sessionId || this.sessions[0]?.id
        if (target) await this.openSession(target)
      } catch (error) {
        if (seq === this.chatSeq && useAuthStore().familyId === familyId) this.chatError = error instanceof Error ? error.message : '会话读取失败'
      }
    },
    async openSession(id: string) {
      if (this.chatSending) return
      if (this.sessionId !== id) this.newChat()
      this.sessionId = id
      await this.loadChatMessages()
    },
    async loadChatMessages(before = '') {
      const familyId = useAuthStore().familyId
      if (!familyId || !this.sessionId) return
      const seq = this.chatSeq
      const sessionId = this.sessionId
      this.chatLoading = true
      try {
        const result = await agentApi.sessionMessages(familyId, sessionId, before ? { before } : {})
        if (seq !== this.chatSeq || useAuthStore().familyId !== familyId) return
        this.chatMessages = mergeAgentMessages(before ? this.chatMessages : [], result.messages)
        this.chatCursor = result.next_cursor
        const incomplete = [...this.chatMessages].reverse().find((m) => m.role === 'user' && ['running', 'failed', 'pending'].includes(m.run_status || '') && m.client_message_id && !completedChatReply(this.chatMessages, m.client_message_id))
        if (incomplete?.client_message_id) {
          this.pendingChat = { session_id: sessionId, client_message_id: incomplete.client_message_id, message: incomplete.body }
        } else if (this.pendingChat && completedChatReply(this.chatMessages, this.pendingChat.client_message_id)) this.pendingChat = null
      } finally {
        if (seq === this.chatSeq) this.chatLoading = false
      }
    },
    async sendChat(message: string) {
      if (this.chatSending || this.pendingChat || !message.trim()) return
      this.pendingChat = { session_id: this.sessionId || undefined, client_message_id: newAgentClientId(), message: message.trim() }
      await this.retryChat()
    },
    async retryChat() {
      const familyId = useAuthStore().familyId
      const pending = this.pendingChat
      if (!familyId || !pending || this.chatSending) return
      const seq = this.chatSeq
      this.chatSending = true
      this.chatError = ''
      try {
        // After a client timeout first recover the saved result, then retry the
        // exact same ID. Never send a second logical turn for a lost response.
        if (this.sessionId) {
          await this.loadChatMessages()
          if (seq !== this.chatSeq || useAuthStore().familyId !== familyId) return
          if (completedChatReply(this.chatMessages, pending.client_message_id)) { this.pendingChat = null; return }
        }
        const result = await agentApi.chat(familyId, pending)
        if (seq !== this.chatSeq || useAuthStore().familyId !== familyId) return
        this.sessionId = result.session_id
        pending.session_id = result.session_id
        if (result.message) this.chatMessages = mergeAgentMessages(this.chatMessages, [result.message])
        await this.loadChatMessages()
        if (seq !== this.chatSeq) return
        if (result.status === 'completed') this.pendingChat = null
        else this.chatError = '本轮仍在处理中，请稍后点击恢复结果；中断轮次会在租约到期后继续。'
        await this.loadSessions()
      } catch (error) {
        if (seq === this.chatSeq && useAuthStore().familyId === familyId) this.chatError = error instanceof Error ? error.message : '发送失败，请恢复结果后重试'
      } finally {
        if (seq === this.chatSeq) this.chatSending = false
      }
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
      const item = [...this.messages, ...this.chatMessages].find((m) => m.id === id)
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
      const item = [...this.messages, ...this.chatMessages].find((m) => m.id === id)
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
      const item = [...this.messages, ...this.chatMessages].find((m) => m.id === id)
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
