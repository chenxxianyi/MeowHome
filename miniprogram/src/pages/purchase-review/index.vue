<script setup lang="ts">
import { computed, reactive, ref } from 'vue'
import { onLoad } from '@dcloudio/uni-app'
import AppIcon from '../../components/app/AppIcon.vue'
import { toApiError } from '../../api/adapter'
import { services } from '../../services'
import { useRouter } from '../../utils/navigation'
import { useProtectedPage } from '../../utils/page'
import type { PurchaseImport, PurchaseLine, ShoppingItem } from '../../types'

const router = useRouter()

const importId = ref('')
const imp = ref<PurchaseImport | null>(null)
const loading = ref(true)
const busy = ref(false)
const error = ref('')
const issues = ref<string[]>([])

/** 可编辑草稿（与服务端 recognition_version 对应）。 */
const draft = reactive({
  platform: '',
  orderDate: '',
  totalAmount: '0',
  lines: [] as Array<{
    id: string
    name: string
    specification: string
    quantity: string
    unit: string
    lineAmount: string
    confidence: string
    includeInStock: boolean
    matchedShoppingItemId: string
  }>
})
const includeExpense = ref(true)
const screenshots = ref<string[]>([])
const shoppingTodo = ref<ShoppingItem[]>([])

/** 确认后的落账摘要。 */
const summary = ref<{
  created: boolean
  expenseAmount?: number
  transactionCount: number
  shoppingCompleted: number
} | null>(null)

const PLATFORMS = [
  { value: '', label: '未知' },
  { value: 'taobao', label: '淘宝' },
  { value: 'jd', label: '京东' },
  { value: 'pdd', label: '拼多多' },
  { value: 'other', label: '其他' }
]
const platformLabel = (value?: string) => PLATFORMS.find((item) => item.value === (value || ''))?.label || '未知'
const platformIndex = computed(() => {
  const index = PLATFORMS.findIndex((item) => item.value === draft.platform)
  return index < 0 ? 0 : index
})

const statusMeta: Record<string, { label: string; tone: string }> = {
  uploaded: { label: '待识别', tone: 'info' },
  processing: { label: '识别中', tone: 'info' },
  parsed: { label: '识别完成待核对', tone: 'ok' },
  partial: { label: '识别部分成功', tone: 'warning' },
  failed: { label: '识别失败，可手动填写', tone: 'danger' },
  confirmed: { label: '已确认', tone: 'ok' },
  cancelled: { label: '已取消', tone: 'muted' }
}

const isLocked = computed(() => {
  const status = imp.value?.status
  return status === 'confirmed' || status === 'cancelled' || status === 'processing'
})

const canConfirm = computed(() => !isLocked.value && !busy.value && draft.lines.length > 0)

/** 待购匹配下拉：0 为“不关联”。 */
const shoppingOptions = computed(() => ['不关联', ...shoppingTodo.value.map((item) => item.name)])
const shoppingValue = (lineId: string): number => {
  const line = draft.lines.find((item) => item.id === lineId)
  if (!line?.matchedShoppingItemId) return 0
  return shoppingTodo.value.findIndex((item) => item.id === line.matchedShoppingItemId) + 1
}

function applyImport(next: PurchaseImport) {
  imp.value = next
  draft.platform = next.platform || ''
  draft.orderDate = next.orderDate || ''
  draft.totalAmount = String(next.totalAmount ?? 0)
  draft.lines = next.lines.map((line: PurchaseLine) => ({
    id: line.id,
    name: line.name,
    specification: line.specification || '',
    quantity: String(line.quantity),
    unit: line.unit || '',
    lineAmount: line.lineAmount == null ? '' : String(line.lineAmount),
    confidence: line.confidence,
    includeInStock: line.includeInStock,
    matchedShoppingItemId: line.matchedShoppingItemId || ''
  }))
}

async function loadScreenshots(next: PurchaseImport) {
  screenshots.value = []
  const urls: string[] = []
  for (let i = 0; i < (next.mediaKeys || []).length; i += 1) {
    try {
      urls.push(await services.getPurchaseMediaDataUrl(next.id, i))
    } catch {
      // 截图读取失败不阻塞复核；界面仍可继续编辑与确认。
    }
  }
  screenshots.value = urls
}

