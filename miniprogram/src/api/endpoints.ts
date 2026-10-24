// 真实后端接口调用层。
//
// 每个函数返回前端既有的领域类型（types/index.ts），
// 后端 DTO 与前端类型的差异（snake_case、派生字段）在这里收敛。
import { aiHttp, http, request, requestEnvelope, uploadFile, apiBase } from './client'
import { API_ORIGIN } from './config'
import type {
  AIParseSession,
  AIAnalysisReport,
  Cat,
  Expense,
  Family,
  FamilyMember,
  FocusItemData,
  InventoryItem,
  InventoryTransaction,
  PurchaseConfirmSummary,
  PurchaseImport,
  PurchaseLine,
  PurchaseRecognizeResult,
  Reminder,
  ShoppingItem,
  TimelineEvent,
  TodayStatusData,
  TrendDataPoint,
  User
} from '../types'
import type {
  AgentChatRequest,
  AgentChatResponse,
  AgentConfirmResponse,
  AgentMessage,
  AgentMessageListParams,
  AgentMessageListResponse,
  AgentReminderEditRequest,
  AgentReminderDraft,
  AgentSessionListResponse,
  AgentSessionMessagesResponse
} from '../types/agent'

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
  created_by?: string
  created_at?: string
  updated_at?: string
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
  rule?: string
  scheduled_at?: string
  timezone?: string
  completed_at?: string
  source_record_id?: string
  completed_record_id?: string
  metadata?: Record<string, unknown>
}

export interface VaccineSuggestionDTO {
  status: 'suggested' | 'needs_input' | 'unsupported'
  cat_id: string
  vaccine_code?: string
  vaccine_name?: string
  dose_number?: string
  age_weeks?: number
  prior_doses_recorded: number
  date_window_start?: string
  date_window_end?: string
  default_reminder_date?: string
  title?: string
  reason: string
  rule_id?: string
  rule_version?: string
  source_url?: string
  timezone: string
  date_source?: string
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
  brand?: string
  specification?: string
  category: string
  quantity: number
  unit: string
  estimatedDays?: number
  status: string
  expiry: string
}

export interface ShoppingItemDTO {
  id: string
  name: string
  brand?: string
  specification?: string
  category?: string
  quantity: number
  unit?: string
  note?: string
  priority: string
  status: string
  matched_inventory_item_id?: string
  completed_at?: string
  created_by?: string
  created_at?: string
}

export interface InventoryTxDTO {
  id: string
  item_id: string
  change: number
  reason: string
  source_type?: string
  source_id?: string
  created_by?: string
  created_at?: string
}

export interface PurchaseLineDTO {
  id: string
  raw_title?: string
  name: string
  specification?: string
  quantity: number
  unit?: string
  line_amount?: number | null
  confidence: string
  evidence?: string
  include_in_stock: boolean
  matched_inventory_item_id?: string
  matched_shopping_item_id?: string
}

export interface PurchaseImportDTO {
  id: string
  status: string
  platform?: string
  order_date?: string
  order_no?: string
  total_amount: number
  currency: string
  media_keys?: string[]
  recognition_version: number
  error_summary?: string
  confirmed_at?: string
  expense_id?: string
  lines?: PurchaseLineDTO[]
  created_at?: string
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
  return { id: dto.id, name: dto.name, createdAt: '', timezone: dto.timezone, members }
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
    avatar: toAvatarUrl(dto.avatar_key),
    diseases: dto.diseases ?? [],
    allergies: dto.allergies ?? [],
    currentMedication: null,
    nextVaccine: null,
    nextDeworm: null
  }
}

