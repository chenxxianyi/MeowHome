import { defineStore } from 'pinia'
import { services } from '../services'
import type { TrendDataPoint } from '../types'

interface HealthState {
  trends: TrendDataPoint[]
  range: 7 | 30 | 90
  catId: string
}

export const useHealthStore = defineStore('health', {
  state: (): HealthState => ({
    trends: [],
    range: 30,
    catId: 'cat-whit'
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
      const res = await services.getTrends(catId, allowedRange)
      this.trends = res.data
    },
    setRange(r: 7 | 30 | 90) {
      this.range = r
      // Re-fetch would happen here in real app; for demo, slice from cached
      if (this.catId) {
        this.trends = this.trends.slice(0, r)
      }
    }
  }
})
