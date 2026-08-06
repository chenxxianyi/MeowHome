<script setup lang="ts">
import { computed } from 'vue'
import BottomNav from '../navigation/BottomNav.vue'

const props = defineProps<{ noNav?: boolean }>()

const showNav = computed(() => !props.noNav)
</script>

<template>
  <div class="page">
    <slot />
    <BottomNav v-if="showNav" />
  </div>
</template>

<style scoped>
.page {
  min-height: 100dvh;
  padding-bottom: calc(64px + var(--safe-bottom));
  display: flex;
  flex-direction: column;
}
</style>
