<script setup lang="ts">
import { computed, ref } from 'vue'
import AppIcon from '../../components/app/AppIcon.vue'
import { toApiError } from '../../api/adapter'
import { services } from '../../services'
import { useInventoryStore } from '../../stores/inventory'
import { useShoppingStore } from '../../stores/shopping'
import { useRouter } from '../../utils/navigation'
import { useProtectedPage } from '../../utils/page'
import type { InventoryItem, ShoppingItem } from '../../types'

const router = useRouter()
const store = useInventoryStore()
const shoppingStore = useShoppingStore()

const items = ref<InventoryItem[]>([])
const shopping = ref<ShoppingItem[]>([])
const loading = ref(true)
const activeTab = ref<'stock' | 'shopping'>('stock')

// 添加库存抽屉
const stockSheetOpen = ref(false)
const stockForm = ref({ name: '', quantity: '1', unit: '', category: '', note: '' })
// 添加待购抽屉
const shoppingSheetOpen = ref(false)
const shoppingForm = ref({ name: '', quantity: '1', unit: '', note: '', priority: 'normal' })
const submitting = ref(false)
const sheetError = ref('')
// 调整中的库存项（禁用重复点击）
const adjustingId = ref('')

const lowItems = computed(() => items.value.filter((item) => item.status === 'low' || item.status === 'expired'))
const todoShopping = computed(() => shopping.value.filter((item) => item.status === 'todo'))
const doneShopping = computed(() => shopping.value.filter((item) => item.status !== 'todo'))

const PRIORITIES = [
  { value: 'low', label: '不急' },
  { value: 'normal', label: '普通' },
  { value: 'high', label: '优先' }
]
const priorityLabel = (value: string) => PRIORITIES.find((item) => item.value === value)?.label || '普通'

async function load() {
  const [inv, shop] = await Promise.all([services.getInventory(), services.getShoppingList()])
  items.value = inv.data
  store.set(inv.data)
  shopping.value = shop.data
  shoppingStore.set(shop.data)
  loading.value = false
}

function goPurchase() {
  router.push('/purchase')
}

// ---- 库存调整 ----

async function adjust(item: InventoryItem, delta: number) {
  if (adjustingId.value) return
  adjustingId.value = item.id
  try {
    const result = await services.adjustInventory(item.id, delta, delta > 0 ? '手动入库' : '手动消耗')
    const index = items.value.findIndex((row) => row.id === result.item.id)
    if (index >= 0) items.value.splice(index, 1, result.item)
  } catch (e) {
    uni.showToast({ title: toApiError(e).message || '调整失败', icon: 'none' })
  } finally {
    adjustingId.value = ''
  }
}

// ---- 添加库存 ----

function openStockSheet() {
  stockForm.value = { name: '', quantity: '1', unit: '', category: '', note: '' }
  sheetError.value = ''
  stockSheetOpen.value = true
}

function closeStockSheet() {
  stockSheetOpen.value = false
}

async function submitStock() {
  if (submitting.value) return
  const name = stockForm.value.name.trim()
  if (!name) {
    sheetError.value = '请填写物品名称'
    return
  }
  submitting.value = true
  sheetError.value = ''
  try {
    await services.createInventoryItem({
      name,
      quantity: Math.max(0, parseInt(stockForm.value.quantity, 10) || 0),
      unit: stockForm.value.unit.trim(),
      category: stockForm.value.category.trim() || '其他'
    })
    stockSheetOpen.value = false
    await load()
    uni.showToast({ title: '已添加', icon: 'success' })
  } catch (e) {
    sheetError.value = toApiError(e).message || '保存失败，请稍后重试'
  } finally {
    submitting.value = false
  }
}

// ---- 添加待购 ----

function openShoppingSheet() {
  shoppingForm.value = { name: '', quantity: '1', unit: '', note: '', priority: 'normal' }
  sheetError.value = ''
  shoppingSheetOpen.value = true
}

function closeShoppingSheet() {
  shoppingSheetOpen.value = false
}

