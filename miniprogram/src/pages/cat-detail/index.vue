<script setup lang="ts">
import { ref, computed } from 'vue'
import { useRoute, useRouter } from '../../utils/navigation'
import AppIcon from '../../components/app/AppIcon.vue'
import { useCatStore } from '../../stores/cat'
import { useAuthStore } from '../../stores/auth'
import { services } from '../../services'
import { catApi, toCat } from '../../api/endpoints'
import { toApiError } from '../../api/adapter'
import { recordTypes } from '../../data/recordTypes'
import { useProtectedPage } from '../../utils/page'
import type { RecordDTO } from '../../api/endpoints'
import type { TodayStatusData } from '../../types'

const route = useRoute()
const router = useRouter()
const store = useCatStore()
const auth = useAuthStore()
const catId = computed(() => (route.params.catId as string) || store.currentCatId || 'cat-whit')
const cat = computed(() => store.cats.find((c) => c.id === catId.value) || store.cats[0])
const activeTab = ref<'overview' | 'records' | 'trends' | 'medical' | 'moments'>('overview')

const tabs = [
  { id: 'overview' as const, label: '概览' },
  { id: 'records' as const, label: '记录' },
  { id: 'trends' as const, label: '趋势' },
  { id: 'medical' as const, label: '医疗' },
  { id: 'moments' as const, label: '时光' }
]

const weightTrends = ref<any[]>([])
const loading = ref(true)
const range = ref<7 | 30 | 90>(7)
const todayStatus = ref<TodayStatusData | null>(null)
const catRecords = ref<RecordDTO[]>([])
const recordsLoading = ref(false)
const avatarUploading = ref(false)

const catRecordGroups = computed(() => {
  const groups = new Map<string, RecordDTO[]>()
  for (const record of catRecords.value) {
    const date = record.occurred_at?.slice(0, 10) || '日期未知'
    const items = groups.get(date) || []
    items.push(record)
    groups.set(date, items)
  }
  return Array.from(groups.entries()).map(([date, items]) => ({ date, items }))
})

const allRecordTypes = [...recordTypes.high, ...recordTypes.health, ...recordTypes.life]

function recordLabel(type: string) {
  return allRecordTypes.find((item) => item.id === type)?.label || type
}

function recordIcon(type: string) {
  return allRecordTypes.find((item) => item.id === type)?.icon || 'record'
}

function recordTime(value: string) {
  return value && value.length >= 16 ? value.slice(11, 16) : ''
}

function chooseAvatar() {
  if (avatarUploading.value) return
  uni.chooseMedia({
    count: 1,
    mediaType: ['image'],
    sourceType: ['album', 'camera'],
    success: (result) => {
      const filePath = result.tempFiles[0]?.tempFilePath
      if (filePath) void uploadAvatar(filePath)
    },
    fail: (error) => {
      if (!String(error.errMsg || '').includes('cancel')) {
        uni.showToast({ title: '无法选择图片，请检查相册权限', icon: 'none' })
      }
    }
  })
}

async function uploadAvatar(filePath: string) {
  const familyId = auth.familyId
  if (!familyId) {
    uni.showToast({ title: '请先加入家庭再上传头像', icon: 'none' })
    return
  }

  avatarUploading.value = true
  uni.showLoading({ title: '正在上传', mask: true })
  try {
    const updatedCat = toCat(await catApi.uploadAvatar(familyId, catId.value, filePath))
    store.setCats(store.cats.map((item) => (item.id === updatedCat.id ? updatedCat : item)))
    uni.showToast({ title: '头像已更新', icon: 'success' })
  } catch (error) {
    uni.showModal({ title: '头像上传失败', content: toApiError(error).message, showCancel: false })
  } finally {
    avatarUploading.value = false
    uni.hideLoading()
  }
}

const statusRows = computed(() => {
  const status = todayStatus.value
  if (!status) return []
  return [
    { icon: 'food', label: '饮食', value: status.food.label, state: status.food.state },
    { icon: 'water', label: '饮水', value: status.water.label, state: status.water.state },
    { icon: 'elimination', label: '排便', value: status.elimination.label, state: status.elimination.state },
    { icon: 'vomit', label: '呕吐', value: status.vomit.label, state: status.vomit.state }
  ]
})

async function load() {
  loading.value = true
  const [trendResponse, todayResponse] = await Promise.all([
    services.getTrends(catId.value, range.value),
    services.getTodayStatus(catId.value)
  ])
  weightTrends.value = trendResponse.data
  todayStatus.value = todayResponse.data
  loading.value = false
}

