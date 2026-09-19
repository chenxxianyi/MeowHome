<script setup lang="ts">
import { computed, ref } from 'vue'
import { useRouter } from '../../utils/navigation'
import AppIcon from '../../components/app/AppIcon.vue'
import AIResultBadge from '../../components/ai/AIResultBadge.vue'
import AIEvidencePanel from '../../components/ai/AIEvidencePanel.vue'
import OfflineBanner from '../../components/app/OfflineBanner.vue'
import { useCatStore } from '../../stores/cat'
import { services } from '../../services'
import { usePageCapabilities, useProtectedPage } from '../../utils/page'
import type { AIAnalysisReport, TimelineEvent } from '../../types'

type MomentFilter = 'all' | 'photo' | 'milestone' | 'care'

const router = useRouter()
const catStore = useCatStore()
const events = ref<TimelineEvent[]>([])
const loading = ref(true)
const showEvidence = ref(false)
const activeFilter = ref<MomentFilter>('all')
const aiSummary = ref<AIAnalysisReport | null>(null)

const filters: { id: MomentFilter; label: string }[] = [
  { id: 'all', label: '全部' },
  { id: 'photo', label: '照片' },
  { id: 'milestone', label: '成长' },
  { id: 'care', label: '健康与互动' }
]

const filteredEvents = computed(() => {
  if (activeFilter.value === 'all') return events.value
  if (activeFilter.value === 'care') {
    return events.value.filter((event) => event.type === 'medical' || event.type === 'interaction')
  }
  return events.value.filter((event) => event.type === activeFilter.value)
})

const monthGroups = computed(() => {
  const groups = new Map<string, TimelineEvent[]>()
  filteredEvents.value.forEach((event) => {
    const key = event.date.slice(0, 7)
    const group = groups.get(key) || []
    group.push(event)
    groups.set(key, group)
  })
  return Array.from(groups.entries()).map(([key, items]) => ({
    key,
    label: formatMonth(key),
    items,
    images: items.reduce((sum, item) => sum + (item.images || 0), 0)
  }))
})

const totalImages = computed(() => events.value.reduce((sum, event) => sum + (event.images || 0), 0))
const recordedDays = computed(() => new Set(events.value.map((event) => event.date)).size)
const featuredEvent = computed(() => events.value[0])

const aiReview = computed(() => ({
  body:
    aiSummary.value?.body ||
    (events.value.length
      ? `这个阶段共收藏 ${recordedDays.value} 天、${totalImages.value} 张照片。${featuredEvent.value?.title || '两只猫留下了许多日常片段'}，平凡的小事正在慢慢变成家的记忆。`
      : '新的回忆正在路上，记录第一张照片或一个值得纪念的日子吧。'),
  generatedAt: aiSummary.value?.generatedAt || new Date().toISOString(),
  evidenceCount: aiSummary.value?.recordCount || events.value.length,
  evidence: aiSummary.value?.evidence || []
}))

function formatMonth(key: string) {
  const [year, month] = key.split('-')
  return `${year}年${Number(month)}月`
}

function formatDay(date: string) {
  const [, month, day] = date.split('-')
  return `${Number(month)}月${Number(day)}日`
}

function catName(catId?: string) {
  if (catId === 'both') {
    const names = catStore.cats.map((c) => c.name).filter(Boolean)
    return names.length ? names.join(' 与 ') : '猫宅'
  }
  return catStore.cats.find((cat) => cat.id === catId)?.name || '猫宅'
}

function eventIcon(type: TimelineEvent['type']) {
  if (type === 'photo') return 'camera'
  if (type === 'milestone') return 'calendar'
  if (type === 'medical') return 'medical'
  return 'users'
}

function eventLabel(type: TimelineEvent['type']) {
  if (type === 'photo') return '生活切片'
  if (type === 'milestone') return '成长纪念'
  if (type === 'medical') return '健康记录'
  return '一起生活'
}

async function load() {
  loading.value = true
  try {
    const [momentsResponse, summaryResponse] = await Promise.all([services.getMoments(), services.getAISummary()])
    events.value = [...momentsResponse.data].sort((a, b) => b.date.localeCompare(a.date))
    aiSummary.value = summaryResponse.data
  } finally {
    loading.value = false
  }
}

useProtectedPage(load)
usePageCapabilities(load, '猫宅 · 我们的时光')
</script>

