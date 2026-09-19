<script setup lang="ts">
import { ref, computed } from 'vue'
import { useRouter } from '../../utils/navigation'
import AppIcon from '../../components/app/AppIcon.vue'
import CatSwitcher from '../../components/cat/CatSwitcher.vue'
import OfflineBanner from '../../components/app/OfflineBanner.vue'
import { useAppStore } from '../../stores/app'
import { useCatStore } from '../../stores/cat'
import { services } from '../../services'
import { recordTypes } from '../../data/recordTypes'
import { usePageCapabilities, useProtectedPage } from '../../utils/page'
import type { RecordDTO } from '../../api/endpoints'

const router = useRouter()
const app = useAppStore()
const catStore = useCatStore()

// 从 AI 确认页“返回修改”回来时，保留上次输入以便继续编辑
const aiInput = ref(app.aiInput || '')
const aiLoading = ref(false)
const aiError = ref('')
const records = ref<RecordDTO[]>([])
const recordsLoading = ref(true)

const recordGroups = computed(() => {
  const groups = new Map<string, RecordDTO[]>()
  for (const record of records.value) {
    const date = (record.occurred_at || '').slice(0, 10) || '未知日期'
    const items = groups.get(date) || []
    items.push(record)
    groups.set(date, items)
  }
  return Array.from(groups.entries())
    .sort(([a], [b]) => b.localeCompare(a))
    .map(([date, items]) => ({ date, items }))
})

function recordLabel(type: string) {
  const all = [...recordTypes.high, ...recordTypes.health, ...recordTypes.life]
  return all.find((item) => item.id === type)?.label || type
}

async function loadRecords() {
  recordsLoading.value = true
  const result = await services.getRecords(catStore.currentCatId)
  records.value = [...result.data].sort((a, b) => b.occurred_at.localeCompare(a.occurred_at))
  recordsLoading.value = false
}

function onCatSelect() {
  loadRecords()
}

const aiExamples = ['小白早上吐了一次黄色的水', '小橘晚饭吃了半罐，喝水正常']

function useAIExample(example: string) {
  aiInput.value = example
  aiError.value = ''
}

const recording = ref(false)
const micError = ref('')

function toggleVoiceInput() {
  recording.value = false
  micError.value = '当前版本尚未接入语音转文字，请先使用文字输入'
  uni.showToast({ title: micError.value, icon: 'none' })
}

// 常用（高频）记录：主路径，首屏大卡片
const commonTypes = computed(() => recordTypes.high)

// 其余分类弱化为两个分组
const moreGroups = computed(() => [
  { key: 'health', title: '健康', items: recordTypes.health },
  { key: 'life', title: '生活', items: recordTypes.life }
])

function selectType(type: string) {
  router.push(`/records/quick/${type}`)
}

async function parseAI() {
  if (!aiInput.value.trim() || aiLoading.value) return
  aiLoading.value = true
  aiError.value = ''
  app.setAIState(aiInput.value, null, true)
  try {
    const res = await services.parseAI(aiInput.value)
    if (res.success) {
      app.setAIState(aiInput.value, res.data, false)
      aiLoading.value = false
      router.push('/records/ai/confirm')
    } else {
      aiError.value = '没有识别到有效内容，请补充描述后重试'
      app.setAIState(aiInput.value, null, false)
      aiLoading.value = false
    }
  } catch {
    aiError.value = 'AI 服务暂时不可用，可先使用下方分类手动记录'
    app.setAIState(aiInput.value, null, false)
    aiLoading.value = false
  }
}

useProtectedPage(loadRecords)
usePageCapabilities(loadRecords, '猫宅 · 记录猫咪的每一天')
</script>

