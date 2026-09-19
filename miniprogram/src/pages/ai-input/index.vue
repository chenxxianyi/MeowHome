<script setup lang="ts">
import { ref } from 'vue'
import { useRouter } from '../../utils/navigation'
import AppHeader from '../../components/app/AppHeader.vue'
import AppIcon from '../../components/app/AppIcon.vue'
import AIResultBadge from '../../components/ai/AIResultBadge.vue'
import { useAppStore } from '../../stores/app'
import { useCatStore } from '../../stores/cat'
import { services } from '../../services'
import { useProtectedPage } from '../../utils/page'

const app = useAppStore()
const router = useRouter()
const catStore = useCatStore()
const input = ref(app.aiInput || '')
const loading = ref(false)
const result = ref<any>(null)
const error = ref('')

async function handleParse() {
  if (!input.value.trim()) return
  loading.value = true
  error.value = ''
  app.setAIState(input.value, null, true)
  try {
    const res = await services.parseAI(input.value)
    if (!res.success || !res.data.records?.length) {
      error.value = '没有识别到有效记录，请补充时间、猫咪和事件后重试'
      app.setAIState(input.value, null, false)
      return
    }
    result.value = res.data
    app.setAIState(input.value, res.data, false)
  } catch {
    error.value = 'AI 服务暂时不可用，请稍后重试'
    app.setAIState(input.value, null, false)
  } finally {
    loading.value = false
  }
}

function unavailable() {
  uni.showToast({ title: '后端尚未接入媒体上传，请先使用文字描述', icon: 'none' })
}

function catName(id?: string) {
  return catStore.cats.find((cat) => cat.id === id)?.name || '未指定猫咪'
}

useProtectedPage()
</script>

<template>
  <AppHeader title="AI 记录" />

  <view class="page-content">
    <view class="ai-input-page">
      <view class="form-group">
        <label class="form-label" for="ai-natural-input">告诉猫宅管家，猫咪今天发生了什么…</label>
        <textarea
          id="ai-natural-input"
          v-model="input"
          class="form-input"
          rows="4"
          placeholder="例如：小白今天早上没怎么吃，下午吐了一次黄色的水，小橘晚上要记得喂药…"
        />
      </view>

      <view class="ai-input-actions">
        <button class="ai-action-btn" aria-label="拍照" @click="unavailable">
          <AppIcon name="camera" :size="20" />
        </button>
        <button class="ai-action-btn" aria-label="上传图片" @click="unavailable">
          <AppIcon name="upload" :size="20" />
        </button>
        <button class="ai-action-btn" aria-label="病历扫描" @click="router.push('/medical/upload')">
          <AppIcon name="medical" :size="20" />
        </button>
        <button class="btn-primary ai-submit" :disabled="!input.trim() || loading" @click="handleParse">
          <AppIcon name="ai" :size="16" /> {{ loading ? '解析中…' : '交给管家整理' }}
        </button>
      </view>

      <view v-if="error" class="ai-assistant-error">{{ error }}</view>

      <!-- Result -->
      <view v-if="result" class="ai-result-page">
        <view class="ai-result-header">
          <AIResultBadge :source="result.model || 'AI 整理'" />
          <text style="font-size: var(--font-size-caption); color: var(--color-text-tertiary)">{{
            result.parsedAt?.slice(11, 16)
          }}</text>
        </view>
        <view class="ai-result-list">
          <view v-for="rec in result.records" :key="rec.id" class="ai-result-card">
            <view class="ai-result-card-title">
              {{ rec.type === 'feeding' ? '喂食' : rec.type === 'vomit' ? '呕吐' : '用药' }}
              <text class="cat-tag">{{ catName(rec.catId) }}</text>
            </view>
            <view class="ai-field-list">
              <view v-for="f in rec.fields" :key="f.key" class="ai-field-row">
                <text class="ai-field-key">{{ f.key }}</text>
                <text
                  class="ai-field-val"
                  :class="f.confidence === 'high' ? 'confirmed' : f.confidence === 'low' ? 'unconfirmed' : ''"
                >
                  {{ f.value || '—' }}
                  <text v-if="f.note" class="ai-field-note">{{ f.note }}</text>
                </text>
              </view>
            </view>
          </view>
        </view>
        <view class="ai-disclaimer"> AI 整理 · 不构成医疗诊断 · 请确认后保存 </view>
        <view class="form-actions-onboarding">
          <button class="btn-secondary" @click="router.push('/records')">返回修改</button>
          <button class="btn-primary" @click="router.push('/records/ai/confirm')">
            确认并保存 {{ result.records.length }} 条
          </button>
        </view>
      </view>
    </view>
  </view>
</template>

<style scoped>
.ai-input-page {
  padding-bottom: 80px;
}
.ai-input-actions {
  display: flex;
  gap: var(--space-8);
  margin-top: var(--space-12);
  align-items: center;
  flex-wrap: wrap;
}
.ai-action-btn {
  width: 44px;
  height: 44px;
  border-radius: var(--radius-sm);
  background: var(--color-bg-subtle);
  display: flex;
  align-items: center;
  justify-content: center;
  color: var(--color-text-secondary);
  cursor: pointer;
}
.ai-submit {
  margin-left: auto;
  width: auto;
  min-height: 44px;
  padding: 0 var(--space-20);
  font-size: var(--font-size-body);
  display: flex;
  align-items: center;
  gap: 6px;
}
.ai-result-page {
  margin-top: var(--space-16);
}
.ai-result-header {
  display: flex;
  align-items: center;
  gap: var(--space-8);
  margin-bottom: var(--space-12);
}
.ai-result-list {
  display: flex;
  flex-direction: column;
  gap: var(--space-12);
}
.ai-result-card {
  background: var(--color-bg-surface);
  border: 1px solid var(--color-divider);
  border-radius: var(--radius-md);
  padding: var(--space-12);
}
.ai-result-card-title {
  font-weight: 600;
  font-size: var(--font-size-body);
  display: flex;
  align-items: center;
  gap: 6px;
  margin-bottom: var(--space-8);
}
.cat-tag {
  font-size: var(--font-size-caption);
  color: var(--color-text-tertiary);
}
.ai-field-list {
  display: flex;
  flex-direction: column;
  gap: 4px;
}
.ai-field-row {
  display: flex;
  justify-content: space-between;
  font-size: var(--font-size-assist);
}
.ai-field-key {
  color: var(--color-text-tertiary);
}
.ai-field-val {
  font-weight: 500;
}
.ai-field-val.confirmed {
  color: var(--color-success);
}
.ai-field-val.unconfirmed {
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
.form-actions-onboarding {
  display: flex;
  gap: var(--space-12);
  margin-top: var(--space-16);
}
.form-actions-onboarding .btn-primary {
  flex: 2;
}
.form-actions-onboarding .btn-secondary {
  flex: 1;
}
</style>
