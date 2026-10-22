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
import { useAgentStore } from '../../stores/agent'
import { services } from '../../services'
import { dateAtTimezone, deviceTimezone, formatZonedDateTime, greeting, todayDateLabel, todayStr } from '../../utils/date'
import { buildMonthGrid, shiftMonth } from '../../utils/recordCalendar'
import { usePageCapabilities, useProtectedPage } from '../../utils/page'
import type { FocusItemData, Reminder, TodayStatusData } from '../../types'

const router = useRouter()
const app = useAppStore()
const catStore = useCatStore()
const agentStore = useAgentStore()
const agentMessages = computed(() => agentStore.important.slice(0, 2))

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
const calendarToday = ref(todayStr())
const reminderCalendarMonth = ref(calendarToday.value.slice(0, 7))
const selectedReminderDate = ref(calendarToday.value)
const reminderWeekdays = ['一', '二', '三', '四', '五', '六', '日']

const currentCat = computed(() => catStore.currentCatId || 'all')
const now = new Date()
const todayNumber = now.getDate()
const todayMonth = `${now.getMonth() + 1}月`

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
  (currentCat.value === 'all'
    ? reminderList.value
    : reminderList.value.filter((item) => item.catId === currentCat.value || item.catId === 'both'))
    .slice()
    .sort((a, b) => reminderOrder(a) - reminderOrder(b) || reminderSortDate(a).localeCompare(reminderSortDate(b)))
)
const remindersByDate = computed(() => {
  const reminders = new Map<string, Reminder[]>()
  for (const item of visibleReminders.value) {
    const date = reminderCalendarDate(item)
    if (!date) continue
    const items = reminders.get(date) || []
    items.push(item)
    reminders.set(date, items)
  }
  return reminders
})
const reminderCalendarCells = computed(() => buildMonthGrid(reminderCalendarMonth.value).map((cell) => {
  if (!cell) return null
  return {
    ...cell,
    count: remindersByDate.value.get(cell.date)?.length || 0
  }
}))
const reminderCalendarMonthLabel = computed(() => {
  const [year, month] = reminderCalendarMonth.value.split('-')
  return `${year}年${Number(month)}月`
})
const selectedDateReminders = computed(() => remindersByDate.value.get(selectedReminderDate.value) || [])
const unscheduledReminders = computed(() => visibleReminders.value.filter((item) => !item.scheduled_at))
const selectedReminderDateLabel = computed(() => {
  const [, month, day] = selectedReminderDate.value.split('-').map(Number)
  return selectedReminderDate.value === calendarToday.value ? `今天 · ${month}月${day}日` : `${month}月${day}日`
})

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
  { id: 'vaccine', label: '疫苗', icon: 'vaccine' },
  { id: 'more', label: '更多', icon: 'more' }
]

function reminderSortDate(item: Reminder): string {
  return item.scheduled_at ? dateAtTimezone(item.scheduled_at, item.timezone || deviceTimezone()) : '9999-99-99'
}

function reminderCalendarDate(item: Reminder): string {
  return item.scheduled_at ? dateAtTimezone(item.scheduled_at, item.timezone || deviceTimezone()) : ''
}

function reminderTimeLabel(item: Reminder): string {
  return item.scheduled_at ? formatZonedDateTime(item.scheduled_at, item.timezone || deviceTimezone()).slice(11) : item.time || '未设时间'
}

function reminderDueLabel(item: Reminder): string {
  if (!item.scheduled_at) return '未设置日期'
  const [, month, day] = reminderCalendarDate(item).split('-').map(Number)
  return `${month}月${day}日 ${reminderTimeLabel(item)}`
}

function moveReminderCalendarMonth(amount: number) {
  const nextMonth = shiftMonth(reminderCalendarMonth.value, amount)
  const [year, month] = nextMonth.split('-').map(Number)
  const maxDay = new Date(Date.UTC(year, month, 0)).getUTCDate()
  const selectedDay = Number(selectedReminderDate.value.slice(-2))
  reminderCalendarMonth.value = nextMonth
  selectedReminderDate.value = `${nextMonth}-${String(Math.min(selectedDay, maxDay)).padStart(2, '0')}`
}

function selectReminderDate(date: string) {
  selectedReminderDate.value = date
}

function goToReminderCalendarToday() {
  calendarToday.value = todayStr()
  reminderCalendarMonth.value = calendarToday.value.slice(0, 7)
  selectedReminderDate.value = calendarToday.value
}

