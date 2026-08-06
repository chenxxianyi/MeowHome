<script setup lang="ts">
import { ref, computed, watch } from 'vue'
import { useRouter } from 'vue-router'
import AppShell from '../../components/app/AppShell.vue'
import AppHeader from '../../components/app/AppHeader.vue'
import AIResultBadge from '../../components/ai/AIResultBadge.vue'
import AIEvidencePanel from '../../components/ai/AIEvidencePanel.vue'
import { useAppStore } from '../../stores/app'
import { services } from '../../services'
import type { AIParseSession, AIParsedRecord } from '../../types'

const app = useAppStore()
const router = useRouter()
const showEvidence = ref(false)

// Fallback session for demo when store empty
const fallbackSession: AIParseSession = {
  id: 'mock-fallback',
  originalInput: '小白今天早上没怎么吃，下午吐了一次黄色的水。小橘晚上要记得喂药。',
  parsedAt: new Date().toISOString(),
  model: 'mock-ai-v1',
  records: [
    { id: 'r1', type: 'feeding', catId: 'cat-whit', fields: [
      { key: 'time', value: '07:30', confidence: 'high' },
      { key: 'food', value: '渴望室内猫粮', confidence: 'medium', note: 'AI推断' },
      { key: 'provided', value: '80g', confidence: 'high' },
      { key: 'consumed', value: '35g', confidence: 'low', note: 'AI推断：不到一半' },
      { key: 'appetite', value: '偏低', confidence: 'medium' }
    ]},
    { id: 'r2', type: 'vomit', catId: 'cat-whit', fields: [
      { key: 'time', value: '14:00', confidence: 'medium' },
      { key: 'count', value: '1', confidence: 'high' },
      { key: 'content', value: '黄色液体', confidence: 'high' },
      { key: 'mentalState', value: '正常', confidence: 'low', note: '未明确提及' }
    ]},
    { id: 'r3', type: 'medication', catId: 'cat-oran', fields: [
      { key: 'time', value: '20:00', confidence: 'medium', note: '推断为晚间' },
      { key: 'medication', value: '肾上腺素抑制剂', confidence: 'high' },
      { key: 'dose', value: null, confidence: 'none', note: '待确认' }
    ]}
  ]
}

const session = ref<AIParseSession>(app.aiSession as AIParseSession || fallbackSession)
const recordCount = computed(() => session.value.records?.length || 0)

const recordTypeLabel: Record<string, string> = {
  feeding: '喂食', vomit: '呕吐', medication: '用药', drinking: '饮水', elimination: '排便'
}

function removeRecord(id: string) {
  session.value = {
    ...session.value,
    records: (session.value.records || []).filter(r => r.id !== id)
  }
}

const evidenceItems = computed(() =>
  session.value.records.flatMap((rec) =>
    rec.fields.filter(f => f.value).map(f => ({
      type: recordTypeLabel[rec.type] || rec.type,
      time: String(f.value),
      content: `${f.key}: ${f.value}`,
      catId: rec.catId
    }))
  )
)

function catName(id?: string): string {
  if (!id) return '未知'
  if (id === 'cat-whit') return '小白'
  if (id === 'cat-oran') return '小橘'
  return '未知'
}

async function confirm() {
  try {
    await services.saveRecord({ type: 'ai-confirmed', records: session.value.records })
    app.clearAIState()
    router.push('/today')
  } catch {
    // keep state for retry
  }
}

function goBack() {
  router.push('/records')
}
</script>

