// Unit tests for MeowHome Vue 3 frontend
import { describe, it, expect } from 'vitest'
import { mount } from '@vue/test-utils'
import { createRouter, createWebHashHistory } from 'vue-router'

// —— 日期工具 ——
describe('date utils', () => {
  it('todayStr returns YYYY-MM-DD format', async () => {
    const mod = await import('@/utils/date')
    const s = mod.todayStr()
    expect(s).toMatch(/^\d{4}-\d{2}-\d{2}$/)
  })

  it('dateStr with 0 returns today', async () => {
    const mod = await import('@/utils/date')
    expect(mod.dateStr(0)).toBe(mod.todayStr())
  })

  it('dateStr with 1 returns yesterday', async () => {
    const mod = await import('@/utils/date')
    const d = new Date(mod.dateStr(1))
    const expected = new Date()
    expected.setDate(expected.getDate() - 1)
    expect(d.getFullYear()).toBe(expected.getFullYear())
    expect(d.getMonth()).toBe(expected.getMonth())
    expect(d.getDate()).toBe(expected.getDate())
  })

  it('greeting returns a string', async () => {
    const mod = await import('@/utils/date')
    expect(typeof mod.greeting()).toBe('string')
  })
})

// —— Mock 数据（保留为静态夹具，供图表占位与记录类型常量使用）——
describe('mock data', () => {
  it('mockCats has at least 2 cats', async () => {
    const { mockCats } = await import('@/mocks/family')
    expect(mockCats.length).toBeGreaterThanOrEqual(2)
  })

  it('mockFamily has members', async () => {
    const { mockFamily } = await import('@/mocks/family')
    expect(mockFamily.members.length).toBeGreaterThanOrEqual(2)
  })

  it('mockReminders has todo items', async () => {
    const { mockReminders } = await import('@/mocks/data')
    const todo = mockReminders.filter(r => r.state === 'todo')
    expect(todo.length).toBeGreaterThan(0)
  })

  it('mockRecordTypes has high/health/life groups', async () => {
    const { mockRecordTypes } = await import('@/mocks/data')
    expect(mockRecordTypes.high.length).toBeGreaterThan(0)
    expect(mockRecordTypes.health.length).toBeGreaterThan(0)
    expect(mockRecordTypes.life.length).toBeGreaterThan(0)
  })

  it('mockAIParseSession has records', async () => {
    const { mockAIParseSession } = await import('@/mocks/data')
    expect(mockAIParseSession.records.length).toBeGreaterThan(0)
    expect(mockAIParseSession.records[0].type).toBeDefined()
  })

  it('generateTrends returns 30 points', async () => {
    const { generateTrends } = await import('@/mocks/trends')
    const data = generateTrends('cat-whit')
    expect(data.length).toBe(30)
    expect(data[0].date).toBeDefined()
    expect(typeof data[0].weight).toBe('number')
  })
})

// —— 响应适配层 ——
describe('api adapter', () => {
  it('unwrap 把后端 Envelope 转成 APIResponse', async () => {
    const { unwrap } = await import('@/api/adapter')
    const r = unwrap({ code: 'SUCCESS', data: { id: 'x' }, request_id: 'rid-1' })
    expect(r.success).toBe(true)
    expect(r.data).toEqual({ id: 'x' })
    expect(r.requestId).toBe('rid-1')
  })

  it('unwrap 对非 SUCCESS 返回 success=false', async () => {
    const { unwrap } = await import('@/api/adapter')
    expect(unwrap({ code: 'CONFLICT', message: 'dup' }).success).toBe(false)
  })

  it('toApiError 解析业务错误体', async () => {
    const { toApiError } = await import('@/api/adapter')
    const err = toApiError({
      response: { status: 403, data: { code: 'FAMILY_FORBIDDEN', message: 'access denied' } }
    })
    expect(err.status).toBe(403)
    expect(err.code).toBe('FAMILY_FORBIDDEN')
    expect(err.isAuthError).toBe(false)
  })

  it('toApiError 把 401 标记为认证错误', async () => {
    const { toApiError } = await import('@/api/adapter')
    const err = toApiError({ response: { status: 401, data: { code: 'TOKEN_EXPIRED' } } })
    expect(err.isAuthError).toBe(true)
  })

  it('toApiError 对网络错误给出兜底码', async () => {
    const { toApiError } = await import('@/api/adapter')
    expect(toApiError(new Error('boom')).code).toBe('NETWORK_ERROR')
  })
})

