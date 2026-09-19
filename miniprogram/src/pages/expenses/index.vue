<script setup lang="ts">
import { ref, computed } from 'vue'
import AppIcon from '../../components/app/AppIcon.vue'
import { useExpenseStore } from '../../stores/expense'
import { services } from '../../services'
import { useProtectedPage } from '../../utils/page'
import type { Expense } from '../../types'

const store = useExpenseStore()
const expenses = ref<Expense[]>([])
const total = ref(0)
const loading = ref(true)
const foodTotal = computed(() =>
  expenses.value.filter((e) => e.category === 'food').reduce((sum, e) => sum + e.amount, 0)
)
const medicalTotal = computed(() =>
  expenses.value.filter((e) => e.category === 'medical').reduce((sum, e) => sum + e.amount, 0)
)

async function load() {
  const res = await services.getExpenses()
  expenses.value = res.data
  store.set(res.data)
  total.value = res.data.reduce((sum: number, e: any) => sum + e.amount, 0)
  loading.value = false
}

useProtectedPage(load)
</script>

<template>
  <view class="page-header">
    <view class="page-title"> 养猫支出 </view>
  </view>

  <view class="page-content">
    <view v-if="loading" aria-busy="true">
      <view class="skeleton" style="height: 120px; margin-bottom: 16px" />
      <view class="skeleton" style="height: 60px; margin-bottom: 12px" />
    </view>
    <template v-else>
      <view class="status-panel">
        <view class="status-panel-title"> 本月支出 </view>
        <view class="status-row">
          <view class="status-label"><AppIcon name="wallet" :size="18" /> 总计</view>
          <text class="status-value">¥{{ total.toFixed(2) }}</text>
        </view>
        <view class="status-row">
          <view class="status-label"><AppIcon name="food" :size="18" /> 食物</view>
          <text class="status-value">¥{{ foodTotal.toFixed(2) }}</text>
        </view>
        <view class="status-row">
          <view class="status-label"><AppIcon name="medical" :size="18" /> 医疗</view>
          <text class="status-value">¥{{ medicalTotal.toFixed(2) }}</text>
        </view>
      </view>

      <view class="section-title"> 支出明细 </view>
      <view v-if="!expenses.length" class="empty-state">暂无支出记录</view>
      <view class="reminder-list">
        <view v-for="e in expenses" :key="e.id" class="reminder-item">
          <view class="reminder-left">
            <view class="reminder-icon wallet">
              <AppIcon name="wallet" :size="18" />
            </view>
            <view class="reminder-text">
              <view class="reminder-title">
                {{ e.label }}
              </view>
              <view class="reminder-subtitle"> {{ e.date }} · {{ e.category }} · 全部 </view>
            </view>
          </view>
          <view class="reminder-time"> ¥{{ e.amount.toFixed(2) }} </view>
        </view>
      </view>
    </template>
  </view>
</template>