<template>
  <AppShell>
    <AppHeader title="AI 解析确认" />

    <div class="page-content">
      <!-- Original input -->
      <div class="ai-original">
        <div class="ai-original-label">
          你的原始输入
        </div>
        <div class="ai-original-text">
          {{ session.originalInput }}
        </div>
      </div>

      <div class="ai-meta-row">
        <span>AI 解析为 {{ recordCount }} 条记录</span>
        <span>{{ session.model }} · {{ session.parsedAt?.slice(11, 16) }}</span>
      </div>

      <div class="warn-banner">
        ⚠ 以下数据未确认，不会写入正式档案。请仔细检查每条记录。
      </div>

      <!-- Record cards -->
      <div
        v-for="rec in session.records"
        :key="rec.id"
        class="ai-record-card"
      >
        <div class="ai-record-header">
          <div class="ai-record-type">
            <AIResultBadge />
            {{ recordTypeLabel[rec.type] || '记录' }}
            <span class="cat-tag">{{ catName(rec.catId) }}</span>
          </div>
          <button
            class="ai-record-delete"
            aria-label="删除此记录"
            @click="removeRecord(rec.id)"
          >
            删除
          </button>
        </div>
        <div class="ai-fields">
          <div
            v-for="f in rec.fields"
            :key="f.key"
            class="ai-field"
          >
            <span class="ai-field-label">{{ f.key }}</span>
            <span
              class="ai-field-value"
              :class="f.confidence === 'high' ? 'confirmed' : 'unconfirmed'"
            >
              {{ f.value || '—' }}
              <span
                v-if="f.note"
                class="ai-field-note"
              >{{ f.note }}</span>
            </span>
          </div>
        </div>
      </div>

      <!-- Disclaimer -->
      <div class="ai-disclaimer">
        AI 整理 · 不构成医疗诊断
      </div>

      <!-- Evidence panel -->
      <AIEvidencePanel
        v-if="showEvidence"
        :items="evidenceItems"
      />

      <!-- Bottom bar -->
      <div class="bottom-bar">
        <button
          class="btn-secondary"
          @click="goBack"
        >
          返回修改
        </button>
        <button
          class="btn-primary"
          @click="confirm"
        >
          确认并保存 {{ recordCount }} 条记录
        </button>
      </div>
    </div>
  </AppShell>
</template>

<style scoped>
.ai-original { background: var(--color-bg-subtle); padding: var(--space-12); border-radius: var(--radius-md); margin-bottom: var(--space-12); }
.ai-original-label { font-size: var(--font-size-caption); color: var(--color-text-tertiary); }
.ai-original-text { margin-top: 4px; font-size: var(--font-size-body); white-space: pre-wrap; }
.ai-meta-row { display: flex; justify-content: space-between; font-size: var(--font-size-caption); color: var(--color-text-tertiary); margin-bottom: var(--space-12); }
.warn-banner { background: var(--color-warning-soft); color: var(--color-warning); padding: var(--space-8) var(--space-12); border-radius: var(--radius-sm); font-size: var(--font-size-assist); margin-bottom: var(--space-16); }
.ai-record-card { background: var(--color-bg-surface); border: 1px solid var(--color-divider); border-radius: var(--radius-md); padding: var(--space-12); margin-bottom: var(--space-12); }
.ai-record-header { display: flex; justify-content: space-between; align-items: center; margin-bottom: var(--space-8); }
.ai-record-type { font-weight: 600; font-size: var(--font-size-body); display: flex; align-items: center; gap: 6px; }
.cat-tag { font-size: var(--font-size-caption); color: var(--color-text-tertiary); }
.ai-record-delete { font-size: var(--font-size-assist); color: var(--color-text-tertiary); cursor: pointer; padding: 4px 8px; }
.ai-fields { display: flex; flex-direction: column; gap: 4px; }
.ai-field { display: flex; justify-content: space-between; font-size: var(--font-size-assist); }
.ai-field-label { color: var(--color-text-tertiary); }
.ai-field-value { font-weight: 500; }
.ai-field-value.confirmed { color: var(--color-success); }
.ai-field-value.unconfirmed { color: var(--color-warning); }
.ai-field-note { font-size: 10px; color: var(--color-warning); }
.ai-disclaimer { font-size: var(--font-size-caption); color: var(--color-text-tertiary); margin-top: var(--space-16); }
.bottom-bar { position: fixed; bottom: calc(64px + var(--safe-bottom)); left: 0; right: 0; padding: var(--space-12) var(--space-16); background: var(--color-bg-surface); border-top: 1px solid var(--color-divider); display: flex; gap: var(--space-12); }
.bottom-bar .btn-primary { flex: 2; margin: 0; }
.bottom-bar .btn-secondary { flex: 1; margin: 0; }
</style>
