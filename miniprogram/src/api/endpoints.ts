// 真实后端接口调用层。
//
// 每个函数返回前端既有的领域类型（types/index.ts），
// 后端 DTO 与前端类型的差异（snake_case、派生字段）在这里收敛。
import { aiHttp, http, request, requestEnvelope, apiBase } from './client'
import type {
  AIParseSession,
  AIAnalysisReport,
  Cat,
  Expense,
  Family,
  FamilyMember,
  FocusItemData,
  InventoryItem,
  Reminder,
  TimelineEvent,
  TodayStatusData,
  TrendDataPoint,
  User
} from '../types'

// ---------------------------------------------------------------- DTO

export interface AuthResultDTO {
  access_token: string
  refresh_token: string
  expires_in: number
  user: { id: string; email: string; name: string }
}

export interface MeDTO {
  id: string
  email: string
  user_name: string
  family_id?: string
  member_id?: string
  role?: string
  timezone?: string
}

export interface MyFamilyDTO {
  id: string
  name: string
  timezone: string
  currency: string
  member_id: string
  role: string
}

export interface CatDTO {
  id: string
  family_id: string
  name: string
  breed?: string
  gender?: string
  birthday?: string
  birth_date?: string
  neutered?: boolean
  avatar_key?: string
  diseases?: string[]
  allergies?: string[]
}

export interface MemberDTO {
  id: string
  family_id: string
  user_id: string
  user_name?: string
  role: string
}

export interface RecordDTO {
  id: string
  family_id: string
  cat_ids: string[]
  type: string
  occurred_at: string
  source: string
  severity: string
  title: string
  note: string
  payload?: Record<string, unknown>
}

export interface ReminderDTO {
  id: string
  catId: string
  type: string
  title: string
  subtitle: string
  time: string
  state: string
  icon: string
}

export interface MomentDTO {
  id: string
  date: string
  type: string
  title: string
  body?: string
  catId?: string
  images?: number
}

export interface InventoryDTO {
  id: string
  name: string
  category: string
  quantity: number
  unit: string
  estimatedDays?: number
  status: string
  expiry: string
}

export interface ExpenseDTO {
  id: string
  date: string
  amount: number
  category: string
  label: string
  catId?: string
}

export interface ParseSessionDTO {
  id: string
  originalInput: string
  parsedAt: string
  model: string
  records: {
    id: string
    type: string
    catId?: string
    fields: { key: string; value: unknown; confidence: string; note?: string }[]
  }[]
}

export interface SummaryDTO {
  id: string
  generatedAt: string
  recordCount: number
  body: string
  evidence: { type: string; catId?: string; time: string; content: string }[]
}

// ---------------------------------------------------------------- 映射

/** 由生日推算年龄（周岁）。 */
function ageFrom(birthday?: string): number {
  if (!birthday) return 0
  const b = new Date(birthday)
  if (Number.isNaN(b.getTime())) return 0
  const now = new Date()
  let age = now.getFullYear() - b.getFullYear()
  const m = now.getMonth() - b.getMonth()
  if (m < 0 || (m === 0 && now.getDate() < b.getDate())) age--
  return age < 0 ? 0 : age
}

export function toUser(dto: MeDTO): User {
  const role = dto.role === 'owner' ? 'owner' : 'member'
  return { id: dto.id, name: dto.user_name, role }
}

export function toFamily(dto: MyFamilyDTO, members: FamilyMember[] = []): Family {
  return { id: dto.id, name: dto.name, createdAt: '', members }
}

export function toMember(dto: MemberDTO): FamilyMember {
  return {
    id: dto.user_id || dto.id,
    name: dto.user_name || dto.id,
    role: dto.role === 'owner' ? 'owner' : 'member'
  }
}

export function toCat(dto: CatDTO): Cat {
  const gender = dto.gender === 'male' || dto.gender === 'female' ? dto.gender : 'unknown'
  const birthday = dto.birthday ?? dto.birth_date ?? ''
  return {
    id: dto.id,
    name: dto.name,
    gender,
    breed: dto.breed ?? '',
    birthday,
    age: ageFrom(birthday),
    neutered: Boolean(dto.neutered),
    avatar: dto.avatar_key ?? '',
    diseases: dto.diseases ?? [],
    allergies: dto.allergies ?? [],
    currentMedication: null,
    nextVaccine: null,
    nextDeworm: null
  }
}

