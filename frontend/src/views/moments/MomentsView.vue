<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useRouter } from 'vue-router'
import AppShell from '../../components/app/AppShell.vue'
import AppIcon from '../../components/app/AppIcon.vue'
import AIResultBadge from '../../components/ai/AIResultBadge.vue'
import AIEvidencePanel from '../../components/ai/AIEvidencePanel.vue'
import { useCatStore } from '../../stores/cat'
import { services } from '../../services'
import { mockAISummary } from '../../mocks/health'
import type { TimelineEvent } from '../../types'

type MomentFilter = 'all' | 'photo' | 'milestone' | 'care'

const router = useRouter()
const catStore = useCatStore()
const events = ref<TimelineEvent[]>([])
const loading = ref(true)
const showEvidence = ref(false)
const activeFilter = ref<MomentFilter>('all')

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
  body: events.value.length
    ? `这个阶段共收藏 ${recordedDays.value} 天、${totalImages.value} 张照片。${featuredEvent.value?.title || '两只猫留下了许多日常片段'}，平凡的小事正在慢慢变成家的记忆。`
    : '新的回忆正在路上，记录第一张照片或一个值得纪念的日子吧。',
  generatedAt: mockAISummary.generatedAt,
  evidenceCount: events.value.length,
  evidence: mockAISummary.evidence
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
  if (catId === 'both') return '小白与小橘'
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

onMounted(async () => {
  loading.value = true
  try {
    const response = await services.getMoments()
    events.value = [...response.data].sort((a, b) => b.date.localeCompare(a.date))
  } finally {
    loading.value = false
  }
})
</script>

<template>
  <AppShell>
    <header class="page-header moments-page-header">
      <div>
        <div class="page-title">
          时光
        </div>
        <div class="page-subtitle">
          收藏一起生活的每个小瞬间
        </div>
      </div>
      <button
        type="button"
        class="moments-add-button"
        aria-label="记录新的时光"
        @click="router.push('/records/quick/photo')"
      >
        <AppIcon
          name="camera"
          :size="20"
        />
      </button>
    </header>

    <main class="page-content moments-page">
      <div
        v-if="loading"
        class="moments-loading"
        aria-label="正在加载时光画册"
        aria-busy="true"
      >
        <div class="skeleton moments-cover-skeleton" />
        <div class="skeleton moments-review-skeleton" />
        <div class="skeleton moments-grid-skeleton" />
      </div>

      <template v-else>
        <section class="moments-cover">
          <div class="moments-cover-copy">
            <span>MEOWHOME MEMORY BOOK · 2026</span>
            <h1>把平常，<br />收藏成时光</h1>
            <p>两只猫的成长、陪伴和每一次靠近，都值得被好好留下。</p>
          </div>

          <div class="moments-cover-collage" aria-hidden="true">
            <span class="moments-polaroid polaroid-one">
              <AppIcon name="cat" :size="28" />
              <i>nap time</i>
            </span>
            <span class="moments-polaroid polaroid-two">
              <AppIcon name="camera" :size="25" />
              <i>together</i>
            </span>
            <span class="moments-tape" />
          </div>

          <div class="moments-cover-stats">
            <span><strong>{{ events.length }}</strong> 段回忆</span>
            <span><strong>{{ totalImages }}</strong> 张照片</span>
            <span><strong>{{ recordedDays }}</strong> 个日子</span>
          </div>
          <button
            type="button"
            class="moments-cover-action"
            @click="router.push('/records/quick/photo')"
          >
            <AppIcon name="camera" :size="17" />
            收藏今天
          </button>
        </section>

        <section
          class="moments-ai-note"
          aria-label="AI 月度回顾"
        >
          <span class="moments-ai-tape" aria-hidden="true" />
          <div class="moments-ai-head">
            <AIResultBadge>AI 月度回顾</AIResultBadge>
            <span>{{ aiReview.evidenceCount }} 段回忆</span>
          </div>
          <h2>这个月，它们这样生活</h2>
          <p>{{ aiReview.body }}</p>
          <div class="moments-ai-footer">
            <small>整理于 {{ aiReview.generatedAt.slice(11, 16) }}</small>
            <button
              type="button"
              @click="showEvidence = !showEvidence"
            >
              {{ showEvidence ? '收起依据' : '查看整理依据' }}
              <AppIcon
                :name="showEvidence ? 'chevronDown' : 'chevronRight'"
                :size="14"
              />
            </button>
          </div>
        </section>

        <AIEvidencePanel
          v-if="showEvidence"
          :items="aiReview.evidence"
        />

        <section class="moments-album-section">
          <div class="moments-section-heading">
            <div>
              <span>OUR LITTLE STORIES</span>
              <h2>记忆画册</h2>
            </div>
            <small>{{ filteredEvents.length }} 个瞬间</small>
          </div>

          <div
            class="moments-filter-bar"
            role="tablist"
            aria-label="筛选时光类型"
          >
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
          </div>

          <div
            v-if="monthGroups.length"
            class="moments-months"
          >
            <section
              v-for="month in monthGroups"
              :key="month.key"
              class="moments-month"
            >
              <header class="moments-month-head">
                <div>
                  <span>{{ month.key.replace('-', ' / ') }}</span>
                  <h3>{{ month.label }}</h3>
                </div>
                <small>{{ month.items.length }} 个瞬间 · {{ month.images }} 张照片</small>
              </header>

              <div class="moments-memory-grid">
                <article
                  v-for="(event, index) in month.items"
                  :key="event.id"
                  class="moments-memory-card"
                  :class="[`type-${event.type}`, { featured: index === 0 }]"
                >
                  <div class="moments-memory-visual">
                    <span class="moments-memory-icon">
                      <AppIcon
                        :name="eventIcon(event.type)"
                        :size="index === 0 ? 32 : 25"
                      />
                    </span>
                    <span class="moments-memory-count">
                      <AppIcon name="photo" :size="12" />
                      {{ event.images || 0 }}
                    </span>
                    <span class="moments-memory-date">{{ formatDay(event.date) }}</span>
                  </div>
                  <div class="moments-memory-copy">
                    <span>{{ eventLabel(event.type) }}</span>
                    <h4>{{ event.title }}</h4>
                    <p v-if="event.body">{{ event.body }}</p>
                    <small>{{ catName(event.catId) }}</small>
                  </div>
                </article>
              </div>
            </section>
          </div>

          <div v-else class="moments-empty">
            <AppIcon name="camera" :size="34" />
            <strong>这个分类还没有回忆</strong>
            <span>记录一张照片，让故事从今天开始。</span>
            <button type="button" @click="router.push('/records/quick/photo')">收藏今天</button>
          </div>
        </section>
      </template>
    </main>
  </AppShell>
</template>
