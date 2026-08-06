import { defineStore } from 'pinia'
import type { Expense } from '../types'
import { mockExpenses } from '../mocks/data'

export const useExpenseStore = defineStore('expense', {
  state: () => ({
    expenses: mockExpenses
  }),
  getters: {
    totalAmount: (s) => s.expenses.reduce((sum, e) => sum + e.amount, 0)
  }
})
