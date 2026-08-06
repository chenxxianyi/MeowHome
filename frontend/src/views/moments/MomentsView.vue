<script setup lang="ts">
import { ref, onMounted } from 'vue'
import AppShell from '../../components/app/AppShell.vue'
import AppIcon from '../../components/app/AppIcon.vue'
import AIResultBadge from '../../components/ai/AIResultBadge.vue'
import AIEvidencePanel from '../../components/ai/AIEvidencePanel.vue'
import { useMomentStore } from '../../stores/moment'
import { services } from '../../services'
import { mockAISummary } from '../../mocks/health'

const store = useMomentStore()
const events = ref<any[]>([])
const loading = ref(true)
const showEvidence = ref(false)

const months = [
  { label: '2026年8月', items: [
    { id: 'm1', date: '08-06', type: 'photo', title: '小橘在阳光下打盹', catId: 'cat-oran', images: 8 },
    { id: 'm2', date: '08-05', type: 'milestone', title: '小橘5岁生日', catId: 'cat-oran', images: 12 }
  ]},
  { label: '2026年7月', items: [
    { id: 'm3', date: '07-28', type: 'photo', title: '小橘在窗边晒太阳', catId: 'cat-oran', images: 12 },
    { id: 'm4', date: '07-25', type: 'interaction', title: '两只猫第一次和平共处', catId: 'both', images: 5 },
    { id: 'm5', date: '07-20', type: 'milestone', title: '小橘5岁生日', catId: 'cat-oran', images: 8 },
    { id: 'm6', date: '07-15', type: 'medical', title: '小白体检', catId: 'cat-whit', images: 3 }
  ]}
]

const aiReview = {
  body: '两只猫共同记录28天，小白体重保持稳定，小橘本月最喜欢在窗边睡觉，共保存36张照片。',
  generatedAt: mockAISummary.generatedAt,
  evidenceCount: mockAISummary.evidenceCount,
  evidence: mockAISummary.evidence
}

onMounted(async () => {
  loading.value = true
  try {
    const res = await services.getMoments()
    events.value = res.data
  } finally {
    loading.value = false
  }
})
</script>

<template>
  <AppShell>
    <div class="page-header">
      <div class="page-title">
        时光
      </div>
    </div>

    <div class="page-content">
      <!-- AI Monthly Review -->
      <section
        v-if="aiReview"
        class="ai-summary"
        aria-label="月度回顾"
      >
        <AIResultBadge />
        <div class="ai-summary-title">
          月度回顾
        </div>
        <div class="ai-summary-body">
          {{ aiReview.body }}
        </div>
        <div class="ai-meta">
          <span>生成于 {{ aiReview.generatedAt.slice(11, 16) }}</span>
          <span
            class="ai-evidence-link"
            @click="showEvidence = !showEvidence"
          >查看依据</span>
        </div>
        <div class="ai-disclaimer">
          AI 整理，不构成医疗诊断。
        </div>
      </section>

      <AIEvidencePanel
        v-if="showEvidence && aiReview"
        :items="aiReview.evidence"
      />

      <!-- Timeline by month -->
      <template v-if="!loading">
        <div
          v-for="month in months"
          :key="month.label"
          class="timeline-month"
        >
          <div class="timeline-month-title">
            {{ month.label }}
          </div>
          <div class="timeline-events">
            <div
              v-for="ev in month.items"
              :key="ev.id"
              class="timeline-item"
            >
              <div class="timeline-dot">
                <AppIcon
                  name="spotlight"
                  :size="18"
                />
              </div>
              <div class="timeline-content">
                <div class="timeline-date">
                  {{ ev.date }}
                </div>
                <div class="timeline-title">
                  {{ ev.title }}
                </div>
                <div class="timeline-body">
                  {{ ev.catId === 'both' ? '两只猫' : '小橘' }} · {{ ev.images }}张照片
                </div>
              </div>
            </div>
          </div>
        </div>
      </template>

      <div
        v-else
        aria-busy="true"
      >
        <div
          v-for="i in 3"
          :key="i"
          class="skeleton"
          style="height:60px;margin-bottom:16px;"
        />
      </div>
    </div>
  </AppShell>
</template>