// —— 后端 DTO → 前端类型 ——
describe('DTO 映射', () => {
  it('toCat 由生日推算年龄', async () => {
    const { toCat } = await import('@/api/endpoints')
    const cat = toCat({
      id: 'c1', family_id: 'f1', name: '小白', gender: 'female',
      breed: '中华田园猫', birthday: '2020-01-01', neutered: true
    })
    expect(cat.name).toBe('小白')
    expect(cat.gender).toBe('female')
    expect(cat.age).toBeGreaterThanOrEqual(5)
    expect(cat.neutered).toBe(true)
  })

  it('toCat 把无法识别的性别归一化为 unknown', async () => {
    const { toCat } = await import('@/api/endpoints')
    expect(toCat({ id: 'c1', family_id: 'f1', name: 'x', gender: 'weird' }).gender).toBe('unknown')
  })

  it('toInventory 保留后端推导的 status，空 expiry 归一为 null', async () => {
    const { toInventory } = await import('@/api/endpoints')
    const item = toInventory({
      id: 'i1', name: '豆腐猫砂', category: '猫砂', quantity: 2, unit: '袋', status: 'low', expiry: ''
    })
    expect(item.status).toBe('low')
    expect(item.expiry).toBeNull()
  })

  it('toExpense 保留小数金额', async () => {
    const { toExpense } = await import('@/api/endpoints')
    const e = toExpense({ id: 'e1', date: '2026-08-01', amount: 168.5, category: '粮食', label: '猫粮' })
    expect(e.amount).toBe(168.5)
  })

  it('toReminder 归一化 state 与 catId', async () => {
    const { toReminder } = await import('@/api/endpoints')
    const r = toReminder({
      id: 'r1', catId: '', type: 'medication', title: '服药', subtitle: '', time: '20:00', state: 'done', icon: ''
    })
    expect(r.state).toBe('done')
    expect(r.catId).toBe('both')
  })

  it('toParseSession 保留字段置信度', async () => {
    const { toParseSession } = await import('@/api/endpoints')
    const s = toParseSession({
      id: 's1', originalInput: 'text', parsedAt: 't', model: 'rule-based-v1',
      records: [{ id: 'rec-1', type: 'feeding', catId: 'c1', fields: [{ key: 'time', value: '08:00', confidence: 'high' }] }]
    })
    expect(s.records[0].fields[0].confidence).toBe('high')
    expect(s.model).toBe('rule-based-v1')
  })
})

// —— 服务层降级（无后端 / 未加入家庭时不抛异常）——
describe('services 降级行为', () => {
  it('未加入家庭时 getCats 返回 success=false 与数组兜底', async () => {
    const { services } = await import('@/services')
    const res = await services.getCats()
    expect(res.success).toBe(false)
    expect(Array.isArray(res.data)).toBe(true)
  })

  it('未加入家庭时 getTodayStatus 返回中性状态', async () => {
    const { services } = await import('@/services')
    const res = await services.getTodayStatus('cat-1')
    expect(res.success).toBe(false)
    expect(res.data.food.state).toBe('none')
    expect(res.data.vomit.count).toBe(0)
  })

  it('未加入家庭时 getReminders 返回空数组', async () => {
    const { services } = await import('@/services')
    const res = await services.getReminders('todo')
    expect(res.success).toBe(false)
    expect(res.data).toEqual([])
  })
})

