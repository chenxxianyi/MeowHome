import test from 'node:test'
import assert from 'node:assert/strict'
import { readFileSync } from 'node:fs'
import { dirname, resolve } from 'node:path'
import { fileURLToPath } from 'node:url'
import vm from 'node:vm'

const buildRoot = fileURLToPath(new URL('../../dist/build/mp-weixin/', import.meta.url))

function fixture() {
  const storage = new Map()
  const requests = []
  const timers = new Map()
  const redirects = []
  let timerId = 0
  const uni = {
    getStorageSync: (key) => storage.get(key),
    setStorageSync: (key, value) => storage.set(key, value),
    removeStorageSync: (key) => storage.delete(key),
    reLaunch: (options) => redirects.push(options),
    request: (options) => {
      const request = { options, aborts: 0 }
      requests.push(request)
      return {
        abort() {
          request.aborts++
          options.fail({ errMsg: 'request:fail abort' })
        }
      }
    }
  }
  const modules = new Map()
  const setTimeout = (callback, delay) => {
    const id = ++timerId
    timers.set(id, { callback, delay })
    return id
  }
  const clearTimeout = (id) => timers.delete(id)
  function load(path) {
    if (path === resolve(buildRoot, 'common/vendor.js')) return { index: uni }
    if (modules.has(path)) return modules.get(path).exports
    const module = { exports: {} }
    modules.set(path, module)
    vm.runInNewContext(
      readFileSync(path, 'utf8'),
      {
        module,
        exports: module.exports,
        require: (name) => load(resolve(dirname(path), name)),
        setTimeout,
        clearTimeout,
        getCurrentPages: () => [{ route: 'pages/today/index' }]
      },
      { filename: path }
    )
    return module.exports
  }
  const client = load(resolve(buildRoot, 'api/client.js'))
  function expire(delay) {
    for (const [id, timer] of [...timers]) {
      if (timer.delay === delay) {
        timers.delete(id)
        timer.callback()
      }
    }
  }
  return { client, requests, timers, storage, redirects, expire }
}

test('WeChat request rejects and aborts when native callbacks never arrive', async () => {
  const f = fixture()
  const pending = f.client.request(f.client.http, { method: 'POST', url: '/auth/register' })
  assert.equal(f.timers.size, 1)
  const rejected = assert.rejects(pending, (error) => error.code === 'REQUEST_TIMEOUT')
  f.expire(15000)
  await rejected
  assert.equal(f.requests[0].aborts, 1)
  assert.equal(f.timers.size, 0)
})

test('WeChat successful response cancels the request deadline', async () => {
  const f = fixture()
  const pending = f.client.request(f.client.http, { method: 'GET', url: '/me' })
  f.requests[0].options.success({ statusCode: 200, data: { code: 'SUCCESS', data: { id: 'user' } } })
  assert.equal((await pending).id, 'user')
  assert.equal(f.timers.size, 0)
  f.expire(15000)
  assert.equal(f.requests[0].aborts, 0)
})

test('WeChat network failure rejects and releases the request deadline', async () => {
  const f = fixture()
  const pending = f.client.request(f.client.http, { method: 'POST', url: '/auth/register' })
  const rejected = assert.rejects(pending, (error) => error.code === 'NETWORK_ERROR')
  f.requests[0].options.fail({ errMsg: 'request:fail connection refused' })
  await rejected
  assert.equal(f.timers.size, 0)
})

test('WeChat ignores a late success after the request has timed out', async () => {
  const f = fixture()
  const pending = f.client.request(f.client.http, { method: 'POST', url: '/auth/register' })
  assert.equal(f.timers.size, 1)
  const rejected = assert.rejects(pending, (error) => error.code === 'REQUEST_TIMEOUT')
  f.expire(15000)
  await rejected
  f.requests[0].options.success({ statusCode: 200, data: { code: 'SUCCESS', data: { id: 'late-user' } } })
  assert.equal(f.timers.size, 0)
  assert.equal(f.storage.size, 0)
})

test('WeChat refresh deadline clears authentication and ignores late credentials', async () => {
  const f = fixture()
  f.client.saveTokens('expired', 'refresh')
  const pending = f.client.request(f.client.http, { method: 'GET', url: '/me' })
  const rejected = assert.rejects(pending, (error) => error.code === 'AUTH_REQUIRED')
  f.requests[0].options.success({ statusCode: 401, data: { code: 'AUTH_REQUIRED' } })
  await Promise.resolve()
  await Promise.resolve()
  assert.equal(f.requests.length, 2)
  assert.equal(f.timers.size, 1)
  f.expire(10000)
  await rejected
  assert.equal(f.requests[1].aborts, 1)
  assert.equal(f.storage.size, 0)
  assert.equal(f.redirects[0].url, '/pages/auth/index')
  f.requests[1].options.success({
    statusCode: 200,
    data: { code: 'SUCCESS', data: { access_token: 'late', refresh_token: 'late-refresh' } }
  })
  assert.equal(f.storage.size, 0)
})
