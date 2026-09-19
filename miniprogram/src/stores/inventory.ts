import { defineStore } from 'pinia'
import type { InventoryItem } from '../types'

/**
 * 库存 store。
 *
 * 与 `expense.ts` 同理：Web 端 `InventoryView.vue` 实例化了本 store
 * 但**从不读取其字段**，故小程序端去掉 mock 种子，改为空数组 + 写入动作。
 */
interface InventoryState {
  items: InventoryItem[]
}

export const useInventoryStore = defineStore('inventory', {
  state: (): InventoryState => ({
    items: []
  }),
  getters: {
    lowStockCount: (s) => s.items.filter((i) => i.status === 'low').length,
    expiredCount: (s) => s.items.filter((i) => i.status === 'expired').length,
    totalItems: (s) => s.items.length
  },
  actions: {
    set(items: InventoryItem[]) {
      this.items = items
    }
  }
})
