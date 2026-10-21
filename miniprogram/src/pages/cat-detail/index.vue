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
import type { TodayStatusData, TrendDataPoint } from '../../types'
import type { TimelineEvent } from '../../types'

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

const weightTrends = ref<TrendDataPoint[]>([])
const loading = ref(true)
const range = ref<7 | 30 | 90>(7)
const todayStatus = ref<TodayStatusData | null>(null)
const catRecords = ref<RecordDTO[]>([])
const recordsLoading = ref(false)
const catMoments = ref<TimelineEvent[]>([])
const momentsLoading = ref(false)
const avatarUploading = ref(false)

const weightObservations = computed(() =>
  [...weightTrends.value]
    .filter((point) => Number.isFinite(point.weight) && point.weight > 0)
    .sort((a, b) => a.date.localeCompare(b.date))
)
const latestWeight = computed(() => weightObservations.value[weightObservations.value.length - 1] || null)
const previousWeight = computed(() =>
  weightObservations.value.length > 1 ? weightObservations.value[weightObservations.value.length - 2] : null
)
const weightChange = computed(() =>
  latestWeight.value && previousWeight.value ? latestWeight.value.weight - previousWeight.value.weight : null
)
const chartAxisMax = computed(() => {
  const max = Math.max(0, ...weightObservations.value.map((point) => point.weight))
  return Math.max(1, Math.ceil((max * 1.2) / 0.5) * 0.5)
})
const chartPoints = computed(() => {
  const points = weightObservations.value
  const count = 7
  const sampled =
    points.length <= count
      ? points
      : Array.from({ length: count }, (_, index) => points[Math.round((index * (points.length - 1)) / (count - 1))])

  return sampled.map((point) => ({
    date: point.date.slice(5),
    weight: point.weight,
    height: `${Math.max(3, Math.min(100, (point.weight / chartAxisMax.value) * 100))}%`
  }))
})

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

