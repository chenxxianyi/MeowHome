<script setup lang="ts">
import { ref, onMounted } from 'vue'
import AppShell from '../../components/app/AppShell.vue'
import AppIcon from '../../components/app/AppIcon.vue'
import { useReminderStore } from '../../stores/reminder'
import { services } from '../../services'
import type { Reminder } from '../../types'

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

function complete(id: string) {
  store.complete(id)
  const item = reminders.value.find((r) => r.id === id)
  if (item) item.state = 'done'
}

onMounted(load)

const todayTodo = () => reminders.value.filter((r) => r.state === 'todo')
const todayDone = () => reminders.value.filter((r) => r.state === 'done')
</script>

<template>
  <AppShell>
    <div class="page-header">
      <div class="page-title">
        提醒中心
      </div>
    </div>

    <div class="page-content">
      <!-- 筛选：待处理 / 已完成 / 全部 —— 与 demo 一致 -->
      <div
        style="display:flex;gap:8px;margin-bottom:16px;"
        role="tablist"
        aria-label="提醒筛选"
      >
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
      </div>

      <!-- 列表 -->
      <div
        v-if="loading"
        aria-busy="true"
      >
        <div
          v-for="i in 4"
          :key="i"
          class="skeleton"
          style="height:60px;margin-bottom:12px;"
        />
      </div>

      <div
        v-else-if="reminders.length === 0"
        class="empty-state"
      >
        <AppIcon
          name="bell"
          :size="40"
        />
        <div class="empty-state-title">
          暂无提醒
        </div>
        <div class="empty-state-desc">
          当前没有{{ store.filter === 'todo' ? '待处理' : '已' }}的提醒
        </div>
      </div>

      <div
        v-else
        class="reminder-list"
      >
        <div
          v-for="item in reminders"
          :key="item.id"
          class="reminder-item"
        >
          <div class="reminder-left">
            <div
              class="reminder-icon"
              :class="item.icon"
            >
              <AppIcon
                :name="(item.icon as any) === 'water' ? 'water' : (item.icon as any)"
                :size="18"
              />
            </div>
            <div class="reminder-text">
              <div class="reminder-title">
                {{ item.title }}
              </div>
              <div class="reminder-subtitle">
                {{ item.subtitle }} · {{ item.time }}
              </div>
            </div>
          </div>
          <template v-if="item.state === 'done'">
            <span class="ai-badge">已完成</span>
          </template>
          <template v-else>
            <div class="reminder-actions">
              <button class="reminder-btn">
                稍后
              </button>
              <button
                class="reminder-btn primary"
                @click="complete(item.id)"
              >
                完成
              </button>
            </div>
          </template>
        </div>
      </div>
    </div>
  </AppShell>
</template>
