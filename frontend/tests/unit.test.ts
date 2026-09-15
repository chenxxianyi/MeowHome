// Unit tests for MeowHome Vue 3 frontend
import { describe, it, expect } from 'vitest'

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
