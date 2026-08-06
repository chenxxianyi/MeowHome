import { defineStore } from 'pinia'
import type { Reminder } from '../types'
import { mockReminders } from '../mocks/data'

interface ReminderState {
  items: Reminder[]
  filter: 'todo' | 'done' | 'all'
}

export const useReminderStore = defineStore('reminder', {
  state: (): ReminderState => ({
    items: mockReminders,
    filter: 'todo'
  }),
  getters: {
    filtered: (s) => {
      if (s.filter === 'todo') return s.items.filter((r) => r.state === 'todo')
      if (s.filter === 'done') return s.items.filter((r) => r.state === 'done')
      return s.items
    },
    todoCount: (s) => s.items.filter((r) => r.state === 'todo').length
  },
  actions: {
    setFilter(f: 'todo' | 'done' | 'all') {
      this.filter = f
    },
    complete(id: string) {
      const item = this.items.find((r) => r.id === id)
      if (item) item.state = 'done'
    }
  }
})