// —— Pinia stores ——
describe('Pinia stores', () => {
  it('app store actions', async () => {
    const { useAppStore } = await import('@/stores/app')
    const store = useAppStore()
    expect(store.currentCat).toBe('cat-whit')
    store.setCurrentCat('cat-oran')
    expect(store.currentCat).toBe('cat-oran')
    store.setAIState('test', null, true)
    expect(store.aiInput).toBe('test')
    store.clearAIState()
    expect(store.aiInput).toBe('')
  })

  it('cat store 由接口数据驱动，并自动回落到首只', async () => {
    const { useCatStore } = await import('@/stores/cat')
    const store = useCatStore()
    expect(store.cats).toEqual([])
    expect(store.currentCatId).toBe('')

    store.setCats([
      {
        id: 'cat-a', name: 'A', gender: 'female', breed: '', birthday: '', age: 1,
        neutered: false, diseases: [], allergies: [], currentMedication: null,
        nextVaccine: null, nextDeworm: null
      }
    ])
    expect(store.currentCatId).toBe('cat-a')
    expect(store.hasCats).toBe(true)

    store.setCat('cat-a')
    expect(store.activeCatId).toBe('cat-a')

    store.reset()
    expect(store.cats).toEqual([])
  })

  it('auth store 初始为未登录，logout 清空状态', async () => {
    const { useAuthStore } = await import('@/stores/auth')
    const store = useAuthStore()
    store.$reset()
    expect(store.isAuthenticated).toBe(false)
    expect(store.user).toBeNull()
  })

  it('reminder store filter', async () => {
    const { useReminderStore } = await import('@/stores/reminder')
    const store = useReminderStore()
    store.setFilter('todo')
    expect(store.filter).toBe('todo')
    expect(store.todoCount).toBeGreaterThan(0)
  })

  it('expense store total', async () => {
    const { useExpenseStore } = await import('@/stores/expense')
    const store = useExpenseStore()
    expect(store.totalAmount).toBeGreaterThan(0)
  })
})

// —— 类型 ——
describe('types', () => {
  it('Cat type shape', async () => {
    const { mockCats } = await import('@/mocks/family')
    const cat = mockCats[0]
    expect(cat.id).toBeDefined()
    expect(['male', 'female', 'unknown']).toContain(cat.gender)
    expect(cat.age).toBeGreaterThanOrEqual(0)
    expect(typeof cat.neutered).toBe('boolean')
  })

  it('Reminder type shape', async () => {
    const { mockReminders } = await import('@/mocks/data')
    const r = mockReminders[0]
    expect(['todo', 'done']).toContain(r.state)
    expect(typeof r.title).toBe('string')
  })
})

