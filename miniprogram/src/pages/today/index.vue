<script setup lang="ts">
import { computed, ref } from 'vue'
import { useRouter } from '../../utils/navigation'
import AppIcon from '../../components/app/AppIcon.vue'
import CatSwitcher from '../../components/cat/CatSwitcher.vue'
import AIResultBadge from '../../components/ai/AIResultBadge.vue'
import AIEvidencePanel from '../../components/ai/AIEvidencePanel.vue'
import OfflineBanner from '../../components/app/OfflineBanner.vue'
import { useAppStore } from '../../stores/app'
import { useCatStore } from '../../stores/cat'
import { services } from '../../services'
import { greeting, todayDateLabel } from '../../utils/date'
import { usePageCapabilities, useProtectedPage } from '../../utils/page'
import type { FocusItemData, Reminder, TodayStatusData } from '../../types'

const router = useRouter()
const app = useAppStore()
const catStore = useCatStore()

interface StatusRow {
  icon: string
  label: string
  value: string
  state: string
  badge: string
}

interface StatusGroup {
  catId: string
  catName: string
  avatar?: string
  rows: StatusRow[]
}

const statusGroups = ref<StatusGroup[]>([])
const focusItems = ref<FocusItemData[]>([])
const aiSummary = ref<any>(null)
const reminderList = ref<Reminder[]>([])
const loading = ref(true)
const loadError = ref('')
const showEvidence = ref(false)

const currentCat = computed(() => catStore.currentCatId || 'all')
const now = new Date()
const todayNumber = now.getDate()
const todayMonth = `${now.getMonth() + 1}月`

const dayOfYear = computed(() => {
  const start = new Date(now.getFullYear(), 0, 0)
  return Math.floor((now.getTime() - start.getTime()) / 86400000)
})

const attentionRows = computed(() =>
  statusGroups.value.flatMap((group) =>
    group.rows
      .filter((row) => row.state === 'warning' || row.state === 'danger')
      .map((row) => ({ ...row, catId: group.catId, catName: group.catName }))
  )
)

const dangerCount = computed(() => attentionRows.value.filter((row) => row.state === 'danger').length)
const visibleFocusItems = computed(() =>
  currentCat.value === 'all' ? focusItems.value : focusItems.value.filter((item) => item.catId === currentCat.value)
)
const visibleReminders = computed(() =>
  currentCat.value === 'all'
    ? reminderList.value
    : reminderList.value.filter((item) => item.catId === currentCat.value || item.catId === 'both')
)

const heroTitle = computed(() => {
  if (dangerCount.value)
    return `${attentionRows.value.find((row) => row.state === 'danger')?.catName || '家里'}需要多看一眼`
  if (attentionRows.value.length) return '今天有几件事要记得'
  return '今天的状态很不错'
})

const heroDescription = computed(() => {
  if (!attentionRows.value.length) return '饮食、饮水和精神状态都保持稳定，继续按平时的节奏照顾就好。'
  const summary = attentionRows.value
    .slice(0, 2)
    .map((row) => `${row.catName}${row.value}`)
    .join('；')
  return `${summary}${attentionRows.value.length > 2 ? `，另有 ${attentionRows.value.length - 2} 项待处理。` : '。'}`
})

const quickRecords = [
  { id: 'feeding', label: '喂食', icon: 'food' },
  { id: 'elimination', label: '排便', icon: 'elimination' },
  { id: 'vomit', label: '呕吐', icon: 'vomit' },
  { id: 'weight', label: '体重', icon: 'weight' },
  { id: 'more', label: '更多', icon: 'more' }
]

function badgeOf(state: string, warnText: string): string {
  if (state === 'normal') return '正常'
  if (state === 'danger') return '需关注'
  if (state === 'warning') return warnText
  return ''
}

