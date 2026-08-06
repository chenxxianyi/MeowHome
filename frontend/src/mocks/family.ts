// Mock 数据 — 猫宅 MeowHome（开发数据，与 H5 demo 保持一致）
import type { Cat, Family, AIParseSession } from '../types'
import type { TodayStatusData, FocusItemData, RecordTypeGroup, Reminder, TimelineEvent, InventoryItem, Expense } from '../types'

// —— 家庭 ——
export const mockFamily: Family = {
  id: 'fam-001',
  name: '小家的猫宅',
  createdAt: '2024-01-10',
  members: [
    { id: 'mem-1', name: '小明', role: 'owner' },
    { id: 'mem-2', name: '小红', role: 'member' }
  ]
}

// —— 猫咪 ——
export const mockCats: Cat[] = [
  {
    id: 'cat-whit',
    name: '小白',
    gender: 'female',
    breed: '中华田园猫',
    birthday: '2021-03-15',
    age: 4,
    neutered: true,
    avatar: '',
    diseases: [],
    allergies: [],
    currentMedication: null,
    nextVaccine: '2026-09-10',
    nextDeworm: '2026-08-20'
  },
  {
    id: 'cat-oran',
    name: '小橘',
    gender: 'male',
    breed: '中华田园猫',
    birthday: '2020-07-22',
    age: 5,
    neutered: true,
    avatar: '',
    diseases: ['慢性肾病'],
    allergies: ['鸡肉'],
    currentMedication: '肾上腺素抑制剂',
    nextVaccine: '2026-10-05',
    nextDeworm: '2026-08-15'
  }
]
