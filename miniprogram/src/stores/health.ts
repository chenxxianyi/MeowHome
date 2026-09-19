import { defineStore } from 'pinia'

import { services } from '../services'
import type { TrendDataPoint } from '../types'

/** 趋势 store，通过统一业务服务层调用真实 `care.trends` 接口。 */
interface HealthState {
  trends: TrendDataPoint[]
  range: 7 | 30 | 90
  catId: string
}

export const useHealthStore = defineStore('health', {
  state: (): HealthState => ({
    trends: [],
    range: 30,
    catId: ''
  }),
  getters: {
    coverage: (s) => s.trends.length,
    weightPoints: (s) => s.trends.map((t) => ({ date: t.date, value: t.weight })),
    foodPoints: (s) => s.trends.map((t) => ({ date: t.date, value: t.food })),
    waterPoints: (s) => s.trends.map((t) => ({ date: t.date, value: t.water })),
    events: (s) => s.trends.filter((t) => (t.events?.length ?? 0) > 0)
  },
  actions: {
    async loadTrends(catId: string, range: number) {
      const allowedRange = range === 7 || range === 90 ? range : 30
      this.catId = catId
      this.range = allowedRange

      const result = await services.getTrends(catId, allowedRange)
      this.trends = result.data
    },
    setRange(r: 7 | 30 | 90) {
      this.range = r
      // 暂时保留 Web 端原行为（对缓存切片）。
      // 注意：按天数切片趋势数组语义上并不正确——正确做法是带新 range
      // 重新 loadTrends。迁移 TrendsView（步骤 3.5）时应改为重新取数。
      if (this.catId) {
        this.trends = this.trends.slice(0, r)
      }
    }
  }
})
