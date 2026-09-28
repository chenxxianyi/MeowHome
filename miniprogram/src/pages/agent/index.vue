<script setup lang="ts">
import { computed } from 'vue'
import { useAgentStore } from '../../stores/agent'
import { useProtectedPage } from '../../utils/page'

const store = useAgentStore()
const messages = computed(() => store.important)

useProtectedPage(() => store.load())

async function dismiss(id: string) {
  try {
    await store.dismiss(id)
    uni.showToast({ title: '已忽略', icon: 'success' })
  } catch {
    uni.showToast({ title: '操作失败，请稍后重试', icon: 'none' })
  }
}

async function confirm(item: (typeof messages.value)[number]) {
  if (item.draft_version === undefined) return
  try {
    await store.confirm(item.id, item.draft_version)
    uni.showToast({ title: '提醒已创建', icon: 'success' })
  } catch {
    uni.showToast({ title: '草稿已变化，请重新打开', icon: 'none' })
  }
}
</script>

<template>
  <view class="page-header">
    <view class="page-title">猫管家</view>
    <view class="page-subtitle">基于家庭真实记录的巡检提示</view>
  </view>
  <view class="page-content">
    <view v-if="store.loading" class="empty-state">正在读取巡检消息…</view>
    <view v-else-if="store.error" class="empty-state">
      <view class="empty-state-title">暂时无法读取</view>
      <view class="empty-state-desc">{{ store.error }}</view>
      <button class="reminder-btn primary" @click="store.load()">重试</button>
    </view>
    <view v-else-if="messages.length === 0" class="empty-state">
      <view class="empty-state-title">暂无新的巡检提示</view>
      <view class="empty-state-desc">有新的记录后，猫管家会在这里展示可核对的提醒。</view>
    </view>
    <view v-else class="agent-list">
      <view v-for="item in messages" :key="item.id" class="agent-card" :class="`agent-${item.severity}`">
        <view class="agent-card-header">
          <text class="agent-card-title">{{ item.title }}</text>
          <text class="ai-badge">{{ item.severity }}</text>
        </view>
        <view class="agent-card-body">{{ item.body }}</view>
        <view v-if="item.evidence?.length" class="agent-evidence">
          <view v-for="evidence in item.evidence" :key="`${item.id}-${evidence.source_id}`">
            {{ evidence.excerpt || evidence.source_type }} · {{ evidence.occurred_at || '时间未记录' }}
          </view>
        </view>
        <view class="reminder-actions">
          <button v-if="item.action_status === 'pending' && item.draft_version" class="reminder-btn primary" @click="confirm(item)">确认提醒</button>
          <button v-if="item.action_status !== 'confirmed' && item.action_status !== 'dismissed'" class="reminder-btn" @click="dismiss(item.id)">忽略</button>
        </view>
      </view>
    </view>
    <button v-if="store.nextCursor" class="reminder-btn" @click="store.load({ before: store.nextCursor })">加载更早消息</button>
  </view>
</template>