// —— 注册页「确认密码」——
describe('AuthView 确认密码', () => {
  /** 挂载登录/注册页。Pinia 由 vitest.setup.ts 全局注入，这里只需补 router。 */
  async function mountAuth() {
    const { default: AuthView } = await import('@/views/auth/AuthView.vue')
    const router = createRouter({
      history: createWebHashHistory(),
      routes: [{ path: '/', component: { template: '<div />' } }]
    })
    await router.push('/')
    await router.isReady()
    return mount(AuthView, { global: { plugins: [router] } })
  }

  /** 切到注册模式（页面上唯一的 .link 按钮就是模式切换）。 */
  async function toRegister(wrapper: Awaited<ReturnType<typeof mountAuth>>) {
    await wrapper.find('button.link').trigger('click')
  }

  const CONFIRM = 'input[placeholder="请再次输入密码"]'

  it('登录模式不显示确认密码，切到注册才显示', async () => {
    const w = await mountAuth()
    expect(w.find(CONFIRM).exists()).toBe(false)

    await toRegister(w)
    expect(w.find(CONFIRM).exists()).toBe(true)
  })

  it('两次密码不一致时给出提示并禁止提交', async () => {
    const w = await mountAuth()
    await toRegister(w)

    await w.find('input[placeholder="家里怎么称呼你"]').setValue('小明')
    await w.find('input[type="email"]').setValue('a@b.com')
    const pwd = w.findAll('input[type="password"]')
    expect(pwd.length).toBe(2)
    await pwd[0].setValue('password123')
    await pwd[1].setValue('password124')

    expect(w.find('#confirm-hint').text()).toContain('不一致')
    expect(w.find('button.primary').attributes('disabled')).toBeDefined()
  })

  it('两次一致且长度达标时解除禁用', async () => {
    const w = await mountAuth()
    await toRegister(w)

    await w.find('input[placeholder="家里怎么称呼你"]').setValue('小明')
    await w.find('input[type="email"]').setValue('a@b.com')
    const pwd = w.findAll('input[type="password"]')
    await pwd[0].setValue('password123')
    await pwd[1].setValue('password123')

    expect(w.find('#confirm-hint').exists()).toBe(false)
    expect(w.find('button.primary').attributes('disabled')).toBeUndefined()
  })

  it('密码不足 8 位时提示，且确认一致也无法提交', async () => {
    const w = await mountAuth()
    await toRegister(w)

    await w.find('input[placeholder="家里怎么称呼你"]').setValue('小明')
    await w.find('input[type="email"]').setValue('a@b.com')
    const pwd = w.findAll('input[type="password"]')
    await pwd[0].setValue('short')
    await pwd[1].setValue('short')

    expect(w.find('#password-hint').text()).toContain('至少 8 位')
    expect(w.find('button.primary').attributes('disabled')).toBeDefined()
  })

  it('切回登录模式会清空确认密码，避免残留', async () => {
    const w = await mountAuth()
    await toRegister(w)
    await w.find(CONFIRM).setValue('password123')
    await w.find('button.link').trigger('click') // 切回登录

    await toRegister(w)
    expect((w.find(CONFIRM).element as HTMLInputElement).value).toBe('')
  })

  it('显示/隐藏切换会同时作用于两个密码框', async () => {
    const w = await mountAuth()
    await toRegister(w)

    expect(w.findAll('input[type="password"]').length).toBe(2)
    await w.find('button.toggle').trigger('click')
    expect(w.findAll('input[type="password"]').length).toBe(0)
    // 昵称 + 家庭名称 + 两个已明文显示的密码框
    expect(w.findAll('input[type="text"]').length).toBe(4)
  })

  it('切换按钮是图标而非文字，且位于输入框容器内', async () => {
    const w = await mountAuth()
    await toRegister(w)

    const toggle = w.find('.input-wrap .toggle')
    expect(toggle.exists()).toBe(true)
    // 没有可见文字，语义全部由 aria-label 承担
    expect(toggle.text()).toBe('')
    expect(toggle.find('svg').exists()).toBe(true)
    expect(toggle.attributes('aria-label')).toBe('显示密码')

    await toggle.trigger('click')
    expect(toggle.attributes('aria-label')).toBe('隐藏密码')
    expect(toggle.attributes('aria-pressed')).toBe('true')
  })

  it('图标按钮与输入框同容器，两个密码框结构一致', async () => {
    const w = await mountAuth()
    await toRegister(w)

    // 修复点：图标由「与输入框并排的 flex 项」改为「叠加在输入框内的定位元素」。
    // 结构上体现为容器内恰好两个子节点，且输入框是第一个。
    const wrap = w.find('.input-wrap')
    expect(wrap.exists()).toBe(true)
    expect(wrap.element.children.length).toBe(2)
    expect(wrap.element.children[0].tagName).toBe('INPUT')
    expect(wrap.element.children[1].tagName).toBe('BUTTON')

    // 两个密码框都没有内联尺寸，宽度完全来自同一条 .field input{width:100%} 规则
    const pwd = w.find('.input-wrap input').element as HTMLInputElement
    const confirm = w.find(CONFIRM).element as HTMLInputElement
    expect(pwd.getAttribute('style')).toBeNull()
    expect(confirm.getAttribute('style')).toBeNull()
    expect(pwd.className).toBe(confirm.className)

    // 注册模式共 5 个输入框：邮箱、昵称、密码、确认密码、家庭名称
    expect(w.findAll('.field input').length).toBe(5)
  })
})
