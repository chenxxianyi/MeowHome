import test from 'node:test'
import assert from 'node:assert/strict'
import { readFileSync } from 'node:fs'
import { dirname, resolve } from 'node:path'
import { fileURLToPath } from 'node:url'
import vm from 'node:vm'

const buildRoot = fileURLToPath(new URL('../../dist/build/mp-weixin/', import.meta.url))

function fixture({ apiBase, throwRequests = false } = {}) {
  const requests = []
  const timers = new Map()
  const logs = []
  const networkCalls = []
  let launch
  let timerId = 0
  const request = (transport) => (options) => {
    if (throwRequests) throw new Error('platform request unavailable')
    const entry = { transport, options, aborts: 0 }
    requests.push(entry)
    return {
      abort() {
        entry.aborts++
        options.fail({ errMsg: 'request:fail abort' })
      }
    }
  }
  const vendor = {
    index: { request: request('uni') },
    wx$1: { request: request('wx') },
    defineComponent: (component) => component,
    onLaunch: (callback) => {
      launch = callback
    },
    createSSRApp: (component) => ({
      use() {},
      mount() {
        component.setup()
      }
    }),
    createPinia: () => ({})
  }
  const dependencies = new Map([
    [resolve(buildRoot, 'common/vendor.js'), vendor],
    [
      resolve(buildRoot, 'stores/app.js'),
      {
        useAppStore: () => ({
          syncNetworkStatus: () => networkCalls.push('sync'),
          listenNetworkStatus: () => networkCalls.push('listen')
        })
      }
    ],
    // Reproduce the missing export reported by the real device. Startup must
    // remain independent of this obsolete diagnostic module.
    [resolve(buildRoot, 'utils/network-diagnostics.js'), {}]
  ])
  if (apiBase) dependencies.set(resolve(buildRoot, 'api/config.js'), { API_BASE: apiBase })
  const modules = new Map()
  function load(path) {
    if (dependencies.has(path)) return dependencies.get(path)
    if (modules.has(path)) return modules.get(path).exports
    const module = { exports: {} }
    modules.set(path, module)
    vm.runInNewContext(
      readFileSync(path, 'utf8'),
      {
        module,
        exports: module.exports,
        require: (name) => load(resolve(dirname(path), name)),
        console: { info: (...args) => logs.push(args) },
        setTimeout: (callback, delay) => {
          const id = ++timerId
          timers.set(id, { callback, delay })
          return id
        },
        clearTimeout: (id) => timers.delete(id)
      },
      { filename: path }
    )
    return module.exports
  }
  load(resolve(buildRoot, 'app.js'))
  return {
    requests,
    timers,
    logs,
    networkCalls,
    launch: () => launch(),
    expire: () => {
      for (const [id, timer] of [...timers]) {
        assert.equal(timer.delay, 6000)
        timers.delete(id)
        timer.callback()
      }
    },
    summary: () => JSON.parse(logs.find(([label]) => label === '[network-probe] summary')[1])
  }
}

const flushPromises = () => new Promise((resolve) => setImmediate(resolve))

test('compiled app launches despite an obsolete diagnostic module missing its export', () => {
  const f = fixture()
  assert.doesNotThrow(f.launch)
  assert.deepEqual(f.networkCalls, ['sync', 'listen'])
  assert.equal(f.requests.length, 4)
  assert.deepEqual(
    f.requests.map(({ transport }) => transport),
    ['uni', 'uni', 'wx', 'wx']
  )
  for (const { options } of f.requests) {
    assert.match(options.url, /^http:\/\/[^/]+\/health\/live$/)
    assert.equal(options.method, 'GET')
    assert.equal(options.data, undefined)
    assert.deepEqual(Object.keys(options.header), ['X-Request-Id'])
  }
  f.launch()
  assert.equal(f.requests.length, 4)
})

test('compiled app initializes network listeners and finishes probes when callbacks never arrive', async () => {
  const f = fixture()
  f.launch()
  assert.deepEqual(f.networkCalls, ['sync', 'listen'])
  assert.equal(f.timers.size, 4)
  f.expire()
  await flushPromises()
  assert.equal(f.timers.size, 0)
  assert.equal(f.summary().length, 4)
  assert.ok(f.summary().every(({ phase }) => phase === 'deadline'))
  assert.ok(f.requests.every(({ aborts }) => aborts === 1))
  for (const { options } of f.requests) options.success({ statusCode: 200 })
  assert.ok(f.summary().every(({ phase }) => phase === 'deadline'))
})

test('compiled app summarizes successful framework and native probes and clears their timers', async () => {
  const f = fixture()
  f.launch()
  for (const { options } of f.requests) options.success({ statusCode: 200 })
  await flushPromises()
  assert.equal(f.timers.size, 0)
  assert.deepEqual(
    f.summary().map(({ probe, phase, status }) => ({ probe, phase, status })),
    ['uni-json', 'uni-json-complete', 'wx-json', 'wx-text'].map((probe) => ({
      probe,
      phase: 'success',
      status: 200
    }))
  )
})

test('compiled app keeps startup working when the platform throws for every diagnostic request', async () => {
  const f = fixture({ throwRequests: true })
  assert.doesNotThrow(f.launch)
  await flushPromises()
  assert.deepEqual(f.networkCalls, ['sync', 'listen'])
  assert.equal(f.timers.size, 0)
  assert.ok(f.summary().every(({ phase }) => phase === 'threw'))
})

test('compiled app with an HTTPS API does not run local diagnostics', () => {
  const f = fixture({ apiBase: 'https://example.test/api/v1' })
  assert.doesNotThrow(f.launch)
  assert.deepEqual(f.networkCalls, ['sync', 'listen'])
  assert.equal(f.requests.length, 0)
  assert.equal(f.timers.size, 0)
  assert.equal(
    f.logs.some(([label]) => label === '[network-probe] auto.start'),
    false
  )
})