function toAvatarUrl(avatarKey?: string) {
  if (!avatarKey) return ''
  if (/^(?:https?:|data:|file:|wxfile:)/i.test(avatarKey)) return avatarKey
  return `${API_ORIGIN}/static/${avatarKey.replace(/^\/+/, '')}`
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
    icon: dto.icon,
    rule: dto.rule,
    scheduled_at: dto.scheduled_at,
    timezone: dto.timezone,
    completed_at: dto.completed_at,
    source_record_id: dto.source_record_id,
    completed_record_id: dto.completed_record_id,
    metadata: dto.metadata
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
    brand: dto.brand,
    specification: dto.specification,
    category: dto.category,
    quantity: dto.quantity,
    unit: dto.unit,
    estimatedDays: dto.estimatedDays,
    status: (dto.status || 'ok') as InventoryItem['status'],
    expiry: dto.expiry || null
  }
}

export function toShoppingItem(dto: ShoppingItemDTO): ShoppingItem {
  return {
    id: dto.id,
    name: dto.name,
    brand: dto.brand,
    specification: dto.specification,
    category: dto.category,
    quantity: dto.quantity,
    unit: dto.unit,
    note: dto.note,
    priority: (dto.priority || 'normal') as ShoppingItem['priority'],
    status: (dto.status || 'todo') as ShoppingItem['status'],
    matchedInventoryItemId: dto.matched_inventory_item_id,
    completedAt: dto.completed_at,
    createdBy: dto.created_by,
    createdAt: dto.created_at
  }
}

export function toInventoryTransaction(dto: InventoryTxDTO): InventoryTransaction {
  return {
    id: dto.id,
    itemId: dto.item_id,
    change: dto.change,
    reason: dto.reason,
    sourceType: dto.source_type,
    sourceId: dto.source_id,
    createdAt: dto.created_at
  }
}

export function toPurchaseLine(dto: PurchaseLineDTO): PurchaseLine {
  return {
    id: dto.id,
    rawTitle: dto.raw_title,
    name: dto.name,
    specification: dto.specification,
    quantity: dto.quantity,
    unit: dto.unit,
    lineAmount: dto.line_amount ?? null,
    confidence: (dto.confidence || 'none') as PurchaseLine['confidence'],
    evidence: dto.evidence,
    includeInStock: Boolean(dto.include_in_stock),
    matchedInventoryItemId: dto.matched_inventory_item_id,
    matchedShoppingItemId: dto.matched_shopping_item_id
  }
}

