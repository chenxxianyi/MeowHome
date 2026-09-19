<script setup lang="ts">
import AppIcon from '../app/AppIcon.vue'

defineProps<{ catId: string; name: string }>()
</script>

<template>
  <view class="cat-avatar" role="img" :aria-label="name + '头像'">
    <AppIcon name="cat" :size="28" />
  </view>
</template>

<style scoped>
.cat-avatar {
  width: 48px;
  height: 48px;
  border-radius: 50%;
  background: var(--color-brand-soft);
  color: var(--color-brand);
  display: flex;
  align-items: center;
  justify-content: center;
  overflow: hidden;
}
image {
  width: 100%;
  height: 100%;
  object-fit: cover;
}
</style>
