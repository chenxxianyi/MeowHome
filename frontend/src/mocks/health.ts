// Mock 数据 — 今日状态与 AI 摘要
import { todayStr } from '../utils/date'

export interface TodayStatusRow {
  state: 'normal' | 'warning' | 'danger' | 'none'
  label: string
  icon?: string
}

export interface TodayStatusData {
  food: { amount: number; expected: number; unit: string; state: string; label: string }
  water: { amount: number; expected: number; unit: string; state: string; label: string }
  elimination: { state: string; label: string }
  vomit: { count: number; state: string; label: string }
  medication: { state: string; label: string; time?: string }
  mental: { state: string; label: string }
}

export const todayStatusByCat: Record<string, TodayStatusData> = {
  'cat-whit': {
    food: { amount: 55, expected: 80, unit: 'g', state: 'warning', label: '早餐食量偏低' },
    water: { amount: 120, expected: 150, unit: 'ml', state: 'normal', label: '饮水正常' },
    elimination: { state: 'normal', label: '排便正常' },
    vomit: { count: 1, state: 'danger', label: '下午 14:45 呕吐 1 次（黄色液体）' },
    medication: { state: 'none', label: '无需用药', time: undefined },
    mental: { state: 'normal', label: '精神状态正常' }
  },
  'cat-oran': {
    food: { amount: 75, expected: 80, unit: 'g', state: 'normal', label: '进食正常' },
    water: { amount: 110, expected: 120, unit: 'ml', state: 'normal', label: '饮水正常' },
    elimination: { state: 'normal', label: '排便正常' },
    vomit: { count: 0, state: 'none', label: '无呕吐' },
    medication: { state: 'warning', label: '今晚需服药：肾上腺素抑制剂', time: '20:00' },
    mental: { state: 'normal', label: '精神状态正常' }
  }
}

export const mockAISummary = {
  id: 'ai-daily-' + todayStr(),
  generatedAt: todayStr() + 'T08:30:00+08:00',
  evidenceCount: 12,
  body: '今日共记录12条数据。小白早餐后食量偏低（55g/80g），下午呕吐1次黄色液体，建议继续观察食欲和排便情况。小橘今日状态稳定，夜间需按时服药。两只猫饮水均正常，排便规律。',
  evidence: [
    { type: 'feeding', catId: 'cat-whit', time: '07:30', content: '早餐食量55g，低于预期' },
    { type: 'vomit', catId: 'cat-whit', time: '14:45', content: '呕吐1次，黄色液体' },
    { type: 'feeding', catId: 'cat-oran', time: '07:00', content: '早餐正常进食75g' },
    { type: 'medication', catId: 'cat-oran', time: '20:00', content: '待服药：肾上腺素抑制剂' },
    { type: 'water', catId: 'cat-whit', time: '全天', content: '饮水120ml，正常' },
    { type: 'water', catId: 'cat-oran', time: '全天', content: '饮水110ml，正常' }
  ]
}
