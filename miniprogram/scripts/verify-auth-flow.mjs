/**
 * 步骤 1.5 认证链路联调 —— 小程序网络层「协议镜像」验证
 *
 * ## 这个脚本验证什么，不验证什么
 *
 * 微信开发者工具里的模拟器无法被自动化驱动，所以本脚本**不是**端到端 UI 测试。
 * 它把 `src/api/client.ts` 的协议行为在 Node 里逐行复刻一遍，再打**真实后端**，
 * 以此证明「小程序发出的请求形态」与「后端期望的形态」是一致的。
 *
 *   ✅ 验证：请求头（Authorization / X-Request-Id / X-HTTP-Method-Override）、
 *            Envelope 解包、401 → 刷新 → 重试、刷新单飞、跨家庭 403
 *   ❌ 不验证：uni.request 本身、页面跳转、UI 渲染 —— 这些只能在 DevTools 里看
 *
 * ## 为什么不用「乱码 token」测 401
 *
 * 冒烟脚本已用乱码/篡改 token 测过 401。但 1.5 要验的是「**令牌过期**」这条路径，
 * 与「令牌无效」在服务端是两个分支（ErrExpired vs ErrSignature）。这里用
 * `backend/.env` 里的真实密钥现铸一个**签名合法但已过期**的 token，命中真正的过期分支。
 *
 * 用法：
 *   node scripts/verify-auth-flow.mjs
 *   node scripts/verify-auth-flow.mjs --origin http://127.0.0.1:8080
 */
import { createHmac } from 'node:crypto'
import { readFileSync, writeFileSync } from 'node:fs'
import { dirname, resolve } from 'node:path'
import { fileURLToPath } from 'node:url'

const __dirname = dirname(fileURLToPath(import.meta.url))

const argOrigin = process.argv.indexOf('--origin')
const ORIGIN = (argOrigin > -1 ? process.argv[argOrigin + 1] : null) || 'http://127.0.0.1:8080'
const apiBase = `${ORIGIN}/api/v1`

// ------------------------------------------------------------------ 测试框架

const results = []
let currentGroup = ''

function group(name) {
  currentGroup = name
}

function check(name, ok, detail) {
  results.push({ group: currentGroup, name, ok: !!ok, detail: detail ?? '' })
  const mark = ok ? 'PASS' : 'FAIL'
  // 控制台只输出 ASCII，中文进报告文件（避免 Windows 终端编码干扰判读）
  console.log(`  [${mark}] ${name}${detail ? `  = ${detail}` : ''}`)
  return !!ok
}

function assertEqual(name, actual, expected) {
  return check(name, actual === expected, `${JSON.stringify(actual)} (expect ${JSON.stringify(expected)})`)
}

// ------------------------------------------------------------------ 密钥

function loadSecret() {
  if (process.env.MEOWHOME_JWT_SECRET) return process.env.MEOWHOME_JWT_SECRET
  try {
    const env = readFileSync(resolve(__dirname, '../../backend/.env'), 'utf8')
    const m = env.match(/^JWT_SECRET=(.+)$/m)
    if (m && m[1].trim()) return m[1].trim()
  } catch {
    /* 落到下方报错 */
  }
  throw new Error('未找到 JWT_SECRET：请设置环境变量 MEOWHOME_JWT_SECRET，或确认 backend/.env 存在')
}

/** 复刻 backend/internal/platform/token/token.go 的 Sign，但把 exp 放到过去。 */
function mintExpiredToken(secret, uid, ageSeconds = 1800) {
  const now = Math.floor(Date.now() / 1000)
  const payload = JSON.stringify({ uid, iat: now - ageSeconds * 2, exp: now - ageSeconds })
  const body = Buffer.from(payload).toString('base64url') // = Go 的 base64.RawURLEncoding
  const sig = createHmac('sha256', secret).update(body).digest('hex')
  return `${body}.${sig}`
}

// ------------------------------------------------------------------ client.ts 镜像

const TOKEN_KEY = 'meowhome.token'
const REFRESH_KEY = 'meowhome.refresh'
const FAMILY_KEY = 'meowhome.familyId'