function reminderOrder(item: Reminder): number {
  if (!item.scheduled_at) return 3
  const due = reminderSortDate(item)
  const today = dateAtTimezone(new Date().toISOString(), item.timezone || deviceTimezone())
  if (due < today) return 0
  if (due === today) return 1
  return 2
}

function reminderStatusLabel(item: Reminder): string {
  if (!item.scheduled_at) return '未设置日期'
  const due = reminderSortDate(item)
  const today = dateAtTimezone(new Date().toISOString(), item.timezone || deviceTimezone())
  if (due < today) return '计划日期已过'
  if (due === today) return '今天到期'
  return '即将到期'
}

function openVaccineRecord(item: Reminder) {
  router.push(`/records/quick/vaccine?catId=${encodeURIComponent(item.catId || '')}&reminderId=${encodeURIComponent(item.id)}`)
}

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
  const latestToday = todayStr()
  if (latestToday !== calendarToday.value) {
    calendarToday.value = latestToday
    reminderCalendarMonth.value = latestToday.slice(0, 7)
    selectedReminderDate.value = latestToday
  }
  void agentStore.load({ limit: 2 })
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

function showToast(message: string) {
  uni.showToast({ title: message, icon: 'none' })
}

useProtectedPage(load)
usePageCapabilities(load, '猫宅 · 今日照顾简报')
</script>

