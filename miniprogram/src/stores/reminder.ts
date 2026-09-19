import { defineStore } from 'pinia'
import type { Reminder } from '../types'

/**
 * 提醒 store。
 *
 * 与 `expense.ts` / `inventory.ts` / `moment.ts` 不同，本 store 在 Web 端
 * **确实被使用**：`RemindersView.vue` 读取 `filter` 并调用 `setFilter()` /
 * `complete()`，因此保留其全部逻辑。
 *
 * 但 `items` 在 Web 端是从 `mocks/data` 播种的，且视图只读 `filter`
 * （列表数据来自 `services.getReminders(filter)`）——`complete()` 对 `items`
 * 的修改实际不产生可见效果。
 *
 * 小程序端：去掉 mock 种子（初始为空数组），保留 filter / setFilter /
 * 各 getter 以维持与 Web 端一致的接口；新增 `set()` 供视图写入真实数据。
 */
interface ReminderState {
  items: Reminder[]
  filter: 'todo' | 'done' | 'all'
}

export const useReminderStore = defineStore('reminder', {
  state: (): ReminderState => ({
    items: [],
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
    set(items: Reminder[]) {
      this.items = items
    },
    setFilter(f: 'todo' | 'done' | 'all') {
      this.filter = f
    },
    complete(id: string) {
      const item = this.items.find((r) => r.id === id)
      if (item) item.state = 'done'
    }
  }
})