/** 镜像 uni 的同步存储（getStorageSync / setStorageSync / removeStorageSync）。 */
const store = new Map()
const storage = {
  get: (k) => {
    const v = store.get(k)
    return typeof v === 'string' && v.length > 0 ? v : null
  },
  set: (k, v) => store.set(k, v),
  remove: (k) => store.delete(k),
  clearAuth() {
    store.delete(TOKEN_KEY)
    store.delete(REFRESH_KEY)
    store.delete(FAMILY_KEY)
  }
}

const counters = { refreshCalls: 0, redirectToLogin: 0, requests: 0 }

function genRequestId() {
  return `req-${Date.now()}-${Math.random().toString(16).slice(2, 10)}`
}

function buildQuery(params) {
  if (!params) return ''
  const parts = []
  for (const key of Object.keys(params)) {
    const v = params[key]
    if (v === undefined || v === null || v === '') continue
    parts.push(`${encodeURIComponent(key)}=${encodeURIComponent(String(v))}`)
  }
  return parts.length ? `?${parts.join('&')}` : ''
}

/** 镜像 rawRequest：同样的头、同样的 PATCH 降级。 */
async function rawRequest(config, token) {
  const header = { 'Content-Type': 'application/json', 'X-Request-Id': genRequestId() }
  if (token) header.Authorization = `Bearer ${token}`

  const isPatch = config.method === 'PATCH'
  const method = isPatch ? 'POST' : config.method
  if (isPatch) header['X-HTTP-Method-Override'] = 'PATCH'

  counters.requests++
  const res = await fetch(apiBase + config.url + buildQuery(config.params), {
    method,
    headers: header,
    body: config.data === undefined ? undefined : JSON.stringify(config.data)
  })
  const text = await res.text()
  let data = null
  if (text) {
    try {
      data = JSON.parse(text)
    } catch {
      data = text
    }
  }
  return { statusCode: res.status, data, sentMethod: method, sentHeader: header }
}

/** 镜像 refreshAccessToken（绕开本方封装，避免递归）。 */
async function refreshAccessToken() {
  const refresh = storage.get(REFRESH_KEY)
  if (!refresh) return null
  counters.refreshCalls++
  try {
    const res = await fetch(`${apiBase}/auth/refresh`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ refresh_token: refresh })
    })
    const body = await res.json().catch(() => null)
    const data = body && body.data
    if (res.status === 200 && data && data.access_token) {
      storage.set(TOKEN_KEY, data.access_token)
      if (data.refresh_token) storage.set(REFRESH_KEY, data.refresh_token)
      return data.access_token
    }
    return null
  } catch {
    return null
  }
}

/** 镜像 send：401 → 单飞刷新 → 重试一次。 */
let refreshing = null
async function send(config) {
  const first = await rawRequest(config, storage.get(TOKEN_KEY))
  if (first.statusCode !== 401) return first

  if (!refreshing) {
    refreshing = refreshAccessToken().finally(() => {
      refreshing = null
    })
  }
  const token = await refreshing
  if (token) return rawRequest(config, token)

  storage.clearAuth()
  counters.redirectToLogin++
  return first
}

/** 镜像 client.ts 的 assertEnvelope：非 Envelope 响应必须抛错，不能静默当成功。 */
function assertEnvelope(res) {
  const body = res.data
  if (!body || typeof body !== 'object' || typeof body.code !== 'string') {
    const err = new Error(
      res.statusCode >= 400
        ? `服务返回异常状态（HTTP ${res.statusCode}）`
        : `响应格式不符合约定（HTTP ${res.statusCode}）`
    )
    err.code = `HTTP_${res.statusCode}`
    err.statusCode = res.statusCode
    err.raw = typeof body === 'string' ? body.slice(0, 60) : body
    throw err
  }
  return body
}

/** 镜像 request：校验 Envelope，解包 data，失败抛错。 */
async function request(config) {
  const res = await send(config)
  if (res.statusCode === 204 || res.data == null) return undefined
  const env = assertEnvelope(res)
  if (env.code !== 'SUCCESS') {
    const err = new Error(`${env.code}: ${env.message}`)
    err.code = env.code
    err.statusCode = res.statusCode
    throw err
  }
  return env.data
}

// ------------------------------------------------------------------ 主流程

