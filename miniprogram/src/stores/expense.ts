import { defineStore } from 'pinia'
import type { Expense } from '../types'

/**
 * 支出 store。
 *
 * 迁移实测：Web 端 `ExpensesView.vue` 虽然调用了 `useExpenseStore()`，
 * 但**从未读取本 store 的任何字段**——它用自己的局部 ref 并从
 * `services.getExpenses()` 取数。因此本 store 在 Web 端实际是残留。
 *
 * 小程序端不复制 `mocks/`（阶段 2/3 的完成标准要求「无 mock 残留」），
 * 故初始值改为空数组。若后续视图确实需要它，应作为**纯缓存层**使用：
 * 由视图取数后调 `set(expenses)` 写入，而非自带种子数据。
 */
export const useExpenseStore = defineStore('expense', {
  state: () => ({
    expenses: [] as Expense[]
  }),
  getters: {
    totalAmount: (s) => s.expenses.reduce((sum, e) => sum + e.amount, 0)
  },
  actions: {
    set(expenses: Expense[]) {
      this.expenses = expenses
    }
  }
})
