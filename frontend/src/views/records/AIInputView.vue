<script setup lang="ts">
import { ref } from 'vue'
import { useRouter } from 'vue-router'
import AppShell from '../../components/app/AppShell.vue'
import AppHeader from '../../components/app/AppHeader.vue'
import AppIcon from '../../components/app/AppIcon.vue'
import AIResultBadge from '../../components/ai/AIResultBadge.vue'
import { useAppStore } from '../../stores/app'
import { services } from '../../services'

const app = useAppStore()
const router = useRouter()
const input = ref(app.aiInput || '')
const loading = ref(false)
const result = ref<any>(null)

async function handleParse() {
  if (!input.value.trim()) return
  loading.value = true
  app.setAIState(input.value, null, true)
  try {
    const res = await services.parseAI(input.value)
    result.value = res.data
    app.setAIState(input.value, res.data, false)
  } catch {
    loading.value = false
    app.setAIState(input.value, null, false)
  }
}
</script>

<template>
  <AppShell>
    <AppHeader title="AI 记录" />

    <div class="page-content">
      <div class="ai-input-page">
        <div class="form-group">
          <label
            class="form-label"
            for="ai-natural-input"
          >告诉猫宅管家，猫咪今天发生了什么…</label>
          <textarea
            id="ai-natural-input"
            v-model="input"
            class="form-input"
            rows="4"
            placeholder="例如：小白今天早上没怎么吃，下午吐了一次黄色的水，小橘晚上要记得喂药…"
          />
        </div>

        <div class="ai-input-actions">
          <button
            class="ai-action-btn"
            aria-label="拍照"
          >
            <AppIcon
              name="camera"
              :size="20"
            />
          </button>
          <button
            class="ai-action-btn"
            aria-label="上传图片"
          >
            <AppIcon
              name="upload"
              :size="20"
            />
          </button>
          <button
            class="ai-action-btn"
            aria-label="病历扫描"
          >
            <AppIcon
              name="medical"
              :size="20"
            />
          </button>
          <button
            class="btn-primary ai-submit"
            :disabled="!input.trim() || loading"
            @click="handleParse"
          >
            <AppIcon
              name="ai"
              :size="16"
            /> {{ loading ? '解析中…' : '交给管家整理' }}
          </button>
        </div>

        <!-- Result -->
        <div
          v-if="result"
          class="ai-result-page"
        >
          <div class="ai-result-header">
            <AIResultBadge source="mock-ai-v1" />
            <span style="font-size:var(--font-size-caption);color:var(--color-text-tertiary);">{{ result.parsedAt?.slice(11, 16) }}</span>
          </div>
          <div class="ai-result-list">
            <div
              v-for="rec in result.records"
              :key="rec.id"
              class="ai-result-card"
            >
              <div class="ai-result-card-title">
                {{ rec.type === 'feeding' ? '喂食' : rec.type === 'vomit' ? '呕吐' : '用药' }}
                <span class="cat-tag">{{ rec.catId === 'cat-whit' ? '小白' : rec.catId === 'cat-oran' ? '小橘' : '未知' }}</span>
              </div>
              <div class="ai-field-list">
                <div
                  v-for="f in rec.fields"
                  :key="f.key"
                  class="ai-field-row"
                >
                  <span class="ai-field-key">{{ f.key }}</span>
                  <span
                    class="ai-field-val"
                    :class="f.confidence === 'high' ? 'confirmed' : f.confidence === 'low' ? 'unconfirmed' : ''"
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
          <div class="form-actions-onboarding">
            <button
              class="btn-secondary"
              @click="router.push('/records')"
            >
              返回修改
            </button>
            <button
              class="btn-primary"
              @click="router.push('/records/ai/confirm')"
            >
              确认并保存 {{ result.records.length }} 条
            </button>
          </div>
        </div>
      </div>
    </div>
  </AppShell>
</template>

<style scoped>
.ai-input-page { padding-bottom: 80px; }
.ai-input-actions { display: flex; gap: var(--space-8); margin-top: var(--space-12); align-items: center; flex-wrap: wrap; }
.ai-action-btn { width: 44px; height: 44px; border-radius: var(--radius-sm); background: var(--color-bg-subtle); display: flex; align-items: center; justify-content: center; color: var(--color-text-secondary); cursor: pointer; }
.ai-submit { margin-left: auto; width: auto; min-height: 44px; padding: 0 var(--space-20); font-size: var(--font-size-body); display: flex; align-items: center; gap: 6px; }
.ai-result-page { margin-top: var(--space-16); }
.ai-result-header { display: flex; align-items: center; gap: var(--space-8); margin-bottom: var(--space-12); }
.ai-result-list { display: flex; flex-direction: column; gap: var(--space-12); }
.ai-result-card { background: var(--color-bg-surface); border: 1px solid var(--color-divider); border-radius: var(--radius-md); padding: var(--space-12); }
.ai-result-card-title { font-weight: 600; font-size: var(--font-size-body); display: flex; align-items: center; gap: 6px; margin-bottom: var(--space-8); }
.cat-tag { font-size: var(--font-size-caption); color: var(--color-text-tertiary); }
.ai-field-list { display: flex; flex-direction: column; gap: 4px; }
.ai-field-row { display: flex; justify-content: space-between; font-size: var(--font-size-assist); }
.ai-field-key { color: var(--color-text-tertiary); }
.ai-field-val { font-weight: 500; }
.ai-field-val.confirmed { color: var(--color-success); }
.ai-field-val.unconfirmed { color: var(--color-warning); }
.ai-field-note { font-size: 10px; color: var(--color-warning); }
.ai-disclaimer { font-size: var(--font-size-caption); color: var(--color-text-tertiary); margin-top: var(--space-16); }
.form-actions-onboarding { display: flex; gap: var(--space-12); margin-top: var(--space-16); }
.form-actions-onboarding .btn-primary { flex: 2; }
.form-actions-onboarding .btn-secondary { flex: 1; }
</style>