async function submitShopping() {
  if (submitting.value) return
  const name = shoppingForm.value.name.trim()
  if (!name) {
    sheetError.value = '请填写物品名称'
    return
  }
  submitting.value = true
  sheetError.value = ''
  try {
    const created = await services.createShoppingItem({
      name,
      quantity: Math.max(1, parseInt(shoppingForm.value.quantity, 10) || 1),
      unit: shoppingForm.value.unit.trim(),
      note: shoppingForm.value.note.trim(),
      priority: shoppingForm.value.priority
    })
    shoppingStore.upsert(created)
    shopping.value = [...shoppingStore.items]
    shoppingSheetOpen.value = false
    uni.showToast({ title: '已加入待购', icon: 'success' })
  } catch (e) {
    sheetError.value = toApiError(e).message || '保存失败，请稍后重试'
  } finally {
    submitting.value = false
  }
}

// ---- 待购状态 ----

async function completeShopping(item: ShoppingItem) {
  try {
    const updated = await services.completeShoppingItem(item.id)
    shoppingStore.upsert(updated)
    shopping.value = [...shoppingStore.items]
    uni.showToast({ title: '已标记购买', icon: 'success' })
  } catch (e) {
    uni.showToast({ title: toApiError(e).message || '操作失败', icon: 'none' })
  }
}

async function dismissShopping(item: ShoppingItem) {
  try {
    const updated = await services.dismissShoppingItem(item.id)
    shoppingStore.upsert(updated)
    shopping.value = [...shoppingStore.items]
  } catch (e) {
    uni.showToast({ title: toApiError(e).message || '操作失败', icon: 'none' })
  }
}

async function removeShopping(item: ShoppingItem) {
  try {
    await services.deleteShoppingItem(item.id)
    shoppingStore.remove(item.id)
    shopping.value = [...shoppingStore.items]
  } catch (e) {
    uni.showToast({ title: toApiError(e).message || '删除失败', icon: 'none' })
  }
}

useProtectedPage(load)
</script>

