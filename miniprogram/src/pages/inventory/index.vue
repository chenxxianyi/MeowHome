<script setup lang="ts">
import { ref } from 'vue'
import AppIcon from '../../components/app/AppIcon.vue'
import { services } from '../../services'
import { useInventoryStore } from '../../stores/inventory'
import { useProtectedPage } from '../../utils/page'
import type { InventoryItem } from '../../types'

const store = useInventoryStore()
const items = ref<InventoryItem[]>([])
const loading = ref(true)

async function load() {
  const res = await services.getInventory()
  items.value = res.data
  store.set(res.data)
  loading.value = false
}

const lowItems = () => items.value.filter((item) => item.status === 'low' || item.status === 'expired')

useProtectedPage(load)
</script>

<template>
  <view class="page-header">
    <view class="page-title"> 库存管理 </view>
  </view>

  <view class="page-content">
    <!-- 需关注 -->
    <view v-if="lowItems().length" class="family-section">
      <view class="section-title"> 需关注 </view>
      <view class="reminder-list">
        <view v-for="item in lowItems()" :key="item.id" class="reminder-item">
          <view class="reminder-left">
            <view class="reminder-icon inventory">
              <AppIcon name="alertTriangle" :size="18" />
            </view>
            <view class="reminder-text">
              <view class="reminder-title">
                {{ item.name }}
              </view>
              <view class="reminder-subtitle">
                {{ item.quantity }} {{ item.unit }} · {{ item.category }} ·
                {{ item.status === 'expired' ? '已过期' : '剩余不足' }}
              </view>
            </view>
          </view>
          <text class="inventory-badge" :class="item.status">{{ item.status === 'low' ? '库存不足' : '已过期' }}</text>
        </view>
      </view>
    </view>

    <!-- 全部库存 -->
    <view class="family-section">
      <view class="section-title"> 全部库存 </view>
      <view class="reminder-list">
        <view v-for="item in items" :key="item.id" class="reminder-item">
          <view class="reminder-left">
            <view class="reminder-icon inventory">
              <AppIcon name="inbox" :size="18" />
            </view>
            <view class="reminder-text">
              <view class="reminder-title">
                {{ item.name }}
              </view>
              <view class="reminder-subtitle">
                {{ item.quantity }} {{ item.unit }} · {{ item.category
                }}{{ item.expiry ? ' · 到期 ' + item.expiry : '' }}
              </view>
            </view>
          </view>
          <text class="inventory-badge" :class="item.status">{{
            item.status === 'ok' ? '正常' : item.status === 'low' ? '不足' : '过期'
          }}</text>
        </view>
      </view>
      <view v-if="!loading && !items.length" class="empty-state">暂无库存记录</view>
    </view>
  </view>
</template>