function setRange(next: 7 | 30 | 90) {
  range.value = next
  load()
}

async function openRecords() {
  const previousTab = activeTab.value
  const selectedCatId = catId.value
  activeTab.value = 'records'
  recordsLoading.value = true
  store.setCat(selectedCatId)
  const result = await services.getRecords(selectedCatId)
  recordsLoading.value = false

  if (!result.success) {
    activeTab.value = previousTab
    uni.showModal({
      title: '记录加载失败',
      content: '暂时无法获取记录，请检查网络后重试。',
      confirmText: '重试',
      success: (modalResult) => {
        if (modalResult.confirm) void openRecords()
      }
    })
    return
  }

  catRecords.value = [...result.data].sort((a, b) => b.occurred_at.localeCompare(a.occurred_at))
  if (!catRecords.value.length) {
    activeTab.value = previousTab
    uni.showModal({
      title: '还没有记录',
      content: `还没有${cat.value?.name || '这只猫'}的记录，要去记录页添加吗？`,
      confirmText: '去记录',
      cancelText: '稍后',
      success: (modalResult) => {
        if (modalResult.confirm) router.push('/records')
      }
    })
    return
  }

  activeTab.value = 'records'
}

async function selectTab(id: typeof activeTab.value) {
  if (id === 'overview' || id === 'trends') {
    activeTab.value = id
    return
  }
  if (id === 'records') {
    await openRecords()
    return
  }
  store.setCat(catId.value)
  if (id === 'medical') router.push('/medical/upload')
  else router.push('/moments')
}

useProtectedPage(load)
</script>

<template>
  <view class="cat-detail-header">
    <button
      type="button"
      class="cat-avatar-button"
      :disabled="avatarUploading"
      :aria-label="avatarUploading ? '正在上传头像' : '点击更换猫咪头像'"
      @click="chooseAvatar"
    >
      <image v-if="cat.avatar" mode="aspectFill" class="cat-detail-avatar" :src="cat.avatar" :alt="cat.name + '头像'" />
      <view v-else class="cat-detail-avatar-placeholder">
        <AppIcon name="cat" :size="40" />
      </view>
      <view class="cat-avatar-edit-badge"><AppIcon name="camera" :size="14" /></view>
    </button>
    <text class="cat-avatar-hint">{{ avatarUploading ? '正在更新头像…' : '点击头像更换照片' }}</text>
    <view class="cat-detail-name">
      {{ cat.name }}
    </view>
    <view class="cat-detail-meta">
      {{ cat.breed }} · {{ cat.age }}岁 · {{ cat.gender === 'female' ? '母' : '公' }}
    </view>
  </view>

  <view class="cat-tabs">
    <button
      v-for="tab in tabs"
      :key="tab.id"
      class="cat-tab"
      :class="{ active: activeTab === tab.id }"
      @click="selectTab(tab.id)"
    >
      {{ tab.label }}
    </button>
  </view>

  <view class="page-content">
    <template v-if="activeTab === 'overview'">
      <view class="status-panel">
        <view class="status-panel-title"> 今日状态 </view>
        <view v-for="row in statusRows" :key="row.label" class="status-row">
          <view class="status-label"><AppIcon :name="row.icon" :size="18" /> {{ row.label }}</view>
          <text class="status-value" :class="row.state">{{ row.value }}</text>
        </view>
      </view>
    </template>

    <template v-if="activeTab === 'records'">
      <view class="cat-records">
        <view v-if="recordsLoading" class="cat-record-loading">正在加载记录…</view>
        <template v-else>
          <view class="cat-records-heading">
            <text>记录动态</text>
            <text class="cat-records-count">{{ catRecords.length }} 条</text>
          </view>
          <view v-for="group in catRecordGroups" :key="group.date" class="cat-record-group">
            <view class="cat-record-date">{{ group.date }}</view>
            <view v-for="item in group.items" :key="item.id" class="cat-record-item">
              <view class="cat-record-icon">
                <AppIcon :name="recordIcon(item.type)" :size="18" />
              </view>
              <view class="cat-record-copy">
                <text class="cat-record-title">{{ item.title || recordLabel(item.type) }}</text>
                <text class="cat-record-note">{{ item.note || recordLabel(item.type) }}</text>
              </view>
              <text class="cat-record-time">{{ recordTime(item.occurred_at) }}</text>
            </view>
          </view>
        </template>
      </view>
    </template>

    <template v-if="activeTab === 'trends'">
      <view class="chart-container">
        <view class="chart-header">
          <view class="chart-title"> 体重趋势 </view>
          <view class="chart-range">
            <button class="chart-range-btn" :class="{ active: range === 7 }" @click="setRange(7)">7天</button>
            <button class="chart-range-btn" :class="{ active: range === 30 }" @click="setRange(30)">30天</button>
            <button class="chart-range-btn" :class="{ active: range === 90 }" @click="setRange(90)">90天</button>
          </view>
        </view>
        <view v-if="loading" class="chart-empty">
          <view class="skeleton" style="height: 150px" />
        </view>
        <view v-else class="chart-canvas">
          <view v-for="(p, i) in weightTrends.slice(-7)" :key="i" class="chart-point">
            <view class="chart-bar" :style="{ height: p.weight + 'px' }" />
            <view class="chart-label">
              {{ p.date.slice(5) }}
            </view>
          </view>
        </view>
      </view>
    </template>
  </view>
