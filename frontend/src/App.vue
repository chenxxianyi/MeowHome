<script setup lang="ts">
import { computed } from 'vue'
import { useRoute } from 'vue-router'

import AppShell from './components/app/AppShell.vue'

const route = useRoute()
// 登录页等公共路由不套应用外壳（无底部导航/头部）
const bare = computed(() => Boolean(route.meta.public))
</script>

<template>
  <router-view v-slot="{ Component }">
    <Suspense>
      <component :is="Component" v-if="bare" />
      <AppShell v-else>
        <component :is="Component" />
      </AppShell>
    </Suspense>
  </router-view>
</template>
