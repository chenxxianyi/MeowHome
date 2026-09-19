<script setup lang="ts">
import { ref } from 'vue'
import AppIcon from '../../components/app/AppIcon.vue'
import { useAuthStore } from '../../stores/auth'
import { useCatStore } from '../../stores/cat'
import { navigate } from '../../utils/guard'
import { useProtectedPage } from '../../utils/page'

const settings = ref<any[]>([
  { icon: 'ai', title: 'AI 管家', desc: '启用/禁用 AI 功能', href: '/settings' },
  { icon: 'eye', title: '隐私设置', desc: '管理数据可见性', href: '/settings' },
  { icon: 'export', title: '数据导出', desc: '导出 JSON 或 CSV', href: '/settings' },
  { icon: 'import', title: '数据导入', desc: '从备份恢复', href: '/settings' },
  { icon: 'refresh', title: '备份与恢复', desc: '本地备份管理', href: '/settings' },
  { icon: 'info', title: '关于猫宅', desc: 'v0.1.0 · 小程序版', href: '/settings' }
])

const auth = useAuthStore()
const catStore = useCatStore()
const loggingOut = ref(false)

function explain(item: { title: string }) {
  uni.showModal({
    title: item.title,
    content:
      item.title === '关于猫宅' ? '猫宅 MeowHome 小程序版 v0.1.0' : '该设置项尚未接入后端配置，本版本暂不可修改。',
    showCancel: false
  })
}

function requestLogout() {
  uni.showModal({
    title: '退出登录',
    content: '确认清除本机登录状态并返回登录页吗？',
    success: async ({ confirm }) => {
      if (!confirm || loggingOut.value) return
      loggingOut.value = true
      await auth.logout()
      catStore.reset()
      loggingOut.value = false
      navigate('/pages/auth/index')
    }
  })
}

useProtectedPage()
</script>

<template>
  <view class="page-header">
    <view class="page-title"> 设置 </view>
  </view>

  <view class="page-content">
    <view class="account-card">
      <view class="account-avatar">{{ auth.displayName.slice(0, 1) || '猫' }}</view>
      <view>
        <text>{{ auth.displayName || '猫宅用户' }}</text>
        <view>{{ auth.isOwner ? '家庭创建者' : '家庭成员' }}</view>
      </view>
    </view>
    <view class="family-section">
      <view class="family-section-title"> 通用 </view>
      <view class="family-list">
        <view
          v-for="item in settings"
          :key="item.title"
          class="family-item"
          role="button"
          :aria-label="item.title"
          @click="explain(item)"
        >
          <view class="family-item-icon">
            <AppIcon :name="item.icon" :size="20" />
          </view>
          <view class="family-item-content">
            <view class="family-item-title">
              {{ item.title }}
            </view>
            <view class="family-item-desc">
              {{ item.desc }}
            </view>
          </view>
          <view class="family-item-arrow"><AppIcon name="chevronRight" :size="20" /></view>
        </view>
      </view>
    </view>
    <button class="logout-button" :disabled="loggingOut" @click="requestLogout">
      {{ loggingOut ? '正在退出…' : '退出登录' }}
    </button>
  </view>
</template>

<style scoped>
.account-card {
  display: flex;
  align-items: center;
  gap: var(--space-12);
  padding: var(--space-16);
  margin-bottom: var(--space-16);
  background: var(--color-bg-surface);
  border-radius: var(--radius-md);
}
.account-avatar {
  width: 48px;
  height: 48px;
  display: flex;
  align-items: center;
  justify-content: center;
  border-radius: 50%;
  color: var(--color-brand);
  background: var(--color-brand-soft);
  font-size: var(--font-size-section);
  font-weight: 700;
}
.account-card text {
  font-weight: 600;
  color: var(--color-text-primary);
}
.account-card view view {
  margin-top: 3px;
  color: var(--color-text-tertiary);
  font-size: var(--font-size-assist);
}
.logout-button {
  width: 100%;
  margin-top: var(--space-24);
  color: var(--color-danger);
  background: var(--color-bg-surface);
  border: 1px solid var(--color-danger-soft);
}
</style>
