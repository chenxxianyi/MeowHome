<script setup lang="ts">
defineProps<{ items: { type: string; catId?: string; time: string; content: string }[] }>()
</script>

<template>
  <view class="evidence-panel" role="region" aria-label="AI 分析依据">
    <view style="font-size: var(--font-size-assist); font-weight: 600; margin-bottom: 8px">
      <text style="color: var(--color-ai-text)">AI 分析依据</text>
    </view>
    <view v-for="(ev, i) in items" :key="i" class="evidence-row">
      <text>{{
        ev.type === 'feeding'
          ? '喂食'
          : ev.type === 'vomit'
            ? '呕吐'
            : ev.type === 'medication'
              ? '用药'
              : ev.type === 'water'
                ? '饮水'
                : ev.type
      }}</text>
      <text style="color: var(--color-text-tertiary)"> · {{ ev.time }}</text>
      <text
        style="display: block; font-size: var(--font-size-assist); color: var(--color-text-secondary); margin-top: 2px"
        >{{ ev.content }}</text
      >
    </view>
  </view>
</template>

<style scoped>
.evidence-row {
  padding: 6px 0;
  border-bottom: 1px solid var(--color-divider);
  font-size: var(--font-size-assist);
}
</style>
