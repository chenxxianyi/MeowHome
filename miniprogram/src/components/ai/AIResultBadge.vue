<script setup lang="ts">
import AppIcon from '../app/AppIcon.vue'

defineProps<{ source?: string }>()
</script>

<template>
  <view class="ai-badge" :title="source">
    <AppIcon name="ai" :size="12" />
    <slot>{{ source || 'AI 整理' }}</slot>
  </view>
</template>

<style scoped>
.ai-badge {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  font-size: 11px;
  color: var(--color-ai-text);
  background: var(--color-ai-bg);
  padding: 2px 8px;
  border-radius: var(--radius-pill);
  border: 1px solid var(--color-ai-border);
}
</style>
