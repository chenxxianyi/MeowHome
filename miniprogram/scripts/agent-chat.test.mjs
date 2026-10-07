import test from 'node:test'
import assert from 'node:assert/strict'
import { mergeAgentMessages, completedChatReply, newAgentClientId } from '../src/utils/agent-chat.ts'

const msg = (id, values = {}) => ({ id, role: 'assistant', type: 'chat_answer', generated_at: '2026-10-06T10:00:00Z', turn_id: 'turn', ...values })

test('recovery merges updated draft state without duplicate messages', () => {
  const current = [msg('draft', { type: 'reminder_draft', draft_version: 1 }), msg('answer')]
  const recovered = mergeAgentMessages(current, [msg('draft', { type: 'reminder_draft', draft_version: 2, action_status: 'confirmed' }), msg('user', { role: 'user' })])
  assert.deepEqual(recovered.map((m) => m.id), ['user', 'draft', 'answer'])
  assert.equal(recovered[1].draft_version, 2)
  assert.equal(recovered[1].action_status, 'confirmed')
})

test('pending user and saved draft do not count as completed answer', () => {
  const messages = [msg('user', { role: 'user', client_message_id: 'key' }), msg('draft', { type: 'reminder_draft', client_message_id: 'key' })]
  assert.equal(completedChatReply(messages, 'key'), undefined)
  const answer = msg('answer', { client_message_id: 'key', degraded: true })
  assert.equal(completedChatReply([...messages, answer], 'key'), answer)
  assert.equal(completedChatReply([...messages, answer], 'another-key'), undefined)
})

test('new submissions use bounded distinct idempotency keys', () => {
  const keys = Array.from({ length: 100 }, newAgentClientId)
  assert.equal(new Set(keys).size, 100)
  assert.ok(keys.every((key) => key.length > 0 && key.length <= 128))
})
