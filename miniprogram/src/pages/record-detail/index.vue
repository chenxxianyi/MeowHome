<script setup lang="ts">
import { computed, ref } from 'vue'
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
const recordDateLabel = computed(() => {
  if (record.value?.type === 'vaccine' && typeof record.value.payload?.vaccination_date === 'string') {
    return record.value.payload.vaccination_date
  }
  return record.value?.occurred_at || ''
})
const vaccineDetails = computed(() => {
  if (record.value?.type !== 'vaccine') return []
  const payload = record.value.payload || {}
  const labels: Record<string, string> = {
    fvrhcp: '猫三联', felv: 'FeLV', rabies: '狂犬疫苗', other: '其他疫苗', unknown: '不确定疫苗'
  }
  const dose = payload.dose_number === 'booster' ? '加强针' : payload.dose_number === 'unknown' ? '不确定' : `第 ${String(payload.dose_number || '?')} 针`
  const rows: [string, unknown][] = [
    ['疫苗类型', typeof payload.vaccine_code === 'string' ? labels[payload.vaccine_code] || payload.vaccine_code : ''],
    ['疫苗名称', payload.vaccine_name],
    ['接种日期', payload.vaccination_date],
    ['针次', dose],
    ['系列阶段', payload.series_stage === 'primary' ? '初免系列' : payload.series_stage === 'booster' ? '加强针' : '不确定'],
    ['接种时年龄', payload.age_weeks !== undefined ? `${String(payload.age_weeks)} 周` : ''],
    ['品牌/产品', payload.product_name],
    ['生产商', payload.manufacturer],
    ['批号', payload.lot_number],
    ['诊所', payload.clinic],
    ['下一次日期', payload.next_due_date],
    ['下一次日期来源', payload.next_due_source]
  ]
  return rows.filter(([, value]) => value !== undefined && value !== null && value !== '').map(([label, value]) => ({ label, value: String(value) }))
})

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
      <view class="page-subtitle">{{ recordDateLabel }} · {{ record.severity }}</view>
      <view class="agent-card-body">{{ record.note || '无补充说明' }}</view>
      <view v-if="record.type === 'vaccine'" class="vaccine-details">
        <view v-for="detail in vaccineDetails" :key="detail.label" class="vaccine-detail-row">
          <text>{{ detail.label }}</text><text>{{ detail.value }}</text>
        </view>
        <view class="page-subtitle">记录用于追溯接种信息；具体安排以兽医意见和产品说明为准。</view>
      </view>
      <view class="page-subtitle">记录编号：{{ record.id }}</view>
    </view>
  </view>
</template>

<style scoped>
.vaccine-details { display: flex; flex-direction: column; gap: 8px; margin: 16px 0; }
.vaccine-detail-row { display: flex; justify-content: space-between; gap: 12px; }
</style>