async function load() {
  if (!importId.value) {
    loading.value = false
    error.value = '缺少导入记录参数'
    return
  }
  loading.value = true
  error.value = ''
  try {
    const [detail, todo] = await Promise.all([
      services.getPurchaseImport(importId.value),
      services.getShoppingList('todo')
    ])
    applyImport(detail)
    shoppingTodo.value = todo.data
    if (detail.status === 'confirmed' && detail.expenseId) {
      summary.value = {
        created: false,
        transactionCount: detail.lines.filter((line) => line.includeInStock).length,
        shoppingCompleted: 0
      }
    }
    await loadScreenshots(detail)
  } catch (e) {
    error.value = toApiError(e).message || '加载失败，请稍后重试'
  } finally {
    loading.value = false
  }
}

async function retryRecognize() {
  if (busy.value || !imp.value) return
  busy.value = true
  error.value = ''
  try {
    const result = await services.recognizePurchaseImport(imp.value.id)
    applyImport(result.import)
    issues.value = result.issues || []
  } catch (e) {
    error.value = toApiError(e).message || '识别失败，请稍后重试'
  } finally {
    busy.value = false
  }
}

// ---- 截图删除与草稿取消（隐私相关操作） ----

function deleteScreenshot(index: number) {
  if (!imp.value || busy.value) return
  uni.showModal({
    title: '删除截图',
    content: '删除后无法恢复，确认删除这张订单截图？',
    success: async (res) => {
      if (!res.confirm || !imp.value) return
      busy.value = true
      try {
        const updated = await services.deletePurchaseMedia(imp.value.id, index)
        applyImport(updated)
        await loadScreenshots(updated)
      } catch (e) {
        error.value = toApiError(e).message || '删除失败，请稍后重试'
      } finally {
        busy.value = false
      }
    }
  })
}

function deleteAllScreenshots() {
  if (!imp.value || busy.value || !imp.value.mediaKeys.length) return
  uni.showModal({
    title: '删除原始截图',
    content: '将删除本次导入的全部原始截图，库存与支出记录不受影响。确认删除？',
    success: async (res) => {
      if (!res.confirm || !imp.value) return
      busy.value = true
      try {
        let updated = imp.value
        while (updated.mediaKeys.length > 0) {
          updated = await services.deletePurchaseMedia(updated.id, 0)
        }
        applyImport(updated)
        screenshots.value = []
        uni.showToast({ title: '截图已删除', icon: 'success' })
      } catch (e) {
        error.value = toApiError(e).message || '删除失败，请稍后重试'
      } finally {
        busy.value = false
      }
    }
  })
}

function cancelDraft() {
  if (!imp.value || busy.value || isLocked.value) return
  uni.showModal({
    title: '取消导入',
    content: '取消后不会写入库存或支出，确认取消这份采购草稿？',
    success: async (res) => {
      if (!res.confirm || !imp.value) return
      busy.value = true
      try {
        const updated = await services.cancelPurchaseImport(imp.value.id)
        applyImport(updated)
        uni.showToast({ title: '已取消', icon: 'success' })
      } catch (e) {
        error.value = toApiError(e).message || '取消失败，请稍后重试'
      } finally {
        busy.value = false
      }
    }
  })
}

/** 保存草稿并返回最新版本号；返回 null 表示失败。 */
async function saveDraft(): Promise<number | null> {
  if (!imp.value) return null
  const lines = draft.lines
    .filter((line) => line.name.trim())
    .map((line) => ({
      id: line.id,
      name: line.name.trim(),
      specification: line.specification.trim(),
      quantity: Math.max(1, parseInt(line.quantity, 10) || 1),
      unit: line.unit.trim(),
      line_amount: line.lineAmount === '' ? undefined : Math.max(0, parseFloat(line.lineAmount) || 0),
      confidence: line.confidence,
      include_in_stock: line.includeInStock,
      matched_shopping_item_id: line.matchedShoppingItemId || undefined
    }))
  const saved = await services.savePurchaseDraft(imp.value.id, {
    expected_version: imp.value.recognitionVersion,
    platform: draft.platform,
    order_date: draft.orderDate,
    total_amount: Math.max(0, parseFloat(draft.totalAmount) || 0),
    lines
  })
  applyImport(saved)
  return saved.recognitionVersion
}

