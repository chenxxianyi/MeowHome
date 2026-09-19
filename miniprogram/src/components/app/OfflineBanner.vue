<script setup lang="ts">
import { storeToRefs } from 'pinia'

import { useAppStore } from '../../stores/app'

const { isOffline } = storeToRefs(useAppStore())
</script>

<template>
  <view v-if="isOffline" class="offline-banner" role="status"> 当前处于离线状态，读取与保存可能失败 </view>
</template>

<style scoped>
.offline-banner {
  position: sticky;
  top: 0;
  z-index: 80;
  padding: 8px 12px;
  color: #7a441f;
  background: #fff0d8;
  border-bottom: 1px solid #e5bf87;
  text-align: center;
  font-size: var(--font-size-assist);
}
</style>
