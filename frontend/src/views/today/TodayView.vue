<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
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
import type { TodayStatusData } from '../../types'
import type { FocusItemData, Reminder } from '../../types'

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

const quickRecords = [
  { id: 'feeding', label: '喂食', icon: 'food' },
  { id: 'elimination', label: '排便', icon: 'elimination' },
  { id: 'vomit', label: '呕吐', icon: 'vomit' },
  { id: 'weight', label: '体重', icon: 'weight' },
  { id: 'more', label: '更多', icon: 'more' }
]

/** 状态徽标文案：normal→正常，danger→需关注，warning 按行类型给语义 */
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
      value:
        status.medication.state === 'none'
          ? '无需用药'
          : status.medication.label + (status.medication.time ? ' · ' + status.medication.time : ''),
      state: status.medication.state,
      badge: status.medication.state === 'none' ? '' : status.medication.state === 'warning' ? '待办' : '需关注'
    },
    { icon: 'mental', label: '精神', value: status.mental.label, state: status.mental.state, badge: badgeOf(status.mental.state, '注意') }
  ]
}

async function load() {
  loading.value = true
  try {
    // 今日状态：单猫 → 该猫；全部 → 两只猫并列，明确显示所属猫咪
    if (currentCat.value === 'all') {
      const cats = catStore.cats
      const results = await Promise.all(cats.map((c) => services.getTodayStatus(c.id)))
      statusGroups.value = cats.map((c, i) => ({
        catId: c.id,
        catName: c.name,
        avatar: c.avatar,
        rows: buildRows(results[i].data)
      }))
    } else {
      const res = await services.getTodayStatus(currentCat.value)
      const cat = catStore.cats.find((c) => c.id === currentCat.value)
      statusGroups.value = [
        { catId: currentCat.value, catName: cat?.name || '', avatar: cat?.avatar, rows: buildRows(res.data) }
      ]
    }

    const [focusRes, aiRes, reminderRes] = await Promise.all([
      services.getFocusItems(),
      services.getAISummary(),
      services.getReminders('todo')
    ])
    focusItems.value = focusRes.data
    aiSummary.value = aiRes.data
    reminderList.value = reminderRes.data.filter((r: Reminder) => !app.completedReminders.includes(r.id))
  } catch {
    /* handled by empty UI */
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
  else if (action === 'view') router.push('/records')
  else if (action === 'observe') router.push('/records')
  else if (action === 'add') router.push('/records/quick/vomit')
  else if (action === 'done') {
    focusItems.value = focusItems.value.filter((i) => i.id !== item.id)
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
  reminderList.value = reminderList.value.filter((r) => r.id !== id)
}

function laterReminder(id: string) {
  reminderList.value = reminderList.value.filter((r) => r.id !== id)
  showToast('已设置稍后提醒')
}

let toastTimer: ReturnType<typeof setTimeout> | null = null
function showToast(msg: string) {
  let el = document.querySelector('.today-toast') as HTMLElement | null
  if (!el) {
    el = document.createElement('div')
    el.className = 'today-toast'
    document.body.appendChild(el)
  }
  el.textContent = msg
  el.classList.add('show')
  if (toastTimer) clearTimeout(toastTimer)
  toastTimer = setTimeout(() => el?.classList.remove('show'), 2000)
}

onMounted(load)
</script>

<template>
  <AppShell>
    <!-- 顶部：问候语 / 家庭名 / 日期 / 提醒入口 / 猫咪切换器 -->
    <div class="page-header">
      <div style="display:flex;align-items:flex-start;justify-content:space-between;width:100%;">
        <div>
          <div style="font-size:var(--font-size-title);font-weight:600;">
            {{ greeting() }}
          </div>
          <div style="font-size:var(--font-size-assist);color:var(--color-text-tertiary);margin-top:4px;">
            小家的猫宅 · {{ todayDateLabel() }}
          </div>
        </div>
        <button
          class="ai-action-btn"
          aria-label="提醒中心"
          @click="router.push('/reminders')"
        >
          <AppIcon
            name="bell"
            :size="20"
          />
        </button>
      </div>
      <CatSwitcher @select="onCatSelect" />
    </div>

    <div class="page-content">
      <!-- 加载 -->
      <div
        v-if="loading"
        aria-busy="true"
      >
        <div
          class="skeleton"
          style="height:120px;margin-bottom:var(--space-16);"
        />
        <div
          class="skeleton"
          style="height:60px;margin-bottom:var(--space-12);"
        />
        <div
          class="skeleton"
          style="height:80px;"
        />
      </div>

      <template v-else>
        <!-- 今日状态：连续状态列表；"全部"模式下两只猫并列 -->
        <section
          class="status-panel"
          aria-label="今日状态"
        >
          <div class="status-panel-title">
            今日状态
          </div>
          <template
            v-for="group in statusGroups"
            :key="group.catId"
          >
            <div
              v-if="statusGroups.length > 1"
              class="status-cat-head"
            >
              <img
                v-if="group.avatar"
                class="cat-avatar-sm"
                :src="group.avatar"
                :alt="group.catName + '头像'"
              />
              <span
                v-else
                class="cat-avatar-sm"
              ><AppIcon
                name="cat"
                :size="14"
              /></span>
              {{ group.catName }}
            </div>
            <div
              v-for="row in group.rows"
              :key="group.catId + '-' + row.icon"
              class="status-row"
            >
              <span class="status-label">
                <AppIcon
                  :name="row.icon as any"
                  :size="18"
                /> {{ row.label }}
              </span>
              <span
                class="status-value"
                :class="row.state"
              >
                {{ row.value }}
                <span
                  v-if="row.badge"
                  class="status-badge"
                  :class="row.state"
                >{{ row.badge }}</span>
              </span>
            </div>
          </template>
        </section>

        <!-- 需要关注 -->
        <section
          v-if="focusItems.length"
          aria-label="需要关注"
        >
          <div class="section-title">
            需要关注
          </div>
          <div
            v-for="item in focusItems"
            :key="item.id"
            class="focus-section"
            :class="item.severity"
          >
            <div class="focus-title">
              <AppIcon
                name="alertTriangle"
                :size="18"
              /> {{ item.title }}
            </div>
            <div class="focus-body">
              {{ item.body }}
            </div>
            <!-- 提示依据：与记录一一对应，可追溯 -->
            <div
              v-if="item.evidence.length"
              class="focus-evidence"
            >
              <div class="focus-evidence-title">
                提示依据
              </div>
              <div
                v-for="ev in item.evidence"
                :key="ev"
                class="focus-evidence-item"
              >
                {{ ev }}
              </div>
            </div>
            <div class="focus-actions">
              <button
                v-for="action in item.actions"
                :key="action.action"
                class="focus-action-btn"
                :class="{ primary: action.action === 'ai' }"
                @click="handleFocusAction(item, action.action)"
              >
                {{ action.label }}
              </button>
            </div>
          </div>
        </section>

        <!-- AI 每日摘要 -->
        <section
          v-if="aiSummary"
          class="ai-summary"
          aria-label="AI 每日摘要"
        >
          <AIResultBadge>AI 每日整理</AIResultBadge>
          <div class="ai-summary-title">
            今日速览
          </div>
          <div class="ai-summary-body">
            {{ aiSummary.body }}
          </div>
          <div class="ai-meta">
            <span>生成于 {{ aiSummary.generatedAt.slice(11, 16) }}</span>
            <span>基于 {{ aiSummary.evidenceCount }} 条记录</span>
            <span
              class="ai-evidence-link"
              @click="showEvidence = !showEvidence"
            >查看依据</span>
          </div>
          <div class="ai-disclaimer">
            以上由 AI 整理，不构成医疗诊断。如有异常请咨询兽医。
          </div>
        </section>

        <!-- 依据面板 -->
        <AIEvidencePanel
          v-if="showEvidence && aiSummary"
          :items="aiSummary.evidence"
        />

        <!-- 今日提醒 -->
        <section
          v-if="reminderList.length"
          aria-label="今日提醒"
        >
          <div class="section-title">
            今日提醒
          </div>
          <div class="reminder-list">
            <div
              v-for="item in reminderList"
              :key="item.id"
              class="reminder-item"
            >
              <div class="reminder-left">
                <div
                  class="reminder-icon"
                  :class="item.icon || item.type"
                >
                  <AppIcon
                    :name="item.icon || item.type"
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
              <div class="reminder-actions">
                <button
                  class="reminder-btn"
                  @click="laterReminder(item.id)"
                >
                  稍后
                </button>
                <button
                  class="reminder-btn primary"
                  @click="completeReminder(item.id)"
                >
                  完成
                </button>
              </div>
            </div>
          </div>
        </section>

        <!-- 快捷记录 -->
        <section
          class="quick-record-section"
          aria-label="快捷记录"
        >
          <div class="quick-record-grid">
            <button
              v-for="btn in quickRecords"
              :key="btn.id"
              class="quick-record-btn"
              @click="handleQuickRecord(btn.id)"
            >
              <AppIcon
                :name="btn.icon"
                :size="22"
              />
              <span>{{ btn.label }}</span>
            </button>
          </div>
        </section>
      </template>
    </div>
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