const secret = loadSecret()
const stamp = Date.now()
const ALICE = `mp-alice-${stamp}@example.com`
const BOB = `mp-bob-${stamp}@example.com`
const PASSWORD = 'miniprogram-2026'

console.log(`\n=== 1.5 认证链路联调（协议镜像） ===\norigin: ${ORIGIN}\n`)

// ---- 1) 注册 → 建家庭 → /me 返回 family_id ----
group('注册 → 建家庭 → /me')

const reg = await request({
  method: 'POST',
  url: '/auth/register',
  data: { email: ALICE, password: PASSWORD, user_name: '爱丽丝' }
})
check('POST /auth/register 返回 access_token', !!reg.access_token)
check('POST /auth/register 返回 refresh_token', !!reg.refresh_token)
assertEqual('access_token 段数', String(reg.access_token).split('.').length, 2)
check('注册返回 user', !!reg.user && !!reg.user.id, `user.id=${reg.user && reg.user.id}`)
assertEqual(
  'user 字段名为 snake_case（与其余接口一致）',
  Object.keys(reg.user || {}).sort().join(','),
  'email,id,name'
)
check(
  '注册响应未下发密码哈希',
  reg.user && !Object.keys(reg.user).some((k) => k.toLowerCase().includes('password')),
  `keys=${Object.keys(reg.user || {}).join(',')}`
)
assertEqual('user.email 与注册邮箱一致', reg.user && reg.user.email, ALICE)
assertEqual('user.name 与注册昵称一致', reg.user && reg.user.name, '爱丽丝')

storage.set(TOKEN_KEY, reg.access_token)
storage.set(REFRESH_KEY, reg.refresh_token)

const meBefore = await request({ method: 'GET', url: '/me' })
check('注册后 /me 可访问', !!meBefore.id)
check('注册后尚未加入家庭（family_id 为空 → 守卫应送去 onboarding）', !meBefore.family_id, `family_id=${JSON.stringify(meBefore.family_id)}`)

const fam = await request({ method: 'POST', url: '/families', data: { name: '爱丽丝的猫宅' } })
check('POST /families 返回家庭 id', !!fam.id, `id=${fam.id}`)
storage.set(FAMILY_KEY, fam.id)

const meAfter = await request({ method: 'GET', url: '/me' })
assertEqual('/me 返回 family_id 且已注入', meAfter.family_id, fam.id)

const cats0 = await request({ method: 'GET', url: `/families/${fam.id}/cats` })
check('GET cats 可访问', Array.isArray(cats0), `length=${cats0 && cats0.length}`)

// ---- 2) 令牌过期 → 401 → 自动刷新 → 重试成功 ----
group('令牌过期 → 自动刷新')

const validAccess = reg.access_token
const expired = mintExpiredToken(secret, meBefore.id)
check('已铸造签名合法但过期的 token', expired.split('.')[0] !== validAccess.split('.')[0])

storage.set(TOKEN_KEY, expired)
const refreshBefore = counters.refreshCalls
const meRefreshed = await request({ method: 'GET', url: '/me' })

check('过期令牌下 /me 仍成功（说明走完了刷新+重试）', !!meRefreshed.id, `id=${meRefreshed.id}`)
assertEqual('刷新被调用 1 次', counters.refreshCalls - refreshBefore, 1)
check('存储中的 access_token 已轮换', storage.get(TOKEN_KEY) !== expired)
check('存储中的 refresh_token 已轮换（后端单次使用）', storage.get(REFRESH_KEY) !== reg.refresh_token)

// ---- 3) 并发 401 只触发一次刷新（单飞） ----
group('并发 401 单飞')

storage.set(TOKEN_KEY, mintExpiredToken(secret, meBefore.id))
const before3 = counters.refreshCalls
const parallel = await Promise.all([
  request({ method: 'GET', url: '/me' }),
  request({ method: 'GET', url: '/me' }),
  request({ method: 'GET', url: '/me' }),
  request({ method: 'GET', url: '/me' })
])
assertEqual('4 个并发请求全部成功', parallel.filter((r) => r && r.id).length, 4)
assertEqual('刷新仅触发 1 次（未被 4 个请求各刷一次）', counters.refreshCalls - before3, 1)

