<script setup lang="ts">
import { ref, computed } from 'vue'
import { useRoute, useRouter } from '../../utils/navigation'
import AppIcon from '../../components/app/AppIcon.vue'
import { useCatStore } from '../../stores/cat'
import { services } from '../../services'
import { useProtectedPage } from '../../utils/page'
import type { TodayStatusData } from '../../types'

const route = useRoute()
const router = useRouter()
const store = useCatStore()
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

function selectTab(id: typeof activeTab.value) {
  if (id === 'overview' || id === 'trends') {
    activeTab.value = id
    return
  }
  store.setCat(catId.value)
  if (id === 'records') router.push('/records')
  else if (id === 'medical') router.push('/medical/upload')
  else router.push('/moments')
}

useProtectedPage(load)
</script>

<template>
  <view class="cat-detail-header">
    <image mode="aspectFill" v-if="cat.avatar" class="cat-detail-avatar" :src="cat.avatar" :alt="cat.name + '头像'" />
    <view v-else class="cat-detail-avatar-placeholder">
      <AppIcon name="cat" :size="40" />
    </view>
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
  margin: 0 auto var(--space-12);
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
  margin: 0 auto var(--space-12);
  display: block;
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
</style>
