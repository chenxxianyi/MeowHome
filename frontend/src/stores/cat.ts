import { defineStore } from 'pinia'
import { mockCats } from '../mocks/family'
import type { Cat } from '../types'

interface CatState {
  cats: Cat[]
  currentCatId: string
}

export const useCatStore = defineStore('cat', {
  state: (): CatState => ({
    cats: mockCats,
    currentCatId: 'cat-whit'
  }),
  getters: {
    currentCat: (s) => s.cats.find((c) => c.id === s.currentCatId) ?? s.cats[0],
    activeCatId: (s) => s.currentCatId
  },
  actions: {
    setCat(id: string) {
      this.currentCatId = id
    }
  }
})
