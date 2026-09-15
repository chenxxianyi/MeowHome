// 核心业务类型 — 猫宅 MeowHome

export type ID = string

export type Gender = 'female' | 'male' | 'unknown'

export type Severity = 'normal' | 'warning' | 'danger'

export type RecordSource = 'manual' | 'ai' | 'ocr' | 'import'

export type AIConfidence = 'high' | 'medium' | 'low' | 'none'

export type DraftStatus = 'local' | 'pending_sync' | 'syncing' | 'synced' | 'failed'

export interface User {
  id: ID
  name: string
  avatar?: string
  role: 'owner' | 'member'
}

export interface FamilyMember {
  id: ID
  name: string
  role: 'owner' | 'member'
  avatar?: string
}

export interface Family {
  id: ID
  name: string
  createdAt: string
  members: FamilyMember[]
}

export interface Cat {
  id: ID
  name: string
  gender: Gender
  breed: string
  birthday: string
  age: number
  neutered: boolean
  avatar?: string
  diseases: string[]
  allergies: string[]
  defaultFeeding?: string
  currentMedication?: string | null
  nextVaccine?: string | null
  nextDeworm?: string | null
}

export interface CatHealthProfile {
  catId: ID
  diseases: string[]
  allergies: string[]
  contraindications: string[]
}

export interface DailyRecordBase {
  id: ID
  catId: ID | ID[]
  createdAt: string
  occurredAt?: string
  source: RecordSource
  notes?: string
}

export interface FeedingRecord extends DailyRecordBase {
  type: 'feeding'
  meal?: string
  food?: string
  providedAmount?: number
  consumedAmount?: number
  appetite?: 'normal' | 'low' | 'very_low' | 'refused'
}

export interface DrinkingRecord extends DailyRecordBase {
  type: 'drinking'
  amount?: number
}

export interface EliminationRecord extends DailyRecordBase {
  type: 'elimination'
  kind?: 'poop' | 'pee' | 'both'
  form?: string
  color?: string
}

export interface VomitingRecord extends DailyRecordBase {
  type: 'vomit'
  count: number
  content?: string
  beforeMeal?: boolean | null
  includedHairball?: boolean
  mentalState?: string
}

export interface WeightRecord extends DailyRecordBase {
  type: 'weight'
  weight: number
}

export interface MentalStateRecord extends DailyRecordBase {
  type: 'mental'
  level: 'normal' | 'low' | 'lethargic'
}

export interface SymptomRecord extends DailyRecordBase {
  type: 'symptom'
  symptom: string
  severity: Severity
}

export type DailyRecord =
  | FeedingRecord
  | DrinkingRecord
  | EliminationRecord
  | VomitingRecord
  | WeightRecord
  | MentalStateRecord
  | SymptomRecord

export interface Medication {
  id: ID
  name: string
  dose?: string
  frequency?: string
  catIds: ID[]
  startedAt?: string
  endedAt?: string
}

export interface MedicationPlan extends Medication {
  type: 'plan'
  reminders: string[]
}

export interface MedicationLog extends DailyRecordBase {
  type: 'medication'
  medicationId: ID
  medicationName: string
  dose?: string
  givenAt: string
}

export interface Vaccination {
  id: ID
  catId: ID
  name: string
  scheduledFor: string
  givenAt?: string
}

export interface DewormingRecord {
  id: ID
  catId: ID
  product: string
  scheduledFor: string
  givenAt?: string
}

export interface MedicalVisit {
  id: ID
  catId: ID
  occurredAt: string
  hospital?: string
  reason?: string
  diagnosis?: string
  followUp?: string | null
}

export interface MedicalTestResult {
  id: ID
  metric: string
  value: string
  unit?: string
  referenceRange?: string
  status: 'normal' | 'borderline' | 'abnormal'
}

export interface MedicalDocument {
  id: ID
  catId: ID
  originalImage?: string
  ocrText?: string
  aiResult?: string
  metrics: MedicalTestResult[]
  status: 'uploaded' | 'processing' | 'ready' | 'failed' | 'confirmed'
  uploadedAt: string
}

export interface Reminder {
  id: ID
  catId?: ID | 'both'
  type: 'medication' | 'vaccine' | 'deworm' | 'checkup' | 'weight' | 'inventory' | 'water' | 'custom'
  title: string
  subtitle: string
  time: string
  state: 'todo' | 'done'
  icon?: string
}

export interface CareTask {
  id: ID
  title: string
  description?: string
  assignee?: ID
  frequency?: string
  nextDue?: string
}

export interface InventoryItem {
  id: ID
  name: string
  category: string
  quantity: number
  unit: string
  estimatedDays?: number
  status: 'low' | 'ok' | 'expired'
  expiry?: string | null
}

export interface Expense {
  id: ID
  date: string
  amount: number
  category: string
  label: string
  catId?: ID | 'both'
}

export interface TimelineEvent {
  id: ID
  date: string
  type: 'photo' | 'interaction' | 'milestone' | 'medical'
  title: string
  body?: string
  catId?: ID | 'both'
  images?: number
}

export interface CatInteraction {
  id: ID
  catIds: ID[]
  date: string
  notes: string
  type: 'play' | 'groom' | 'sleep' | 'other'
}

export interface AIParsedField {
  key: string
  value: string | number | boolean | null
  confidence: AIConfidence
  note?: string
}

export interface AIParsedRecord {
  id: ID
  type: string
  catId?: ID
  fields: AIParsedField[]
}

export interface AIParseSession {
  id: ID
  originalInput: string
  parsedAt: string
  model: string
  records: AIParsedRecord[]
}

export interface AIEvidence {
  type: string
  catId?: ID
  time: string
  content: string
}

export interface AIEvidence {
  type: string
  catId?: ID
  time: string
  content: string
}

export interface TodayStatusData {
  food: { amount: number; expected: number; unit: string; state: string; label: string }
  water: { amount: number; expected: number; unit: string; state: string; label: string }
  elimination: { state: string; label: string }
  vomit: { count: number; state: string; label: string }
  medication: { state: string; label: string; time?: string }
  mental: { state: string; label: string }
}

export interface AIAnalysisReport {
  id: ID
  generatedAt: string
  recordCount: number
  body: string
  evidence: AIEvidence[]
}

export interface TrendEventMarker {
  type: string
  label: string
}

export interface TrendDataPoint {
  date: string
  weight: number
  food: number
  water: number
  poop: number
  vomit: number
  mental: 'normal' | 'low'
  events: TrendEventMarker[]
}

export interface FocusItemData {
  id: ID
  catId: ID
  type: string
  severity: 'warn' | 'info' | 'danger'
  title: string
  body: string
  evidence: string[]
  actions: { label: string; action: string }[]
}

export interface RecordTypeItem {
  id: ID
  label: string
  icon: string
}

export interface RecordTypeGroup {
  high: RecordTypeItem[]
  health: RecordTypeItem[]
  life: RecordTypeItem[]
}

export interface APIResponse<T> {
  success: boolean
  data: T
  requestId?: string
}

export interface PaginatedResponse<T> {
  items: T[]
  total: number
  page: number
  pageSize: number
}
