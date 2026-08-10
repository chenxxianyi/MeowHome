<script setup lang="ts">
import { ref, computed, onBeforeUnmount } from 'vue'
import { useRouter } from 'vue-router'
import AppShell from '../../components/app/AppShell.vue'
import AppIcon from '../../components/app/AppIcon.vue'
import CatSwitcher from '../../components/cat/CatSwitcher.vue'
import { useAppStore } from '../../stores/app'
import { services } from '../../services'
import { mockRecordTypes } from '../../mocks/data'

const router = useRouter()
const app = useAppStore()

// 从 AI 确认页“返回修改”回来时，保留上次输入以便继续编辑
const aiInput = ref(app.aiInput || '')
const aiLoading = ref(false)
const aiError = ref('')

// —— 语音输入（Web Speech API，不支持时降级为提示）——
interface SpeechRecognitionEventLike {
  results: {
    length: number
    [index: number]: {
      isFinal: boolean
      length: number
      [index: number]: { transcript: string }
    }
  }
}
interface SpeechRecognitionErrorEventLike {
  error: string
}
interface SpeechRecognitionLike {
  lang: string
  interimResults: boolean
  continuous: boolean
  onresult: ((event: SpeechRecognitionEventLike) => void) | null
  onend: (() => void) | null
  onerror: ((event: SpeechRecognitionErrorEventLike) => void) | null
  start(): void
  stop(): void
  abort(): void
}
type SpeechRecognitionCtorLike = new () => SpeechRecognitionLike

const win = window as {
  SpeechRecognition?: SpeechRecognitionCtorLike
  webkitSpeechRecognition?: SpeechRecognitionCtorLike
}
const recognitionCtor = win.SpeechRecognition || win.webkitSpeechRecognition

const recording = ref(false)
const recognition = ref<SpeechRecognitionLike | null>(null)
const micError = ref('')

function toggleVoiceInput() {
  micError.value = ''
  if (recording.value) {
    recognition.value?.stop()
    return
  }
  if (!recognitionCtor) {
    micError.value = '当前浏览器不支持语音输入，请使用 Chrome / Edge，或直接手动输入'
    return
  }
  const rec = new recognitionCtor()
  rec.lang = 'zh-CN'
  rec.interimResults = true
  rec.continuous = false
  // 录音开始时保留输入框已有内容作为前缀
  const baseText = aiInput.value.trim()
  rec.onresult = (event) => {
    let finalText = ''
    let interimText = ''
    for (let i = 0; i < event.results.length; i++) {
      const result = event.results[i]
      const transcript = result[0]?.transcript ?? ''
      if (result.isFinal) finalText += transcript
      else interimText += transcript
    }
    aiInput.value = [baseText, finalText, interimText].filter(Boolean).join(' ')
  }
  rec.onend = () => {
    if (recognition.value !== rec) return
    recording.value = false
  }
  rec.onerror = (event) => {
    if (recognition.value !== rec) return
    recording.value = false
    if (event.error === 'not-allowed' || event.error === 'service-not-allowed') {
      micError.value = '麦克风权限被拒绝，请在浏览器设置中允许后重试'
    } else if (event.error === 'no-speech') {
      micError.value = '没有听到声音，请靠近麦克风再试'
    } else if (event.error === 'network') {
      micError.value = '语音识别服务连接失败，请检查网络后重试'
    } else {
      micError.value = '语音识别出错，请重试'
    }
  }
  recognition.value = rec
  rec.start()
  recording.value = true
}

onBeforeUnmount(() => {
  recognition.value?.abort()
})

// 常用（高频）记录：主路径，首屏大卡片
const commonTypes = computed(() => mockRecordTypes.high)

// 其余分类弱化为两个分组
const moreGroups = computed(() => [
  { key: 'health', title: '健康', items: mockRecordTypes.health },
  { key: 'life', title: '生活', items: mockRecordTypes.life }
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
</script>

<template>
  <AppShell>
    <div class="page-header records-header">
      <div class="page-title">
        记录
      </div>
      <CatSwitcher />
    </div>

    <div class="page-content">
      <!-- 常用记录：主路径，首屏直达 -->
      <section
        class="records-hero"
        aria-label="常用记录"
      >
        <div class="records-section-title">
          常用
        </div>
        <div class="record-hero-grid">
          <button
            v-for="item in commonTypes"
            :key="item.id"
            class="record-hero-btn"
            @click="selectType(item.id)"
          >
            <AppIcon
              :name="item.icon"
              :size="26"
            />
            <span>{{ item.label }}</span>
          </button>
        </div>
      </section>

      <!-- AI 管家：紧凑卡片，说句话即可记录 -->
      <section
        class="ai-card"
        aria-label="AI 管家"
      >
        <div class="ai-card-head">
          <AppIcon
            name="ai"
            :size="20"
          />
          <div>
            <div class="ai-card-title">
              AI 管家
            </div>
            <div class="ai-card-sub">
              说句话，自动整理成结构化记录
            </div>
          </div>
        </div>
        <div class="ai-card-input-row">
          <textarea
            v-model="aiInput"
            class="ai-card-input"
            rows="2"
            :placeholder="recording ? '正在聆听，请说话…' : '例如：小白早上吐了一次黄色的水…'"
            aria-label="AI 自然语言输入"
            @keydown.enter.exact.prevent="parseAI"
          />
          <button
            class="ai-card-mic"
            :class="{ recording }"
            :aria-label="recording ? '停止语音输入' : '语音输入'"
            :aria-pressed="recording"
            @click="toggleVoiceInput"
          >
            <AppIcon
              name="mic"
              :size="20"
            />
          </button>
          <button
            class="btn-primary ai-card-submit"
            :disabled="!aiInput.trim() || aiLoading"
            @click="parseAI"
          >
            <AppIcon
              name="ai"
              :size="16"
            /> {{ aiLoading ? '解析中…' : '交给管家整理' }}
          </button>
        </div>
        <p
          v-if="micError"
          class="ai-card-error"
          aria-live="polite"
        >
          {{ micError }}
        </p>
        <p
          v-if="aiError"
          class="ai-card-error"
          aria-live="polite"
        >
          {{ aiError }}
        </p>
        <button
          class="ai-card-scan"
          @click="router.push('/medical/upload')"
        >
          <AppIcon
            name="medical"
            :size="16"
          /> 病历扫描 · 自动提取检查信息
        </button>
      </section>

      <!-- 健康 / 生活：其余分类 -->
      <section
        v-for="group in moreGroups"
        :key="group.key"
        class="record-type-section"
      >
        <div class="record-type-group-title">
          {{ group.title }}
        </div>
        <div class="record-type-grid">
          <button
            v-for="item in group.items"
            :key="item.id"
            class="record-type-btn"
            @click="selectType(item.id)"
          >
            <AppIcon
              :name="item.icon"
              :size="20"
            />
            <span>{{ item.label }}</span>
          </button>
        </div>
      </section>
    </div>
  </AppShell>
</template>