<template>
  <OfflineBanner />
  <view class="page-header records-header">
    <view class="page-title"> 记录 </view>
    <CatSwitcher @select="onCatSelect" />
  </view>

  <view class="page-content">
    <!-- 常用记录：主路径，首屏直达 -->
    <view class="records-hero" aria-label="常用记录">
      <view class="records-section-title"> 常用 </view>
      <view class="record-hero-grid">
        <button v-for="item in commonTypes" :key="item.id" class="record-hero-btn" @click="selectType(item.id)">
          <AppIcon :name="item.icon" :size="26" />
          <text>{{ item.label }}</text>
        </button>
      </view>
    </view>

    <!-- AI 管家：自然语言优先的记录工作台 -->
    <view class="ai-assistant" aria-label="AI 管家">
      <view class="ai-assistant-head">
        <view class="ai-assistant-mark">
          <AppIcon name="ai" :size="22" />
        </view>
        <view class="ai-assistant-heading">
          <text class="ai-assistant-kicker">AI 智能记录</text>
          <view>说一说，管家帮你记</view>
        </view>
      </view>

      <view class="ai-assistant-subtitle"> 不用先选分类，直接描述发生了什么，AI 会提取时间、事件和数值。 </view>

      <view class="ai-example-list" aria-label="输入示例">
        <text class="ai-example-label">试试这样说</text>
        <button
          v-for="example in aiExamples"
          :key="example"
          type="button"
          class="ai-example-chip"
          @click="useAIExample(example)"
        >
          {{ example }}
        </button>
      </view>

      <view class="ai-composer" :class="{ 'is-recording': recording }">
        <textarea
          v-model="aiInput"
          class="ai-composer-input"
          rows="3"
          maxlength="300"
          :placeholder="recording ? '正在聆听，请说话…' : '例如：小白今天早上没怎么吃，下午吐了一次黄色的水…'"
          aria-label="AI 自然语言输入"
          @keydown.ctrl.enter.prevent="parseAI"
          @keydown.meta.enter.prevent="parseAI"
        />
        <view class="ai-composer-footer">
          <text class="ai-composer-hint">Ctrl / ⌘ + Enter 发送</text>
          <view class="ai-composer-actions">
            <button
              type="button"
              class="ai-voice-btn"
              :class="{ recording }"
              :aria-label="recording ? '停止语音输入' : '语音输入'"
              :aria-pressed="recording"
              @click="toggleVoiceInput"
            >
              <AppIcon name="mic" :size="18" />
              <text>{{ recording ? '停止' : '语音' }}</text>
            </button>
            <button
              type="button"
              class="ai-submit-btn"
              :class="{ 'is-loading': aiLoading }"
              :disabled="!aiInput.trim() || aiLoading"
              @click="parseAI"
            >
              <AppIcon :name="aiLoading ? 'loading' : 'ai'" :size="17" />
              {{ aiLoading ? '正在整理…' : '智能整理' }}
            </button>
          </view>
        </view>
      </view>
      <view v-if="micError" class="ai-assistant-error" aria-live="polite">
        {{ micError }}
      </view>
      <view v-if="aiError" class="ai-assistant-error" aria-live="polite">
        {{ aiError }}
      </view>
      <button type="button" class="ai-medical-entry" @click="router.push('/medical/upload')">
        <view class="ai-medical-icon">
          <AppIcon name="medical" :size="19" />
        </view>
        <view class="ai-medical-copy">
          <text>导入病历或检查单</text>
          <view>拍照扫描，自动提取检查信息</view>
        </view>
        <AppIcon class="ai-medical-arrow" name="chevronRight" :size="18" />
      </button>
    </view>

    <!-- 健康 / 生活：其余分类 -->
    <view v-for="group in moreGroups" :key="group.key" class="record-type-section">
      <view class="record-type-group-title">
        {{ group.title }}
      </view>
      <view class="record-type-grid">
        <button v-for="item in group.items" :key="item.id" class="record-type-btn" @click="selectType(item.id)">
          <AppIcon :name="item.icon" :size="20" />
          <text>{{ item.label }}</text>
        </button>
      </view>
    </view>

    <view class="records-history">
      <view class="records-section-title">最近记录</view>
      <view v-if="recordsLoading" class="skeleton" style="height: 96px" />
      <view v-else-if="!recordGroups.length" class="empty-state">还没有记录，从上方选择一种开始吧。</view>
      <view v-for="group in recordGroups" v-else :key="group.date" class="records-day-group">
        <view class="records-day-title">{{ group.date }}</view>
        <view v-for="item in group.items" :key="item.id" class="records-history-item">
          <AppIcon :name="recordTypes.high.find((type) => type.id === item.type)?.icon || 'record'" :size="18" />
          <view class="records-history-copy">
            <text>{{ item.title || recordLabel(item.type) }}</text>
            <view>{{ item.note || recordLabel(item.type) }} · {{ item.occurred_at.slice(11, 16) }}</view>
          </view>
        </view>
      </view>
    </view>
  </view>
</template>

<style scoped>
.records-history {
  margin-top: var(--space-24);
}
.records-day-title {
  margin: var(--space-16) 0 var(--space-8);
  color: var(--color-text-tertiary);
  font-size: var(--font-size-assist);
}
.records-history-item {
  display: flex;
  gap: var(--space-12);
  align-items: center;
  padding: var(--space-12);
  border-bottom: 1px solid var(--color-divider);
  background: var(--color-bg-surface);
}
.records-history-copy {
  flex: 1;
}
.records-history-copy > text {
  color: var(--color-text-primary);
  font-weight: 600;
}
.records-history-copy > view {
  margin-top: 3px;
  color: var(--color-text-tertiary);
  font-size: var(--font-size-caption);
}
.empty-state {
  padding: var(--space-24);
  text-align: center;
  color: var(--color-text-tertiary);
}
</style>
