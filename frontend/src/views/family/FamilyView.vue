<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useRouter } from 'vue-router'
import AppShell from '../../components/app/AppShell.vue'
import AppIcon from '../../components/app/AppIcon.vue'
import { useCatStore } from '../../stores/cat'
import { services } from '../../services'
import { todayStr } from '../../utils/date'
import type { Expense, Family, InventoryItem } from '../../types'

const router = useRouter()
const catStore = useCatStore()

const family = ref<Family>({
  id: '',
  name: '小家的猫宅',
  createdAt: '',
  members: []
})
const inventory = ref<InventoryItem[]>([])
const expenses = ref<Expense[]>([])
const loading = ref(true)

const cats = computed(() => catStore.cats)
const lowCount = computed(() => inventory.value.filter((item) => item.status === 'low' || item.status === 'expired').length)
const currentMonth = todayStr().slice(0, 7)
const monthExpense = computed(() => expenses.value
  .filter((expense) => expense.date.startsWith(currentMonth))
  .reduce((sum, expense) => sum + expense.amount, 0))
const homeStatus = computed(() => lowCount.value
  ? `${lowCount.value} 项物资需要关注`
  : '今日照顾状态良好')

const careEntries = computed(() => [
  {
    icon: 'inbox',
    title: '物资库存',
    desc: lowCount.value ? `${lowCount.value} 项需要补充或处理` : '常用物资都很充足',
    value: `${inventory.value.length} 项`,
    href: '/family/inventory',
    tone: lowCount.value ? 'warning' : 'sage'
  },
  {
    icon: 'wallet',
    title: '本月支出',
    desc: '查看分类与支出明细',
    value: `¥${monthExpense.value.toFixed(0)}`,
    href: '/family/expenses',
    tone: 'clay'
  },
  {
    icon: 'bell',
    title: '照顾提醒',
    desc: '用药、驱虫与日常待办',
    value: '查看日程',
    href: '/reminders',
    tone: 'blue'
  },
  {
    icon: 'users',
    title: '家庭成员',
    desc: '共同照顾与权限管理',
    value: `${family.value.members.length} 人`,
    href: '/settings',
    tone: 'rose'
  }
])

const settingsEntries = [
  { icon: 'ai', title: 'AI 管家设置', desc: '权限与数据范围' },
  { icon: 'import', title: '数据管理', desc: '导入、导出与备份' },
  { icon: 'eye', title: '隐私设置', desc: '数据可见性' }
]

onMounted(async () => {
  try {
    const [famRes, invRes, expRes] = await Promise.all([
      services.getFamily(),
      services.getInventory(),
      services.getExpenses()
    ])
    family.value = famRes.data
    inventory.value = invRes.data
    expenses.value = expRes.data
  } finally {
    loading.value = false
  }
})

function go(href: string) {
  router.push(href)
}
</script>

