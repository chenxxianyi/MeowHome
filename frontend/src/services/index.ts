// 业务服务 — 调用真实后端接口。
//
// 约定：本层不向外抛异常。视图是按 mock 时代写的（只读 res.data、没有 try/catch），
// 一旦抛出会让 loading 永远停在骨架屏。因此统一在 api() 里兜底成
// { success:false, data:<中性兜底> } 并记录日志；认证失效由 client 拦截器接管跳转。
import { toApiError } from '../api/adapter'
import {
  aiApi,
  careApi,
  catApi,
  expenseApi,
  familyApi,
  inventoryApi,
  momentApi,
  recordApi,
  reminderApi,
  toCat,
  toExpense,
  toFamily,
  toInventory,
  toMember,
  toMoment,
  toParseSession,
  toReminder,
  toSummary
} from '../api/endpoints'
import type { RecordInput } from '../api/endpoints'
import { mockRecordTypes } from '../mocks/data'
import { useAuthStore } from '../stores/auth'
import { useCatStore } from '../stores/cat'
import type {
  AIParseSession,
  AIAnalysisReport,
  APIResponse,
  Cat,
  Expense,
  Family,
  FocusItemData,
  InventoryItem,
  Reminder,
  TimelineEvent,
  TodayStatusData,
  TrendDataPoint
} from '../types'

/** 尚未加入家庭时抛出的错误码。 */
export const NO_FAMILY = 'NO_FAMILY'

function familyId(): string {
  const id = useAuthStore().familyId
  if (!id) {
    const err = new Error('尚未加入任何家庭') as Error & { code?: string }
    err.code = NO_FAMILY
    throw err
  }
  return id
}

/** 统一兜底：成功返回数据，失败返回 fallback 并记录日志。 */
async function api<T>(fallback: T, fn: () => Promise<T>): Promise<APIResponse<T>> {
  try {
    return { success: true, data: await fn() }
  } catch (e) {
    const err = toApiError(e)
    // 认证问题已由拦截器处理（清空登录态并跳转），这里只记录
    console.warn(`[services] ${err.code}: ${err.message}`)
    return { success: false, data: fallback }
  }
}

/** 中性的今日状态，作为接口失败时的兜底展示。 */
function emptyToday(): TodayStatusData {
  return {
    food: { amount: 0, expected: 0, unit: 'g', state: 'none', label: '暂无喂食记录' },
    water: { amount: 0, expected: 0, unit: 'ml', state: 'none', label: '暂无饮水记录' },
    elimination: { state: 'none', label: '暂无排便记录' },
    vomit: { count: 0, state: 'none', label: '无呕吐' },
    medication: { state: 'none', label: '无需用药' },
    mental: { state: 'none', label: '暂无精神记录' }
  }
}

function emptySummary(): AIAnalysisReport {
  return { id: '', generatedAt: new Date().toISOString(), recordCount: 0, body: '', evidence: [] }
}

function emptyParseSession(input = ''): AIParseSession {
  return { id: '', originalInput: input, parsedAt: new Date().toISOString(), model: '', records: [] }
}

// —— 记录载荷映射 ——

const META_KEYS = new Set([
  'type', 'catId', 'catIds', 'cat_ids', 'createdAt', 'source', 'time',
  'occurredAt', 'occurred_at', 'note', 'title', 'severity', 'records', 'id'
])

/** 按记录类型推断严重度（用于焦点项聚合）。 */
function inferSeverity(type: string, payload: Record<string, unknown>): string {
  if (type === 'vomit' || type === 'symptom') {
    return payload.severity === 'normal' ? 'warning' : 'danger'
  }
  if (type === 'mental') {
    const level = payload.level
    return level && level !== 'normal' ? 'warning' : 'normal'
  }
  return 'normal'
}

function normalizeCatIds(raw: Record<string, unknown>): string[] {
  const rawIds = raw.catIds ?? raw.cat_ids
  if (Array.isArray(rawIds)) return rawIds.filter((x): x is string => typeof x === 'string')
  if (typeof rawIds === 'string' && rawIds) return [rawIds]
  if (typeof raw.catId === 'string' && raw.catId) return [raw.catId]
  return []
}

function toISO(value: unknown): string | undefined {
  if (typeof value !== 'string' || !value) return undefined
  const d = new Date(value)
  return Number.isNaN(d.getTime()) ? undefined : d.toISOString()
}

/** 把视图传入的松散对象转成后端 RecordInput。 */
function buildRecordInput(raw: Record<string, unknown>): RecordInput {
  const type = typeof raw.type === 'string' && raw.type ? raw.type : 'custom'
  const payload: Record<string, unknown> = {}
  for (const [k, v] of Object.entries(raw)) {
    if (META_KEYS.has(k)) continue
    if (v === '' || v === null || v === undefined) continue
    payload[k] = v
  }

  const input: RecordInput = {
    type,
    cat_ids: normalizeCatIds(raw),
    source: typeof raw.source === 'string' ? raw.source : 'manual'
  }
  const occurred = toISO(raw.occurredAt ?? raw.time ?? raw.createdAt)
  if (occurred) input.occurred_at = occurred
  if (typeof raw.title === 'string' && raw.title) input.title = raw.title
  if (typeof raw.note === 'string' && raw.note) input.note = raw.note
  if (Object.keys(payload).length) input.payload = payload
  input.severity = typeof raw.severity === 'string' && raw.severity ? raw.severity : inferSeverity(type, payload)
  return input
}

