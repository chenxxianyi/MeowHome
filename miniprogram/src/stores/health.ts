import { defineStore } from 'pinia'

import { toApiError } from '../api/adapter'
import { getStoredFamilyId } from '../api/client'
import { careApi } from '../api/endpoints'
import type { TrendDataPoint } from '../types'

/**
 * 趋势 store。
 *
 * 与 Web 端的差异：Web 端调 `services.getTrends()`（由 services 层解析
 * familyId，并把异常兜底成 `{success,data}`）。小程序端的 `services` 层
 * 属阶段 2/3 范围，此处先直连 `careApi` + 从 storage 取 familyId。
 *
 * **待办**：阶段 2/3 建好 `services` 后，本文件应改回调用
 * `services.getTrends()`，以与 Web 端架构保持一致。
 */
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

      const familyId = getStoredFamilyId()
      if (!familyId) {
        this.trends = []
        return
      }
      try {
        this.trends = await careApi.trends(familyId, catId, allowedRange)
      } catch (e) {
        console.warn(`[health] loadTrends: ${toApiError(e).message}`)
        this.trends = []
      }
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
