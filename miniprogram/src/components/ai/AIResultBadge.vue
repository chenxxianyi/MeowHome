<script setup lang="ts">
import AppIcon from '../app/AppIcon.vue'

defineProps<{ source?: string }>()
</script>

<template>
  <view class="ai-badge" :title="source">
    <view class="ai-badge-icon">
      <AppIcon name="ai" :size="12" />
    </view>
    <slot>{{ source || 'AI 整理' }}</slot>
  </view>
</template>

<style scoped>
.ai-badge {
  display: inline-flex;
  align-items: center;
  justify-content: flex-start;
  box-sizing: border-box;
  min-height: 28px;
  gap: 4px;
  font-size: 11px;
  line-height: 1.2;
  color: var(--color-ai-text);
  background: var(--color-ai-bg);
  padding: 2px 8px;
  border-radius: var(--radius-pill);
  border: 1px solid var(--color-ai-border);
}

.ai-badge-icon {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 12px;
  height: 16px;
  flex: 0 0 12px;
}
</style>
