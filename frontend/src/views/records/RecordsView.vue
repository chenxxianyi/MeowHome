<script setup lang="ts">
import { ref } from 'vue'
import { useRouter } from 'vue-router'
import AppShell from '../../components/app/AppShell.vue'
import AppIcon from '../../components/app/AppIcon.vue'
import AIResultBadge from '../../components/ai/AIResultBadge.vue'
import { useAppStore } from '../../stores/app'
import { services } from '../../services'
import { mockRecordTypes } from '../../mocks/data'
import type { RecordTypeGroup } from '../../types'

const router = useRouter()
const app = useAppStore()
const recordType = ref('feeding')
const recordTypes = ref<RecordTypeGroup>(mockRecordTypes)
const aiInput = ref('')
const aiResult = ref<any>(null)
const aiLoading = ref(false)
const showAI = ref(false)

function selectType(type: string) {
  router.push(`/records/quick/${type}`)
}

async function parseAI() {
  if (!aiInput.value.trim()) return
  aiLoading.value = true
  try {
    const res = await services.parseAI(aiInput.value)
    aiResult.value = res.data
    showAI.value = true
  } finally {
    aiLoading.value = false
  }
}
</script>

<template>
  <AppShell>
    <div class="page-header">
      <div class="page-title">
        记录
      </div>
    </div>

    <div class="page-content">
      <!-- AI 自然语言输入，复用 demo 结构 -->
      <section class="ai-input-section">
        <label
          class="ai-input-label"
          for="ai-natural-input"
        >告诉猫宅管家，猫咪今天发生了什么…</label>
        <div class="ai-input-row">
          <textarea
            id="ai-natural-input"
            v-model="aiInput"
            class="ai-input-field"
            rows="3"
            placeholder="例如：小白今天早上没怎么吃，下午吐了一次黄色的水…"
            aria-label="AI 自然语言输入"
          />
          <button
            class="ai-action-btn"
            aria-label="拍照"
            @click="router.push('/medical/upload')"
          >
            <AppIcon
              name="camera"
              :size="20"
            />
          </button>
          <button
            class="ai-action-btn"
            aria-label="上传图片"
            @click="router.push('/medical/upload')"
          >
            <AppIcon
              name="upload"
              :size="20"
            />
          </button>
          <button
            class="ai-action-btn"
            aria-label="病历扫描"
            @click="router.push('/medical/upload')"
          >
            <AppIcon
              name="medical"
              :size="20"
            />
          </button>
          <button
            class="btn-primary ai-submit"
            :disabled="!aiInput.trim() || aiLoading"
            @click="parseAI"
          >
            <AppIcon
              name="ai"
              :size="16"
            /> {{ aiLoading ? '解析中…' : '交给管家整理' }}
          </button>
        </div>
      </section>

      <!-- AI 解析结果 -->
      <div
        v-if="showAI && aiResult"
        class="ai-confirm-section"
      >
        <AIResultBadge />
        <div class="ai-confirm-list">
          <div
            v-for="rec in aiResult.records"
            :key="rec.id"
            class="ai-record-card"
          >
            <div class="ai-record-header">
              <div class="ai-record-type">
                {{ rec.type === 'feeding' ? '喂食' : rec.type === 'vomit' ? '呕吐' : '用药' }}
                <span class="cat-tag">{{ rec.catId === 'cat-whit' ? '小白' : '小橘' }}</span>
              </div>
              <button class="ai-record-delete">
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
        </div>
        <div class="ai-disclaimer">
          AI 整理 · 不构成医疗诊断 · 请确认后保存
        </div>
        <div class="ai-actions">
          <button
            class="btn-secondary"
            @click="showAI = false"
          >
            返回修改
          </button>
          <button
            class="btn-primary"
            @click="router.push('/records/ai/confirm')"
          >
            确认并保存
          </button>
        </div>
      </div>

      <!-- 记录类型 -->
      <section
        v-for="(group, key) in recordTypes"
        :key="key"
        class="record-type-section"
      >
        <div class="record-type-group-title">
          {{ key === 'high' ? '常用' : key === 'health' ? '健康' : '生活' }}
        </div>
        <div class="record-type-grid">
          <button
            v-for="item in group"
            :key="item.id"
            class="record-type-btn"
            @click="selectType(item.id)"
          >
            <AppIcon
              :name="item.icon"
              :size="24"
            />
            <span>{{ item.label }}</span>
          </button>
        </div>
      </section>
    </div>
  </AppShell>
</template>

<style scoped>
.ai-submit { margin-left: auto; width: auto; min-height: 44px; padding: 0 var(--space-20); font-size: var(--font-size-body); display: flex; align-items: center; gap: 6px; }
.ai-confirm-section { margin-top: var(--space-16); }
.ai-confirm-list { display: flex; flex-direction: column; gap: var(--space-12); }
.ai-record-card { background: var(--color-bg-surface); border: 1px solid var(--color-divider); border-radius: var(--radius-lg); padding: var(--space-16); margin-bottom: var(--space-12); position: relative; }
.ai-record-header { display: flex; align-items: center; justify-content: space-between; margin-bottom: var(--space-12); }
.ai-record-type { display: flex; align-items: center; gap: var(--space-8); font-size: var(--font-size-body); font-weight: 500; color: var(--color-text-primary); }
.ai-record-type svg { width: 18px; height: 18px; color: var(--color-brand); }
.ai-record-delete { padding: var(--space-4) var(--space-8); font-size: var(--font-size-caption); color: var(--color-danger); border: 1px solid var(--color-danger-soft); border-radius: var(--radius-sm); cursor: pointer; }
.ai-fields { display: flex; flex-direction: column; gap: 4px; }
.ai-field { display: flex; justify-content: space-between; padding: var(--space-8) 0; border-bottom: 1px solid var(--color-divider); font-size: var(--font-size-assist); }
.ai-field:last-child { border-bottom: none; }
.ai-field-label { color: var(--color-text-tertiary); }
.ai-field-value { color: var(--color-text-primary); font-weight: 500; text-align: right; max-width: 60%; white-space: nowrap; overflow: hidden; text-overflow: ellipsis; }
.ai-field-value.confirmed { color: var(--color-success); }
.ai-field-value.unconfirmed { color: var(--color-warning); background: var(--color-warning-soft); padding: 2px 8px; border-radius: var(--radius-sm); }
.ai-field-note { font-size: 10px; color: var(--color-warning); }
.ai-disclaimer { font-size: var(--font-size-caption); color: var(--color-text-tertiary); margin-top: var(--space-16); }
.ai-actions { display: flex; gap: var(--space-12); margin-top: var(--space-16); }
.ai-actions .btn-primary { flex: 2; margin: 0; }
.ai-actions .btn-secondary { flex: 1; margin: 0; }
</style>
