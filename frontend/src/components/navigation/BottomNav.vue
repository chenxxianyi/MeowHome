<script setup lang="ts">
import { useRouter, useRoute } from 'vue-router'
import AppIcon from '../app/AppIcon.vue'

const router = useRouter()
const route = useRoute()

const navItems = [
  { id: 'today', label: '今日', icon: 'home', path: '/today' },
  { id: 'records', label: '记录', icon: 'record', path: '/records', recordNav: true },
  { id: 'cats', label: '猫咪', icon: 'cat', path: '/cats' },
  { id: 'moments', label: '时光', icon: 'timeline', path: '/moments' },
  { id: 'family', label: '家庭', icon: 'family', path: '/family' }
]

function isActive(path: string) {
  return route.path.startsWith(path)
}

function go(path: string) {
  router.push(path)
}
</script>

<template>
  <nav
    class="bottom-nav"
    aria-label="主导航"
  >
    <button
      v-for="item in navItems"
      :key="item.id"
      class="nav-item"
      :class="{ active: isActive(item.path), 'record-nav': item.recordNav }"
      :aria-label="item.label"
      :aria-current="isActive(item.path) ? 'page' : 'false'"
      @click="go(item.path)"
    >
      <AppIcon
        :name="item.icon"
        :size="24"
      />
      <span>{{ item.label }}</span>
    </button>
  </nav>
</template>

<style scoped>
/* 图标包裹在 nav-item 内的 svg 与 demo 用 span 不同，通过 scoped 调整 */
.nav-item :deep(svg) { width: 24px; height: 24px; }
</style>
