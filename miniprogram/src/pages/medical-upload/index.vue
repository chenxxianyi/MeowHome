<script setup lang="ts">
import { ref } from 'vue'

import AppHeader from '../../components/app/AppHeader.vue'
import AppIcon from '../../components/app/AppIcon.vue'
import { useCatStore } from '../../stores/cat'
import { useProtectedPage } from '../../utils/page'

const catStore = useCatStore()
const images = ref<string[]>([])
const selectedCatId = ref('')

function selectImages() {
  uni.chooseMedia({
    count: 9,
    mediaType: ['image'],
    sourceType: ['album', 'camera'],
    success: (result) => {
      images.value = result.tempFiles.map((file) => file.tempFilePath)
    },
    fail: (error) => {
      if (!String(error.errMsg).includes('cancel')) {
        uni.showToast({ title: '无法读取图片，请检查相册权限', icon: 'none' })
      }
    }
  })
}

function explainUnavailable() {
  uni.showModal({
    title: '暂未开放上传',
    content:
      '当前后端还没有媒体上传与 OCR 接口。图片只在本机临时预览，不会上传或保存；可先用“AI 记录”输入检查结果文字。',
    showCancel: false
  })
}

useProtectedPage(() => {
  selectedCatId.value = catStore.currentCatId === 'all' ? catStore.cats[0]?.id || '' : catStore.currentCatId
})
</script>

<template>
  <AppHeader title="病历上传" subtitle="拍摄或选择检查单" />

  <view class="page-content medical-page">
    <view class="capability-notice">
      <AppIcon name="info" :size="19" />
      <view>
        <text>媒体上传与 OCR 暂未接入</text>
        <view>当前可选择图片做本地预览，但不会上传、识别或保存。</view>
      </view>
    </view>

    <view class="section-title">关联猫咪</view>
    <view class="cat-pick">
      <button
        v-for="cat in catStore.cats"
        :key="cat.id"
        class="cat-pick-btn"
        :class="{ active: selectedCatId === cat.id }"
        @click="selectedCatId = cat.id"
      >
        {{ cat.name }}
      </button>
    </view>

    <button class="upload-area" @click="selectImages">
      <AppIcon name="camera" :size="40" />
      <text>{{ images.length ? '重新选择图片' : '拍照或从相册选择' }}</text>
      <view class="upload-hint">支持微信可读取的图片格式，最多 9 张</view>
    </button>

    <scroll-view v-if="images.length" class="preview-list" scroll-x>
      <image
        v-for="(image, index) in images"
        :key="image"
        class="preview-image"
        :src="image"
        mode="aspectFill"
        :aria-label="`病历图片 ${index + 1}`"
      />
    </scroll-view>

    <button class="btn-primary disabled-action" :disabled="!images.length" @click="explainUnavailable">
      上传并识别（暂未开放）
    </button>

    <button class="text-action" @click="explainUnavailable">为什么暂时不能识别？</button>
  </view>
</template>

<style scoped>
.medical-page {
  padding-bottom: calc(24px + var(--safe-bottom));
}
.capability-notice {
  display: flex;
  gap: var(--space-12);
  padding: var(--space-12);
  margin-bottom: var(--space-20);
  color: var(--color-warning);
  background: var(--color-warning-soft);
  border-radius: var(--radius-md);
}
.capability-notice text {
  font-weight: 600;
}
.capability-notice view view {
  margin-top: 4px;
  color: var(--color-text-secondary);
  font-size: var(--font-size-assist);
  line-height: 1.5;
}
.cat-pick {
  display: flex;
  flex-wrap: wrap;
  gap: var(--space-8);
  margin: var(--space-8) 0 var(--space-16);
}
.cat-pick-btn {
  padding: var(--space-8) var(--space-16);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-pill);
  color: var(--color-text-secondary);
  background: var(--color-bg-subtle);
  font-size: var(--font-size-assist);
}
.cat-pick-btn.active {
  color: var(--color-brand);
  border-color: var(--color-brand);
  background: var(--color-brand-soft);
}
.upload-area {
  width: 100%;
  min-height: 180px;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: var(--space-8);
  padding: var(--space-24);
  color: var(--color-text-secondary);
  background: var(--color-bg-surface);
  border: 2px dashed var(--color-border);
  border-radius: var(--radius-md);
}
.upload-hint {
  color: var(--color-text-tertiary);
  font-size: var(--font-size-caption);
}
.preview-list {
  width: 100%;
  margin: var(--space-16) 0;
  white-space: nowrap;
}
.preview-image {
  width: 140px;
  height: 140px;
  margin-right: var(--space-8);
  border-radius: var(--radius-md);
}
.disabled-action {
  margin-top: var(--space-16);
}
.text-action {
  width: 100%;
  margin-top: var(--space-8);
  color: var(--color-brand);
  background: transparent;
  font-size: var(--font-size-assist);
}
</style>
