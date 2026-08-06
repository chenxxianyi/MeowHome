<script setup lang="ts">
import { ref, onMounted } from 'vue'
import AppShell from '../../components/app/AppShell.vue'
import AppIcon from '../../components/app/AppIcon.vue'
import { services } from '../../services'
import { useInventoryStore } from '../../stores/inventory'

const store = useInventoryStore()
const items = ref<any[]>([])
const loading = ref(true)

onMounted(async () => {
  const res = await services.getInventory()
  items.value = res.data
  loading.value = false
})

const lowItems = () => items.value.filter((i: any) => i.status === 'low' || i.status === 'expired')
</script>

<template>
  <AppShell>
    <div class="page-header">
      <div class="page-title">
        库存管理
      </div>
    </div>

    <div class="page-content">
      <!-- 需关注 -->
      <div
        v-if="lowItems().length"
        class="family-section"
      >
        <div class="section-title">
          需关注
        </div>
        <div class="reminder-list">
          <div
            v-for="item in lowItems()"
            :key="item.id"
            class="reminder-item"
          >
            <div class="reminder-left">
              <div class="reminder-icon inventory">
                <AppIcon
                  name="alertTriangle"
                  :size="18"
                />
              </div>
              <div class="reminder-text">
                <div class="reminder-title">
                  {{ item.name }}
                </div>
                <div class="reminder-subtitle">
                  {{ item.quantity }} {{ item.unit }} · {{ item.category }} · 剩余不足
                </div>
              </div>
            </div>
            <span
              class="inventory-badge"
              :class="item.status"
            >{{ item.status === 'low' ? '库存不足' : '即将过期' }}</span>
          </div>
        </div>
      </div>

      <!-- 全部库存 -->
      <div class="family-section">
        <div class="section-title">
          全部库存
        </div>
        <div class="reminder-list">
          <div
            v-for="item in items"
            :key="item.id"
            class="reminder-item"
          >
            <div class="reminder-left">
              <div class="reminder-icon inventory">
                <AppIcon
                  name="inbox"
                  :size="18"
                />
              </div>
              <div class="reminder-text">
                <div class="reminder-title">
                  {{ item.name }}
                </div>
                <div class="reminder-subtitle">
                  {{ item.quantity }} {{ item.unit }} · {{ item.category }}{{ item.expiry ? ' · 到期 ' + item.expiry : '' }}
                </div>
              </div>
            </div>
            <span
              class="inventory-badge"
              :class="item.status"
            >{{ item.status === 'normal' ? '正常' : item.status === 'low' ? '不足' : '过期' }}</span>
          </div>
        </div>
      </div>
    </div>
  </AppShell>
</template>
