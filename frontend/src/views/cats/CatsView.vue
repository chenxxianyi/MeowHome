<script setup lang="ts">
import { ref, onMounted } from 'vue'
import AppShell from '../../components/app/AppShell.vue'
import AppIcon from '../../components/app/AppIcon.vue'
import { services } from '../../services'
import { useCatStore } from '../../stores/cat'

const store = useCatStore()
const cats = ref(store.cats)
const loading = ref(true)

onMounted(async () => {
  const res = await services.getCats()
  cats.value = res.data
  loading.value = false
})
</script>

<template>
  <AppShell>
    <div class="page-header">
      <div class="page-title">
        猫咪
      </div>
    </div>

    <div class="page-content">
      <div
        v-if="loading"
        aria-busy="true"
      >
        <div
          v-for="i in 3"
          :key="i"
          class="skeleton"
          style="height:72px;margin-bottom:12px;"
        />
      </div>

      <template v-else>
        <a
          v-for="cat in cats"
          :key="cat.id"
          class="cat-list-item"
          :href="`#/cats/${cat.id}`"
          :aria-label="'查看 ' + cat.name + ' 详情'"
        >
          <img
            class="cat-avatar"
            :src="cat.avatar || ''"
            :alt="cat.name + '头像'"
          />
          <div class="cat-info">
            <div class="cat-name">{{ cat.name }}</div>
            <div class="cat-meta">
              {{ cat.age }}岁 · {{ cat.gender === 'female' ? '母' : '公' }} · {{ cat.breed }}
              {{ cat.neutered ? ' · 已绝育' : '' }}
            </div>
            <span class="cat-status normal">健康</span>
          </div>
          <span class="family-item-arrow"><AppIcon
            name="chevronRight"
            :size="20"
          /></span>
        </a>

        <button
          class="btn-primary"
          style="margin-top:16px;"
        >
          添加猫咪
        </button>
      </template>
    </div>
  </AppShell>
</template>