function buildRows(status: TodayStatusData): StatusRow[] {
  return [
    {
      icon: 'food',
      label: '饮食',
      value: status.food.label,
      state: status.food.state,
      badge: badgeOf(status.food.state, '偏低')
    },
    {
      icon: 'water',
      label: '饮水',
      value: status.water.label,
      state: status.water.state,
      badge: badgeOf(status.water.state, '偏低')
    },
    {
      icon: 'elimination',
      label: '排便',
      value: status.elimination.label,
      state: status.elimination.state,
      badge: badgeOf(status.elimination.state, '注意')
    },
    {
      icon: 'vomit',
      label: '呕吐',
      value: status.vomit.state === 'none' ? '无呕吐' : status.vomit.label,
      state: status.vomit.state,
      badge: badgeOf(status.vomit.state, '注意')
    },
    {
      icon: 'medication',
      label: '用药',
      value:
        status.medication.state === 'none'
          ? '无需用药'
          : status.medication.label + (status.medication.time ? ` · ${status.medication.time}` : ''),
      state: status.medication.state,
      badge: status.medication.state === 'none' ? '' : status.medication.state === 'warning' ? '待办' : '需关注'
    },
    {
      icon: 'mental',
      label: '精神',
      value: status.mental.label,
      state: status.mental.state,
      badge: badgeOf(status.mental.state, '注意')
    }
  ]
}

function groupAttention(group: StatusGroup) {
  return group.rows.filter((row) => row.state === 'warning' || row.state === 'danger')
}

function groupNormal(group: StatusGroup) {
  return group.rows.filter((row) => row.state === 'normal' || row.state === 'none')
}

function groupLevel(group: StatusGroup) {
  if (group.rows.some((row) => row.state === 'danger')) return 'danger'
  if (group.rows.some((row) => row.state === 'warning')) return 'warning'
  return 'normal'
}

async function load() {
  loading.value = true
  loadError.value = ''
  try {
    let statusSuccess = true
    if (currentCat.value === 'all') {
      const cats = catStore.cats
      const results = await Promise.all(cats.map((cat) => services.getTodayStatus(cat.id)))
      statusSuccess = results.every((result) => result.success)
      statusGroups.value = cats.map((cat, index) => ({
        catId: cat.id,
        catName: cat.name,
        avatar: cat.avatar,
        rows: buildRows(results[index].data)
      }))
    } else {
      const res = await services.getTodayStatus(currentCat.value)
      statusSuccess = res.success
      const cat = catStore.cats.find((item) => item.id === currentCat.value)
      statusGroups.value = [
        {
          catId: currentCat.value,
          catName: cat?.name || '',
          avatar: cat?.avatar,
          rows: buildRows(res.data)
        }
      ]
    }

    const [focusRes, aiRes, reminderRes] = await Promise.all([
      services.getFocusItems(),
      services.getAISummary(),
      services.getReminders('todo')
    ])
    focusItems.value = focusRes.data
    aiSummary.value = aiRes.data
    reminderList.value = reminderRes.data.filter((item: Reminder) => !app.completedReminders.includes(item.id))
    if (!statusSuccess && !focusRes.success && !aiRes.success && !reminderRes.success) {
      loadError.value = '今日数据暂时无法加载，请稍后重试'
    }
  } catch {
    statusGroups.value = []
    loadError.value = '今日数据暂时无法加载，请稍后重试'
  } finally {
    loading.value = false
  }
}

function onCatSelect(catId: string) {
  app.setCurrentCat(catId)
  load()
}

function handleFocusAction(item: FocusItemData, action: string) {
  if (action === 'ai') router.push('/records/ai')
  else if (action === 'view' || action === 'observe') router.push('/records')
  else if (action === 'add') router.push('/records/quick/vomit')
  else if (action === 'done') {
    focusItems.value = focusItems.value.filter((focus) => focus.id !== item.id)
    showToast('已标记完成')
  } else if (action === 'later') {
    showToast('已设置稍后提醒（1 小时后）')
  } else if (action === 'plan') {
    router.push('/reminders')
  }
}

function handleQuickRecord(id: string) {
  if (id === 'more') router.push('/records')
  else router.push(`/records/quick/${id}`)
}

async function completeReminder(id: string) {
  const result = await services.completeReminder(id)
  if (!result.success) {
    showToast('完成失败，请稍后重试')
    return
  }
  app.completedReminders.push(id)
  reminderList.value = reminderList.value.filter((item) => item.id !== id)
  showToast('已完成')
}

function laterReminder(id: string) {
  showToast('稍后提醒功能暂未开放')
}

function showToast(message: string) {
  uni.showToast({ title: message, icon: 'none' })
}

useProtectedPage(load)
usePageCapabilities(load, '猫宅 · 今日照顾简报')
</script>

