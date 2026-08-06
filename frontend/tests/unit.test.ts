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

// —— Mock 数据 ——
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

// —— 服务 ——
describe('services', () => {
  it('getCats returns data', async () => {
    const { services } = await import('@/services')
    const res = await services.getCats()
    expect(res.success).toBe(true)
    expect(Array.isArray(res.data)).toBe(true)
  })

  it('getFamily returns data', async () => {
    const { services } = await import('@/services')
    const res = await services.getFamily()
    expect(res.success).toBe(true)
    expect(res.data.id).toBeDefined()
  })

  it('getReminders with todo filter', async () => {
    const { services } = await import('@/services')
    const res = await services.getReminders('todo')
    expect(res.success).toBe(true)
    expect(res.data.every(r => r.state === 'todo')).toBe(true)
  })

  it('getAISummary returns evidence', async () => {
    const { services } = await import('@/services')
    const res = await services.getAISummary()
    expect(res.success).toBe(true)
    expect(res.data.evidence.length).toBeGreaterThan(0)
  })

  it('parseAI with empty string returns false success', async () => {
    const { services } = await import('@/services')
    const res = await services.parseAI('')
    expect(res.success).toBe(false)
  })

  it('saveRecord returns new id', async () => {
    const { services } = await import('@/services')
    const res = await services.saveRecord({ type: 'feeding', food: 'test' })
    expect(res.success).toBe(true)
    expect(res.data.id).toContain('rec-')
  })

  it('completeReminder returns done state', async () => {
    const { services } = await import('@/services')
    const res = await services.completeReminder('r1')
    expect(res.success).toBe(true)
    expect(res.data.state).toBe('done')
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

  it('cat store', async () => {
    const { useCatStore } = await import('@/stores/cat')
    const store = useCatStore()
    expect(store.cats.length).toBeGreaterThanOrEqual(2)
    store.setCat('cat-oran')
    expect(store.currentCatId).toBe('cat-oran')
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
    expect(['male', 'female']).toContain(cat.gender)
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
