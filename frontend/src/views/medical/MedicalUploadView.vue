<script setup lang="ts">
import { ref } from 'vue'
import AppShell from '../../components/app/AppShell.vue'
import AppHeader from '../../components/app/AppHeader.vue'
import AppIcon from '../../components/app/AppIcon.vue'
import { useCatStore } from '../../stores/cat'
import { services } from '../../services'

const catStore = useCatStore()
const files = ref<File[]>([])
const images = ref<string[]>([])
const ocrText = ref('')
const aiResult = ref('')
const uploading = ref(false)
const step = ref<'upload' | 'ocr' | 'review' | 'confirm'>('upload')

function selectFile(e: Event) {
  const input = e.target as HTMLInputElement
  files.value = Array.from(input.files || [])
  images.value = files.value.map(f => URL.createObjectURL(f))
}

async function simulateOCR() {
  step.value = 'ocr'
  uploading.value = true
  await new Promise(r => setTimeout(r, 1200))
  ocrText.value = '血常规检查\n红细胞计数：5.2×10^12/L\n白细胞计数：12.3×10^9/L (参考值：5.5-19.5)\n血小板计数：280×10^9/L'
  uploading.value = false
  step.value = 'review'
}

async function simulateAI() {
  step.value = 'confirm'
  uploading.value = true
  await new Promise(r => setTimeout(r, 800))
  aiResult.value = '检查结果：血常规正常，各项指标均在参考范围内。'
  uploading.value = false
}

function confirmSave() {
  step.value = 'ocr'
  // In real app, save to server
  setTimeout(() => {
    const toast = document.createElement('div')
    toast.className = 'toast show success'
    toast.textContent = '病历已保存'
    document.body.appendChild(toast)
    setTimeout(() => toast.remove(), 2000)
  }, 300)
}
</script>

<template>
  <AppShell>
    <AppHeader title="病历上传" />

    <div class="page-content">
      <!-- Step indicator -->
      <div
        class="step-bar"
        role="progressbar"
        aria-label="处理进度"
      >
        <div
          class="step"
          :class="{ active: step === 'upload' || step === 'ocr' || step === 'review' || step === 'confirm' }"
        >
          1. 上传
        </div>
        <div
          class="step"
          :class="{ active: step === 'ocr' || step === 'review' || step === 'confirm' }"
        >
          2. OCR
        </div>
        <div
          class="step"
          :class="{ active: step === 'review' || step === 'confirm' }"
        >
          3. AI整理
        </div>
        <div
          class="step"
          :class="{ active: step === 'confirm' }"
        >
          4. 确认
        </div>
      </div>

      <!-- Upload -->
      <section
        v-if="step === 'upload'"
        class="upload-section"
      >
        <label class="upload-label">上传或拍摄病历图片</label>
        <div
          class="upload-area"
          :class="{ dragging: false }"
        >
          <input
            type="file"
            accept="image/*"
            capture="environment"
            class="upload-input"
            aria-label="上传图片"
            @change="selectFile"
          />
          <div
            v-if="images.length === 0"
            class="upload-placeholder"
          >
            <AppIcon
              name="camera"
              :size="40"
            />
            <div>点击拍照或上传</div>
            <div class="upload-hint">
              支持 JPG、PNG、PDF
            </div>
          </div>
          <div
            v-for="(img, i) in images"
            :key="i"
            class="upload-preview"
          >
            <img
              :src="img"
              :alt="'病历图片 ' + (i+1)"
              class="preview-img"
            />
          </div>
        </div>
        <button
          v-if="images.length"
          class="btn-primary"
          style="margin-top:16px;"
          @click="simulateOCR"
        >
          开始识别
        </button>
      </section>

      <!-- OCR result -->
      <section
        v-if="step === 'ocr' || step === 'review' || step === 'confirm'"
        class="ocr-section"
      >
        <div class="section-title">
          OCR 原文
        </div>
        <div
          class="ocr-text"
          aria-label="OCR 识别原文"
        >
          <pre style="white-space:pre-wrap;font-family:inherit;font-size:var(--font-size-assist);color:var(--color-text-secondary);">{{ ocrText || '识别中…' }}</pre>
        </div>
        <button
          v-if="step === 'ocr'"
          class="btn-primary"
          style="margin-top:12px;"
          @click="simulateAI"
        >
          <AppIcon
            name="ai"
            :size="16"
          /> AI 整理
        </button>
      </section>

      <!-- AI result -->
      <section
        v-if="step === 'confirm'"
        class="ai-result-section"
      >
        <div class="section-title">
          AI 整理结果
        </div>
        <div class="ai-result-box">
          <AIResultBadge />
          <div class="ai-result-body">
            {{ aiResult }}
          </div>
          <div class="ai-disclaimer">
            AI 整理 · 不构成医疗诊断
          </div>
        </div>
        <div style="margin-top:16px;">
          <div class="section-title">
            关联猫咪
          </div>
          <div class="cat-pick">
            <button
              v-for="cat in catStore.cats"
              :key="cat.id"
              class="cat-pick-btn"
              :class="{ active: true }"
              @click="catStore.setCat(cat.id)"
            >
              {{ cat.name }}
            </button>
          </div>
        </div>
        <div class="form-actions">
          <button
            class="btn-secondary"
            @click="step = 'upload'"
          >
            返回修改
          </button>
          <button
            class="btn-primary"
            @click="confirmSave"
          >
            确认保存
          </button>
        </div>
      </section>

      <!-- Loading overlay -->
      <div
        v-if="uploading"
        class="loading-overlay"
        aria-live="polite"
      >
        <div class="spinner" />
        <div>{{ step === 'ocr' ? 'OCR 识别中…' : 'AI 整理中…' }}</div>
      </div>
    </div>
  </AppShell>
