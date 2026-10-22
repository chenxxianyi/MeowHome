<script setup lang="ts">
import { computed, ref } from 'vue'
import AppIcon from '../../components/app/AppIcon.vue'
import { useReminderStore } from '../../stores/reminder'
import { services } from '../../services'
import type { Reminder } from '../../types'
import { useProtectedPage } from '../../utils/page'
import { dateAtTimezone, deviceTimezone, formatZonedDateTime, todayStr } from '../../utils/date'
import { useRouter } from '../../utils/navigation'

const store = useReminderStore()
const router = useRouter()
const reminders = ref<Reminder[]>([])
const loading = ref(true)
type ReminderGroupKey = 'overdue' | 'today' | 'upcoming' | 'unscheduled' | 'done'
const groupLabels: Record<ReminderGroupKey, string> = {
  overdue: '计划日期已过', today: '今天到期', upcoming: '即将到期', unscheduled: '未设置计划日期', done: '已完成'
}

function groupKey(item: Reminder): ReminderGroupKey {
  if (item.state === 'done') return 'done'
  if (!item.scheduled_at) return 'unscheduled'
  const dueDate = dateAtTimezone(item.scheduled_at, item.timezone)
  const today = dateAtTimezone(new Date().toISOString(), item.timezone)
  if (dueDate < today) return 'overdue'
  if (dueDate === today) return 'today'
  return 'upcoming'
}

const sections = computed(() => {
  const order: ReminderGroupKey[] = ['overdue', 'today', 'upcoming', 'unscheduled', 'done']
  return order.map((key) => ({
    key,
    title: groupLabels[key],
    items: reminders.value
      .filter((item) => groupKey(item) === key)
      .sort((a, b) => {
        if (key === 'done') return (b.completed_at || '').localeCompare(a.completed_at || '')
        const aDate = a.scheduled_at ? dateAtTimezone(a.scheduled_at, a.timezone) : '9999-99-99'
        const bDate = b.scheduled_at ? dateAtTimezone(b.scheduled_at, b.timezone) : '9999-99-99'
        return aDate.localeCompare(bDate)
      })
  })).filter((section) => section.items.length > 0)
})

async function load() {
  loading.value = true
  const res = await services.getReminders(store.filter)
  reminders.value = res.data
  loading.value = false
}

function onFilterChange(f: 'todo' | 'done' | 'all') {
  store.setFilter(f)
  load()
}

async function complete(id: string) {
  const result = await services.completeReminder(id)
  if (!result.success) {
    uni.showToast({ title: '操作失败，请稍后重试', icon: 'none' })
    return
  }
  store.complete(id)
  const item = reminders.value.find((r) => r.id === id)
  if (item) item.state = 'done'
  if (store.filter === 'todo') reminders.value = reminders.value.filter((r) => r.id !== id)
  uni.showToast({ title: '已完成', icon: 'success' })
}

function later() {
  uni.showToast({ title: '稍后提醒功能暂未开放', icon: 'none' })
}

function plannedDate(item: Reminder): string {
  return item.scheduled_at ? formatZonedDateTime(item.scheduled_at, item.timezone) : item.time
}

function openVaccineRecord(item: Reminder) {
  const query = `catId=${encodeURIComponent(item.catId || '')}&reminderId=${encodeURIComponent(item.id)}`
  router.push(`/records/quick/vaccine?${query}`)
}

function changeSchedule(item: Reminder, event: unknown) {
  const date = (event as { detail?: { value?: string } }).detail?.value
  if (!date || typeof date !== 'string') return
  const timezone = item.timezone || deviceTimezone()
  uni.showModal({
    title: '调整疫苗待办日期',
    content: `将计划日期调整为 ${date}？`,
    success: async ({ confirm }) => {
      if (!confirm) return
      const result = await services.rescheduleVaccineReminder(item.id, date, timezone)
      if (!result.success) {
        uni.showToast({ title: '日期未更新，请选择未来日期后重试', icon: 'none' })
        return
      }
      await load()
      uni.showToast({ title: '日期已更新', icon: 'success' })
    }
  })
}

function cancelVaccine(item: Reminder) {
  uni.showModal({
    title: '取消疫苗待办',
    content: '取消后不会再显示这条提醒。接种记录仍会保留。',
    success: async ({ confirm }) => {
      if (!confirm) return
      const result = await services.cancelVaccineReminder(item.id)
      if (!result.success) {
        uni.showToast({ title: '取消失败，请稍后重试', icon: 'none' })
        return
      }
      reminders.value = reminders.value.filter((row) => row.id !== item.id)
      uni.showToast({ title: '待办已取消', icon: 'success' })
    }
  })
}

function openCompletedRecord(item: Reminder) {
  if (item.completed_record_id) router.push(`/records/detail/${encodeURIComponent(item.completed_record_id)}`)
}

useProtectedPage(load)

const todayTodo = () => reminders.value.filter((r) => r.state === 'todo')
const todayDone = () => reminders.value.filter((r) => r.state === 'done')
</script>

