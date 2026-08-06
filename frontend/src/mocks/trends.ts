// Mock 数据 — 趋势数据
import type { TrendDataPoint } from '../types'
import { dateStr } from '../utils/date'

export function generateTrends(catId: string) {
  const data: TrendDataPoint[] = []
  for (let i = 29; i >= 0; i--) {
    const date = dateStr(i)
    const base = catId === 'cat-whit' ? 4.0 : 4.9
    const weight = Math.round((base + (Math.random() - 0.5) * 0.3) * 10) / 10
    const food = Math.round(60 + Math.random() * 30)
    const water = Math.round(80 + Math.random() * 70)
    const poop = Math.random() > 0.1 ? 1 : 0
    const vomit = (catId === 'cat-whit' && i === 0) ? 1 : (Math.random() > 0.95 ? 1 : 0)
    const mental = Math.random() > 0.1 ? 'normal' : 'low'
    const events = i === 15
      ? [{ type: '换粮', label: '换粮：渴望室内猫粮' }]
      : i === 5
        ? [{ type: '就诊', label: '血常规检查' }]
        : []
    data.push({ date, weight, food, water, poop, vomit, mental, events })
  }
  return data
}
