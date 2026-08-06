<script setup lang="ts">
import { ref, onMounted } from 'vue'
import AppShell from '../../components/app/AppShell.vue'
import AppIcon from '../../components/app/AppIcon.vue'
import { services } from '../../services'

const settings = ref<any[]>([
  { icon: 'ai', title: 'AI 管家', desc: '启用/禁用 AI 功能', href: '/settings' },
  { icon: 'eye', title: '隐私设置', desc: '管理数据可见性', href: '/settings' },
  { icon: 'export', title: '数据导出', desc: '导出 JSON 或 CSV', href: '/settings' },
  { icon: 'import', title: '数据导入', desc: '从备份恢复', href: '/settings' },
  { icon: 'refresh', title: '备份与恢复', desc: '本地备份管理', href: '/settings' },
  { icon: 'info', title: '关于猫宅', desc: 'v0.1.0-demo · 开发数据', href: '/settings' }
])

const loading = ref(true)
onMounted(() => {
  setTimeout(() => { loading.value = false }, 300)
})
</script>

<template>
  <AppShell>
    <div class="page-header">
      <div class="page-title">
        设置
      </div>
    </div>

    <div class="page-content">
      <div class="family-section">
        <div class="family-section-title">
          通用
        </div>
        <div class="family-list">
          <div
            v-for="item in settings"
            :key="item.title"
            class="family-item"
            role="button"
            :aria-label="item.title"
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
              </div>
              <div class="family-item-desc">
                {{ item.desc }}
              </div>
            </div>
            <span class="family-item-arrow"><AppIcon
              name="chevronRight"
              :size="20"
            /></span>
          </div>
        </div>
      </div>
    </div>
  </AppShell>
</template>
