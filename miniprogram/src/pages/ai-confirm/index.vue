<script setup lang="ts">
import { ref, computed } from 'vue'
import { useRouter } from '../../utils/navigation'
import AppHeader from '../../components/app/AppHeader.vue'
import AIResultBadge from '../../components/ai/AIResultBadge.vue'
import AIEvidencePanel from '../../components/ai/AIEvidencePanel.vue'
import { useAppStore } from '../../stores/app'
import { useCatStore } from '../../stores/cat'
import { services } from '../../services'
import { useProtectedPage } from '../../utils/page'
import type { AIParseSession } from '../../types'

const app = useAppStore()
const catStore = useCatStore()
const router = useRouter()
const showEvidence = ref(false)
const saving = ref(false)
const error = ref('')

const emptySession: AIParseSession = {
  id: '',
  originalInput: '',
  parsedAt: '',
  model: '',
  records: []
}
const session = ref<AIParseSession>((app.aiSession as AIParseSession | null) || emptySession)
const recordCount = computed(() => session.value.records?.length || 0)

const recordTypeLabel: Record<string, string> = {
  feeding: '喂食',
  vomit: '呕吐',
  medication: '用药',
  drinking: '饮水',
  elimination: '排便'
}

function removeRecord(id: string) {
  session.value = {
    ...session.value,
    records: (session.value.records || []).filter((r) => r.id !== id)
  }
}

const evidenceItems = computed(() =>
  session.value.records.flatMap((rec) =>
    rec.fields
      .filter((f) => f.value)
      .map((f) => ({
        type: recordTypeLabel[rec.type] || rec.type,
        time: String(f.value),
        content: `${f.key}: ${f.value}`,
        catId: rec.catId
      }))
  )
)

function catName(id?: string): string {
  return catStore.cats.find((cat) => cat.id === id)?.name || '未指定猫咪'
}

async function confirm() {
  if (!recordCount.value || saving.value) return
  saving.value = true
  error.value = ''
  try {
    const result = await services.saveRecord({ type: 'ai-confirmed', records: session.value.records })
    if (!result.success) {
      error.value = '保存失败，请检查网络后重试'
      return
    }
    app.clearAIState()
    router.push('/today')
  } catch {
    error.value = '保存失败，请检查网络后重试'
  } finally {
    saving.value = false
  }
}

function goBack() {
  router.push('/records')
}

useProtectedPage(() => {
  if (!app.aiSession) {
    uni.showToast({ title: '没有待确认的 AI 记录', icon: 'none' })
    router.replace('/records')
  }
})
</script>

<template>
  <AppHeader title="AI 解析确认" />

  <view class="page-content">
    <!-- Original input -->
    <view class="ai-original">
      <view class="ai-original-label"> 你的原始输入 </view>
      <view class="ai-original-text">
        {{ session.originalInput }}
      </view>
    </view>

    <view class="ai-meta-row">
      <text>AI 解析为 {{ recordCount }} 条记录</text>
      <text>{{ session.model }} · {{ session.parsedAt?.slice(11, 16) }}</text>
    </view>

    <view class="warn-banner"> ⚠ 以下数据未确认，不会写入正式档案。请仔细检查每条记录。 </view>

    <!-- Record cards -->
    <view v-for="rec in session.records" :key="rec.id" class="ai-record-card">
      <view class="ai-record-header">
        <view class="ai-record-type">
          <AIResultBadge />
          {{ recordTypeLabel[rec.type] || '记录' }}
          <text class="cat-tag">{{ catName(rec.catId) }}</text>
        </view>
        <button class="ai-record-delete" aria-label="删除此记录" @click="removeRecord(rec.id)">删除</button>
      </view>
      <view class="ai-fields">
        <view v-for="f in rec.fields" :key="f.key" class="ai-field">
          <text class="ai-field-label">{{ f.key }}</text>
          <text class="ai-field-value" :class="f.confidence === 'high' ? 'confirmed' : 'unconfirmed'">
            {{ f.value || '—' }}
            <text v-if="f.note" class="ai-field-note">{{ f.note }}</text>
          </text>
        </view>
      </view>
    </view>

    <!-- Disclaimer -->
    <view class="ai-disclaimer">
      AI 整理 · 不构成医疗诊断
      <button class="ai-evidence-link" @click="showEvidence = !showEvidence">
        {{ showEvidence ? '收起依据' : '查看解析依据' }}
      </button>
    </view>

    <!-- Evidence panel -->
    <AIEvidencePanel v-if="showEvidence" :items="evidenceItems" />

    <!-- Bottom bar -->
    <view class="bottom-bar">
      <view v-if="error" class="form-error">{{ error }}</view>
      <button class="btn-secondary" @click="goBack">返回修改</button>
      <button class="btn-primary" :disabled="!recordCount || saving" @click="confirm">
        {{ saving ? '保存中…' : `确认并保存 ${recordCount} 条记录` }}
      </button>
    </view>
  </view>
