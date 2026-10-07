import type { AgentMessage } from '../types/agent'

export function mergeAgentMessages(current: AgentMessage[], incoming: AgentMessage[]): AgentMessage[] {
  const byId = new Map(current.map((message) => [message.id, message]))
  incoming.forEach((message) => byId.set(message.id, message))
  return [...byId.values()].sort((a, b) => {
    const time = a.generated_at.localeCompare(b.generated_at)
    if (time) return time
    if (a.turn_id && a.turn_id === b.turn_id) {
      const rank = (m: AgentMessage) => m.role === 'user' ? 0 : m.type === 'reminder_draft' ? 1 : 2
      return rank(a) - rank(b) || a.id.localeCompare(b.id)
    }
    return a.id.localeCompare(b.id)
  })
}

export function completedChatReply(messages: AgentMessage[], clientId: string): AgentMessage | undefined {
  return messages.find((message) => message.role === 'assistant' && message.client_message_id === clientId && message.type === 'chat_answer')
}

export function newAgentClientId(): string {
  return `chat-${Date.now().toString(36)}-${Math.random().toString(36).slice(2)}-${Math.random().toString(36).slice(2)}`
}
