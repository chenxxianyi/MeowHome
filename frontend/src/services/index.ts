// 业务服务 — 从 Mock Service 拉取数据
// 后端完成后，仅需将 Mock 返回值替换为对应的 http 请求即可（接口定义保持一致）。
import {
  mockReminders, mockMoments, mockInventory, mockExpenses,
  mockFocusItems, mockRecordTypes, mockAIParseSession
} from '../mocks/data'
import { mockFamily } from '../mocks/family'
import { mockCats } from '../mocks/family'
import { mockAISummary, todayStatusByCat } from '../mocks/health'
import { generateTrends } from '../mocks/trends'
import type { APIResponse } from '../types'

const delay = (ms: number) => new Promise((resolve) => setTimeout(resolve, ms))

export const services = {
  async getFamily() {
    await delay(200)
    return { success: true, data: mockFamily }
  },

  async getCats() {
    await delay(200)
    return { success: true, data: mockCats }
  },

  async getTodayStatus(catId: string) {
    await delay(200)
    return { success: true, data: todayStatusByCat[catId] || todayStatusByCat['cat-whit'] }
  },

  async getFocusItems() {
    await delay(200)
    return { success: true, data: mockFocusItems }
  },

  async getAISummary() {
    await delay(500)
    return { success: true, data: mockAISummary }
  },

  async getReminders(filter?: string) {
    await delay(200)
    let list = [...mockReminders]
    if (filter === 'todo') list = list.filter((r) => r.state === 'todo')
    if (filter === 'done') list = list.filter((r) => r.state === 'done')
    return { success: true, data: list }
  },

  async getTrends(catId: string, range: number) {
    await delay(200)
    const all = generateTrends(catId)
    const sliced = all.slice(0, range)
    return { success: true, data: sliced }
  },

  async getMoments() {
    await delay(200)
    return { success: true, data: mockMoments }
  },

  async getInventory() {
    await delay(200)
    return { success: true, data: mockInventory }
  },

  async getExpenses() {
    await delay(200)
    return { success: true, data: mockExpenses }
  },

  async parseAI(text: string): Promise<APIResponse<typeof mockAIParseSession>> {
    await delay(800)
    if (!text || text.trim().length < 3) {
      return { success: false, data: mockAIParseSession, requestId: 'mock' }
    }
    return { success: true, data: mockAIParseSession, requestId: 'mock-ai-' + Date.now() }
  },

  async saveRecord(record: Record<string, unknown>) {
    await delay(300)
    return { success: true, data: { ...record, id: 'rec-' + Date.now() } }
  },

  async completeReminder(id: string) {
    await delay(200)
    return { success: true, data: { id, state: 'done' } }
  }
}

export const recordTypes = mockRecordTypes