</template>

<style scoped>
.ai-original {
  background: var(--color-bg-subtle);
  padding: var(--space-12);
  border-radius: var(--radius-md);
  margin-bottom: var(--space-12);
}
.ai-original-label {
  font-size: var(--font-size-caption);
  color: var(--color-text-tertiary);
}
.ai-original-text {
  margin-top: 4px;
  font-size: var(--font-size-body);
  white-space: pre-wrap;
}
.ai-meta-row {
  display: flex;
  justify-content: space-between;
  font-size: var(--font-size-caption);
  color: var(--color-text-tertiary);
  margin-bottom: var(--space-12);
}
.warn-banner {
  background: var(--color-warning-soft);
  color: var(--color-warning);
  padding: var(--space-8) var(--space-12);
  border-radius: var(--radius-sm);
  font-size: var(--font-size-assist);
  margin-bottom: var(--space-16);
}
.ai-record-card {
  background: var(--color-bg-surface);
  border: 1px solid var(--color-divider);
  border-radius: var(--radius-md);
  padding: var(--space-12);
  margin-bottom: var(--space-12);
}
.ai-record-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: var(--space-8);
}
.ai-record-type {
  font-weight: 600;
  font-size: var(--font-size-body);
  display: flex;
  align-items: center;
  gap: 6px;
}
.cat-tag {
  font-size: var(--font-size-caption);
  color: var(--color-text-tertiary);
}
.ai-record-delete {
  font-size: var(--font-size-assist);
  color: var(--color-text-tertiary);
  cursor: pointer;
  padding: 4px 8px;
}
.ai-fields {
  display: flex;
  flex-direction: column;
  gap: 4px;
}
.ai-field {
  display: flex;
  justify-content: space-between;
  font-size: var(--font-size-assist);
}
.ai-field-label {
  color: var(--color-text-tertiary);
}
.ai-field-value {
  font-weight: 500;
}
.ai-field-value.confirmed {
  color: var(--color-success);
}
.ai-field-value.unconfirmed {
  color: var(--color-warning);
}
.ai-field-note {
  font-size: 10px;
  color: var(--color-warning);
}
.ai-disclaimer {
  font-size: var(--font-size-caption);
  color: var(--color-text-tertiary);
  margin-top: var(--space-16);
}
.bottom-bar {
  position: fixed;
  bottom: var(--safe-bottom);
  left: 0;
  right: 0;
  padding: var(--space-12) var(--space-16);
  background: var(--color-bg-surface);
  border-top: 1px solid var(--color-divider);
  display: flex;
  flex-wrap: wrap;
  gap: var(--space-12);
}
.bottom-bar .form-error {
  flex-basis: 100%;
  color: var(--color-danger);
  font-size: var(--font-size-assist);
}
.bottom-bar .btn-primary {
  flex: 2;
  margin: 0;
}
.bottom-bar .btn-secondary {
  flex: 1;
  margin: 0;
}
</style>
