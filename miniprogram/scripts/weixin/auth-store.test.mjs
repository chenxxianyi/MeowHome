import test from 'node:test'
import assert from 'node:assert/strict'
import { readFileSync } from 'node:fs'
import { dirname, resolve } from 'node:path'
import { fileURLToPath } from 'node:url'
import vm from 'node:vm'
import { createPinia, defineStore, setActivePinia } from 'pinia'

const buildRoot = fileURLToPath(new URL('../../dist/build/mp-weixin/', import.meta.url))

// Execute the actual WeChat CommonJS output, including both store modules and
// their circular references. Only network/storage and the platform vendor are
// supplied by the fixture; the Pinia implementation is real.
function fixture({ token = 'valid', familyId = 'family', entry = 'auth' } = {}) {
  const storage = new Map([
    ['token', token],
    ['refresh', 'refresh'],
    ['family', familyId]
  ])
  const api = {
    me: async () => ({ id: 'user', name: '猫主人', role: 'owner', family_id: familyId }),
    myFamilies: async () => [{ id: familyId }],
    createFamily: async () => ({ id: 'new-family' }),
    logout: async () => {}
  }
  const client = {
    getToken: () => storage.get('token'),
    getRefreshToken: () => storage.get('refresh'),
    getStoredFamilyId: () => storage.get('family'),
    saveFamilyId: (id) => storage.set('family', id),
    saveTokens: (access, refresh) => {
      storage.set('token', access)
      if (refresh) storage.set('refresh', refresh)
    },
    clearAuthStorage: () => storage.clear()
  }
  const dependencies = new Map([
    [resolve(buildRoot, 'common/vendor.js'), { defineStore }],
    [resolve(buildRoot, 'api/client.js'), client],
    [
      resolve(buildRoot, 'api/endpoints.js'),
      {
        authApi: api,
        agentApi: {},
        toUser: ({ id, name, role }) => ({ id, name, role })
      }
    ]
  ])
  const modules = new Map()
  function load(path) {
    if (dependencies.has(path)) return dependencies.get(path)
    if (modules.has(path)) return modules.get(path).exports
    const module = { exports: {} }
    modules.set(path, module)
    const source = readFileSync(path, 'utf8')
    const require = (name) => load(resolve(dirname(path), name))
    vm.runInNewContext(source, { module, exports: module.exports, require, console }, { filename: path })
    return module.exports
  }

  setActivePinia(createPinia())
  load(resolve(buildRoot, `stores/${entry}.js`))
  const auth = load(resolve(buildRoot, 'stores/auth.js')).useAuthStore()
  const agent = load(resolve(buildRoot, 'stores/agent.js')).useAgentStore()
  agent.messages = [{ id: 'old-patrol' }]
  agent.sessions = [{ id: 'old-session' }]
  agent.sessionId = 'old-session'
  agent.chatMessages = [{ id: 'old-answer', body: '旧账号的私有消息' }]
  agent.pendingChat = { client_message_id: 'old-key', message: '旧账号的问题' }
  return { auth, agent, api, storage }
}

function assertAgentCleared(agent) {
  assert.equal(agent.messages.length, 0)
  assert.equal(agent.sessions.length, 0)
  assert.equal(agent.chatMessages.length, 0)
  assert.equal(agent.sessionId, '')
  assert.equal(agent.pendingChat, null)
  assert.equal(agent.chatSending, false)
  assert.equal(agent.chatLoading, false)
  assert.ok(agent.requestSeq > 0)
  assert.ok(agent.chatSeq > 0)
}

for (const entry of ['auth', 'agent']) {
  test(`WeChat output boots without a token when ${entry} is imported first`, async () => {
    const f = fixture({ token: null, entry })
    f.api.me = async () => {
      assert.fail('Logged-out startup must not request /me')
    }
    await f.auth.bootstrap()
    assert.equal(f.auth.ready, true)
    assert.equal(f.auth.isAuthenticated, false)
    assertAgentCleared(f.agent)
  })
}

test('WeChat output clears expired authentication and private Agent state', async () => {
  const f = fixture()
  f.api.me = async () => {
    throw new Error('401')
  }
  await f.auth.bootstrap()
  assert.equal(f.auth.ready, true)
  assert.equal(f.auth.isAuthenticated, false)
  assert.equal(f.auth.familyId, null)
  assert.equal(f.storage.size, 0)
  assertAgentCleared(f.agent)
})

test('WeChat output clears private Agent state when /me changes the family', async () => {
  const f = fixture()
  f.api.me = async () => ({ id: 'user', family_id: 'new-family' })
  await f.auth.bootstrap()
  assert.equal(f.auth.isAuthenticated, true)
  assert.equal(f.auth.familyId, 'new-family')
  assert.equal(f.storage.get('family'), 'new-family')
  assertAgentCleared(f.agent)
})

test('WeChat output clears private Agent state when resolving another family', async () => {
  const f = fixture()
  f.api.myFamilies = async () => [{ id: 'new-family' }]
  assert.equal(await f.auth.resolveFamily(), 'new-family')
  assert.equal(f.storage.get('family'), 'new-family')
  assertAgentCleared(f.agent)
})

test('WeChat output clears private Agent state when creating a family', async () => {
  const f = fixture()
  f.api.me = async () => ({ id: 'user', family_id: 'new-family' })
  assert.equal(await f.auth.createFamily('新家庭'), 'new-family')
  assert.equal(f.auth.familyId, 'new-family')
  assert.equal(f.storage.get('family'), 'new-family')
  assertAgentCleared(f.agent)
})

test('WeChat output clears the session even when remote logout fails', async () => {
  const f = fixture()
  f.auth.isAuthenticated = true
  f.api.logout = async () => {
    throw new Error('offline')
  }
  await f.auth.logout()
  assert.equal(f.auth.isAuthenticated, false)
  assert.equal(f.auth.user, null)
  assert.equal(f.auth.familyId, null)
  assert.equal(f.storage.size, 0)
  assertAgentCleared(f.agent)
})
