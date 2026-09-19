<script setup lang="ts">
import { computed, ref } from 'vue'
import { useRouter } from '../../utils/navigation'
import AppIcon from '../../components/app/AppIcon.vue'
import OfflineBanner from '../../components/app/OfflineBanner.vue'
import { useCatStore } from '../../stores/cat'
import { services } from '../../services'
import { todayStr } from '../../utils/date'
import { usePageCapabilities, useProtectedPage } from '../../utils/page'
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
const loadError = ref('')

const cats = computed(() => catStore.cats)
const lowCount = computed(
  () => inventory.value.filter((item) => item.status === 'low' || item.status === 'expired').length
)
const currentMonth = todayStr().slice(0, 7)
const monthExpense = computed(() =>
  expenses.value
    .filter((expense) => expense.date.startsWith(currentMonth))
    .reduce((sum, expense) => sum + expense.amount, 0)
)
const homeStatus = computed(() => (lowCount.value ? `${lowCount.value} 项物资需要关注` : '今日照顾状态良好'))

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

async function load() {
  loading.value = true
  loadError.value = ''
  try {
    const [famRes, invRes, expRes] = await Promise.all([
      services.getFamily(),
      services.getInventory(),
      services.getExpenses()
    ])
    family.value = famRes.data
    inventory.value = invRes.data
    expenses.value = expRes.data
    if (!famRes.success && !invRes.success && !expRes.success) {
      loadError.value = '家庭数据暂时无法加载，请稍后重试'
    }
  } finally {
    loading.value = false
  }
}

function go(href: string) {
  router.push(href)
}

useProtectedPage(load)
usePageCapabilities(load, '猫宅 · 一起照顾，彼此放心')
</script>

