import { defineStore } from 'pinia'

import type { Cat } from '../types'

interface CatState {
  cats: Cat[]
  currentCatId: string
}

export const useCatStore = defineStore('cat', {
  // 初始为空：数据由 /families/:id/cats 拉取（见 router 守卫与 services.getCats）
  state: (): CatState => ({
    cats: [],
    currentCatId: 'all'
  }),
  getters: {
    currentCat: (s) => s.cats.find((c) => c.id === s.currentCatId) ?? s.cats[0],
    activeCatId: (s) => s.currentCatId,
    hasCats: (s) => s.cats.length > 0
  },
  actions: {
    setCats(cats: Cat[]) {
      this.cats = cats
      // 当前选中猫咪被删除或首次加载时，回落到第一只
      if (this.currentCatId !== 'all' && (!this.currentCatId || !cats.some((c) => c.id === this.currentCatId))) {
        this.currentCatId = cats[0]?.id ?? 'all'
      }
    },
    setCat(id: string) {
      this.currentCatId = id
    },
    reset() {
      this.cats = []
      this.currentCatId = 'all'
    }
  }
})