async function saveOnly() {
  if (busy.value || isLocked.value) return
  busy.value = true
  error.value = ''
  try {
    await saveDraft()
    uni.showToast({ title: '草稿已保存', icon: 'success' })
  } catch (e) {
    error.value = toApiError(e).message || '保存失败，请稍后重试'
  } finally {
    busy.value = false
  }
}

async function confirmPurchase() {
  if (!canConfirm.value || !imp.value) return
  busy.value = true
  error.value = ''
  issues.value = []
  try {
    // 先保存用户修订（版本+1），再以最新版本确认，避免草稿版本冲突。
    const version = await saveDraft()
    if (version == null) throw new Error('保存草稿失败')
    const result = await services.confirmPurchaseImport(imp.value.id, {
      expected_version: version,
      include_expense: includeExpense.value
    })
    applyImport(result.import)
    summary.value = {
      created: result.created,
      expenseAmount: result.expenseAmount,
      transactionCount: result.transactions?.length || 0,
      shoppingCompleted: result.shoppingCompleted?.length || 0
    }
    uni.showToast({ title: result.created ? '采购已确认' : '已确认过，未重复落账', icon: 'success' })
  } catch (e) {
    error.value = toApiError(e).message || '确认失败，请稍后重试'
  } finally {
    busy.value = false
  }
}

function onPlatformChange(event: { detail: { value: string } }) {
  const index = Number(event.detail.value)
  draft.platform = PLATFORMS[index]?.value || ''
}

function onDateChange(event: { detail: { value: string } }) {
  draft.orderDate = event.detail.value
}

function onMatchChange(lineId: string, event: { detail: { value: string } }) {
  const index = Number(event.detail.value)
  const line = draft.lines.find((item) => item.id === lineId)
  if (!line) return
  line.matchedShoppingItemId = index > 0 ? shoppingTodo.value[index - 1]?.id || '' : ''
}

function addLine() {
  if (isLocked.value) return
  draft.lines.push({
    id: `local-${Date.now()}`,
    name: '',
    specification: '',
    quantity: '1',
    unit: '',
    lineAmount: '',
    confidence: 'none',
    includeInStock: true,
    matchedShoppingItemId: ''
  })
}

function removeLine(lineId: string) {
  if (isLocked.value) return
  draft.lines = draft.lines.filter((line) => line.id !== lineId)
}

function toggleInStock(line: { includeInStock: boolean }) {
  if (isLocked.value) return
  line.includeInStock = !line.includeInStock
}

function confidenceTone(value: string) {
  if (value === 'high') return 'ok'
  if (value === 'medium') return 'info'
  if (value === 'low') return 'warning'
  return 'muted'
}

function goInventory() {
  router.push('/family/inventory')
}

onLoad((options) => {
  importId.value = String(options?.id || '')
})

useProtectedPage(load)
</script>