<template>
  <OfflineBanner />
  <view class="page-header family-page-header">
    <view>
      <view class="page-title"> 家庭 </view>
      <view class="page-subtitle"> 一起照顾，彼此放心 </view>
    </view>
    <button type="button" class="family-header-action" aria-label="打开家庭设置" @click="go('/settings')">
      <AppIcon name="settings" :size="20" />
    </button>
  </view>

  <view class="page-content family-dashboard">
    <view v-if="loading" class="family-dashboard-loading" aria-label="正在加载家庭信息" aria-busy="true">
      <view class="skeleton family-hero-skeleton" />
      <view class="skeleton family-cats-skeleton" />
      <view class="skeleton family-actions-skeleton" />
    </view>

    <view v-else-if="loadError" class="empty-state">
      <view>{{ loadError }}</view>
      <button class="btn-secondary" @click="load">重新加载</button>
    </view>

    <template v-else>
      <view class="family-home-card">
        <view class="family-home-topline">
          <text class="family-home-eyebrow">MEOWHOME · OUR HOME</text>
          <button type="button" class="family-home-manage" @click="go('/settings')">
            管理
            <AppIcon name="chevronRight" :size="14" />
          </button>
        </view>
        <view>{{ family.name }}</view>
        <view>{{ cats.length }} 只猫咪 · {{ family.members.length }} 位家人在共同照顾</view>
        <view class="family-home-bottom">
          <view class="family-home-cat-stack" aria-label="家庭猫咪">
            <view
              v-for="(cat, index) in cats"
              :key="cat.id"
              class="family-home-cat-avatar"
              :class="`tone-${index % 2}`"
              :title="cat.name"
            >
              <image mode="aspectFill" v-if="cat.avatar" :src="cat.avatar" :alt="cat.name" />
              <AppIcon v-else name="cat" :size="18" />
            </view>
          </view>
          <text class="family-home-status" :class="{ warning: lowCount }"> <text />{{ homeStatus }} </text>
        </view>
      </view>

      <view class="family-dashboard-section">
        <view class="family-dashboard-heading">
          <view>
            <text>FAMILY MEMBERS</text>
            <view>共同照顾的家人</view>
          </view>
        </view>
        <view v-if="!family.members.length" class="empty-state">暂无家庭成员</view>
        <view v-for="member in family.members" :key="member.id" class="family-member-item">
          <view class="family-member-avatar">{{ member.name.slice(0, 1) }}</view>
          <view class="family-member-copy">
            <text>{{ member.name }}</text>
            <view>{{ member.role === 'owner' ? '家庭创建者' : '家庭成员' }}</view>
          </view>
        </view>
      </view>

      <view class="family-dashboard-section">
        <view class="family-dashboard-heading">
          <view>
            <text>OUR CATS</text>
            <view>家里的小成员</view>
          </view>
          <button type="button" @click="go('/cats')">
            全部档案
            <AppIcon name="chevronRight" :size="14" />
          </button>
        </view>

        <view class="family-cat-grid">
          <button
            v-for="(cat, index) in cats"
            :key="cat.id"
            type="button"
            class="family-cat-card"
            :class="`tone-${index % 2}`"
            :aria-label="`查看 ${cat.name} 的健康档案`"
            @click="go(`/cats/${cat.id}`)"
          >
            <view class="family-cat-portrait">
              <image mode="aspectFill" v-if="cat.avatar" :src="cat.avatar" :alt="cat.name" />
              <AppIcon v-else name="cat" :size="32" />
            </view>
            <text class="family-cat-health" :class="{ attention: cat.diseases.length }">
              {{ cat.diseases.length ? cat.diseases[0] : '健康状态良好' }}
            </text>
            <text>{{ cat.name }}</text>
            <view>{{ cat.breed }} · {{ cat.age }} 岁</view>
            <view class="family-cat-link">
              健康档案
              <AppIcon name="chevronRight" :size="14" />
            </view>
          </button>
        </view>
      </view>

      <view class="family-dashboard-section">
        <view class="family-dashboard-heading">
          <view>
            <text>CARE TOGETHER</text>
            <view>家庭管家</view>
          </view>
        </view>

        <view class="family-care-grid">
          <button
            v-for="item in careEntries"
            :key="item.title"
            type="button"
            class="family-care-card"
            :class="`tone-${item.tone}`"
            @click="go(item.href)"
          >
            <view class="family-care-icon">
              <AppIcon :name="item.icon" :size="20" />
            </view>
            <text class="family-care-value">{{ item.value }}</text>
            <text>{{ item.title }}</text>
            <view>{{ item.desc }}</view>
          </button>
        </view>
      </view>

      <view class="family-settings-section">
        <view class="family-dashboard-heading">
          <view>
            <text>HOUSEHOLD SETTINGS</text>
            <view>家庭与数据</view>
          </view>
        </view>
        <view class="family-settings-list">
          <button
            v-for="item in settingsEntries"
            :key="item.title"
            type="button"
            class="family-settings-item"
            @click="go('/settings')"
          >
            <view class="family-settings-icon">
              <AppIcon :name="item.icon" :size="18" />
            </view>
            <view>
              <text>{{ item.title }}</text>
              <view>{{ item.desc }}</view>
            </view>
            <AppIcon class="family-settings-arrow" name="chevronRight" :size="17" />
          </button>
        </view>
      </view>
    </template>
  </view>
</template>

<style scoped>
.empty-state {
  padding: var(--space-24);
  text-align: center;
  color: var(--color-text-tertiary);
}
.empty-state .btn-secondary {
  margin-top: var(--space-12);
}
.family-member-item {
  display: flex;
  align-items: center;
  gap: var(--space-12);
  padding: var(--space-12);
  border-bottom: 1px solid var(--color-divider);
  background: var(--color-bg-surface);
}
.family-member-avatar {
  width: 38px;
  height: 38px;
  display: flex;
  align-items: center;
  justify-content: center;
  border-radius: 50%;
  color: var(--color-brand);
  background: var(--color-brand-soft);
}
.family-member-copy {
  flex: 1;
}
.family-member-copy > text {
  font-weight: 600;
  color: var(--color-text-primary);
}
.family-member-copy > view {
  margin-top: 2px;
  color: var(--color-text-tertiary);
  font-size: var(--font-size-caption);
}
</style>