<template>
  <view class="page-header">
    <view class="page-title"> 库存管理 </view>
  </view>

  <view class="page-content">
    <view class="segmented inventory-tabs">
      <button type="button" class="segment" :class="{ active: activeTab === 'stock' }" @click="activeTab = 'stock'">
        库存
      </button>
      <button
        type="button"
        class="segment"
        :class="{ active: activeTab === 'shopping' }"
        @click="activeTab = 'shopping'"
      >
        待购买{{ todoShopping.length ? ' · ' + todoShopping.length : '' }}
      </button>
    </view>

    <view v-if="loading" aria-busy="true">
      <view class="skeleton" style="height: 120px; margin-bottom: 16px" />
      <view class="skeleton" style="height: 60px; margin-bottom: 12px" />
    </view>

    <!-- 库存 -->
    <template v-else-if="activeTab === 'stock'">
      <view v-if="lowItems.length" class="family-section">
        <view class="section-title"> 需关注 </view>
        <view class="reminder-list">
          <view v-for="item in lowItems" :key="item.id" class="reminder-item">
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
            <view class="inventory-actions">
              <button
                type="button"
                class="inventory-adjust-btn"
                :disabled="adjustingId === item.id"
                aria-label="减少一件"
                @click.stop="adjust(item, -1)"
              >
                −
              </button>
              <button
                type="button"
                class="inventory-adjust-btn"
                :disabled="adjustingId === item.id"
                aria-label="增加一件"
                @click.stop="adjust(item, 1)"
              >
                +
              </button>
            </view>
          </view>
        </view>
        <view v-if="!items.length" class="empty-state">暂无库存记录</view>
        <view class="inventory-tip">每次加减都会记录一条库存流水，用于追溯数量变化。</view>
      </view>

      <button class="btn-primary inventory-add" @click="openStockSheet"> 添加物资 </button>
      <button class="btn-secondary inventory-add" @click="goPurchase"> 识别订单截图 </button>
    </template>

    <!-- 待购买 -->
    <template v-else>
      <view class="family-section">
        <view class="section-title"> 待购买 </view>
        <view class="reminder-list">
          <view v-for="item in todoShopping" :key="item.id" class="reminder-item shopping-item">
            <view class="reminder-left">
              <view class="reminder-icon inventory">
                <AppIcon name="inbox" :size="18" />
              </view>
              <view class="reminder-text">
                <view class="reminder-title">
                  {{ item.name }}
                  <text class="shopping-priority" :class="item.priority">{{ priorityLabel(item.priority) }}</text>
                </view>
                <view class="reminder-subtitle">
                  {{ item.quantity }} {{ item.unit || '件' }}
                  <text v-if="item.note"> · {{ item.note }}</text>
                </view>
              </view>
            </view>
            <view class="shopping-actions">
              <button type="button" class="shopping-btn done" @click="completeShopping(item)"> 已购买 </button>
              <button type="button" class="shopping-btn" @click="dismissShopping(item)"> 不买了 </button>
              <button type="button" class="shopping-btn" @click="removeShopping(item)"> 移除 </button>
            </view>
          </view>
        </view>
        <view v-if="!todoShopping.length" class="empty-state">待购清单是空的，需要买点什么？</view>
      </view>

      <view v-if="doneShopping.length" class="family-section">
        <view class="section-title"> 已处理 </view>
        <view class="reminder-list">
          <view v-for="item in doneShopping" :key="item.id" class="reminder-item shopping-item done-item">
            <view class="reminder-left">
              <view class="reminder-icon inventory">
                <AppIcon name="check" :size="18" />
              </view>
              <view class="reminder-text">
                <view class="reminder-title">
                  {{ item.name }}
                </view>
                <view class="reminder-subtitle">
                  {{ item.quantity }} {{ item.unit || '件' }} ·
                  {{ item.status === 'purchased' ? '已购买' : '已移除' }}
                </view>
              </view>
            </view>
          </view>
        </view>
      </view>

      <button class="btn-primary inventory-add" @click="openShoppingSheet"> 添加待购 </button>
      <button class="btn-secondary inventory-add" @click="goPurchase"> 识别订单截图 </button>
    </template>
  </view>

  <!-- 添加物资抽屉 -->
  <view class="sheet-overlay" :class="{ show: stockSheetOpen }" @click.self="closeStockSheet">
    <view class="sheet-content" role="dialog" aria-modal="true" aria-labelledby="add-stock-title">
      <view class="sheet-handle" />
      <view id="add-stock-title" class="sheet-title"> 添加物资 </view>
      <view class="sheet-body">
        <label class="field">
          <text class="field-label">名称 <text class="req">*</text></text>
          <input v-model="stockForm.name" class="field-input" type="text" maxlength="255" placeholder="例如：皇家成猫粮" />
        </label>
        <view class="field-row-2">
          <label class="field">
            <text class="field-label">数量</text>
            <input v-model="stockForm.quantity" class="field-input" type="number" placeholder="0" />
          </label>
          <label class="field">
            <text class="field-label">单位</text>
            <input v-model="stockForm.unit" class="field-input" type="text" maxlength="32" placeholder="袋" />
          </label>
        </view>
        <label class="field">
          <text class="field-label">分类</text>
          <input v-model="stockForm.category" class="field-input" type="text" maxlength="64" placeholder="food / litter / supplies" />
        </label>
        <text v-if="sheetError" class="form-error">{{ sheetError }}</text>
      </view>
      <view class="sheet-actions">
        <button class="btn-secondary" :disabled="submitting" @click="closeStockSheet"> 取消 </button>
        <button class="btn-primary" :disabled="submitting" @click="submitStock"> 保存 </button>
      </view>
    </view>
  </view>

  <!-- 添加待购抽屉 -->
  <view class="sheet-overlay" :class="{ show: shoppingSheetOpen }" @click.self="closeShoppingSheet">
    <view class="sheet-content" role="dialog" aria-modal="true" aria-labelledby="add-shopping-title">
      <view class="sheet-handle" />
      <view id="add-shopping-title" class="sheet-title"> 添加待购 </view>
      <view class="sheet-body">
        <label class="field">
          <text class="field-label">名称 <text class="req">*</text></text>
          <input v-model="shoppingForm.name" class="field-input" type="text" maxlength="255" placeholder="例如：猫砂" />
        </label>
        <view class="field-row-2">
          <label class="field">
            <text class="field-label">数量</text>
            <input v-model="shoppingForm.quantity" class="field-input" type="number" placeholder="1" />
          </label>
          <label class="field">
            <text class="field-label">单位</text>
            <input v-model="shoppingForm.unit" class="field-input" type="text" maxlength="32" placeholder="袋" />
          </label>
        </view>
        <label class="field">
          <text class="field-label">备注</text>
          <input v-model="shoppingForm.note" class="field-input" type="text" maxlength="200" placeholder="可选，例如：买 2.5kg 那款" />
        </label>
        <view class="field">
          <text class="field-label">优先级</text>
          <view class="segmented">
            <button
              v-for="option in PRIORITIES"
              :key="option.value"
              type="button"
              class="segment"
              :class="{ active: shoppingForm.priority === option.value }"
              @click="shoppingForm.priority = option.value"
            >
              {{ option.label }}
            </button>
          </view>
        </view>
        <text v-if="sheetError" class="form-error">{{ sheetError }}</text>
      </view>
      <view class="sheet-actions">
        <button class="btn-secondary" :disabled="submitting" @click="closeShoppingSheet"> 取消 </button>
        <button class="btn-primary" :disabled="submitting" @click="submitShopping"> 保存 </button>
      </view>
    </view>
  </view>
