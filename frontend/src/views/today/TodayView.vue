<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useRouter } from 'vue-router'
import AppShell from '../../components/app/AppShell.vue'
import AppIcon from '../../components/app/AppIcon.vue'
import CatSwitcher from '../../components/cat/CatSwitcher.vue'
import AIResultBadge from '../../components/ai/AIResultBadge.vue'
import AIEvidencePanel from '../../components/ai/AIEvidencePanel.vue'
import { useAppStore } from '../../stores/app'
import { useCatStore } from '../../stores/cat'
import { services } from '../../services'
import { greeting, todayDateLabel } from '../../utils/date'
import type { FocusItemData, Reminder, TodayStatusData } from '../../types'

const router = useRouter()
const app = useAppStore()
const catStore = useCatStore()

interface StatusRow {
  icon: string
  label: string
  value: string
  state: string
  badge: string
}

interface StatusGroup {
  catId: string
  catName: string
  avatar?: string
  rows: StatusRow[]
}

const statusGroups = ref<StatusGroup[]>([])
const focusItems = ref<FocusItemData[]>([])
const aiSummary = ref<any>(null)
const reminderList = ref<Reminder[]>([])
const loading = ref(true)
const showEvidence = ref(false)

const currentCat = computed(() => app.currentCat)
const now = new Date()
const todayNumber = now.getDate()
const todayMonth = `${now.getMonth() + 1}月`

const dayOfYear = computed(() => {
  const start = new Date(now.getFullYear(), 0, 0)
  return Math.floor((now.getTime() - start.getTime()) / 86400000)
})

const attentionRows = computed(() => statusGroups.value.flatMap((group) =>
  group.rows
    .filter((row) => row.state === 'warning' || row.state === 'danger')
    .map((row) => ({ ...row, catId: group.catId, catName: group.catName }))
))

const dangerCount = computed(() => attentionRows.value.filter((row) => row.state === 'danger').length)
const visibleFocusItems = computed(() => currentCat.value === 'all'
  ? focusItems.value
  : focusItems.value.filter((item) => item.catId === currentCat.value))
const visibleReminders = computed(() => currentCat.value === 'all'
  ? reminderList.value
  : reminderList.value.filter((item) => item.catId === currentCat.value || item.catId === 'both'))

const heroTitle = computed(() => {
  if (dangerCount.value) return `${attentionRows.value.find((row) => row.state === 'danger')?.catName || '家里'}需要多看一眼`
  if (attentionRows.value.length) return '今天有几件事要记得'
  return '今天的状态很不错'
})

const heroDescription = computed(() => {
  if (!attentionRows.value.length) return '饮食、饮水和精神状态都保持稳定，继续按平时的节奏照顾就好。'
  const summary = attentionRows.value
    .slice(0, 2)
    .map((row) => `${row.catName}${row.value}`)
    .join('；')
  return `${summary}${attentionRows.value.length > 2 ? `，另有 ${attentionRows.value.length - 2} 项待处理。` : '。'}`
})

const quickRecords = [
  { id: 'feeding', label: '喂食', icon: 'food' },
  { id: 'elimination', label: '排便', icon: 'elimination' },
  { id: 'vomit', label: '呕吐', icon: 'vomit' },
  { id: 'weight', label: '体重', icon: 'weight' },
  { id: 'more', label: '更多', icon: 'more' }
]

function badgeOf(state: string, warnText: string): string {
  if (state === 'normal') return '正常'
  if (state === 'danger') return '需关注'
  if (state === 'warning') return warnText
  return ''
}

function buildRows(status: TodayStatusData): StatusRow[] {
  return [
    { icon: 'food', label: '饮食', value: status.food.label, state: status.food.state, badge: badgeOf(status.food.state, '偏低') },
    { icon: 'water', label: '饮水', value: status.water.label, state: status.water.state, badge: badgeOf(status.water.state, '偏低') },
    { icon: 'elimination', label: '排便', value: status.elimination.label, state: status.elimination.state, badge: badgeOf(status.elimination.state, '注意') },
    { icon: 'vomit', label: '呕吐', value: status.vomit.state === 'none' ? '无呕吐' : status.vomit.label, state: status.vomit.state, badge: badgeOf(status.vomit.state, '注意') },
    {
      icon: 'medication',
      label: '用药',
      value: status.medication.state === 'none'
        ? '无需用药'
        : status.medication.label + (status.medication.time ? ` · ${status.medication.time}` : ''),
      state: status.medication.state,
      badge: status.medication.state === 'none' ? '' : status.medication.state === 'warning' ? '待办' : '需关注'
    },
    { icon: 'mental', label: '精神', value: status.mental.label, state: status.mental.state, badge: badgeOf(status.mental.state, '注意') }
  ]
}

