import { defineStore } from 'pinia'
import type { TimelineEvent } from '../types'

/**
 * 时光 store。
 *
 * 迁移时实测：本 store 在 Web 端**被引用 0 次**，是完全的死代码
 * （`MomentsView.vue` 直接调 `services.getMoments()`）。
 * 小程序端保留结构但去掉 mock 种子，避免复制 `mocks/`。
 */
interface MomentState {
  events: TimelineEvent[]
  month: string | null
}

export const useMomentStore = defineStore('moment', {
  state: (): MomentState => ({
    events: [],
    month: null
  }),
  actions: {
    setMonth(m: string) {
      this.month = m
    },
    set(events: TimelineEvent[]) {
      this.events = events
    }
  }
})
