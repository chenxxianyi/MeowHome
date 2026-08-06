import { defineStore } from 'pinia'

interface DraftRecord {
  id: string
  type: string
  catId: string
  content: Record<string, unknown>
  createdAt: string
  status: 'local' | 'pending_sync' | 'syncing' | 'synced' | 'failed'
}

interface RecordState {
  drafts: DraftRecord[]
  recentRecordType: string
}

export const useRecordStore = defineStore('record', {
  state: (): RecordState => ({
    drafts: [],
    recentRecordType: 'feeding'
  }),
  actions: {
    addDraft(draft: DraftRecord) {
      this.drafts.push(draft)
    },
    removeDraft(id: string) {
      this.drafts = this.drafts.filter((d) => d.id !== id)
    },
    markSynced(id: string) {
      const d = this.drafts.find((x) => x.id === id)
      if (d) d.status = 'synced'
    },
    setRecentRecordType(type: string) {
      this.recentRecordType = type
    }
  }
})
