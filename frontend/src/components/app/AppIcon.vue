<script setup lang="ts">
import { computed } from 'vue'
import { iconPaths, type AppIconName } from './iconPaths'

const props = withDefaults(defineProps<{ name: AppIconName; size?: number }>(), {
  size: 20
})

const content = computed(() => iconPaths[props.name] || '')
</script>

<template>
  <svg
    xmlns="http://www.w3.org/2000/svg"
    :width="size"
    :height="size"
    viewBox="0 0 24 24"
    fill="none"
    stroke="currentColor"
    stroke-width="1.8"
    stroke-linecap="round"
    stroke-linejoin="round"
    aria-hidden="true"
    v-html="content"
  />
</template>