export function toReminder(dto: ReminderDTO): Reminder {
  return {
    id: dto.id,
    catId: (dto.catId || 'both') as Reminder['catId'],
    type: dto.type as Reminder['type'],
    title: dto.title,
    subtitle: dto.subtitle,
    time: dto.time,
    state: dto.state === 'done' ? 'done' : 'todo',
    icon: dto.icon
  }
}

export function toMoment(dto: MomentDTO): TimelineEvent {
  return {
    id: dto.id,
    date: dto.date,
    type: (dto.type || 'photo') as TimelineEvent['type'],
    title: dto.title,
    body: dto.body,
    catId: (dto.catId || 'both') as TimelineEvent['catId'],
    images: dto.images
  }
}

export function toInventory(dto: InventoryDTO): InventoryItem {
  return {
    id: dto.id,
    name: dto.name,
    category: dto.category,
    quantity: dto.quantity,
    unit: dto.unit,
    estimatedDays: dto.estimatedDays,
    status: (dto.status || 'ok') as InventoryItem['status'],
    expiry: dto.expiry || null
  }
}

export function toExpense(dto: ExpenseDTO): Expense {
  return {
    id: dto.id,
    date: dto.date,
    amount: dto.amount,
    category: dto.category,
    label: dto.label,
    catId: (dto.catId || 'both') as Expense['catId']
  }
}

export function toParseSession(dto: ParseSessionDTO): AIParseSession {
  return {
    id: dto.id,
    originalInput: dto.originalInput,
    parsedAt: dto.parsedAt,
    model: dto.model,
    records: (dto.records ?? []).map((r) => ({
      id: r.id,
      type: r.type,
      catId: r.catId,
      fields: (r.fields ?? []).map((f) => ({
        key: f.key,
        value: (f.value ?? null) as string | number | boolean | null,
        confidence: (f.confidence || 'none') as 'high' | 'medium' | 'low' | 'none',
        note: f.note
      }))
    }))
  }
}

export function toSummary(dto: SummaryDTO): AIAnalysisReport {
  return {
    id: dto.id,
    generatedAt: dto.generatedAt,
    recordCount: dto.recordCount,
    body: dto.body,
    evidence: dto.evidence ?? []
  }
}

// ---------------------------------------------------------------- 认证

export const authApi = {
  register(email: string, password: string, userName: string, familyName?: string) {
    return request<AuthResultDTO>(http, {
      method: 'POST',
      url: '/auth/register',
      data: { email, password, user_name: userName, family_name: familyName }
    })
  },
  login(email: string, password: string) {
    return request<AuthResultDTO>(http, {
      method: 'POST',
      url: '/auth/login',
      data: { email, password }
    })
  },
  logout(refreshToken: string) {
    return request<void>(http, { method: 'POST', url: '/auth/logout', data: { refresh_token: refreshToken } })
  },
  me() {
    return request<MeDTO>(http, { method: 'GET', url: '/me' })
  },
  myFamilies() {
    return request<MyFamilyDTO[]>(http, { method: 'GET', url: '/families' })
  },
  createFamily(name: string, timezone?: string, currency?: string) {
    return request<{ id: string; name: string; timezone: string; currency: string }>(http, {
      method: 'POST',
      url: '/families',
      data: { name, timezone, currency }
    })
  }
}

// ---------------------------------------------------------------- 家庭与猫咪

export const familyApi = {
  get(familyId: string) {
    return request<{ id: string; name: string; timezone: string; currency: string }>(http, {
      method: 'GET',
      url: `/families/${familyId}`
    })
  },
  update(familyId: string, patch: { name?: string; timezone?: string; currency?: string }) {
    return request<{ id: string; name: string; timezone: string; currency: string }>(http, {
      method: 'PATCH',
      url: `/families/${familyId}`,
      data: patch
    })
  },
  members(familyId: string) {
    return request<MemberDTO[]>(http, { method: 'GET', url: `/families/${familyId}/members` })
  }
}