<template>
  <view class="page-header">
    <view class="page-title"> 提醒中心 </view>
  </view>

  <view class="page-content">
    <!-- 筛选：待处理 / 已完成 / 全部 —— 与 demo 一致 -->
    <view style="display: flex; gap: 8px; margin-bottom: 16px" role="tablist" aria-label="提醒筛选">
      <button
        class="reminder-btn"
        :class="{ active: store.filter === 'todo' }"
        role="tab"
        :aria-selected="store.filter === 'todo'"
        @click="onFilterChange('todo')"
      >
        待处理 ({{ todayTodo().length }})
      </button>
      <button
        class="reminder-btn"
        :class="{ active: store.filter === 'done' }"
        role="tab"
        :aria-selected="store.filter === 'done'"
        @click="onFilterChange('done')"
      >
        已完成 ({{ todayDone().length }})
      </button>
      <button
        class="reminder-btn"
        :class="{ active: store.filter === 'all' }"
        role="tab"
        :aria-selected="store.filter === 'all'"
        @click="onFilterChange('all')"
      >
        全部 ({{ reminders.length }})
      </button>
    </view>

    <!-- 列表 -->
    <view v-if="loading" aria-busy="true">
      <view v-for="i in 4" :key="i" class="skeleton" style="height: 60px; margin-bottom: 12px" />
    </view>

    <view v-else-if="reminders.length === 0" class="empty-state">
      <AppIcon name="bell" :size="40" />
      <view class="empty-state-title"> 暂无提醒 </view>
      <view class="empty-state-desc"> 当前没有{{ store.filter === 'todo' ? '待处理' : '已' }}的提醒 </view>
    </view>

    <view v-else class="reminder-list">
      <view v-for="section in sections" :key="section.key" class="reminder-section">
        <view class="reminder-section-title">{{ section.title }}</view>
        <view v-for="item in section.items" :key="item.id" class="reminder-item">
          <view class="reminder-main">
            <view class="reminder-left">
              <view class="reminder-icon" :class="item.icon">
                <AppIcon :name="(item.icon as any) === 'water' ? 'water' : (item.icon as any)" :size="18" />
              </view>
              <view class="reminder-text">
                <view class="reminder-title">{{ item.title }}</view>
                <view class="reminder-subtitle">{{ item.subtitle }} · {{ plannedDate(item) }}</view>
                <view v-if="item.type === 'vaccine' && item.state === 'todo'" class="reminder-disclaimer">
                  逾期仅表示计划日期已过，不代表猫咪漏种。
                </view>
              </view>
            </view>
            <text v-if="item.state === 'done'" class="ai-badge">已完成</text>
            <text v-else class="due-badge" :class="`due-${groupKey(item)}`">{{ groupLabels[groupKey(item)] }}</text>
          </view>
          <view v-if="item.state === 'done'" class="reminder-actions">
            <button v-if="item.completed_record_id" class="reminder-btn" @click="openCompletedRecord(item)">查看接种记录</button>
          </view>
          <view v-else-if="item.type === 'vaccine'" class="reminder-actions">
            <button class="reminder-btn primary" @click="openVaccineRecord(item)">记录接种并完成</button>
            <view class="reminder-action-picker">
              <picker mode="date" :value="item.scheduled_at ? dateAtTimezone(item.scheduled_at, item.timezone) : todayStr()" @change="changeSchedule(item, $event)">
                <button class="reminder-btn">调整日期</button>
              </picker>
            </view>
            <button class="reminder-btn" @click="cancelVaccine(item)">取消</button>
          </view>
          <view v-else class="reminder-actions">
            <button class="reminder-btn" @click="later">稍后</button>
            <button class="reminder-btn primary" @click="complete(item.id)">完成</button>
          </view>
        </view>
      </view>
    </view>
  </view>
</template>

<style scoped>
.reminder-list { display: flex; flex-direction: column; gap: 20px; }
.reminder-section { display: flex; flex-direction: column; gap: 10px; }
.reminder-section-title { font-weight: 600; color: var(--color-text-primary); }
.reminder-item { display: flex; flex-direction: column; gap: 12px; padding: 14px; border: 1px solid var(--color-divider); border-radius: 12px; background: var(--color-bg-surface); }
.reminder-main { display: flex; justify-content: space-between; align-items: flex-start; gap: 12px; }
.reminder-left { display: flex; gap: 10px; min-width: 0; }
.reminder-text { min-width: 0; }
.reminder-title { font-weight: 600; }
.reminder-subtitle, .reminder-disclaimer { color: var(--color-text-secondary); font-size: var(--font-size-assist); margin-top: 4px; }
.reminder-actions { display: flex; align-items: center; flex-wrap: wrap; gap: 8px; }
.reminder-actions > .reminder-btn,
.reminder-action-picker .reminder-btn {
  display: flex;
  align-items: center;
  justify-content: center;
  box-sizing: border-box;
  height: 44px;
  min-height: 44px;
  margin: 0;
  line-height: 1.2;
  text-align: center;
  white-space: nowrap;
}
.reminder-action-picker {
  display: flex;
  align-items: center;
  flex: none;
  height: 44px;
}
.reminder-action-picker picker {
  display: flex;
  align-items: center;
  height: 44px;
}
.due-badge { font-size: var(--font-size-assist); white-space: nowrap; }
.due-overdue { color: var(--color-danger); }
.due-today { color: var(--color-warning); }
</style>
