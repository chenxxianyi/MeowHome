<script setup lang="ts">
import { computed } from 'vue'
import { useCatStore } from '../../stores/cat'
import AppIcon from '../app/AppIcon.vue'

const store = useCatStore()
const emit = defineEmits<{ select: [catId: string] }>()

const cats = computed(() => store.cats)
const active = computed(() => store.currentCatId)

function select(catId: string) {
  store.setCat(catId)
  emit('select', catId)
}
</script>

<template>
  <div
    class="cat-switcher"
    role="tablist"
    aria-label="切换猫咪"
  >
    <button
      class="cat-switcher-btn"
      :class="{ active: active === 'all' }"
      :aria-pressed="active === 'all'"
      @click="select('all')"
    >
      <AppIcon
        name="cat"
        :size="20"
      /> 全部
    </button>
    <button
      v-for="cat in cats"
      :key="cat.id"
      class="cat-switcher-btn"
      :class="{ active: active === cat.id }"
      :aria-pressed="active === cat.id"
      @click="select(cat.id)"
    >
      <img
        v-if="cat.avatar"
        class="cat-avatar-sm"
        :src="cat.avatar"
        :alt="cat.name + '头像'"
      />
      <span
        v-else
        class="cat-avatar-sm"
      ><AppIcon
        name="cat"
        :size="14"
      /></span>
      {{ cat.name }}
    </button>
  </div>
</template>