<template>
  <view class="page-header">
    <view class="page-title"> 核对采购 </view>
  </view>

  <view class="page-content">
    <view v-if="loading" aria-busy="true">
      <view class="skeleton" style="height: 140px; margin-bottom: 16px" />
      <view class="skeleton" style="height: 240px" />
    </view>

    <view v-else-if="error && !imp" class="empty-state">
      <view>{{ error }}</view>
      <button class="btn-secondary" @click="load">重新加载</button>
    </view>

    <template v-else-if="imp">
      <!-- 状态与摘要 -->
      <view class="family-section">
        <view class="review-status-row">
          <text class="review-status" :class="statusMeta[imp.status]?.tone">
            {{ statusMeta[imp.status]?.label || imp.status }}
          </text>
          <text v-if="imp.recognitionVersion" class="review-version">草稿 v{{ imp.recognitionVersion }}</text>
        </view>
        <view v-if="imp.errorSummary" class="review-error">{{ imp.errorSummary }}</view>
        <view v-if="issues.length" class="review-issues">
          <view v-for="(issue, index) in issues" :key="index" class="review-issue">
            <AppIcon name="alertTriangle" :size="14" />
            <text>{{ issue }}</text>
          </view>
        </view>
        <view v-if="screenshots.length" class="review-shots">
          <view v-for="(src, index) in screenshots" :key="index" class="review-shot-wrap">
            <image :src="src" mode="aspectFill" class="review-shot" />
            <button
              type="button"
              class="review-shot-remove"
              :disabled="busy"
              aria-label="删除截图"
              @click="deleteScreenshot(index)"
            >
              ×
            </button>
          </view>
        </view>
      </view>

      <!-- 订单信息 -->
      <view class="family-section">
        <view class="section-title"> 订单信息 </view>
        <view class="review-field-row">
          <text class="review-field-label">平台</text>
          <picker :range="PLATFORMS.map((item) => item.label)" :value="platformIndex" @change="onPlatformChange">
            <view class="field-input picker-value">{{ platformLabel(draft.platform) }}</view>
          </picker>
        </view>
        <view class="review-field-row">
          <text class="review-field-label">订单日期</text>
          <picker mode="date" :value="draft.orderDate" @change="onDateChange">
            <view class="field-input picker-value">{{ draft.orderDate || '选择日期' }}</view>
          </picker>
        </view>
        <view class="review-field-row">
          <text class="review-field-label">实付总额</text>
          <input v-model="draft.totalAmount" class="field-input review-amount" type="digit" :disabled="isLocked" placeholder="0.00" />
          <text class="review-unit">元</text>
        </view>
      </view>

      <!-- 商品明细 -->
      <view class="family-section">
        <view class="section-title"> 商品明细 </view>
        <view v-if="!draft.lines.length" class="empty-state">
          未识别到商品明细，请手动添加或重新识别
        </view>
        <view v-for="line in draft.lines" :key="line.id" class="review-line">
          <view class="review-line-top">
            <input v-model="line.name" class="field-input review-line-name" type="text" maxlength="255" :disabled="isLocked" />
            <text class="review-line-confidence" :class="confidenceTone(line.confidence)">
              {{ line.confidence === 'high' ? '高' : line.confidence === 'medium' ? '中' : line.confidence === 'low' ? '低' : '待核对' }}
            </text>
            <button
              v-if="!isLocked"
              type="button"
              class="review-line-remove"
              aria-label="删除该商品行"
              @click="removeLine(line.id)"
            >
              ×
            </button>
          </view>
          <view class="review-line-row">
            <label class="review-inline-field">
              <text class="review-field-label">数量</text>
              <input v-model="line.quantity" class="field-input review-qty" type="number" :disabled="isLocked" />
            </label>
            <label class="review-inline-field">
              <text class="review-field-label">单位</text>
              <input v-model="line.unit" class="field-input review-unit-input" type="text" maxlength="32" :disabled="isLocked" />
            </label>
            <label class="review-inline-field">
              <text class="review-field-label">金额</text>
              <input v-model="line.lineAmount" class="field-input review-amount" type="digit" :disabled="isLocked" placeholder="0.00" />
            </label>
          </view>
          <view class="review-line-row">
            <picker
              :range="shoppingOptions"
              :value="shoppingValue(line.id)"
              :disabled="isLocked"
              @change="onMatchChange(line.id, $event)"
            >
              <view class="field-input picker-value review-match">
                待购：{{ line.matchedShoppingItemId ? shoppingOptions[shoppingValue(line.id)] : '不关联' }}
              </view>
            </picker>
            <button
              type="button"
              class="review-instock"
              :class="{ active: line.includeInStock, disabled: isLocked }"
              :disabled="isLocked"
              @click="toggleInStock(line)"
            >
              <AppIcon :name="line.includeInStock ? 'check' : 'inbox'" :size="14" />
              {{ line.includeInStock ? '确认入库' : '不入库' }}
            </button>
          </view>
        </view>
        <view class="review-line-actions">
          <button v-if="!isLocked" type="button" class="btn-secondary review-add-line" @click="addLine">
            添加商品行
          </button>
          <button
            v-if="!isLocked && (imp.status === 'failed' || imp.status === 'uploaded' || imp.status === 'partial')"
            type="button"
            class="btn-secondary review-add-line"
            :disabled="busy"
            @click="retryRecognize"
          >
            重新识别
          </button>
        </view>
        <view class="review-hint"> 只有「确认入库」的商品才会增加库存；支出按订单实付总额记一笔。 </view>
      </view>

      <!-- 确认 -->
      <view class="family-section review-confirm-card">
        <view class="review-expense-toggle">
          <text>同时记入支出</text>
          <button
            type="button"
            class="review-switch"
            :class="{ active: includeExpense, disabled: isLocked }"
            :disabled="isLocked"
            @click="includeExpense = !includeExpense"
          >
            {{ includeExpense ? '记录 ¥' + draft.totalAmount : '不记账' }}
          </button>
        </view>
        <button class="btn-secondary review-save" :disabled="busy || isLocked" @click="saveOnly"> 保存草稿 </button>
        <button class="btn-primary review-confirm" :disabled="!canConfirm" @click="confirmPurchase">
          {{ busy ? '处理中…' : '确认采购' }}
        </button>
        <button v-if="!isLocked" type="button" class="review-cancel" :disabled="busy" @click="cancelDraft">
          取消这份导入
        </button>
        <text v-if="error" class="form-error">{{ error }}</text>
      </view>

      <!-- 落账摘要 -->
      <view v-if="summary" class="family-section review-summary">
        <view class="section-title"> 落账摘要 </view>
        <view class="review-summary-line">
          {{ summary.created ? '本次已落账' : '此前已确认（未重复落账）' }}
        </view>
        <view v-if="summary.expenseAmount" class="review-summary-line">
          支出：¥{{ summary.expenseAmount.toFixed(2) }}
        </view>
        <view class="review-summary-line"> 入库流水：{{ summary.transactionCount }} 条 </view>
        <view v-if="summary.shoppingCompleted" class="review-summary-line">
          完成待购：{{ summary.shoppingCompleted }} 项
        </view>
        <button class="btn-primary" @click="goInventory"> 查看库存 </button>
        <button
          v-if="imp.mediaKeys.length && !busy"
          type="button"
          class="btn-secondary review-delete-media"
          @click="deleteAllScreenshots"
        >
          删除原始截图
        </button>
      </view>
    </template>
  </view>
