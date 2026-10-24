import { defineStore } from 'pinia'
import type { ShoppingItem } from '../types'

/**
 * 待购买清单 store。
 *
 * 与 inventory / expense store 同一模式：视图负责调用 services，
 * store 只保存最近一次服务端数据并提供派生计数。
 */
interface ShoppingState {
  items: ShoppingItem[]
}

export const useShoppingStore = defineStore('shopping', {
  state: (): ShoppingState => ({
    items: []
  }),
  getters: {
    todoItems: (s) => s.items.filter((item) => item.status === 'todo'),
    doneItems: (s) => s.items.filter((item) => item.status !== 'todo'),
    todoCount: (s) => s.items.filter((item) => item.status === 'todo').length
  },
  actions: {
    set(items: ShoppingItem[]) {
      this.items = items
    },
    upsert(item: ShoppingItem) {
      const index = this.items.findIndex((existing) => existing.id === item.id)
      if (index >= 0) this.items.splice(index, 1, item)
      else this.items.unshift(item)
    },
    remove(id: string) {
      this.items = this.items.filter((item) => item.id !== id)
    }
  }
})