</template>

<style scoped>
.cat-detail-avatar-placeholder {
  width: 88px;
  height: 88px;
  border-radius: 50%;
  background: var(--color-bg-subtle);
  border: 3px solid var(--color-brand-soft);
  margin: 0;
  display: flex;
  align-items: center;
  justify-content: center;
}
.cat-detail-avatar {
  width: 88px;
  height: 88px;
  border-radius: 50%;
  object-fit: cover;
  border: 3px solid var(--color-brand-soft);
  margin: 0;
  display: block;
}
.cat-avatar-button {
  position: relative;
  display: block;
  width: 88px;
  height: 88px;
  padding: 0;
  margin: 0 auto var(--space-8);
  overflow: visible;
  border: 0;
  border-radius: 50%;
  background: transparent;
  line-height: 1;
}
.cat-avatar-button::after {
  border: 0;
}
.cat-avatar-edit-badge {
  position: absolute;
  right: -2px;
  bottom: -2px;
  width: 26px;
  height: 26px;
  display: flex;
  align-items: center;
  justify-content: center;
  border: 2px solid var(--color-bg-surface);
  border-radius: 50%;
  color: #fff;
  background: var(--color-brand);
}
.cat-avatar-hint {
  display: block;
  margin-bottom: var(--space-12);
  color: var(--color-text-tertiary);
  font-size: var(--font-size-caption);
}
.chart-point {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 4px;
  flex: 1;
}
.chart-bar {
  width: 100%;
  background: var(--color-brand);
  border-radius: 4px 4px 0 0;
  min-height: 4px;
}
.chart-label {
  font-size: var(--font-size-caption);
  color: var(--color-text-tertiary);
}
.cat-records {
  padding-bottom: var(--space-24);
}
.cat-records-heading {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: var(--space-16);
  color: var(--color-text-primary);
  font-size: var(--font-size-title);
  font-weight: 700;
}
.cat-records-count {
  color: var(--color-text-tertiary);
  font-size: var(--font-size-caption);
  font-weight: 400;
}
.cat-record-group + .cat-record-group {
  margin-top: var(--space-20);
}
.cat-record-date {
  margin-bottom: var(--space-8);
  color: var(--color-text-secondary);
  font-size: var(--font-size-assist);
  font-weight: 600;
}
.cat-record-item {
  display: flex;
  align-items: center;
  gap: var(--space-12);
  padding: var(--space-12);
  border-bottom: 1px solid var(--color-divider);
  background: var(--color-bg-surface);
}
.cat-record-icon {
  width: 36px;
  height: 36px;
  flex: 0 0 36px;
  display: flex;
  align-items: center;
  justify-content: center;
  border-radius: 50%;
  background: var(--color-bg-subtle);
  color: var(--color-brand);
}
.cat-record-copy {
  min-width: 0;
  flex: 1;
  display: flex;
  flex-direction: column;
  gap: 4px;
}
.cat-record-title {
  overflow: hidden;
  color: var(--color-text-primary);
  font-weight: 600;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.cat-record-note {
  overflow: hidden;
  color: var(--color-text-tertiary);
  font-size: var(--font-size-caption);
  text-overflow: ellipsis;
  white-space: nowrap;
}
.cat-record-time {
  flex: 0 0 auto;
  color: var(--color-text-tertiary);
  font-size: var(--font-size-caption);
}
.cat-record-loading {
  padding: var(--space-32) 0;
  color: var(--color-text-tertiary);
  text-align: center;
}
</style>