export function toPurchaseImport(dto: PurchaseImportDTO): PurchaseImport {
  return {
    id: dto.id,
    status: (dto.status || 'uploaded') as PurchaseImport['status'],
    platform: dto.platform,
    orderDate: dto.order_date,
    orderNo: dto.order_no,
    totalAmount: dto.total_amount,
    currency: dto.currency,
    mediaKeys: dto.media_keys ?? [],
    recognitionVersion: dto.recognition_version,
    errorSummary: dto.error_summary,
    confirmedAt: dto.confirmed_at,
    expenseId: dto.expense_id,
    lines: (dto.lines ?? []).map(toPurchaseLine),
    createdAt: dto.created_at
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
  uploadAvatar(familyId: string, catId: string, filePath: string) {
    return uploadFile<CatDTO>(http, {
      url: `/families/${encodeURIComponent(familyId)}/cats/${encodeURIComponent(catId)}/avatar`,
      filePath,
      name: 'file'
    })
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

export interface RecordUpdateRequest extends RecordInput {
  expected_updated_at: string
}

export const recordApi = {
  get(familyId: string, recordId: string) {
    return request<RecordDTO>(http, { method: 'GET', url: `/families/${familyId}/records/${recordId}` })
  },
  list(familyId: string, params: { cat_id?: string; type?: string; from?: string; to?: string; limit?: number; offset?: number } = {}) {
    return request<RecordDTO[]>(http, { method: 'GET', url: `/families/${familyId}/records`, params })
  },
  create(familyId: string, data: RecordInput) {
    return request<RecordDTO>(http, { method: 'POST', url: `/families/${familyId}/records`, data })
  },
  update(familyId: string, recordId: string, data: RecordUpdateRequest) {
    return request<RecordDTO>(http, { method: 'PATCH', url: `/families/${familyId}/records/${recordId}`, data })
  },
  delete(familyId: string, recordId: string) {
    return request<void>(http, { method: 'DELETE', url: `/families/${familyId}/records/${recordId}` })
  },
  createBatch(familyId: string, records: RecordInput[]) {
    return request<RecordDTO[]>(http, { method: 'POST', url: `/families/${familyId}/records/batch`, data: { records } })
  },
  vaccineSuggestion(familyId: string, recordId: string) {
    return request<VaccineSuggestionDTO>(http, {
      method: 'GET',
      url: `/families/${familyId}/records/${recordId}/vaccine-suggestion`
    })
  },
  createVaccineReminder(
    familyId: string,
    recordId: string,
    data: { scheduled_date: string; date_source: string; timezone: string }
  ) {
    return request<ReminderDTO>(http, {
      method: 'POST',
      url: `/families/${familyId}/records/${recordId}/vaccine-reminders`,
      data
    })
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
  },
  completeWithRecord(familyId: string, reminderId: string, recordId: string) {
    return request<ReminderDTO>(http, {
      method: 'PATCH',
      url: `/families/${familyId}/reminders/${reminderId}/complete-with-record`,
      data: { record_id: recordId }
    })
  },
  updateVaccineSchedule(familyId: string, reminderId: string, data: { scheduled_date: string; timezone: string }) {
    return request<ReminderDTO>(http, {
      method: 'PATCH',
      url: `/families/${familyId}/reminders/${reminderId}/schedule`,
      data
    })
  },
  cancelVaccine(familyId: string, reminderId: string) {
    return request<void>(http, { method: 'DELETE', url: `/families/${familyId}/reminders/${reminderId}` })
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
      brand?: string
      specification?: string
      category?: string
      quantity?: number
      unit?: string
      low_stock_threshold?: number
      expiry?: string
    }
  ) {
    return request<InventoryDTO>(http, { method: 'POST', url: `/families/${familyId}/inventory`, data })
  },
  adjust(familyId: string, itemId: string, data: { delta: number; reason?: string }) {
    return request<{ item: InventoryDTO; transaction: InventoryTxDTO }>(http, {
      method: 'POST',
      url: `/families/${familyId}/inventory/${itemId}/adjustments`,
      data
    })
  },
  transactions(familyId: string, params: { item_id?: string; limit?: number } = {}) {
    return request<InventoryTxDTO[]>(http, { method: 'GET', url: `/families/${familyId}/inventory/transactions`, params })
  }
}

export const shoppingApi = {
  list(familyId: string, status?: string) {
    return request<ShoppingItemDTO[]>(http, {
      method: 'GET',
      url: `/families/${familyId}/shopping-list`,
      params: { status }
    })
  },
  create(
    familyId: string,
    data: {
      name: string
      brand?: string
      specification?: string
      category?: string
      quantity?: number
      unit?: string
      note?: string
      priority?: string
    }
  ) {
    return request<ShoppingItemDTO>(http, { method: 'POST', url: `/families/${familyId}/shopping-list`, data })
  },
  update(familyId: string, itemId: string, patch: Record<string, unknown>) {
    return request<ShoppingItemDTO>(http, {
      method: 'PATCH',
      url: `/families/${familyId}/shopping-list/${itemId}`,
      data: patch
    })
  },
  remove(familyId: string, itemId: string) {
    return request<void>(http, { method: 'DELETE', url: `/families/${familyId}/shopping-list/${itemId}` })
  }
}

export const purchaseApi = {
  /** 微信小程序逐张上传：返回私有媒体键。 */
  uploadMedia(familyId: string, filePath: string) {
    return uploadFile<{ media_key: string }>(http, {
      url: `/families/${encodeURIComponent(familyId)}/purchase-media`,
      filePath,
      name: 'file'
    })
  },
  create(familyId: string, mediaKeys: string[]) {
    return request<PurchaseImportDTO>(http, {
      method: 'POST',
      url: `/families/${familyId}/purchase-imports`,
      data: { media_keys: mediaKeys }
    })
  },
  list(familyId: string, params: { limit?: number } = {}) {
    return request<PurchaseImportDTO[]>(http, { method: 'GET', url: `/families/${familyId}/purchase-imports`, params })
  },
  get(familyId: string, importId: string) {
    return request<PurchaseImportDTO>(http, { method: 'GET', url: `/families/${familyId}/purchase-imports/${importId}` })
  },
  recognize(familyId: string, importId: string) {
    return request<{ import: PurchaseImportDTO; success: boolean; issues?: string[] }>(http, {
      method: 'POST',
      url: `/families/${familyId}/purchase-imports/${importId}/recognize`
    })
  },
  saveDraft(
    familyId: string,
    importId: string,
    data: {
      expected_version?: number
      platform?: string
      order_date?: string
      total_amount?: number
      currency?: string
      lines?: {
        id?: string
        name: string
        specification?: string
        quantity?: number
        unit?: string
        line_amount?: number
        confidence?: string
        evidence?: string
        include_in_stock?: boolean
        matched_inventory_item_id?: string
        matched_shopping_item_id?: string
      }[]
    }
  ) {
    return request<PurchaseImportDTO>(http, {
      method: 'PATCH',
      url: `/families/${familyId}/purchase-imports/${importId}`,
      data
    })
  },
  confirm(familyId: string, importId: string, data: { expected_version?: number; include_expense?: boolean }) {
    return request<{
      import: PurchaseImportDTO
      created: boolean
      expense_id?: string
      expense_amount?: number
      transactions?: InventoryTxDTO[]
      shopping_completed?: string[]
    }>(http, {
      method: 'POST',
      url: `/families/${familyId}/purchase-imports/${importId}/confirm`,
      data
    })
  },
  cancel(familyId: string, importId: string) {
    return request<PurchaseImportDTO>(http, {
      method: 'POST',
      url: `/families/${familyId}/purchase-imports/${importId}/cancel`
    })
  },
  removeMedia(familyId: string, importId: string, index: number) {
    return request<PurchaseImportDTO>(http, {
      method: 'DELETE',
      url: `/families/${familyId}/purchase-imports/${importId}/media/${index}`
    })
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

// ---------------------------------------------------------------- Agent
export const agentApi = {
  listMessages(familyId: string, params: AgentMessageListParams = {}) {
    return request<AgentMessageListResponse>(http, { method: 'GET', url: `/families/${familyId}/agent/messages`, params: params as Record<string, unknown> })
  },
  getMessage(familyId: string, messageId: string) {
    return request<AgentMessage>(http, { method: 'GET', url: `/families/${familyId}/agent/messages/${messageId}` })
  },
  editDraft(familyId: string, messageId: string, data: AgentReminderEditRequest) {
    return request<AgentReminderDraft>(http, { method: 'PATCH', url: `/families/${familyId}/agent/messages/${messageId}/draft`, data })
  },
  confirm(familyId: string, messageId: string, expected_version: number) {
    return request<AgentConfirmResponse>(http, { method: 'POST', url: `/families/${familyId}/agent/messages/${messageId}/confirm`, data: { expected_version } })
  },
  dismiss(familyId: string, messageId: string) {
    return request<void>(http, { method: 'POST', url: `/families/${familyId}/agent/messages/${messageId}/dismiss` })
  },
  chat(familyId: string, data: AgentChatRequest) {
    return request<AgentChatResponse>(aiHttp, { method: 'POST', url: `/families/${familyId}/agent/chat`, data })
  },
  sessions(familyId: string, params: { before?: string; limit?: number } = {}) {
    return request<AgentSessionListResponse>(http, { method: 'GET', url: `/families/${familyId}/agent/sessions`, params })
  },
  sessionMessages(familyId: string, sessionId: string, params: { limit?: number; before?: string } = {}) {
    return request<AgentSessionMessagesResponse>(http, { method: 'GET', url: `/families/${familyId}/agent/sessions/${sessionId}/messages`, params })
  }
}

export { apiBase }
