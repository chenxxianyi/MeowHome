<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import AppShell from '../../components/app/AppShell.vue'
import AppIcon from '../../components/app/AppIcon.vue'
import { useExpenseStore } from '../../stores/expense'
import { services } from '../../services'

const store = useExpenseStore()
const expenses = ref<any[]>([])
const total = ref(0)
const loading = ref(true)
const foodTotal = computed(() => expenses.value.filter((e: any) => e.category === 'food').reduce((s: number, e: any) => s + e.amount, 0))
const medicalTotal = computed(() => expenses.value.filter((e: any) => e.category === 'medical').reduce((s: number, e: any) => s + e.amount, 0))

onMounted(async () => {
  const res = await services.getExpenses()
  expenses.value = res.data
  total.value = res.data.reduce((sum: number, e: any) => sum + e.amount, 0)
  loading.value = false
})
</script>

<template>
  <AppShell>
    <div class="page-header">
      <div class="page-title">
        养猫支出
      </div>
    </div>

    <div class="page-content">
      <div
        v-if="loading"
        aria-busy="true"
      >
        <div
          class="skeleton"
          style="height:120px;margin-bottom:16px;"
        />
        <div
          class="skeleton"
          style="height:60px;margin-bottom:12px;"
        />
      </div>
      <template v-else>
        <div class="status-panel">
          <div class="status-panel-title">
            本月支出
          </div>
          <div class="status-row">
            <span class="status-label"><AppIcon
              name="wallet"
              :size="18"
            /> 总计</span>
            <span class="status-value">¥{{ total.toFixed(2) }}</span>
          </div>
          <div class="status-row">
            <span class="status-label"><AppIcon
              name="food"
              :size="18"
            /> 食物</span>
            <span class="status-value">¥{{ foodTotal.toFixed(2) }}</span>
          </div>
          <div class="status-row">
            <span class="status-label"><AppIcon
              name="medical"
              :size="18"
            /> 医疗</span>
            <span class="status-value">¥{{ medicalTotal.toFixed(2) }}</span>
          </div>
        </div>

        <div class="section-title">
          支出明细
        </div>
        <div class="reminder-list">
          <div
            v-for="e in expenses"
            :key="e.id"
            class="reminder-item"
          >
            <div class="reminder-left">
              <div class="reminder-icon wallet">
                <AppIcon
                  name="wallet"
                  :size="18"
                />
              </div>
              <div class="reminder-text">
                <div class="reminder-title">
                  {{ e.label }}
                </div>
                <div class="reminder-subtitle">
                  {{ e.date }} · {{ e.category }} · 全部
                </div>
              </div>
            </div>
            <div class="reminder-time">
              ¥{{ e.amount.toFixed(2) }}
            </div>
          </div>
        </div>
      </template>
    </div>
  </AppShell>
</template>