// ---- 4) 刷新令牌失效 → 清空存储 + 跳登录页 ----
group('刷新令牌失效')

storage.set(TOKEN_KEY, mintExpiredToken(secret, meBefore.id))
storage.set(REFRESH_KEY, 'not-a-real-refresh-token')
const redirectBefore = counters.redirectToLogin

let threw = null
try {
  await request({ method: 'GET', url: '/me' })
} catch (e) {
  threw = e
}
check('请求最终失败（未静默成功）', !!threw, threw ? threw.code : 'no throw')
assertEqual('access_token 已被清空', storage.get(TOKEN_KEY), null)
assertEqual('refresh_token 已被清空', storage.get(REFRESH_KEY), null)
assertEqual('familyId 已被清空', storage.get(FAMILY_KEY), null)
assertEqual('已触发跳登录页', counters.redirectToLogin - redirectBefore, 1)

// ---- 5) 跨家庭隔离 403 ----
group('跨家庭隔离')

const regBob = await request({
  method: 'POST',
  url: '/auth/register',
  data: { email: BOB, password: PASSWORD, user_name: '鲍勃' }
})
storage.set(TOKEN_KEY, regBob.access_token)
storage.set(REFRESH_KEY, regBob.refresh_token)

let aliceFamily = null
try {
  aliceFamily = await request({ method: 'GET', url: `/families/${fam.id}` })
} catch (e) {
  aliceFamily = e
}
check('读他人家庭被拒 403', aliceFamily instanceof Error && aliceFamily.statusCode === 403, aliceFamily instanceof Error ? `${aliceFamily.code} / HTTP ${aliceFamily.statusCode}` : 'unexpectedly succeeded')

let aliceCats = null
try {
  aliceCats = await request({ method: 'GET', url: `/families/${fam.id}/cats` })
} catch (e) {
  aliceCats = e
}
check('读他人猫咪列表被拒 403', aliceCats instanceof Error && aliceCats.statusCode === 403, aliceCats instanceof Error ? `${aliceCats.code} / HTTP ${aliceCats.statusCode}` : 'unexpectedly succeeded')

const bobFamilies = await request({ method: 'GET', url: '/families' })
assertEqual('鲍勃的家庭列表为空（未泄漏他人家庭）', Array.isArray(bobFamilies) ? bobFamilies.length : -1, 0)

// ---- 6) 非 Envelope 响应必须判为失败（client.ts 的静默吞错修复） ----
group('非 Envelope 响应处理')

// 打一个未注册的路径：Gin 返回 `404 page not found` 纯文本，不是 Envelope。
// 旧实现会在 `env.code` 为 undefined 时继续 return，把这串字符串当成功结果。
let nonEnvErr = null
try {
  await request({ method: 'GET', url: '/definitely-not-a-route' })
} catch (e) {
  nonEnvErr = e
}
check(
  '未注册路径会抛错（不再把 404 文本当成功）',
  !!nonEnvErr,
  nonEnvErr ? `code=${nonEnvErr.code} raw="${nonEnvErr.raw}"` : '静默成功了 —— client.ts 的修复失效'
)
assertEqual('抛出的是 HTTP_404 而非业务码', nonEnvErr && nonEnvErr.code, 'HTTP_404')

// 200 但响应体不是 Envelope，同样要抛错（防止约定被破坏时静默返回垃圾数据）
const bogusOk = await fetch(`${ORIGIN}/health/live`)
check('对照：健康检查确实不是 Envelope（验证上面用例的前提成立）', bogusOk.status === 200, `HTTP ${bogusOk.status}`)

// ---- 7) §11 阻塞项 #7：X-HTTP-Method-Override ----
group('阻塞项 #7：PATCH 覆盖头')

storage.set(TOKEN_KEY, validAccess)
storage.set(REFRESH_KEY, reg.refresh_token)

const nameBefore = (await request({ method: 'GET', url: `/families/${fam.id}` })).name

let patchErr = null
try {
  await request({ method: 'PATCH', url: `/families/${fam.id}`, data: { name: '覆盖名' } })
} catch (e) {
  patchErr = e
}
const nameAfterOverride = (await request({ method: 'GET', url: `/families/${fam.id}` })).name

