<script setup lang="ts">
import { ref, onMounted } from 'vue'
import AppShell from '../../components/app/AppShell.vue'
import AppIcon from '../../components/app/AppIcon.vue'
import { useInventoryStore } from '../../stores/inventory'
import { useExpenseStore } from '../../stores/expense'
import { useCatStore } from '../../stores/cat'
import { services } from '../../services'

const inventoryStore = useInventoryStore()
const expenseStore = useExpenseStore()
const catStore = useCatStore()
const family = ref({ name: '小家的猫宅' })
const inventory = ref<any[]>([])
const expenses = ref<any[]>([])
const loading = ref(true)

onMounted(async () => {
  const famRes = await services.getFamily()
  family.value = famRes.data
  const [invRes, expRes] = await Promise.all([
    services.getInventory(),
    services.getExpenses()
  ])
  inventory.value = invRes.data
  expenses.value = expRes.data
  loading.value = false
})

const lowCount = () => inventory.value.filter((i: any) => i.status === 'low' || i.status === 'expired').length

const menuGroups = [
  {
    title: '照顾与库存',
    items: [
      { icon: 'inbox', title: '库存', desc: `${lowCount()} 项需关注`, href: '/family/inventory', badge: lowCount() ? { type: 'warning', text: String(lowCount()) } : null },
      { icon: 'users', title: '照顾任务', desc: '查看家庭协作任务', href: '', badge: null },
      { icon: 'wallet', title: '养猫支出', desc: '统计本月支出', href: '/family/expenses', badge: null },
      { icon: 'bell', title: '提醒设置', desc: '自定义提醒规则', href: '', badge: null }
    ]
  },
  {
    title: '设置',
    items: [
      { icon: 'ai', title: 'AI 设置', desc: '管理 AI 管家权限与数据', href: '/settings', badge: null },
      { icon: 'users', title: '家庭成员', desc: '管理成员与权限', href: '', badge: null },
      { icon: 'import', title: '数据导入导出', desc: '备份与恢复数据', href: '/settings', badge: null },
      { icon: 'refresh', title: '备份与恢复', desc: '本地备份管理', href: '/settings', badge: null },
      { icon: 'eye', title: '隐私设置', desc: '管理数据可见性', href: '/settings', badge: null },
      { icon: 'family', title: '关于猫宅', desc: '版本与说明', href: '/settings', badge: null }
    ]
  }
]

function go(href: string) {
  if (!href) return
  const router = (window as any).__vue_router__
  if (router) router.push(href)
}
</script>

<template>
  <AppShell>
    <div class="page-header">
      <div class="page-title">
        家庭
      </div>
      <div class="page-subtitle">
        {{ family.name }}
      </div>
    </div>

    <div class="page-content">
      <!-- 猫咪管理 -->
      <div class="family-section">
        <div class="family-section-title">
          猫咪管理
        </div>
        <div class="family-list">
          <a
            v-for="cat in catStore.cats"
            :key="cat.id"
            class="family-item"
            :href="'#/cats/' + cat.id"
            :aria-label="'查看 ' + cat.name"
          >
            <div class="family-item-icon">
              <img
                v-if="cat.avatar"
                :src="cat.avatar || ''"
                :alt="cat.name"
                style="width:20px;height:20px;border-radius:50%;"
              />
              <AppIcon
                v-else
                name="cat"
                :size="20"
              />
            </div>
            <div class="family-item-content">
              <div class="family-item-title">{{ cat.name }}</div>
              <div class="family-item-desc">{{ cat.breed }} · {{ cat.age }}岁</div>
            </div>
            <span class="family-item-arrow"><AppIcon
              name="chevronRight"
              :size="20"
            /></span>
          </a>
        </div>
      </div>

      <!-- 照顾与库存 + 设置 -->
      <div
        v-for="group in menuGroups"
        :key="group.title"
        class="family-section"
      >
        <div class="family-section-title">
          {{ group.title }}
        </div>
        <div class="family-list">
          <a
            v-for="item in group.items"
            :key="item.title"
            class="family-item"
            :href="item.href ? '#/family/inventory' : '#'"
            :aria-label="item.title"
            @click.prevent="go(item.href)"
          >
            <div class="family-item-icon">
              <AppIcon
                :name="item.icon"
                :size="20"
              />
            </div>
            <div class="family-item-content">
              <div class="family-item-title">
                {{ item.title }}
                <span
                  v-if="item.badge"
                  class="inventory-badge"
                  :class="item.badge.type"
                >{{ item.badge.text }}</span>
              </div>
              <div class="family-item-desc">{{ item.desc }}</div>
            </div>
            <span class="family-item-arrow"><AppIcon
              name="chevronRight"
              :size="20"
            /></span>
          </a>
        </div>
      </div>
    </div>
  </AppShell>
</template>