export const catApi = {
  list(familyId: string) {
    return request<CatDTO[]>(http, { method: 'GET', url: `/families/${familyId}/cats` })
  },
  get(familyId: string, catId: string) {
    return request<CatDTO>(http, { method: 'GET', url: `/families/${familyId}/cats/${catId}` })
  },
  create(
    familyId: string,
    data: {
      name: string
      breed?: string
      gender?: string
      birth_date?: string
      neutered?: boolean
      diseases?: string[]
      allergies?: string[]
    }
  ) {
    return request<CatDTO>(http, { method: 'POST', url: `/families/${familyId}/cats`, data })
  },
  update(
    familyId: string,
    catId: string,
    data: { name?: string; breed?: string; gender?: string; birth_date?: string; neutered?: boolean }
  ) {
    return request<CatDTO>(http, { method: 'PATCH', url: `/families/${familyId}/cats/${catId}`, data })
  },
  remove(familyId: string, catId: string) {
    return request<void>(http, { method: 'DELETE', url: `/families/${familyId}/cats/${catId}` })
  }
}

// ---------------------------------------------------------------- 记录与聚合

export interface RecordInput {
  type: string
  cat_ids: string[]
  occurred_at?: string
  severity?: string
  title?: string
  note?: string
  source?: string
  payload?: Record<string, unknown>
}

export const recordApi = {
  list(familyId: string, params: { cat_id?: string; type?: string; from?: string; to?: string; limit?: number } = {}) {
    return request<RecordDTO[]>(http, { method: 'GET', url: `/families/${familyId}/records`, params })
  },
  create(familyId: string, data: RecordInput) {
    return request<RecordDTO>(http, { method: 'POST', url: `/families/${familyId}/records`, data })
  },
  createBatch(familyId: string, records: RecordInput[]) {
    return request<RecordDTO[]>(http, { method: 'POST', url: `/families/${familyId}/records/batch`, data: { records } })
  }
}

export const careApi = {
  today(familyId: string, catId: string) {
    return request<TodayStatusData>(http, {
      method: 'GET',
      url: `/families/${familyId}/today`,
      params: { cat_id: catId }
    })
  },
  focus(familyId: string) {
    return request<FocusItemData[]>(http, { method: 'GET', url: `/families/${familyId}/focus` })
  },
  trends(familyId: string, catId: string, days: number) {
    return request<TrendDataPoint[]>(http, {
      method: 'GET',
      url: `/families/${familyId}/trends`,
      params: { cat_id: catId, days }
    })
  }
}

export const reminderApi = {
  list(familyId: string, state?: string) {
    return request<ReminderDTO[]>(http, { method: 'GET', url: `/families/${familyId}/reminders`, params: { state } })
  },
  create(familyId: string, data: Partial<ReminderDTO>) {
    return request<ReminderDTO>(http, { method: 'POST', url: `/families/${familyId}/reminders`, data })
  },
  complete(familyId: string, reminderId: string) {
    return requestEnvelope<ReminderDTO>(http, {
      method: 'PATCH',
      url: `/families/${familyId}/reminders/${reminderId}/complete`
    })
  }
}

export const momentApi = {
  list(familyId: string) {
    return request<MomentDTO[]>(http, { method: 'GET', url: `/families/${familyId}/moments` })
  },
  create(
    familyId: string,
    data: { type?: string; title: string; body?: string; date?: string; cat_ids?: string[]; images?: number }
  ) {
    return request<MomentDTO>(http, { method: 'POST', url: `/families/${familyId}/moments`, data })
  }
}

export const inventoryApi = {
  list(familyId: string) {
    return request<InventoryDTO[]>(http, { method: 'GET', url: `/families/${familyId}/inventory` })
  },
  create(
    familyId: string,
    data: {
      name: string
      category?: string
      quantity?: number
      unit?: string
      low_stock_threshold?: number
      expiry?: string
    }
  ) {
    return request<InventoryDTO>(http, { method: 'POST', url: `/families/${familyId}/inventory`, data })
  }
}

export const expenseApi = {
  list(familyId: string, month?: string) {
    return request<ExpenseDTO[]>(http, { method: 'GET', url: `/families/${familyId}/expenses`, params: { month } })
  },
  create(
    familyId: string,
    data: { date?: string; amount: number; category?: string; label?: string; cat_ids?: string[] }
  ) {
    return request<ExpenseDTO>(http, { method: 'POST', url: `/families/${familyId}/expenses`, data })
  }
}

export const aiApi = {
  parse(familyId: string, input: string) {
    return request<ParseSessionDTO>(aiHttp, { method: 'POST', url: `/families/${familyId}/ai/parse`, data: { input } })
  },
  summary(familyId: string) {
    return request<SummaryDTO>(aiHttp, { method: 'GET', url: `/families/${familyId}/ai/summary` })
  }
}

export { apiBase }