<template>
  <AppShell>
    <header class="page-header family-page-header">
      <div>
        <div class="page-title">
          家庭
        </div>
        <div class="page-subtitle">
          一起照顾，彼此放心
        </div>
      </div>
      <button
        type="button"
        class="family-header-action"
        aria-label="打开家庭设置"
        @click="go('/settings')"
      >
        <AppIcon
          name="settings"
          :size="20"
        />
      </button>
    </header>

    <main class="page-content family-dashboard">
      <div
        v-if="loading"
        class="family-dashboard-loading"
        aria-label="正在加载家庭信息"
        aria-busy="true"
      >
        <div class="skeleton family-hero-skeleton" />
        <div class="skeleton family-cats-skeleton" />
        <div class="skeleton family-actions-skeleton" />
      </div>

      <template v-else>
        <section class="family-home-card">
          <div class="family-home-topline">
            <span class="family-home-eyebrow">MEOWHOME · OUR HOME</span>
            <button
              type="button"
              class="family-home-manage"
              @click="go('/settings')"
            >
              管理
              <AppIcon
                name="chevronRight"
                :size="14"
              />
            </button>
          </div>
          <h1>{{ family.name }}</h1>
          <p>{{ cats.length }} 只猫咪 · {{ family.members.length }} 位家人在共同照顾</p>
          <div class="family-home-bottom">
            <div
              class="family-home-cat-stack"
              aria-label="家庭猫咪"
            >
              <span
                v-for="(cat, index) in cats"
                :key="cat.id"
                class="family-home-cat-avatar"
                :class="`tone-${index % 2}`"
                :title="cat.name"
              >
                <img
                  v-if="cat.avatar"
                  :src="cat.avatar"
                  :alt="cat.name"
                />
                <AppIcon
                  v-else
                  name="cat"
                  :size="18"
                />
              </span>
            </div>
            <span
              class="family-home-status"
              :class="{ warning: lowCount }"
            >
              <i />{{ homeStatus }}
            </span>
          </div>
        </section>

        <section class="family-dashboard-section">
          <div class="family-dashboard-heading">
            <div>
              <span>OUR CATS</span>
              <h2>家里的小成员</h2>
            </div>
            <button
              type="button"
              @click="go('/cats')"
            >
              全部档案
              <AppIcon
                name="chevronRight"
                :size="14"
              />
            </button>
          </div>

          <div class="family-cat-grid">
            <button
              v-for="(cat, index) in cats"
              :key="cat.id"
              type="button"
              class="family-cat-card"
              :class="`tone-${index % 2}`"
              :aria-label="`查看 ${cat.name} 的健康档案`"
              @click="go(`/cats/${cat.id}`)"
            >
              <span class="family-cat-portrait">
                <img
                  v-if="cat.avatar"
                  :src="cat.avatar"
                  :alt="cat.name"
                />
                <AppIcon
                  v-else
                  name="cat"
                  :size="32"
                />
              </span>
              <span class="family-cat-health" :class="{ attention: cat.diseases.length }">
                {{ cat.diseases.length ? cat.diseases[0] : '健康状态良好' }}
              </span>
              <strong>{{ cat.name }}</strong>
              <small>{{ cat.breed }} · {{ cat.age }} 岁</small>
              <span class="family-cat-link">
                健康档案
                <AppIcon
                  name="chevronRight"
                  :size="14"
                />
              </span>
            </button>
          </div>
        </section>

        <section class="family-dashboard-section">
          <div class="family-dashboard-heading">
            <div>
              <span>CARE TOGETHER</span>
              <h2>家庭管家</h2>
            </div>
          </div>

          <div class="family-care-grid">
            <button
              v-for="item in careEntries"
              :key="item.title"
              type="button"
              class="family-care-card"
              :class="`tone-${item.tone}`"
              @click="go(item.href)"
            >
              <span class="family-care-icon">
                <AppIcon
                  :name="item.icon"
                  :size="20"
                />
              </span>
              <span class="family-care-value">{{ item.value }}</span>
              <strong>{{ item.title }}</strong>
              <small>{{ item.desc }}</small>
            </button>
          </div>
        </section>

        <section class="family-settings-section">
          <div class="family-dashboard-heading">
            <div>
              <span>HOUSEHOLD SETTINGS</span>
              <h2>家庭与数据</h2>
            </div>
          </div>
          <div class="family-settings-list">
            <button
              v-for="item in settingsEntries"
              :key="item.title"
              type="button"
              class="family-settings-item"
              @click="go('/settings')"
            >
              <span class="family-settings-icon">
                <AppIcon
                  :name="item.icon"
                  :size="18"
                />
              </span>
              <span>
                <strong>{{ item.title }}</strong>
                <small>{{ item.desc }}</small>
              </span>
              <AppIcon
                class="family-settings-arrow"
                name="chevronRight"
                :size="17"
              />
            </button>
          </div>
        </section>
      </template>
    </main>
  </AppShell>
</template>
