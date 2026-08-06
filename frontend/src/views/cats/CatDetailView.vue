<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { useRoute } from 'vue-router'
import AppShell from '../../components/app/AppShell.vue'
import AppIcon from '../../components/app/AppIcon.vue'
import { useCatStore } from '../../stores/cat'
import { services } from '../../services'
import { generateTrends } from '../../mocks/trends'

const route = useRoute()
const store = useCatStore()
const catId = computed(() => (route.params.catId as string) || store.currentCatId || 'cat-whit')
const cat = computed(() => store.cats.find(c => c.id === catId.value) || store.cats[0])
const activeTab = ref<'overview' | 'records' | 'trends' | 'medical' | 'moments'>('overview')

const tabs = [
  { id: 'overview' as const, label: '概览' },
  { id: 'records' as const, label: '记录' },
  { id: 'trends' as const, label: '趋势' },
  { id: 'medical' as const, label: '医疗' },
  { id: 'moments' as const, label: '时光' }
]

const weightTrends = ref<any[]>([])
const loading = ref(true)

onMounted(async () => {
  loading.value = true
  weightTrends.value = generateTrends(catId.value)
  loading.value = false
})
</script>

<template>
  <AppShell>
    <div class="cat-detail-header">
      <img
        v-if="cat.avatar"
        class="cat-detail-avatar"
        :src="cat.avatar"
        :alt="cat.name + '头像'"
      />
      <div class="cat-detail-avatar-placeholder">
        <AppIcon
          name="cat"
          :size="40"
        />
      </div>
      <div class="cat-detail-name">
        {{ cat.name }}
      </div>
      <div class="cat-detail-meta">
        {{ cat.breed }} · {{ cat.age }}岁 · {{ cat.gender === 'female' ? '母' : '公' }}
      </div>
    </div>

    <div class="cat-tabs">
      <button
        v-for="tab in tabs"
        :key="tab.id"
        class="cat-tab"
        :class="{ active: activeTab === tab.id }"
        @click="activeTab = tab.id"
      >
        {{ tab.label }}
      </button>
    </div>

    <div class="page-content">
      <template v-if="activeTab === 'overview'">
        <div class="status-panel">
          <div class="status-panel-title">
            今日状态
          </div>
          <div class="status-row">
            <span class="status-label"><AppIcon
              name="food"
              :size="18"
            /> 饮食</span>
            <span class="status-value normal">正常</span>
          </div>
          <div class="status-row">
            <span class="status-label"><AppIcon
              name="water"
              :size="18"
            /> 饮水</span>
            <span class="status-value normal">正常</span>
          </div>
          <div class="status-row">
            <span class="status-label"><AppIcon
              name="elimination"
              :size="18"
            /> 排便</span>
            <span class="status-value warning">偏干</span>
          </div>
          <div class="status-row">
            <span class="status-label"><AppIcon
              name="vomit"
              :size="18"
            /> 呕吐</span>
            <span class="status-value">无呕吐</span>
          </div>
        </div>
      </template>

      <template v-if="activeTab === 'trends'">
        <div class="chart-container">
          <div class="chart-header">
            <div class="chart-title">
              体重趋势
            </div>
            <div class="chart-range">
              <button
                class="chart-range-btn"
                :class="{ active: true }"
              >
                7天
              </button>
              <button class="chart-range-btn">
                30天
              </button>
              <button class="chart-range-btn">
                90天
              </button>
            </div>
          </div>
          <div
            v-if="loading"
            class="chart-empty"
          >
            <div
              class="skeleton"
              style="height:150px;"
            />
          </div>
          <div
            v-else
            class="chart-canvas"
          >
            <div
              v-for="(p, i) in weightTrends.slice(-7)"
              :key="i"
              class="chart-point"
            >
              <div
                class="chart-bar"
                :style="{ height: p.weight + 'px' }"
              />
              <div class="chart-label">
                {{ p.date.slice(5) }}
              </div>
            </div>
          </div>
        </div>
      </template>

      <template v-else-if="activeTab === 'records'">
        <div class="section-title">
          最近记录
        </div>
        <div class="reminder-list">
          <div
            v-for="i in 3"
            :key="i"
            class="reminder-item"
          >
            <div class="reminder-left">
              <div class="reminder-icon feeding">
                <AppIcon
                  name="food"
                  :size="18"
                />
              </div>
              <div class="reminder-text">
                <div class="reminder-title">
                  喂食
                </div>
                <div class="reminder-subtitle">
                  今天 08:00 · 渴望室内猫粮 80g
                </div>
              </div>
            </div>
          </div>
        </div>
      </template>
    </div>
  </AppShell>
</template>

<style scoped>
.cat-detail-avatar-placeholder { width: 88px; height: 88px; border-radius: 50%; background: var(--color-bg-subtle); border: 3px solid var(--color-brand-soft); margin: 0 auto var(--space-12); display: flex; align-items: center; justify-content: center; }
.cat-detail-avatar { width: 88px; height: 88px; border-radius: 50%; object-fit: cover; border: 3px solid var(--color-brand-soft); margin: 0 auto var(--space-12); display: block; }
.chart-point { display: flex; flex-direction: column; align-items: center; gap: 4px; flex: 1; }
.chart-bar { width: 100%; background: var(--color-brand); border-radius: 4px 4px 0 0; min-height: 4px; }
.chart-label { font-size: var(--font-size-caption); color: var(--color-text-tertiary); }
</style>