function groupAttention(group: StatusGroup) {
  return group.rows.filter((row) => row.state === 'warning' || row.state === 'danger')
}

function groupNormal(group: StatusGroup) {
  return group.rows.filter((row) => row.state === 'normal' || row.state === 'none')
}

function groupLevel(group: StatusGroup) {
  if (group.rows.some((row) => row.state === 'danger')) return 'danger'
  if (group.rows.some((row) => row.state === 'warning')) return 'warning'
  return 'normal'
}

async function load() {
  loading.value = true
  try {
    if (currentCat.value === 'all') {
      const cats = catStore.cats
      const results = await Promise.all(cats.map((cat) => services.getTodayStatus(cat.id)))
      statusGroups.value = cats.map((cat, index) => ({
        catId: cat.id,
        catName: cat.name,
        avatar: cat.avatar,
        rows: buildRows(results[index].data)
      }))
    } else {
      const res = await services.getTodayStatus(currentCat.value)
      const cat = catStore.cats.find((item) => item.id === currentCat.value)
      statusGroups.value = [{
        catId: currentCat.value,
        catName: cat?.name || '',
        avatar: cat?.avatar,
        rows: buildRows(res.data)
      }]
    }

    const [focusRes, aiRes, reminderRes] = await Promise.all([
      services.getFocusItems(),
      services.getAISummary(),
      services.getReminders('todo')
    ])
    focusItems.value = focusRes.data
    aiSummary.value = aiRes.data
    reminderList.value = reminderRes.data.filter((item: Reminder) => !app.completedReminders.includes(item.id))
  } catch {
    statusGroups.value = []
  } finally {
    loading.value = false
  }
}

function onCatSelect(catId: string) {
  app.setCurrentCat(catId)
  load()
}

function handleFocusAction(item: FocusItemData, action: string) {
  if (action === 'ai') router.push('/records/ai')
  else if (action === 'view' || action === 'observe') router.push('/records')
  else if (action === 'add') router.push('/records/quick/vomit')
  else if (action === 'done') {
    focusItems.value = focusItems.value.filter((focus) => focus.id !== item.id)
    showToast('已标记完成')
  } else if (action === 'later') {
    showToast('已设置稍后提醒（1 小时后）')
  } else if (action === 'plan') {
    router.push('/reminders')
  }
}

function handleQuickRecord(id: string) {
  if (id === 'more') router.push('/records')
  else router.push(`/records/quick/${id}`)
}

function completeReminder(id: string) {
  app.completedReminders.push(id)
  reminderList.value = reminderList.value.filter((item) => item.id !== id)
}

function laterReminder(id: string) {
  reminderList.value = reminderList.value.filter((item) => item.id !== id)
  showToast('已设置稍后提醒')
}

let toastTimer: ReturnType<typeof setTimeout> | null = null
function showToast(message: string) {
  let element = document.querySelector('.today-toast') as HTMLElement | null
  if (!element) {
    element = document.createElement('div')
    element.className = 'today-toast'
    document.body.appendChild(element)
  }
  element.textContent = message
  element.classList.add('show')
  if (toastTimer) clearTimeout(toastTimer)
  toastTimer = setTimeout(() => element?.classList.remove('show'), 2000)
}

onMounted(load)
</script>

