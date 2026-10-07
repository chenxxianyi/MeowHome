import test from 'node:test'
import assert from 'node:assert/strict'
import fs from 'node:fs'
import vm from 'node:vm'
import ts from 'typescript'
import { completedChatReply, mergeAgentMessages, newAgentClientId } from '../src/utils/agent-chat.ts'

function fixture(overrides = {}) {
  const auth = { familyId: 'family', user: { id: 'user' } }
  const answer = { id: 'answer', turn_id: 'turn', role: 'assistant', type: 'chat_answer', client_message_id: 'key', body: '真实保存结果', generated_at: '2026-10-06T10:00:00Z' }
  const api = { sessions: async () => ({ sessions: [], next_cursor: '' }), sessionMessages: async () => ({ messages: [], next_cursor: '' }), ...overrides }
  const defineStore = (_id, options) => () => {
    const state = options.state()
    Object.entries(options.actions).forEach(([name, action]) => { state[name] = action.bind(state) })
    return state
  }
  const source = fs.readFileSync(new URL('../src/stores/agent.ts', import.meta.url), 'utf8')
  const output = ts.transpileModule(source, { compilerOptions: { module: ts.ModuleKind.CommonJS, target: ts.ScriptTarget.ES2020 } }).outputText
  const exports = {}
  const require = (name) => {
    if (name === 'pinia') return { defineStore }
    if (name === '../api/endpoints') return { agentApi: api }
    if (name === './auth') return { useAuthStore: () => auth }
    if (name === './reminder') return { useReminderStore: () => ({ items: [] }) }
    if (name === '../utils/agent-chat') return { completedChatReply, mergeAgentMessages, newAgentClientId }
    throw new Error(`Unexpected import: ${name}`)
  }
  vm.runInNewContext(output, { exports, require, console, Error })
  return { store: exports.useAgentStore(), auth, api, answer }
}

test('lost first response retries the same id and preserves input', async () => {
  const sent = []
  const f = fixture()
  f.api.chat = async (_family, request) => {
    sent.push({ ...request })
    if (sent.length === 1) throw new Error('timeout')
    return { session_id: 'session', status: 'completed', message: { ...f.answer, client_message_id: request.client_message_id } }
  }
  await f.store.sendChat('查询小白记录')
  assert.equal(f.store.pendingChat.message, '查询小白记录')
  assert.match(f.store.chatError, /timeout/)
  await f.store.retryChat()
  assert.equal(sent.length, 2)
  assert.equal(sent[0].client_message_id, sent[1].client_message_id)
  assert.equal(f.store.sessionId, 'session')
  assert.equal(f.store.pendingChat, null)
})

test('known-session timeout recovers answer before another POST', async () => {
  const f = fixture()
  let posts = 0
  let saved
  f.store.sessionId = 'session'
  f.api.chat = async (_family, request) => { posts++; saved = { ...f.answer, client_message_id: request.client_message_id }; throw new Error('timeout') }
  f.api.sessionMessages = async () => ({ messages: saved ? [saved] : [], next_cursor: '' })
  await f.store.sendChat('查询记录')
  await f.store.retryChat()
  assert.equal(posts, 1)
  assert.equal(f.store.pendingChat, null)
  assert.equal(f.store.chatMessages[0].id, 'answer')
})

test('running response remains recoverable and blocks duplicate send', async () => {
  const f = fixture()
  let release
  let calls = 0
  f.api.chat = async () => { calls++; await new Promise((resolve) => { release = resolve }); return { session_id: 'session', status: 'running', message: null } }
  const first = f.store.sendChat('查询记录')
  await f.store.sendChat('重复点击')
  assert.equal(calls, 1)
  release()
  await first
  assert.ok(f.store.pendingChat)
  assert.match(f.store.chatError, /处理中/)
  assert.equal(f.store.chatSending, false)
})

test('family switch discards delayed chat results', async () => {
  const f = fixture()
  let release
  f.api.chat = async () => { await new Promise((resolve) => { release = resolve }); return { session_id: 'old-session', status: 'completed', message: f.answer } }
  const first = f.store.sendChat('旧家庭问题')
  f.auth.familyId = 'new-family'
  f.store.reset()
  release()
  await first
  assert.equal(f.store.sessionId, '')
  assert.equal(f.store.chatMessages.length, 0)
  assert.equal(f.store.pendingChat, null)
  assert.equal(f.store.chatSending, false)
})

test('new conversation discards a delayed history response', async () => {
  const f = fixture()
  let release
  f.store.sessionId = 'session'
  f.api.sessionMessages = async () => { await new Promise((resolve) => { release = resolve }); return { messages: [f.answer], next_cursor: '' } }
  const loading = f.store.loadChatMessages()
  f.store.newChat()
  release()
  await loading
  assert.equal(f.store.chatMessages.length, 0)
  assert.equal(f.store.sessionId, '')
})

test('re-entry reconstructs the original request from running user message', async () => {
  const f = fixture({ sessions: async () => ({ sessions: [{ id: 'session' }], next_cursor: '' }), sessionMessages: async () => ({ messages: [{ id: 'user', role: 'user', type: 'chat_answer', client_message_id: 'key', body: '原问题', run_status: 'running', generated_at: '2026-10-06T10:00:00Z' }], next_cursor: '' }) })
  await f.store.resumeChat()
  assert.equal(f.store.sessionId, 'session')
  assert.equal(f.store.pendingChat.client_message_id, 'key')
  assert.equal(f.store.pendingChat.message, '原问题')
})

test('expired authentication clears private chat during bootstrap', async () => {
  const f = fixture()
  f.store.chatMessages = [f.answer]
  f.store.sessionId = 'session'
  f.store.pendingChat = { client_message_id: 'key', message: '私有问题' }
  const source = fs.readFileSync(new URL('../src/stores/auth.ts', import.meta.url), 'utf8')
  const output = ts.transpileModule(source, { compilerOptions: { module: ts.ModuleKind.CommonJS, target: ts.ScriptTarget.ES2020 } }).outputText
  const exports = {}
  const defineStore = (_id, options) => () => {
    const state = options.state()
    Object.entries(options.actions).forEach(([name, action]) => { state[name] = action.bind(state) })
    return state
  }
  const require = (name) => {
    if (name === 'pinia') return { defineStore }
    if (name === '../api/client') return { getToken: () => 'expired', getStoredFamilyId: () => 'family', clearAuthStorage: () => {} }
    if (name === '../api/endpoints') return { authApi: { me: async () => { throw new Error('401') } } }
    if (name === './agent') return { useAgentStore: () => f.store }
    throw new Error(`Unexpected import: ${name}`)
  }
  vm.runInNewContext(output, { exports, require, console, Error })
  const authStore = exports.useAuthStore()
  await authStore.bootstrap()
  assert.equal(authStore.familyId, null)
  assert.equal(authStore.isAuthenticated, false)
  assert.equal(f.store.chatMessages.length, 0)
  assert.equal(f.store.pendingChat, null)
})
