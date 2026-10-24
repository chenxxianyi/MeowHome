<script setup lang="ts">
import { computed, ref } from 'vue'
import AppIcon from '../../components/app/AppIcon.vue'
import { toApiError } from '../../api/adapter'
import { services } from '../../services'
import { useRouter } from '../../utils/navigation'

const router = useRouter()

/** 已选择的本地截图路径（uni.chooseImage 返回的临时路径）。 */
const images = ref<string[]>([])
const busy = ref(false)
const stage = ref('')
const error = ref('')

const canSubmit = computed(() => images.value.length > 0 && !busy.value)

function chooseImages() {
  if (busy.value) return
  const remain = 5 - images.value.length
  if (remain <= 0) {
    uni.showToast({ title: '单次最多 5 张截图', icon: 'none' })
    return
  }
  uni.chooseImage({
    count: remain,
    sizeType: ['compressed'],
    sourceType: ['album', 'camera'],
    success: (res) => {
      const picked = (res.tempFilePaths || []).slice(0, remain)
      images.value = [...images.value, ...picked].slice(0, 5)
    }
  })
}

function removeImage(index: number) {
  if (busy.value) return
  images.value = images.value.filter((_, i) => i !== index)
}

async function submit() {
  if (!canSubmit.value) return
  busy.value = true
  error.value = ''
  try {
    const keys: string[] = []
    for (let i = 0; i < images.value.length; i += 1) {
      stage.value = `正在上传第 ${i + 1}/${images.value.length} 张截图…`
      keys.push(await services.uploadPurchaseMedia(images.value[i] ?? ''))
    }
    stage.value = '正在创建订单草稿…'
    const created = await services.createPurchaseImport(keys)
    stage.value = '正在识别订单内容…'
    // 识别失败也进入复核页：导入记录保留，可手动填写后确认。
    await services.recognizePurchaseImport(created.id)
    router.push(`/inventory/purchase/review/${encodeURIComponent(created.id)}`)
  } catch (e) {
    error.value = toApiError(e).message || '上传失败，请稍后重试'
    stage.value = ''
  } finally {
    busy.value = false
  }
}
</script>

<template>
  <view class="page-header">
    <view class="page-title"> 识别订单截图 </view>
  </view>

  <view class="page-content">
    <view class="purchase-intro">
      <view class="purchase-intro-title"> 从淘宝 / 京东订单详情页截图开始 </view>
      <view class="purchase-intro-desc">
        选择 1–5 张同一订单的截图，优先包含商品明细与实付金额。识别只生成可编辑草稿，
        确认后才会更新库存与支出。
      </view>
    </view>

    <view class="purchase-grid">
      <view v-for="(path, index) in images" :key="path" class="purchase-thumb">
        <image :src="path" mode="aspectFill" class="purchase-thumb-image" />
        <button type="button" class="purchase-remove" :disabled="busy" aria-label="移除截图" @click="removeImage(index)">
          ×
        </button>
      </view>
      <button
        v-if="images.length < 5"
        type="button"
        class="purchase-add-tile"
        :disabled="busy"
        @click="chooseImages"
      >
        <AppIcon name="camera" :size="22" />
        <text>添加截图</text>
      </button>
    </view>

    <view v-if="stage" class="purchase-stage">
      {{ stage }}
    </view>
    <text v-if="error" class="form-error">{{ error }}</text>

    <button class="btn-primary purchase-submit" :disabled="!canSubmit" @click="submit">
      {{ busy ? '处理中…' : '上传并识别' }}
    </button>
    <view class="purchase-tip"> 截图只允许家庭成员访问，不会直接写入库存或支出。 </view>
  </view>
</template>

<style scoped>
.purchase-intro {
  margin-bottom: var(--space-16);
  padding: var(--space-12);
  border-radius: var(--radius-lg);
  background: var(--color-bg-surface);
}

.purchase-intro-title {
  font-weight: 600;
  color: var(--color-text-primary);
}

.purchase-intro-desc {
  margin-top: var(--space-4);
  font-size: var(--font-size-caption);
  color: var(--color-text-tertiary);
  line-height: 1.5;
}

.purchase-grid {
  display: flex;
  flex-wrap: wrap;
  gap: var(--space-12);
}

.purchase-thumb {
  position: relative;
  width: 104px;
  height: 104px;
  border-radius: var(--radius-md);
  overflow: hidden;
  background: var(--color-bg-muted);
}

.purchase-thumb-image {
  width: 100%;
  height: 100%;
}

.purchase-remove {
  position: absolute;
  top: 2px;
  right: 2px;
  width: 24px;
  height: 24px;
  min-width: 24px;
  padding: 0;
  border-radius: 50%;
  background: rgba(0, 0, 0, 0.55);
  color: #fff;
  font-size: 16px;
  line-height: 22px;
}

.purchase-add-tile {
  width: 104px;
  height: 104px;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: var(--space-4);
  border: 1px dashed var(--color-divider);
  border-radius: var(--radius-md);
  background: var(--color-bg-surface);
  color: var(--color-text-tertiary);
  font-size: var(--font-size-caption);
}

.purchase-stage {
  margin-top: var(--space-16);
  padding: var(--space-8) var(--space-12);
  border-radius: var(--radius-md);
  background: var(--color-brand-soft);
  color: var(--color-brand);
  font-size: var(--font-size-caption);
}

.purchase-submit {
  margin-top: var(--space-16);
  width: 100%;
}

.purchase-tip {
  margin-top: var(--space-8);
  font-size: var(--font-size-caption);
  color: var(--color-text-tertiary);
}
</style>