check(
  'POST + X-HTTP-Method-Override: PATCH 能真正改到数据',
  !patchErr && nameAfterOverride === '覆盖名',
  `name ${nameBefore} -> ${nameAfterOverride}`
)

// 对照：原生 PATCH 是否可用（浏览器/Node 可用，小程序不可用）——用于区分
// 「路由不存在」与「只是方法名不被支持」，给 §11 阻塞项 #7 一个准确的结论。
let nativePatchStatus
let nativeName = null
try {
  const res = await fetch(`${apiBase}/families/${fam.id}`, {
    method: 'PATCH',
    headers: { 'Content-Type': 'application/json', Authorization: `Bearer ${validAccess}` },
    body: JSON.stringify({ name: '原生改名' })
  })
  nativePatchStatus = res.status
} catch (e) {
  nativePatchStatus = `error: ${e.message}`
}
nativeName = (await request({ method: 'GET', url: `/families/${fam.id}` })).name
check(
  '对照：原生 PATCH 返回 200 且真正生效',
  nativePatchStatus === 200 && nativeName === '原生改名',
  `HTTP ${nativePatchStatus}，name=${nativeName}`
)

// 安全回归：覆盖头只允许取 PATCH，不能被用来把 POST 变成 DELETE。
// （Gin 会按改写后的方法匹配路由，若不限制，等于给所有写接口开后门。）
const cat = await request({
  method: 'POST',
  url: `/families/${fam.id}/cats`,
  data: { name: '安全回归猫', gender: 'female' }
})
// 手工带上覆盖头，模拟把 POST 冒充成 DELETE 的客户端
const evil = await fetch(`${apiBase}/families/${fam.id}/cats/${cat.id}`, {
  method: 'POST',
  headers: {
    'Content-Type': 'application/json',
    Authorization: `Bearer ${validAccess}`,
    'X-HTTP-Method-Override': 'DELETE'
  }
})
const catsAfterEvil = await request({ method: 'GET', url: `/families/${fam.id}/cats` })
check(
  '覆盖头只放行 PATCH：POST+override:DELETE 不会误删数据',
  evil.status !== 204 && catsAfterEvil.some((c) => c.id === cat.id),
  `HTTP ${evil.status}，猫仍在=${catsAfterEvil.some((c) => c.id === cat.id)}`
)

check(
  '结论：§11 #7 已解除（覆盖头可用）',
  !patchErr && nameAfterOverride === '覆盖名',
  patchErr ? `仍失败：HTTP ${patchErr.statusCode}` : 'PATCH 降级链路打通'
)

// ------------------------------------------------------------------ 汇总与报告

const passed = results.filter((r) => r.ok).length
const failed = results.length - passed
const pct = ((passed / results.length) * 100).toFixed(1)

console.log(`\n================================`)
console.log(`通过 ${passed} / 失败 ${failed}  （${pct}%）`)
console.log(`请求数 ${counters.requests}，刷新次数 ${counters.refreshCalls}\n`)

const reportPath = resolve(__dirname, 'auth-flow-report.md')
const lines = [
  '# 1.5 认证链路联调 —— 协议镜像验证报告',
  '',
  `- 运行时间：${new Date().toISOString()}`,
  `- 目标后端：\`${ORIGIN}\``,
  `- 用例：**通过 ${passed} / 失败 ${failed}**（${pct}%）`,
  `- 请求数：${counters.requests}，刷新调用：${counters.refreshCalls}`,
  '',
  '> 说明：本报告由 `scripts/verify-auth-flow.mjs` 生成。它复刻 `src/api/client.ts` 的',
  '> 协议行为打真实后端，**不覆盖** uni.request 本身与页面跳转（那部分需 DevTools 目视）。',
  '',
  '| 分组 | 检查项 | 结果 | 实测值 |',
  '|---|---|---|---|',
  ...results.map((r) => `| ${r.group} | ${r.name} | ${r.ok ? '✅' : '❌'} | ${String(r.detail).replace(/\|/g, '\\|')} |`),
  ''
]
writeFileSync(reportPath, lines.join('\n'), 'utf8')
console.log(`报告已写入：${reportPath}`)

process.exit(failed === 0 ? 0 : 1)
