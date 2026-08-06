<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { useRoute } from 'vue-router'
import AppShell from '../../components/app/AppShell.vue'
import AppHeader from '../../components/app/AppHeader.vue'
import AppIcon from '../../components/app/AppIcon.vue'
import { useCatStore } from '../../stores/cat'
import { useHealthStore } from '../../stores/health'
import { services } from '../../services'

const route = useRoute()
const catStore = useCatStore()
const healthStore = useHealthStore()
const catId = computed(() => route.params.catId as string)
const range = ref(30)
const loading = ref(true)
const chartData = ref<any[]>([])

const ranges = [7, 30, 90] as const

async function load() {
  loading.value = true
  try {
    await healthStore.loadTrends(catId.value, range.value)
    chartData.value = healthStore.trends
  } catch { /* handled */ }
  finally { loading.value = false }
}

function setRange(r: number) {
  range.value = r as 7 | 30 | 90
  load()
}

onMounted(load)
</script>

<template>
  <AppShell>
    <AppHeader
      title="健康趋势"
      :subtitle="(catStore.cats.find(c => c.id === catId)?.name || '') + ' · ' + range + '天'"
    />

    <div class="page-content">
      <!-- Range selector -->
      <div
        style="display:flex;gap:8px;margin-bottom:var(--space-16);"
        role="tablist"
        aria-label="时间范围"
      >
        <button
          v-for="r in ranges"
          :key="r"
          class="range-btn"
          :class="{ active: range === r }"
          role="tab"
          :aria-selected="range === r"
          @click="setRange(r)"
        >
          {{ r }}天
        </button>
      </div>

      <!-- Weight chart -->
      <section
        class="chart-container"
        aria-label="体重趋势图表"
      >
        <div class="chart-header">
          <div class="chart-title">
            体重变化
          </div>
        </div>
        <div
          id="weight-chart"
          class="chart-canvas"
        >
          <div
            v-if="loading"
            class="chart-empty"
          >
            <AppIcon
              name="loading"
              :size="40"
            />
            <span style="font-size:var(--font-size-assist);">加载中…</span>
          </div>
          <div
            v-else-if="chartData.length === 0"
            class="chart-empty"
          >
            <AppIcon
              name="trend"
              :size="40"
            />
            <span style="font-size:var(--font-size-assist);">暂无数据</span>
          </div>
          <div
            v-else
            class="chart-visual"
          >
            <!-- Simple bar chart representation -->
            <div class="chart-bars">
              <div
                v-for="(d, i) in chartData"
                :key="i"
                class="chart-bar-col"
              >
                <div
                  class="chart-bar"
                  :style="{ height: Math.max(4, (d.weight / 6) * 100) + '%' }"
                  :title="d.date + ': ' + d.weight + 'kg'"
                />
                <div class="chart-bar-label">
                  {{ d.date.slice(5) }}
                </div>
              </div>
            </div>
          </div>
        </div>
        <div style="font-size:var(--font-size-caption);color:var(--color-text-tertiary);margin-top:8px;">
          数据覆盖率：{{ chartData.length }}/{{ range }} 天
        </div>
      </section>

      <!-- Events -->
      <section
        v-if="healthStore.events.length"
        class="events-section"
      >
        <div class="section-title">
          关联事件
        </div>
        <div class="timeline-month">
          <div
            v-for="ev in healthStore.events.slice(0, 5)"
            :key="ev.date"
            class="timeline-item"
          >
            <div class="timeline-dot">
              <AppIcon
                name="calendar"
                :size="18"
              />
            </div>
            <div class="timeline-content">
              <div class="timeline-date">
                {{ ev.date }}
              </div>
              <div class="timeline-title">
                {{ ev.events.map((e: any) => e.label).join(' · ') }}
              </div>
            </div>
          </div>
        </div>
      </section>
    </div>
  </AppShell>
</template>

<style scoped>
.range-btn {
  padding: var(--space-8) var(--space-16); border-radius: var(--radius-pill);
  font-size: var(--font-size-assist); background: var(--color-bg-subtle);
  border: 1px solid var(--color-border); color: var(--color-text-secondary);
  cursor: pointer; min-height: 36px; transition: all 0.15s ease;
}
.range-btn.active { background: var(--color-brand); color: white; border-color: var(--color-brand); }
.chart-container { margin-bottom: var(--space-16); }
.chart-header { display: flex; justify-content: space-between; align-items: center; margin-bottom: var(--space-8); }
.chart-title { font-size: var(--font-size-body-lg); font-weight: 600; }
.chart-canvas { background: var(--color-bg-surface); border: 1px solid var(--color-divider); border-radius: var(--radius-md); padding: var(--space-16); min-height: 180px; }
.chart-empty { display: flex; flex-direction: column; align-items: center; justify-content: center; height: 180px; gap: 8px; color: var(--color-text-tertiary); }
.chart-bars { display: flex; align-items: flex-end; gap: 2px; height: 120px; }
.chart-bar-col { flex: 1; display: flex; flex-direction: column; align-items: center; height: 100%; justify-content: flex-end; }
.chart-bar { width: 100%; max-width: 20px; background: var(--color-brand); border-radius: 2px 2px 0 0; transition: height 0.3s ease; min-height: 4px; }
.chart-bar-label { font-size: 9px; color: var(--color-text-tertiary); margin-top: 4px; }
.chart-visual { height: 120px; }
.events-section { margin-top: var(--space-16); }
.timeline-item { display: flex; gap: var(--space-12); padding: var(--space-8) 0; border-bottom: 1px solid var(--color-divider); }
.timeline-dot { flex-shrink: 0; margin-top: 2px; }
.timeline-content { flex: 1; }
.timeline-date { font-size: var(--font-size-caption); color: var(--color-text-tertiary); }
.timeline-title { font-size: var(--font-size-assist); }
</style>