/** 把 AI 解析结果转成后端 RecordInput。 */
function parsedToInput(rec: AIParseSession['records'][number], fallbackCatId: string): RecordInput {
  const payload: Record<string, unknown> = {}
  for (const f of rec.fields ?? []) {
    if (f.value === null || f.value === '' || f.value === undefined) continue
    payload[f.key] = f.value
  }
  const catIds = rec.catId ? [rec.catId] : fallbackCatId ? [fallbackCatId] : []
  return {
    type: rec.type,
    cat_ids: catIds,
    source: 'ai',
    payload,
    severity: inferSeverity(rec.type, payload),
    title: `AI 记录：${recordTypesLabel(rec.type)}`
  }
}

function recordTypesLabel(type: string): string {
  const all = [...mockRecordTypes.high, ...mockRecordTypes.health, ...mockRecordTypes.life]
  return all.find((t) => t.id === type)?.label ?? type
}

// —— 服务 ——

export const services = {
  async getFamily(): Promise<APIResponse<Family>> {
    return api<Family>({ id: '', name: '', createdAt: '', members: [] }, async () => {
      const fid = familyId()
      const [family, members] = await Promise.all([familyApi.get(fid), familyApi.members(fid)])
      return toFamily(
        { ...family, member_id: '', role: '' },
        members.map(toMember)
      )
    })
  },

  async getCats(): Promise<APIResponse<Cat[]>> {
    return api<Cat[]>([], async () => {
      const fid = familyId()
      const list = await catApi.list(fid)
      const cats = list.map(toCat)
      // 同步到全局 store，供猫咪切换器等组件使用
      useCatStore().setCats(cats)
      return cats
    })
  },

  async getTodayStatus(catId: string): Promise<APIResponse<TodayStatusData>> {
    return api<TodayStatusData>(emptyToday(), () => careApi.today(familyId(), catId))
  },

  async getFocusItems(): Promise<APIResponse<FocusItemData[]>> {
    return api<FocusItemData[]>([], () => careApi.focus(familyId()))
  },

  async getAISummary(): Promise<APIResponse<AIAnalysisReport>> {
    return api<AIAnalysisReport>(emptySummary(), async () => toSummary(await aiApi.summary(familyId())))
  },

  async getReminders(filter?: string): Promise<APIResponse<Reminder[]>> {
    return api<Reminder[]>([], async () => {
      const state = filter === 'todo' || filter === 'done' ? filter : undefined
      const list = await reminderApi.list(familyId(), state)
      return list.map(toReminder)
    })
  },

  async getTrends(catId: string, range: number): Promise<APIResponse<TrendDataPoint[]>> {
    return api<TrendDataPoint[]>([], () => careApi.trends(familyId(), catId, range))
  },

  async getMoments(): Promise<APIResponse<TimelineEvent[]>> {
    return api<TimelineEvent[]>([], async () => {
      const list = await momentApi.list(familyId())
      return list.map(toMoment)
    })
  },

  async getInventory(): Promise<APIResponse<InventoryItem[]>> {
    return api<InventoryItem[]>([], async () => {
      const list = await inventoryApi.list(familyId())
      return list.map(toInventory)
    })
  },

  async getExpenses(): Promise<APIResponse<Expense[]>> {
    return api<Expense[]>([], async () => {
      const list = await expenseApi.list(familyId())
      return list.map(toExpense)
    })
  },

  async parseAI(text: string): Promise<APIResponse<AIParseSession>> {
    return api<AIParseSession>(emptyParseSession(text), async () =>
      toParseSession(await aiApi.parse(familyId(), text))
    )
  },

  /** 保存记录。支持快捷记录与「AI 确认入库」两种载荷。 */
  async saveRecord(record: Record<string, unknown>): Promise<APIResponse<unknown>> {
    return api<unknown>(null, async () => {
      const fid = familyId()

      if (record.type === 'ai-confirmed') {
        const parsed = (record.records ?? []) as AIParseSession['records']
        const fallbackCatId = useCatStore().currentCatId
        const inputs = parsed.map((r) => parsedToInput(r, fallbackCatId)).filter((r) => r.cat_ids.length > 0)
        if (inputs.length === 0) throw new Error('没有可入库的记录')
        return recordApi.createBatch(fid, inputs)
      }

      const input = buildRecordInput(record)
      if (input.cat_ids.length === 0) {
        input.cat_ids = [useCatStore().currentCatId]
      }
      return recordApi.create(fid, input)
    })
  },

  async completeReminder(id: string): Promise<APIResponse<{ id: string; state: string }>> {
    return api<{ id: string; state: string }>({ id, state: 'done' }, async () => {
      const res = await reminderApi.complete(familyId(), id)
      return { id, state: res.data?.state ?? 'done' }
    })
  }
}

export const recordTypes = mockRecordTypes