</template>

<style scoped>
/* 抽屉内表单：复用全局 .sheet-content 外观，只覆盖布局与字段样式 */
.sheet-content {
  display: flex;
  flex-direction: column;
  max-height: 86vh; /* WXSS 不支持 dvh */
  overflow: hidden;
}

.sheet-body {
  flex: 1;
  min-height: 0;
  overflow-y: auto;
}

.sheet-actions {
  flex: none;
  flex-direction: row;
  gap: 10px;
  margin-top: 12px;
}

.field {
  display: block;
  margin-bottom: 14px;
}

.field-label {
  display: block;
  margin-bottom: 6px;
  font-size: 13px;
  color: #7a6f5d;
}

.req {
  color: #b4453a;
}

.field-input {
  width: 100%;
  box-sizing: border-box;
  padding: 11px 14px;
  font-size: 15px;
  color: #3d3426;
  background: #fffdfa;
  border: 1px solid #e8e1d5;
  border-radius: 12px;
}

.picker-value {
  color: #3d3426;
}

.segmented {
  display: flex;
  gap: 8px;
}

.segment {
  flex: 1;
  padding: 10px 0;
  font-size: 14px;
  line-height: 1.2;
  color: #7a6f5d;
  background: #fffdfa;
  border: 1px solid #e8e1d5;
  border-radius: 12px;
}

.segment.active {
  color: #fff;
  background: #d9a441;
  border-color: #d9a441;
}

.form-error {
  display: block;
  margin: 4px 0 12px;
  padding: 9px 12px;
  font-size: 13px;
  color: #b4453a;
  background: #fdf1ef;
  border-radius: 10px;
}

.inventory-tabs {
  margin-bottom: var(--space-16);
}

.inventory-actions {
  display: flex;
  gap: var(--space-8);
}

.inventory-adjust-btn {
  width: 32px;
  height: 32px;
  min-width: 32px;
  padding: 0;
  border-radius: 50%;
  border: 1px solid var(--color-divider);
  background: var(--color-bg-surface);
  color: var(--color-text-primary);
  font-size: 18px;
  line-height: 30px;
}

.inventory-adjust-btn:disabled {
  opacity: 0.4;
}

.inventory-tip {
  margin-top: var(--space-8);
  font-size: var(--font-size-caption);
  color: var(--color-text-tertiary);
}

.inventory-add {
  margin-top: var(--space-12);
  width: 100%;
}

.shopping-item {
  align-items: flex-start;
}

.shopping-actions {
  display: flex;
  flex-direction: column;
  gap: var(--space-4);
  align-items: flex-end;
}

.shopping-btn {
  padding: 2px var(--space-8);
  font-size: var(--font-size-caption);
  border-radius: 999px;
  border: 1px solid var(--color-divider);
  background: var(--color-bg-surface);
  color: var(--color-text-secondary);
  line-height: 20px;
}

.shopping-btn.done {
  color: var(--color-brand);
  border-color: var(--color-brand-soft);
}

.shopping-priority {
  margin-left: var(--space-4);
  padding: 0 6px;
  border-radius: 999px;
  font-size: 10px;
  font-weight: 500;
  color: var(--color-text-secondary);
  background: var(--color-bg-muted);
}

.shopping-priority.high {
  color: var(--color-danger);
  background: var(--color-danger-soft);
}

.shopping-priority.low {
  color: var(--color-text-tertiary);
}

.done-item {
  opacity: 0.65;
}

.field-row-2 {
  display: flex;
  gap: var(--space-12);
}

.field-row-2 .field {
  flex: 1;
}
</style>
