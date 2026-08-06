<script setup lang="ts">
import AppShell from './components/app/AppShell.vue'
</script>

<template>
  <router-view v-slot="{ Component }">
    <AppShell>
      <Suspense>
        <component :is="Component" />
      </Suspense>
    </AppShell>
  </router-view>
</template>