<template>
  <OfflineBanner />
  <view class="page-header today-page-header">
    <view class="today-header-main">
      <view>
        <view class="greet-title">
          {{ greeting() }}
        </view>
        <view class="greet-sub">
          {{ todayDateLabel() }}
        </view>
      </view>
      <view class="today-reminder-wrap">
        <button type="button" class="today-reminder-button" aria-label="打开提醒中心" @click="router.push('/reminders')">
          <AppIcon name="bell" :size="19" />
        </button>
        <text v-if="visibleReminders.length" class="today-reminder-badge" aria-hidden="true">{{ visibleReminders.length }}</text>
      </view>
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

      <view class="today-section today-reminders-section" aria-label="待办日历">
        <view class="today-section-heading">
          <view>
            <text>PLAN YOUR DAY</text>
            <view>待办日历</view>
          </view>
          <button type="button" @click="router.push('/reminders')">
            全部提醒（{{ visibleReminders.length }}）
            <AppIcon name="chevronRight" :size="14" />
          </button>
        </view>
        <view class="today-calendar-card">
          <view class="today-calendar-toolbar">
            <text class="today-calendar-month">{{ reminderCalendarMonthLabel }}</text>
            <view class="today-calendar-controls">
              <button type="button" class="today-calendar-today" @click="goToReminderCalendarToday">今天</button>
              <button type="button" class="today-calendar-nav" aria-label="上个月" @click="moveReminderCalendarMonth(-1)">‹</button>
              <button type="button" class="today-calendar-nav" aria-label="下个月" @click="moveReminderCalendarMonth(1)">›</button>
            </view>
          </view>

          <view class="today-calendar-grid today-calendar-weekdays" aria-hidden="true">
            <text v-for="weekday in reminderWeekdays" :key="weekday">{{ weekday }}</text>
          </view>
          <view class="today-calendar-grid today-calendar-days">
            <template v-for="(cell, index) in reminderCalendarCells" :key="cell?.date || `blank-${index}`">
              <button
                v-if="cell"
                type="button"
                class="today-calendar-day"
                :class="{
                  'is-today': cell.date === calendarToday,
                  'is-selected': cell.date === selectedReminderDate,
                  'has-reminders': cell.count > 0
                }"
                :aria-label="`${cell.date}，${cell.count}项待办${cell.date === calendarToday ? '，今天' : ''}`"
                :aria-pressed="cell.date === selectedReminderDate"
                @click="selectReminderDate(cell.date)"
              >
                <text class="today-calendar-day-number">{{ cell.day }}</text>
                <view v-if="cell.count" class="today-calendar-dot" />
              </button>
              <view v-else class="today-calendar-day today-calendar-day--blank" aria-hidden="true" />
            </template>
          </view>

          <view class="today-calendar-selected-heading">
            <text>{{ selectedReminderDateLabel }}</text>
            <text>{{ selectedDateReminders.length }} 项待办</text>
          </view>

          <view v-if="selectedDateReminders.length || visibleReminders.length" class="today-calendar-task-list">
            <text v-if="!selectedDateReminders.length" class="today-calendar-preview-label">
              {{ selectedReminderDate === calendarToday ? '今天暂无安排，最近待办' : '所选日期暂无安排，待办预览' }}
            </text>
            <view
              v-for="item in selectedDateReminders.length ? selectedDateReminders.slice(0, 4) : visibleReminders.slice(0, 2)"
              :key="item.id"
              class="today-calendar-task"
            >
              <text class="today-calendar-task-time">{{ selectedDateReminders.length ? reminderTimeLabel(item) : reminderDueLabel(item) }}</text>
              <view class="today-reminder-icon">
                <AppIcon :name="item.icon || item.type" :size="17" />
              </view>
              <view class="today-calendar-task-copy">
                <text>{{ item.title }}</text>
                <text>{{ item.subtitle }} · {{ reminderStatusLabel(item) }}</text>
              </view>
              <button
                v-if="item.type === 'vaccine'"
                type="button"
                class="today-calendar-task-action is-primary"
                @click="openVaccineRecord(item)"
              >接种</button>
              <button
                v-else
                type="button"
                class="today-calendar-task-action is-primary"
                @click="completeReminder(item.id)"
              >完成</button>
            </view>
          </view>

          <view v-else class="today-calendar-empty">
            <AppIcon name="calendar" :size="19" />
            <text>暂无待办事项，新的提醒会显示在这里</text>
          </view>

          <button v-if="unscheduledReminders.length" type="button" class="today-calendar-unscheduled" @click="router.push('/reminders')">
            {{ unscheduledReminders.length }} 项待办未设置日期
            <AppIcon name="chevronRight" :size="14" />
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

      <view v-if="agentMessages.length" class="today-section" aria-label="猫管家巡检消息">
        <view class="today-section-heading"><view class="records-section-title">猫管家提醒</view><button class="reminder-btn" @click="router.push('/agent')">查看全部</button></view>
        <button v-for="item in agentMessages" :key="item.id" class="agent-card" :class="`agent-${item.severity}`" @click="router.push(`/agent?messageId=${encodeURIComponent(item.id)}`)">
          <view class="agent-card-header"><text class="agent-card-title">{{ item.title }}</text><text class="ai-badge">{{ item.severity }}</text></view>
          <view class="agent-card-body">{{ item.body }}</view>
        </button>
      </view>
      <view v-if="statusGroups.length" class="today-section" aria-label="猫咪今日状态">
        <view class="today-section-heading">
          <view>
            <text>AT A GLANCE</text>
            <view>今日状态</view>
          </view>
          <text class="today-section-note">只突出需要处理的变化</text>
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
              <view class="today-cat-card-copy">
                <text class="today-cat-name">{{ group.catName }}</text>
                <text class="today-cat-status">{{
                  groupAttention(group).length ? `${groupAttention(group).length} 项需要留意` : '状态稳定'
                }}</text>
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
              <view v-for="row in groupNormal(group)" :key="row.icon" class="today-cat-signal" :title="row.value">
                <view class="today-cat-signal-dot" />
                <text>{{ row.label }}</text>
              </view>
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
        <view class="today-ai-subtitle">今日小结</view>
        <view class="today-ai-body">{{ aiSummary.body }}</view>
        <view class="today-ai-footer">
          <view>生成于 {{ aiSummary.generatedAt.slice(11, 16) }} · 不构成医疗诊断</view>
          <button type="button" @click="showEvidence = !showEvidence">
            {{ showEvidence ? '收起依据' : '查看依据' }}
            <AppIcon :name="showEvidence ? 'chevronDown' : 'chevronRight'" :size="14" />
          </button>
        </view>
      </view>

      <AIEvidencePanel v-if="showEvidence && aiSummary" :items="aiSummary.evidence" />



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
.today-calendar-card {
  padding: 14px 12px 10px;
  border: 1px solid var(--color-border);
  border-radius: 18px;
  background: var(--color-bg-surface);
  box-shadow: 0 7px 20px rgba(45, 41, 37, 0.05);
}
.today-calendar-toolbar,
.today-calendar-controls,
.today-calendar-selected-heading {
  display: flex;
  align-items: center;
  justify-content: space-between;
}
.today-calendar-toolbar { margin-bottom: 10px; }
.today-calendar-month { color: var(--color-text-primary); font-weight: 700; }
.today-calendar-controls { flex: none; gap: 8px; }
.today-calendar-today,
.today-calendar-nav {
  display: flex;
  align-items: center;
  justify-content: center;
  box-sizing: border-box;
  height: 44px;
  min-height: 44px;
  color: var(--color-text-secondary);
  border: 0;
  background: transparent;
  box-shadow: none;
}
.today-calendar-today {
  min-width: 44px;
  padding: 0 4px;
  color: var(--color-brand);
  font-size: 12px;
  font-weight: 700;
  line-height: 1;
  text-align: center;
  white-space: nowrap;
}
.today-calendar-nav {
  width: 44px;
  min-width: 44px;
  padding: 0;
  color: var(--color-text-secondary);
  font-size: 22px;
  line-height: 1;
}
.today-calendar-grid { display: grid; grid-template-columns: repeat(7, minmax(0, 1fr)); gap: 3px; }
.today-calendar-weekdays {
  margin-bottom: 3px;
  color: var(--color-text-tertiary);
  text-align: center;
  font-size: 10px;
}
.today-calendar-weekdays text { padding: 5px 0; }
.today-calendar-day {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: flex-start;
  gap: 2px;
  box-sizing: border-box;
  width: 100%;
  height: 40px;
  min-width: 0;
  padding: 4px 0 2px;
  border: 1px solid transparent;
  border-radius: 10px;
  color: var(--color-text-primary);
  background: transparent;
}
.today-calendar-day-number { font-size: 12px; line-height: 17px; }
.today-calendar-day.is-today { border-color: var(--color-brand); color: var(--color-brand); font-weight: 700; }
.today-calendar-day.is-selected { border-color: var(--color-brand); color: #fff; background: var(--color-brand); }
.today-calendar-day.is-selected.is-today { color: #fff; }
.today-calendar-dot { width: 4px; height: 4px; border-radius: 50%; background: var(--color-brand); }
.today-calendar-day.is-selected .today-calendar-dot { background: #fff; }
.today-calendar-day--blank { pointer-events: none; }
.today-calendar-selected-heading {
  gap: 8px;
  margin-top: 10px;
  padding: 10px 0 8px;
  border-top: 1px solid var(--color-divider);
}
.today-calendar-selected-heading text:first-child { color: var(--color-text-primary); font-weight: 700; }
.today-calendar-selected-heading text:last-child { color: var(--color-text-tertiary); font-size: var(--font-size-caption); }
.today-calendar-task-list { display: flex; flex-direction: column; }
.today-calendar-preview-label {
  padding: 4px 0 6px;
  color: var(--color-text-tertiary);
  font-size: var(--font-size-caption);
}
.today-calendar-task {
  display: grid;
  grid-template-columns: 40px 30px minmax(0, 1fr) auto;
  align-items: center;
  gap: 7px;
  min-width: 0;
  padding: 8px 0;
  border-top: 1px solid var(--color-divider);
}
.today-calendar-task-time { color: var(--color-text-tertiary); font-size: 10px; text-align: center; }
.today-calendar-task-copy { display: flex; flex-direction: column; min-width: 0; }
.today-calendar-task-copy text:first-child {
  overflow: hidden;
  color: var(--color-text-primary);
  font-size: var(--font-size-assist);
  font-weight: 600;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.today-calendar-task-copy text:last-child {
  overflow: hidden;
  color: var(--color-text-tertiary);
  font-size: 9px;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.today-calendar-task-action {
  display: flex;
  align-items: center;
  justify-content: center;
  box-sizing: border-box;
  width: 46px;
  height: 29px;
  padding: 0 5px;
  border-radius: 8px;
  color: var(--color-text-secondary);
  background: var(--color-bg-subtle);
  font-size: 10px;
  white-space: nowrap;
}
.today-calendar-task-action.is-primary { color: #fff; background: var(--color-brand); }
.today-calendar-empty {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 8px 0 4px;
  color: var(--color-text-tertiary);
  font-size: var(--font-size-caption);
}
.today-calendar-unscheduled {
  display: flex;
  align-items: center;
  justify-content: space-between;
  width: 100%;
  margin-top: 8px;
  padding: 9px 0 2px;
  border-top: 1px solid var(--color-divider);
  color: var(--color-brand);
  background: transparent;
  font-size: var(--font-size-caption);
  text-align: left;
}
</style>
