<script setup lang="ts">
defineProps<{ title: string; subtitle?: string }>()
</script>

<template>
  <header class="page-header">
    <div class="page-title">
      {{ title }}
    </div>
    <div
      v-if="subtitle"
      class="page-subtitle"
    >
      {{ subtitle }}
    </div>
  </header>
</template>

<style scoped>
.page-header {
  padding: calc(12px + var(--safe-top)) var(--space-16) var(--space-16);
  background: var(--color-bg-surface);
  border-bottom: 1px solid var(--color-divider);
}
.page-title { font-size: var(--font-size-section); font-weight: 600; color: var(--color-text-primary); }
.page-subtitle { font-size: var(--font-size-assist); color: var(--color-text-secondary); margin-top: 2px; }
</style>