</template>

<style scoped>
/* 通用输入框（与 inventory 页保持一致的样式；页面内 scoped） */
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

.form-error {
  display: block;
  margin: 4px 0 12px;
  padding: 9px 12px;
  font-size: 13px;
  color: #b4453a;
  background: #fdf1ef;
  border-radius: 10px;
}

.review-status-row {
  display: flex;
  align-items: center;
  gap: var(--space-8);
  margin-bottom: var(--space-8);
}

.review-status {
  padding: 2px 10px;
  border-radius: 999px;
  font-size: var(--font-size-caption);
  font-weight: 500;
  background: var(--color-bg-muted);
  color: var(--color-text-secondary);
}

.review-status.ok {
  background: var(--color-success-soft, #e7f5ec);
  color: var(--color-success, #2e7d4f);
}

.review-status.info {
  background: var(--color-brand-soft);
  color: var(--color-brand);
}

.review-status.warning {
  background: var(--color-warning-soft, #fdf3e1);
  color: var(--color-warning, #a9661a);
}

.review-status.danger {
  background: var(--color-danger-soft);
  color: var(--color-danger);
}

.review-status.muted {
  color: var(--color-text-tertiary);
}

.review-version {
  font-size: var(--font-size-caption);
  color: var(--color-text-tertiary);
}

.review-error {
  padding: var(--space-8);
  border-radius: var(--radius-md);
  background: var(--color-danger-soft);
  color: var(--color-danger);
  font-size: var(--font-size-caption);
}

.review-issues {
  margin-top: var(--space-8);
}

.review-issue {
  display: flex;
  align-items: center;
  gap: var(--space-4);
  font-size: var(--font-size-caption);
  color: var(--color-warning, #a9661a);
}

.review-shots {
  display: flex;
  gap: var(--space-8);
  margin-top: var(--space-12);
  overflow-x: auto;
}

.review-shot-wrap {
  position: relative;
  flex: 0 0 auto;
}

.review-shot {
  width: 84px;
  height: 84px;
  border-radius: var(--radius-md);
  background: var(--color-bg-muted);
  display: block;
}

.review-shot-remove {
  position: absolute;
  top: 2px;
  right: 2px;
  width: 22px;
  height: 22px;
  min-width: 22px;
  padding: 0;
  border-radius: 50%;
  background: rgba(0, 0, 0, 0.55);
  color: #fff;
  font-size: 14px;
  line-height: 20px;
}

.review-cancel {
  width: 100%;
  padding: 10px 0;
  font-size: var(--font-size-caption);
  color: var(--color-text-tertiary);
  background: transparent;
  border: 1px dashed var(--color-divider);
  border-radius: var(--radius-md);
}

.review-field-row {
  display: flex;
  align-items: center;
  gap: var(--space-8);
  padding: var(--space-8) 0;
}

.review-field-label {
  width: 72px;
  flex: 0 0 auto;
  font-size: var(--font-size-caption);
  color: var(--color-text-tertiary);
}

.review-amount {
  width: 96px;
}

.review-unit {
  color: var(--color-text-tertiary);
}

.review-line {
  padding: var(--space-12) 0;
  border-bottom: 1px solid var(--color-divider);
}

.review-line-top {
  display: flex;
  align-items: center;
  gap: var(--space-8);
}

.review-line-remove {
  width: 24px;
  height: 24px;
  min-width: 24px;
  padding: 0;
  border-radius: 50%;
  border: 1px solid var(--color-divider);
  background: var(--color-bg-surface);
  color: var(--color-text-tertiary);
  font-size: 14px;
  line-height: 22px;
}

.review-line-actions {
  display: flex;
  gap: var(--space-8);
  margin-top: var(--space-12);
}

.review-add-line {
  flex: 1;
  font-size: var(--font-size-caption);
}

.review-line-name {
  flex: 1;
}

.review-line-confidence {
  padding: 0 6px;
  border-radius: 999px;
  font-size: 10px;
  background: var(--color-bg-muted);
  color: var(--color-text-secondary);
}

.review-line-confidence.ok {
  color: var(--color-success, #2e7d4f);
}

.review-line-confidence.warning {
  color: var(--color-warning, #a9661a);
}

.review-line-row {
  display: flex;
  align-items: center;
  gap: var(--space-8);
  margin-top: var(--space-8);
}

.review-inline-field {
  display: flex;
  align-items: center;
  gap: var(--space-4);
  flex: 1;
}

.review-inline-field .review-field-label {
  width: auto;
}

.review-qty {
  width: 56px;
}

.review-unit-input {
  width: 56px;
}

.review-match {
  flex: 1;
}

.review-instock {
  display: flex;
  align-items: center;
  gap: var(--space-4);
  padding: 4px var(--space-8);
  border-radius: 999px;
  border: 1px solid var(--color-divider);
  background: var(--color-bg-surface);
  color: var(--color-text-tertiary);
  font-size: var(--font-size-caption);
}

.review-instock.active {
  border-color: var(--color-brand);
  color: var(--color-brand);
  background: var(--color-brand-soft);
}

.review-instock.disabled {
  opacity: 0.5;
}

.review-hint {
  margin-top: var(--space-8);
  font-size: var(--font-size-caption);
  color: var(--color-text-tertiary);
}

.review-confirm-card {
  display: flex;
  flex-direction: column;
  gap: var(--space-8);
}

.review-expense-toggle {
  display: flex;
  align-items: center;
  justify-content: space-between;
}

.review-switch {
  padding: 4px var(--space-12);
  border-radius: 999px;
  border: 1px solid var(--color-divider);
  background: var(--color-bg-surface);
  color: var(--color-text-tertiary);
  font-size: var(--font-size-caption);
}

.review-switch.active {
  border-color: var(--color-brand);
  color: var(--color-brand);
  background: var(--color-brand-soft);
}

.review-save,
.review-confirm {
  width: 100%;
}

.review-summary-line {
  padding: var(--space-4) 0;
  color: var(--color-text-secondary);
}

.review-summary .btn-primary {
  margin-top: var(--space-12);
  width: 100%;
}

.review-delete-media {
  margin-top: var(--space-8);
  width: 100%;
}
</style>