const catMomentGroups = computed(() => {
  const groups = new Map<string, TimelineEvent[]>()
  for (const moment of catMoments.value) {
    const date = moment.date?.slice(0, 10) || '日期未知'
    const items = groups.get(date) || []
    items.push(moment)
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

function momentLabel(type: TimelineEvent['type']) {
  if (type === 'photo') return '生活切片'
  if (type === 'milestone') return '成长纪念'
  if (type === 'medical') return '健康记录'
  return '一起生活'
}

function momentIcon(type: TimelineEvent['type']) {
  if (type === 'photo') return 'camera'
  if (type === 'milestone') return 'calendar'
  if (type === 'medical') return 'medical'
  return 'users'
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

async function openMoments() {
  const previousTab = activeTab.value
  const selectedCatId = catId.value
  activeTab.value = 'moments'
  momentsLoading.value = true
  store.setCat(selectedCatId)
  const result = await services.getMoments()
  momentsLoading.value = false

  if (!result.success) {
    activeTab.value = previousTab
    uni.showModal({
      title: '时光加载失败',
      content: '暂时无法获取时光内容，请检查网络后重试。',
      confirmText: '重试',
      success: (modalResult) => {
        if (modalResult.confirm) void openMoments()
      }
    })
    return
  }

  catMoments.value = result.data
    .filter((moment) => !moment.catId || moment.catId === 'both' || moment.catId === selectedCatId)
    .sort((a, b) => b.date.localeCompare(a.date))

  if (!catMoments.value.length) {
    activeTab.value = previousTab
    uni.showModal({
      title: '还没有时光',
      content: `${cat.value?.name || '这只猫'}还没有时光内容，要去时光页添加吗？`,
      confirmText: '去时光页',
      cancelText: '稍后',
      success: (modalResult) => {
        if (modalResult.confirm) router.push('/moments')
      }
    })
    return
  }

  activeTab.value = 'moments'
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
  if (id === 'moments') {
    await openMoments()
    return
  }
  store.setCat(catId.value)
  if (id === 'medical') router.push('/medical/upload')
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

    <template v-if="activeTab === 'moments'">
      <view class="cat-moments">
        <view v-if="momentsLoading" class="cat-record-loading">正在加载时光…</view>
        <template v-else>
          <view class="cat-moments-heading">
            <view>
              <text class="cat-moments-kicker">{{ cat.name }} · 成长与陪伴</text>
              <view class="cat-moments-title">时光片段</view>
            </view>
            <text class="cat-moments-count">{{ catMoments.length }} 段回忆</text>
          </view>
          <view class="cat-moments-list">
            <view v-for="group in catMomentGroups" :key="group.date" class="cat-moment-group">
              <view class="cat-moment-date">{{ group.date }}</view>
              <view v-for="moment in group.items" :key="moment.id" class="cat-moment-item">
                <view class="cat-moment-rail">
                  <view class="cat-moment-icon"><AppIcon :name="momentIcon(moment.type)" :size="18" /></view>
                  <view class="cat-moment-line" />
                </view>
                <view class="cat-moment-card">
                  <view class="cat-moment-meta">
                    <text class="cat-moment-type">{{ momentLabel(moment.type) }}</text>
                    <text v-if="recordTime(moment.date)">{{ recordTime(moment.date) }}</text>
                  </view>
                  <text class="cat-moment-title">{{ moment.title }}</text>
                  <text v-if="moment.body" class="cat-moment-body">{{ moment.body }}</text>
                  <view v-if="moment.images" class="cat-moment-images">
                    <AppIcon name="photo" :size="14" />
                    {{ moment.images }} 张照片
                  </view>
                </view>
              </view>
            </view>
          </view>
        </template>
      </view>
    </template>

    <template v-if="activeTab === 'trends'">
      <view class="weight-trend-page">
        <view class="weight-trend-heading">
          <view>
            <text class="weight-trend-kicker">{{ cat.name }} · 健康趋势</text>
            <view class="weight-trend-title">体重趋势</view>
          </view>
          <view class="weight-range" role="tablist" aria-label="趋势时间范围">
            <button class="weight-range-btn" :class="{ active: range === 7 }" @click="setRange(7)">7天</button>
            <button class="weight-range-btn" :class="{ active: range === 30 }" @click="setRange(30)">30天</button>
            <button class="weight-range-btn" :class="{ active: range === 90 }" @click="setRange(90)">90天</button>
          </view>
        </view>

        <view class="weight-summary-card">
          <view class="weight-summary-topline">
            <text>最近一次称重</text>
            <view class="weight-summary-mark"><AppIcon name="trend" :size="18" /></view>
          </view>
          <view class="weight-summary-value">
            <text>{{ latestWeight ? latestWeight.weight.toFixed(1) : '—' }}</text>
            <text class="weight-summary-unit">kg</text>
          </view>
          <view class="weight-summary-bottomline">
            <text>{{ latestWeight ? latestWeight.date : '记录体重后查看变化' }}</text>
            <view v-if="weightChange !== null" class="weight-change" :class="weightChange > 0 ? 'up' : weightChange < 0 ? 'down' : 'steady'">
              {{ weightChange > 0 ? '+' : '' }}{{ weightChange.toFixed(1) }} kg
              <text>较上次</text>
            </view>
            <text v-else class="weight-first-record">{{ latestWeight ? '区间首次' : '等待第一条记录' }}</text>
          </view>
        </view>

        <view class="weight-chart-card">
          <view class="weight-chart-heading">
            <view>
              <text class="weight-chart-title">称重变化</text>
              <text class="weight-chart-caption">近 {{ range }} 天 · {{ weightObservations.length }} 次记录</text>
            </view>
            <text class="weight-chart-unit">KG</text>
          </view>

          <view v-if="loading" class="weight-chart-empty">
            <view class="skeleton" style="height: 190px; border-radius: 14px" />
          </view>
          <view v-else-if="!weightObservations.length" class="weight-chart-empty">
            <view class="weight-empty-icon"><AppIcon name="weight" :size="28" /></view>
            <text class="weight-empty-title">还没有体重记录</text>
            <text class="weight-empty-copy">添加称重记录后，这里会展示体重变化</text>
            <button class="weight-empty-action" @click="router.push('/records/quick/weight')">去记录体重</button>
          </view>
          <view v-else class="weight-chart-plot">
            <view class="weight-chart-guides">
              <view class="weight-chart-guide top" />
              <view class="weight-chart-guide middle" />
              <view class="weight-chart-guide bottom" />
            </view>
            <view class="weight-chart-axis">
              <text>{{ chartAxisMax.toFixed(1) }}</text>
              <text>{{ (chartAxisMax / 2).toFixed(1) }}</text>
              <text>0</text>
            </view>
            <view class="weight-chart-columns">
              <view v-for="(point, index) in chartPoints" :key="`${point.date}-${index}`" class="weight-chart-column">
                <text class="weight-chart-value">{{ point.weight.toFixed(1) }}</text>
                <view class="weight-chart-bar-slot">
                  <view class="weight-chart-bar" :style="{ height: point.height }" />
                </view>
                <text class="weight-chart-date">{{ point.date }}</text>
              </view>
            </view>
          </view>
        </view>

        <view class="weight-trend-note">
          <AppIcon name="info" :size="15" />
          <text>仅展示所选时间范围内已有的称重记录</text>
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
.cat-moments {
  padding: var(--space-16) 0 var(--space-24);
}
.cat-moments-heading {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: var(--space-8);
  margin-bottom: var(--space-20);
}
.cat-moments-heading > view {
  min-width: 0;
}
.cat-moments-kicker {
  color: var(--color-text-tertiary);
  font-size: var(--font-size-caption);
}
.cat-moments-title {
  margin-top: 4px;
  color: var(--color-text-primary);
  font-size: var(--font-size-title);
  font-weight: 700;
}
.cat-moments-count {
  flex: 0 0 auto;
  padding: 6px 10px;
  border-radius: var(--radius-pill);
  color: var(--color-brand);
  background: var(--color-brand-soft);
  font-size: var(--font-size-caption);
}
.cat-moment-group + .cat-moment-group {
  margin-top: var(--space-20);
}
.cat-moment-date {
  margin-bottom: var(--space-8);
  color: var(--color-text-secondary);
  font-size: var(--font-size-assist);
  font-weight: 600;
}
.cat-moment-item {
  display: flex;
  align-items: stretch;
  gap: var(--space-12);
}
.cat-moment-item + .cat-moment-item {
  margin-top: var(--space-8);
}
.cat-moment-rail {
  width: 36px;
  flex: 0 0 36px;
  display: flex;
  flex-direction: column;
  align-items: center;
}
.cat-moment-icon {
  width: 36px;
  height: 36px;
  flex: 0 0 36px;
  display: flex;
  align-items: center;
  justify-content: center;
  border-radius: 50%;
  color: var(--color-brand);
  background: var(--color-brand-soft);
}
.cat-moment-line {
  width: 1px;
  flex: 1;
  min-height: 12px;
  margin-top: 5px;
  background: var(--color-divider);
}
.cat-moment-item:last-child .cat-moment-line {
  display: none;
}
.cat-moment-card {
  min-width: 0;
  flex: 1;
  padding: var(--space-12);
  border: 1px solid var(--color-divider);
  border-radius: var(--radius-md);
  background: var(--color-bg-surface);
  box-shadow: 0 4px 14px rgba(54, 42, 30, 0.035);
}
.cat-moment-meta {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: var(--space-8);
  color: var(--color-text-tertiary);
  font-size: var(--font-size-assist);
}
.cat-moment-type {
  color: var(--color-brand);
  font-weight: 600;
}
.cat-moment-title {
  display: block;
  margin-top: 6px;
  color: var(--color-text-primary);
  font-size: var(--font-size-body);
  font-weight: 600;
  line-height: 1.45;
}
.cat-moment-body {
  display: block;
  margin-top: 5px;
  color: var(--color-text-secondary);
  font-size: var(--font-size-caption);
  line-height: 1.6;
  word-break: break-word;
}
.cat-moment-images {
  display: flex;
  align-items: center;
  gap: 4px;
  margin-top: 9px;
  color: var(--color-text-tertiary);
  font-size: var(--font-size-assist);
}
.weight-trend-page {
  padding: var(--space-16) 0 var(--space-24);
}
.weight-trend-heading {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: var(--space-8);
  margin-bottom: var(--space-16);
}
.weight-trend-kicker {
  color: var(--color-text-tertiary);
  font-size: var(--font-size-caption);
}
.weight-trend-title {
  margin-top: 4px;
  color: var(--color-text-primary);
  font-size: var(--font-size-title);
  font-weight: 700;
}
.weight-range {
  display: flex;
  align-items: center;
  gap: 4px;
  height: 40px;
  padding: 3px;
  box-sizing: border-box;
  border-radius: var(--radius-pill);
  background: var(--color-bg-subtle);
}
.weight-range-btn {
  display: flex;
  flex: 1;
  align-items: center;
  justify-content: center;
  min-width: 42px;
  height: 32px;
  min-height: 32px;
  margin: 0;
  padding: 0 8px;
  box-sizing: border-box;
  border: 0;
  border-radius: var(--radius-pill);
  color: var(--color-text-secondary);
  background: transparent;
  font-size: var(--font-size-caption);
  line-height: 1;
  text-align: center;
  white-space: nowrap;
}
.weight-range-btn::after {
  border: 0;
}
.weight-range-btn.active {
  color: #fff;
  background: var(--color-brand);
  box-shadow: 0 3px 8px rgba(184, 95, 54, 0.2);
}
.weight-summary-card {
  padding: var(--space-16);
  margin-bottom: var(--space-16);
  border: 1px solid rgba(200, 115, 69, 0.16);
  border-radius: var(--radius-lg);
  background: linear-gradient(135deg, #fffaf4 0%, #f8eadc 100%);
}
.weight-summary-topline,
.weight-summary-bottomline {
  display: flex;
  align-items: center;
  justify-content: space-between;
}
.weight-summary-topline {
  color: var(--color-text-secondary);
  font-size: var(--font-size-caption);
}
.weight-summary-mark {
  width: 32px;
  height: 32px;
  display: flex;
  align-items: center;
  justify-content: center;
  border-radius: 50%;
  color: var(--color-brand);
  background: rgba(255, 255, 255, 0.72);
}
.weight-summary-value {
  display: flex;
  align-items: baseline;
  gap: 6px;
  margin: 2px 0 var(--space-8);
  color: var(--color-text-primary);
  font-size: 38px;
  font-weight: 700;
  line-height: 1.15;
}
.weight-summary-unit {
  color: var(--color-text-secondary);
  font-size: var(--font-size-body);
  font-weight: 500;
}
.weight-summary-bottomline {
  color: var(--color-text-tertiary);
  font-size: var(--font-size-caption);
}
.weight-change,
.weight-first-record {
  padding: 5px 9px;
  border-radius: var(--radius-pill);
  background: rgba(255, 255, 255, 0.74);
  font-weight: 600;
}
.weight-change text {
  margin-left: 3px;
  font-weight: 400;
}
.weight-change.up {
  color: #b45d42;
}
.weight-change.down {
  color: #55836e;
}
.weight-change.steady,
.weight-first-record {
  color: var(--color-text-secondary);
}
.weight-chart-card {
  padding: var(--space-16) var(--space-12);
  border: 1px solid var(--color-divider);
  border-radius: var(--radius-lg);
  background: var(--color-bg-surface);
  box-shadow: 0 8px 24px rgba(54, 42, 30, 0.04);
}
.weight-chart-heading {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  padding: 0 var(--space-4);
}
.weight-chart-heading > view {
  display: flex;
  flex-direction: column;
  gap: 4px;
}
.weight-chart-title {
  color: var(--color-text-primary);
  font-size: var(--font-size-body);
  font-weight: 700;
}
.weight-chart-caption {
  color: var(--color-text-tertiary);
  font-size: var(--font-size-caption);
}
.weight-chart-unit {
  color: var(--color-text-tertiary);
  font-size: 10px;
  letter-spacing: 1px;
}
.weight-chart-empty {
  min-height: 190px;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  padding: var(--space-16) 0;
  text-align: center;
}
.weight-empty-icon {
  width: 52px;
  height: 52px;
  display: flex;
  align-items: center;
  justify-content: center;
  margin-bottom: var(--space-8);
  border-radius: 18px;
  color: var(--color-brand);
  background: var(--color-brand-soft);
}
.weight-empty-title {
  color: var(--color-text-primary);
  font-size: var(--font-size-body);
  font-weight: 600;
}
.weight-empty-copy {
  margin-top: 5px;
  color: var(--color-text-tertiary);
  font-size: var(--font-size-caption);
}
.weight-empty-action {
  padding: 7px 14px;
  margin-top: var(--space-12);
  border: 0;
  border-radius: var(--radius-pill);
  color: #fff;
  background: var(--color-brand);
  font-size: var(--font-size-caption);
}
.weight-chart-plot {
  position: relative;
  height: 216px;
  margin-top: var(--space-12);
}
.weight-chart-guides {
  position: absolute;
  top: 10px;
  right: 2px;
  bottom: 27px;
  left: 36px;
  display: flex;
  flex-direction: column;
  justify-content: space-between;
}
.weight-chart-guide {
  width: 100%;
  border-top: 1px dashed var(--color-divider);
}
.weight-chart-axis {
  position: absolute;
  top: 3px;
  bottom: 20px;
  left: 0;
  display: flex;
  flex-direction: column;
  justify-content: space-between;
  color: var(--color-text-tertiary);
  font-size: 10px;
}
.weight-chart-columns {
  position: absolute;
  top: 0;
  right: 0;
  bottom: 0;
  left: 36px;
  display: flex;
  justify-content: space-around;
  gap: 4px;
}
.weight-chart-column {
  min-width: 0;
  flex: 1;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: flex-end;
}
.weight-chart-value {
  margin-bottom: 5px;
  color: var(--color-text-secondary);
  font-size: 10px;
  font-weight: 600;
}
.weight-chart-bar-slot {
  width: 100%;
  height: 142px;
  display: flex;
  align-items: flex-end;
  justify-content: center;
}
.weight-chart-bar {
  width: 18px;
  min-height: 5px;
  border-radius: 8px 8px 3px 3px;
  background: linear-gradient(180deg, #e9a05e 0%, var(--color-brand) 100%);
  box-shadow: 0 3px 8px rgba(200, 115, 69, 0.18);
}
.weight-chart-date {
  margin-top: 7px;
  color: var(--color-text-tertiary);
  font-size: 9px;
  white-space: nowrap;
}
.weight-trend-note {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: var(--space-12) var(--space-4);
  color: var(--color-text-tertiary);
  font-size: var(--font-size-caption);
}
</style>
