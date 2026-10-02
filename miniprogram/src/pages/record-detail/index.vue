<script setup lang="ts">
import { ref } from 'vue'
import { recordApi } from '../../api/endpoints'
import type { RecordDTO } from '../../api/endpoints'
import { useAuthStore } from '../../stores/auth'
import { useProtectedPage } from '../../utils/page'
import { useRoute } from '../../utils/navigation'

const route = useRoute()
const auth = useAuthStore()
const record = ref<RecordDTO | null>(null)
const loading = ref(true)
const error = ref('')

async function load() {
  const familyId = auth.familyId
  const recordId = route.query.id
  if (!familyId || !recordId) { error.value = '缺少记录编号'; loading.value = false; return }
  loading.value = true
  error.value = ''
  record.value = null
  try {
    const result = await recordApi.get(familyId, recordId)
    if (auth.familyId === familyId) record.value = result
  } catch {
    record.value = null
    error.value = '记录不存在或当前家庭无权查看'
  } finally {
    loading.value = false
  }
}

useProtectedPage(load)
</script>

<template>
  <view class="page-header"><view class="page-title">记录详情</view></view>
  <view class="page-content">
    <view v-if="loading" class="empty-state">正在读取记录…</view>
    <view v-else-if="error" class="empty-state">{{ error }}</view>
    <view v-else-if="record" class="agent-card">
      <view class="agent-card-title">{{ record.title || record.type }}</view>
      <view class="page-subtitle">{{ record.occurred_at }} · {{ record.severity }}</view>
      <view class="agent-card-body">{{ record.note || '无补充说明' }}</view>
      <view class="page-subtitle">记录编号：{{ record.id }}</view>
    </view>
  </view>
</template>