<template>
  <OfflineBanner />
  <view class="page-header today-page-header">
    <view class="today-masthead">
      <view class="today-masthead-brand">
        <AppIcon name="catEar" :size="17" />
        MEOWHOME DAILY
      </view>
      <text>NO. {{ dayOfYear }}</text>
    </view>
    <view class="today-header-main">
      <view>
        <view class="greet-title">
          {{ greeting() }}
        </view>
        <view class="greet-sub">
          {{ todayDateLabel() }}
        </view>
      </view>
      <button type="button" class="today-reminder-button" aria-label="打开提醒中心" @click="router.push('/reminders')">
        <AppIcon name="bell" :size="19" />
        <text v-if="visibleReminders.length">{{ visibleReminders.length }}</text>
      </button>
    </view>
    <CatSwitcher @select="onCatSelect" />
  </view>

  <view class="page-content today-dashboard">
    <view v-if="loading" class="today-loading" aria-label="正在加载今日简报" aria-busy="true">
      <view class="skeleton today-cover-skeleton" />
      <view class="skeleton today-actions-skeleton" />
      <view class="skeleton today-cats-skeleton" />
    </view>

    <view v-else-if="loadError" class="empty-state">
      <AppIcon name="alertTriangle" :size="30" />
      <view>{{ loadError }}</view>
      <button class="btn-secondary" @click="load">重新加载</button>
    </view>

    <template v-else>
      <view class="today-cover" :class="{ 'has-danger': dangerCount }" aria-label="今日照顾简报">
        <view class="today-cover-date" aria-hidden="true">
          <text>{{ todayMonth }}</text>
          <text>{{ todayNumber }}</text>
        </view>
        <view class="today-cover-copy">
          <text class="today-cover-kicker">TODAY'S CARE BRIEFING</text>
          <view>{{ heroTitle }}</view>
          <view>{{ heroDescription }}</view>
        </view>
        <view class="today-cover-metrics">
          <text
            ><text>{{ statusGroups.length }}</text> 只猫咪</text
          >
          <text
            ><text>{{ attentionRows.length }}</text> 项关注</text
          >
          <text
            ><text>{{ visibleReminders.length }}</text> 个待办</text
          >
        </view>
        <view class="today-cover-actions">
          <button type="button" class="today-cover-primary" @click="router.push('/records')">
            <AppIcon name="record" :size="17" />
            记一条动态
          </button>
          <button type="button" class="today-cover-secondary" @click="router.push('/reminders')">
            查看今日安排
            <AppIcon name="chevronRight" :size="15" />
          </button>
        </view>
      </view>

      <view class="today-quick-section" aria-label="快捷记录">
        <view class="today-quick-label">
          <text>QUICK NOTE</text>
          快速记一笔
        </view>
        <view class="today-quick-grid">
          <button
            v-for="button in quickRecords"
            :key="button.id"
            type="button"
            class="today-quick-button"
            @click="handleQuickRecord(button.id)"
          >
            <view>
              <AppIcon :name="button.icon" :size="19" />
            </view>
            {{ button.label }}
          </button>
        </view>
      </view>

      <view v-if="statusGroups.length" class="today-section" aria-label="猫咪今日状态">
        <view class="today-section-heading">
          <view>
            <text>AT A GLANCE</text>
            <view>今日状态</view>
          </view>
          <view>只突出需要处理的变化</view>
        </view>

        <view class="today-cat-deck" :class="{ 'is-single': statusGroups.length === 1 }">
          <view
            v-for="(group, index) in statusGroups"
            :key="group.catId"
            class="today-cat-card"
            :class="[`tone-${index % 2}`, `state-${groupLevel(group)}`]"
          >
            <view class="today-cat-card-head">
              <view class="today-cat-avatar">
                <image mode="aspectFill" v-if="group.avatar" :src="group.avatar" :alt="group.catName" />
                <AppIcon v-else name="cat" :size="24" />
              </view>
              <view>
                <text>{{ group.catName }}</text>
                <view>{{
                  groupAttention(group).length ? `${groupAttention(group).length} 项需要留意` : '状态稳定'
                }}</view>
              </view>
              <text class="today-cat-level" />
            </view>

            <view v-if="groupAttention(group).length" class="today-cat-alerts">
              <view
                v-for="row in groupAttention(group).slice(0, 2)"
                :key="row.icon"
                class="today-cat-alert"
                :class="row.state"
              >
                <AppIcon :name="row.icon as any" :size="16" />
                <view>
                  <text>{{ row.label }}</text>
                  <view>{{ row.value }}</view>
                </view>
              </view>
            </view>
            <view v-else class="today-cat-all-good">
              <AppIcon name="success" :size="20" />
              饮食、饮水和精神状态都很好
            </view>

            <view class="today-cat-signals">
              <text v-for="row in groupNormal(group)" :key="row.icon" :title="row.value">
                <text />{{ row.label }}
              </text>
            </view>
            <button type="button" class="today-cat-profile" @click="router.push(`/cats/${group.catId}`)">
              查看健康档案
              <AppIcon name="chevronRight" :size="14" />
            </button>
          </view>
        </view>
      </view>

      <view v-if="visibleFocusItems.length" class="today-section" aria-label="需要关注">
        <view class="today-section-heading">
          <view>
            <text>KEEP AN EYE ON</text>
            <view>需要关注</view>
          </view>
        </view>

        <view v-for="item in visibleFocusItems" :key="item.id" class="today-focus-card" :class="item.severity">
          <view class="today-focus-icon">
            <AppIcon :name="item.severity === 'danger' ? 'alertTriangle' : 'bell'" :size="20" />
          </view>
          <view class="today-focus-copy">
            <text>{{ item.severity === 'danger' ? '异常变化' : '今日待办' }}</text>
            <view>{{ item.title }}</view>
            <view>{{ item.body }}</view>
            <details v-if="item.evidence.length" class="today-focus-evidence">
              <summary>查看提示依据</summary>
              <view v-for="evidence in item.evidence" :key="evidence">{{ evidence }}</view>
            </details>
            <view class="today-focus-actions">
              <button
                v-for="action in item.actions"
                :key="action.action"
                type="button"
                :class="{ primary: action.action === 'ai' || action.action === 'done' }"
                @click="handleFocusAction(item, action.action)"
              >
                {{ action.label }}
              </button>
            </view>
          </view>
        </view>
      </view>

      <view v-if="aiSummary" class="today-ai-brief" aria-label="AI 每日摘要">
        <view class="today-ai-head">
          <AIResultBadge>AI 每日整理</AIResultBadge>
          <text>基于 {{ aiSummary.evidenceCount }} 条记录</text>
        </view>
        <view>管家的今日小结</view>
        <view>{{ aiSummary.body }}</view>
        <view class="today-ai-footer">
          <view>生成于 {{ aiSummary.generatedAt.slice(11, 16) }} · 不构成医疗诊断</view>
          <button type="button" @click="showEvidence = !showEvidence">
            {{ showEvidence ? '收起依据' : '查看依据' }}
            <AppIcon :name="showEvidence ? 'chevronDown' : 'chevronRight'" :size="14" />
          </button>
        </view>
      </view>

      <AIEvidencePanel v-if="showEvidence && aiSummary" :items="aiSummary.evidence" />

      <view v-if="visibleReminders.length" class="today-section today-reminders-section" aria-label="今日提醒">
        <view class="today-section-heading">
          <view>
            <text>TO-DO TODAY</text>
            <view>今日提醒</view>
          </view>
          <button type="button" @click="router.push('/reminders')">
            全部
            <AppIcon name="chevronRight" :size="14" />
          </button>
        </view>
        <view class="today-reminder-list">
          <view v-for="item in visibleReminders.slice(0, 4)" :key="item.id" class="today-reminder-item">
            <text class="today-reminder-time">{{ item.time }}</text>
            <view class="today-reminder-icon">
              <AppIcon :name="item.icon || item.type" :size="18" />
            </view>
            <view class="today-reminder-copy">
              <text>{{ item.title }}</text>
              <view>{{ item.subtitle }}</view>
            </view>
            <view class="today-reminder-actions">
              <button type="button" @click="laterReminder(item.id)">稍后</button>
              <button type="button" class="primary" @click="completeReminder(item.id)">完成</button>
            </view>
          </view>
        </view>
      </view>
    </template>
  </view>
</template>

<style scoped>
.empty-state {
  padding: 64px 20px;
  text-align: center;
  color: var(--color-text-secondary);
}
.empty-state .btn-secondary {
  margin-top: var(--space-16);
}
</style>
