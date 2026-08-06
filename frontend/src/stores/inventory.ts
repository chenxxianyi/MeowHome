import { defineStore } from 'pinia'
import type { InventoryItem } from '../types'
import { mockInventory } from '../mocks/data'

interface InventoryState {
  items: InventoryItem[]
}

export const useInventoryStore = defineStore('inventory', {
  state: (): InventoryState => ({
    items: mockInventory
  }),
  getters: {
    lowStockCount: (s) => s.items.filter((i) => i.status === 'low').length,
    expiredCount: (s) => s.items.filter((i) => i.status === 'expired').length,
    totalItems: (s) => s.items.length
  }
})
