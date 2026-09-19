<script setup lang="ts">
import { ref } from 'vue'
import AppIcon from '../../components/app/AppIcon.vue'
import { useReminderStore } from '../../stores/reminder'
import { services } from '../../services'
import type { Reminder } from '../../types'
import { useProtectedPage } from '../../utils/page'

const store = useReminderStore()
const reminders = ref<Reminder[]>([])
const loading = ref(true)

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
      <view v-for="item in reminders" :key="item.id" class="reminder-item">
        <view class="reminder-left">
          <view class="reminder-icon" :class="item.icon">
            <AppIcon :name="(item.icon as any) === 'water' ? 'water' : (item.icon as any)" :size="18" />
          </view>
          <view class="reminder-text">
            <view class="reminder-title">
              {{ item.title }}
            </view>
            <view class="reminder-subtitle"> {{ item.subtitle }} · {{ item.time }} </view>
          </view>
        </view>
        <template v-if="item.state === 'done'">
          <text class="ai-badge">已完成</text>
        </template>
        <template v-else>
          <view class="reminder-actions">
            <button class="reminder-btn" @click="later">稍后</button>
            <button class="reminder-btn primary" @click="complete(item.id)">完成</button>
          </view>
        </template>
      </view>
    </view>
  </view>
</template>