<template>
  <AppShell>
    <header class="page-header today-page-header">
      <div class="today-masthead">
        <span class="today-masthead-brand">
          <AppIcon
            name="catEar"
            :size="17"
          />
          MEOWHOME DAILY
        </span>
        <span>NO. {{ dayOfYear }}</span>
      </div>
      <div class="today-header-main">
        <div>
          <div class="greet-title">
            {{ greeting() }}
          </div>
          <div class="greet-sub">
            {{ todayDateLabel() }}
          </div>
        </div>
        <button
          type="button"
          class="today-reminder-button"
          aria-label="打开提醒中心"
          @click="router.push('/reminders')"
        >
          <AppIcon
            name="bell"
            :size="19"
          />
          <span v-if="visibleReminders.length">{{ visibleReminders.length }}</span>
        </button>
      </div>
      <CatSwitcher @select="onCatSelect" />
    </header>

    <main class="page-content today-dashboard">
      <div
        v-if="loading"
        class="today-loading"
        aria-label="正在加载今日简报"
        aria-busy="true"
      >
        <div class="skeleton today-cover-skeleton" />
        <div class="skeleton today-actions-skeleton" />
        <div class="skeleton today-cats-skeleton" />
      </div>

      <template v-else>
        <section
          class="today-cover"
          :class="{ 'has-danger': dangerCount }"
          aria-label="今日照顾简报"
        >
          <div class="today-cover-date" aria-hidden="true">
            <span>{{ todayMonth }}</span>
            <strong>{{ todayNumber }}</strong>
          </div>
          <div class="today-cover-copy">
            <span class="today-cover-kicker">TODAY'S CARE BRIEFING</span>
            <h1>{{ heroTitle }}</h1>
            <p>{{ heroDescription }}</p>
          </div>
          <div class="today-cover-metrics">
            <span><strong>{{ statusGroups.length }}</strong> 只猫咪</span>
            <span><strong>{{ attentionRows.length }}</strong> 项关注</span>
            <span><strong>{{ visibleReminders.length }}</strong> 个待办</span>
          </div>
          <div class="today-cover-actions">
            <button
              type="button"
              class="today-cover-primary"
              @click="router.push('/records')"
            >
              <AppIcon
                name="record"
                :size="17"
              />
              记一条动态
            </button>
            <button
              type="button"
              class="today-cover-secondary"
              @click="router.push('/reminders')"
            >
              查看今日安排
              <AppIcon
                name="chevronRight"
                :size="15"
              />
            </button>
          </div>
        </section>

        <section
          class="today-quick-section"
          aria-label="快捷记录"
        >
          <div class="today-quick-label">
            <span>QUICK NOTE</span>
            快速记一笔
          </div>
          <div class="today-quick-grid">
            <button
              v-for="button in quickRecords"
              :key="button.id"
              type="button"
              class="today-quick-button"
              @click="handleQuickRecord(button.id)"
            >
              <span>
                <AppIcon
                  :name="button.icon"
                  :size="19"
                />
              </span>
              {{ button.label }}
            </button>
          </div>
        </section>

        <section
          v-if="statusGroups.length"
          class="today-section"
          aria-label="猫咪今日状态"
        >
          <div class="today-section-heading">
            <div>
              <span>AT A GLANCE</span>
              <h2>今日状态</h2>
            </div>
            <small>只突出需要处理的变化</small>
          </div>

          <div
            class="today-cat-deck"
            :class="{ 'is-single': statusGroups.length === 1 }"
          >
            <article
              v-for="(group, index) in statusGroups"
              :key="group.catId"
              class="today-cat-card"
              :class="[`tone-${index % 2}`, `state-${groupLevel(group)}`]"
            >
              <header class="today-cat-card-head">
                <span class="today-cat-avatar">
                  <img
                    v-if="group.avatar"
                    :src="group.avatar"
                    :alt="group.catName"
                  />
                  <AppIcon
                    v-else
                    name="cat"
                    :size="24"
                  />
                </span>
                <span>
                  <strong>{{ group.catName }}</strong>
                  <small>{{ groupAttention(group).length ? `${groupAttention(group).length} 项需要留意` : '状态稳定' }}</small>
                </span>
                <i class="today-cat-level" />
              </header>

              <div
                v-if="groupAttention(group).length"
                class="today-cat-alerts"
              >
                <div
                  v-for="row in groupAttention(group).slice(0, 2)"
                  :key="row.icon"
                  class="today-cat-alert"
                  :class="row.state"
                >
                  <AppIcon
                    :name="row.icon as any"
                    :size="16"
                  />
                  <span>
                    <strong>{{ row.label }}</strong>
                    <small>{{ row.value }}</small>
                  </span>
                </div>
              </div>
              <div
                v-else
                class="today-cat-all-good"
              >
                <AppIcon
                  name="success"
                  :size="20"
                />
                饮食、饮水和精神状态都很好
              </div>

              <div class="today-cat-signals">
                <span
                  v-for="row in groupNormal(group)"
                  :key="row.icon"
                  :title="row.value"
                >
                  <i />{{ row.label }}
                </span>
              </div>
              <button
                type="button"
                class="today-cat-profile"
                @click="router.push(`/cats/${group.catId}`)"
              >
                查看健康档案
                <AppIcon
                  name="chevronRight"
                  :size="14"
                />
              </button>
            </article>
          </div>
        </section>

        <section
          v-if="visibleFocusItems.length"
          class="today-section"
          aria-label="需要关注"
        >
          <div class="today-section-heading">
            <div>
              <span>KEEP AN EYE ON</span>
              <h2>需要关注</h2>
            </div>
          </div>

          <article
            v-for="item in visibleFocusItems"
            :key="item.id"
            class="today-focus-card"
            :class="item.severity"
          >
            <div class="today-focus-icon">
              <AppIcon
                :name="item.severity === 'danger' ? 'alertTriangle' : 'bell'"
                :size="20"
              />
            </div>
            <div class="today-focus-copy">
              <span>{{ item.severity === 'danger' ? '异常变化' : '今日待办' }}</span>
              <h3>{{ item.title }}</h3>
              <p>{{ item.body }}</p>
              <details v-if="item.evidence.length" class="today-focus-evidence">
                <summary>查看提示依据</summary>
                <div v-for="evidence in item.evidence" :key="evidence">{{ evidence }}</div>
              </details>
              <div class="today-focus-actions">
                <button
                  v-for="action in item.actions"
                  :key="action.action"
                  type="button"
                  :class="{ primary: action.action === 'ai' || action.action === 'done' }"
                  @click="handleFocusAction(item, action.action)"
                >
                  {{ action.label }}
                </button>
              </div>
            </div>
          </article>
        </section>

        <section
          v-if="aiSummary"
          class="today-ai-brief"
          aria-label="AI 每日摘要"
        >
          <div class="today-ai-head">
            <AIResultBadge>AI 每日整理</AIResultBadge>
            <span>基于 {{ aiSummary.evidenceCount }} 条记录</span>
          </div>
          <h2>管家的今日小结</h2>
          <p>{{ aiSummary.body }}</p>
          <div class="today-ai-footer">
            <small>生成于 {{ aiSummary.generatedAt.slice(11, 16) }} · 不构成医疗诊断</small>
            <button
              type="button"
              @click="showEvidence = !showEvidence"
            >
              {{ showEvidence ? '收起依据' : '查看依据' }}
              <AppIcon
                :name="showEvidence ? 'chevronDown' : 'chevronRight'"
                :size="14"
              />
            </button>
          </div>
        </section>

        <AIEvidencePanel
          v-if="showEvidence && aiSummary"
          :items="aiSummary.evidence"
        />

        <section
          v-if="visibleReminders.length"
          class="today-section today-reminders-section"
          aria-label="今日提醒"
        >
          <div class="today-section-heading">
            <div>
              <span>TO-DO TODAY</span>
              <h2>今日提醒</h2>
            </div>
            <button
              type="button"
              @click="router.push('/reminders')"
            >
              全部
              <AppIcon
                name="chevronRight"
                :size="14"
              />
            </button>
          </div>
          <div class="today-reminder-list">
            <article
              v-for="item in visibleReminders.slice(0, 4)"
              :key="item.id"
              class="today-reminder-item"
            >
              <span class="today-reminder-time">{{ item.time }}</span>
              <span class="today-reminder-icon">
                <AppIcon
                  :name="item.icon || item.type"
                  :size="18"
                />
              </span>
              <span class="today-reminder-copy">
                <strong>{{ item.title }}</strong>
                <small>{{ item.subtitle }}</small>
              </span>
              <span class="today-reminder-actions">
                <button type="button" @click="laterReminder(item.id)">稍后</button>
                <button type="button" class="primary" @click="completeReminder(item.id)">完成</button>
              </span>
            </article>
          </div>
        </section>
      </template>
    </main>
  </AppShell>
</template>

<style scoped>
.today-toast {
  position: fixed;
  left: 50%;
  bottom: calc(96px + var(--safe-bottom));
  transform: translateX(-50%);
  background: rgba(45, 41, 37, 0.92);
  color: #fff;
  font-size: var(--font-size-assist);
  padding: 10px 16px;
  border-radius: var(--radius-pill);
  opacity: 0;
  pointer-events: none;
  transition: opacity 0.25s ease;
  z-index: 90;
}

.today-toast.show {
  opacity: 1;
}
</style>