</template>

<style scoped>
.step-bar { display: flex; gap: 4px; margin-bottom: var(--space-16); }
.step { flex: 1; padding: var(--space-8); text-align: center; font-size: var(--font-size-caption); border-radius: var(--radius-sm); background: var(--color-bg-subtle); color: var(--color-text-tertiary); }
.step.active { background: var(--color-brand-soft); color: var(--color-brand); }
.upload-area { border: 2px dashed var(--color-border); border-radius: var(--radius-md); padding: var(--space-24); text-align: center; margin-top: var(--space-12); position: relative; }
.upload-input { position: absolute; inset: 0; opacity: 0; cursor: pointer; }
.upload-placeholder { display: flex; flex-direction: column; align-items: center; gap: var(--space-8); color: var(--color-text-tertiary); }
.upload-hint { font-size: var(--font-size-caption); }
.upload-preview { margin-top: var(--space-8); }
.preview-img { width: 100%; max-width: 200px; border-radius: var(--radius-md); border: 1px solid var(--color-divider); }
.ocr-text { background: var(--color-bg-subtle); padding: var(--space-12); border-radius: var(--radius-sm); overflow-x: auto; }
.ai-result-box { background: var(--color-ai-bg); border: 1px solid var(--color-ai-border); border-radius: var(--radius-md); padding: var(--space-12); }
.ai-result-body { margin-top: var(--space-8); font-size: var(--font-size-body); color: var(--color-text-secondary); }
.ai-result-section { margin-top: var(--space-16); }
.cat-pick { display: flex; gap: var(--space-8); margin-top: var(--space-8); }
.cat-pick-btn { padding: var(--space-8) var(--space-16); border-radius: var(--radius-pill); background: var(--color-bg-subtle); border: 1px solid var(--color-border); font-size: var(--font-size-assist); cursor: pointer; }
.cat-pick-btn.active { background: var(--color-brand-soft); border-color: var(--color-brand); color: var(--color-brand); }
.form-actions { display: flex; gap: var(--space-12); margin-top: var(--space-16); }
.form-actions .btn-primary { flex: 2; }
.form-actions .btn-secondary { flex: 1; }
.loading-overlay { position: fixed; inset: 0; background: rgba(247,244,238,0.9); display: flex; flex-direction: column; align-items: center; justify-content: center; gap: var(--space-16); z-index: 200; }
.spinner { width: 40px; height: 40px; border: 3px solid var(--color-border); border-top-color: var(--color-brand); border-radius: 50%; animation: spin 0.8s linear infinite; }
@keyframes spin { to { transform: rotate(360deg); } }
</style>