<template>
  <OfflineBanner />
  <view class="page-header moments-page-header">
    <view>
      <view class="page-title"> 时光 </view>
      <view class="page-subtitle"> 收藏一起生活的每个小瞬间 </view>
    </view>
    <button
      type="button"
      class="moments-add-button"
      aria-label="记录新的时光"
      @click="router.push('/records/quick/photo')"
    >
      <AppIcon name="camera" :size="20" />
    </button>
  </view>

  <view class="page-content moments-page">
    <view v-if="loading" class="moments-loading" aria-label="正在加载时光画册" aria-busy="true">
      <view class="skeleton moments-cover-skeleton" />
      <view class="skeleton moments-review-skeleton" />
      <view class="skeleton moments-grid-skeleton" />
    </view>

    <template v-else>
      <view class="moments-cover">
        <view class="moments-cover-copy">
          <text>MEOWHOME MEMORY BOOK · 2026</text>
          <view>把平常，<br />收藏成时光</view>
          <view>两只猫的成长、陪伴和每一次靠近，都值得被好好留下。</view>
        </view>

        <view class="moments-cover-collage" aria-hidden="true">
          <view class="moments-polaroid polaroid-one">
            <AppIcon name="cat" :size="28" />
            <text>nap time</text>
          </view>
          <view class="moments-polaroid polaroid-two">
            <AppIcon name="camera" :size="25" />
            <text>together</text>
          </view>
          <text class="moments-tape" />
        </view>

        <view class="moments-cover-stats">
          <text
            ><text>{{ events.length }}</text> 段回忆</text
          >
          <text
            ><text>{{ totalImages }}</text> 张照片</text
          >
          <text
            ><text>{{ recordedDays }}</text> 个日子</text
          >
        </view>
        <button type="button" class="moments-cover-action" @click="router.push('/records/quick/photo')">
          <AppIcon name="camera" :size="17" />
          收藏今天
        </button>
      </view>

      <view class="moments-ai-note" aria-label="AI 月度回顾">
        <text class="moments-ai-tape" aria-hidden="true" />
        <view class="moments-ai-head">
          <AIResultBadge>AI 月度回顾</AIResultBadge>
          <text>{{ aiReview.evidenceCount }} 段回忆</text>
        </view>
        <view>这个月，它们这样生活</view>
        <view>{{ aiReview.body }}</view>
        <view class="moments-ai-footer">
          <view>整理于 {{ aiReview.generatedAt.slice(11, 16) }}</view>
          <button type="button" @click="showEvidence = !showEvidence">
            {{ showEvidence ? '收起依据' : '查看整理依据' }}
            <AppIcon :name="showEvidence ? 'chevronDown' : 'chevronRight'" :size="14" />
          </button>
        </view>
      </view>

      <AIEvidencePanel v-if="showEvidence" :items="aiReview.evidence" />

      <view class="moments-album-section">
        <view class="moments-section-heading">
          <view>
            <text>OUR LITTLE STORIES</text>
            <view>记忆画册</view>
          </view>
          <view>{{ filteredEvents.length }} 个瞬间</view>
        </view>

        <view class="moments-filter-bar" role="tablist" aria-label="筛选时光类型">
          <button
            v-for="filter in filters"
            :key="filter.id"
            type="button"
            role="tab"
            :aria-selected="activeFilter === filter.id"
            :class="{ active: activeFilter === filter.id }"
            @click="activeFilter = filter.id"
          >
            {{ filter.label }}
          </button>
        </view>

        <view v-if="monthGroups.length" class="moments-months">
          <view v-for="month in monthGroups" :key="month.key" class="moments-month">
            <view class="moments-month-head">
              <view>
                <text>{{ month.key.replace('-', ' / ') }}</text>
                <view>{{ month.label }}</view>
              </view>
              <view>{{ month.items.length }} 个瞬间 · {{ month.images }} 张照片</view>
            </view>

            <view class="moments-memory-grid">
              <view
                v-for="(event, index) in month.items"
                :key="event.id"
                class="moments-memory-card"
                :class="[`type-${event.type}`, { featured: index === 0 }]"
              >
                <view class="moments-memory-visual">
                  <text class="moments-memory-index"> NO. {{ String(index + 1).padStart(2, '0') }} </text>
                  <view class="moments-memory-icon">
                    <AppIcon :name="eventIcon(event.type)" :size="index === 0 ? 32 : 25" />
                  </view>
                  <view class="moments-memory-count">
                    <AppIcon name="photo" :size="12" />
                    {{ event.images || 0 }}
                  </view>
                  <text class="moments-memory-date">{{ formatDay(event.date) }}</text>
                </view>
                <view class="moments-memory-copy">
                  <text>{{ eventLabel(event.type) }}</text>
                  <view>{{ event.title }}</view>
                  <view v-if="event.body">{{ event.body }}</view>
                  <view>{{ catName(event.catId) }}</view>
                </view>
              </view>
            </view>
          </view>
        </view>

        <view v-else class="moments-empty">
          <AppIcon name="camera" :size="34" />
          <text>这个分类还没有回忆</text>
          <text>记录一张照片，让故事从今天开始。</text>
          <button type="button" @click="router.push('/records/quick/photo')">收藏今天</button>
        </view>
      </view>
    </template>
  </view>
</template>
