import { defineStore } from 'pinia'
import type { TimelineEvent } from '../types'
import { mockMoments } from '../mocks/data'

interface MomentState {
  events: TimelineEvent[]
  month: string | null
}

export const useMomentStore = defineStore('moment', {
  state: (): MomentState => ({
    events: mockMoments,
    month: null
  }),
  actions: {
    setMonth(m: string) { this.month = m }
  }
})
