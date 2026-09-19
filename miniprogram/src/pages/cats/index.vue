<script setup lang="ts">
/**
 * 猫咪列表页（小程序版）—— 步骤 0.6 技术验证页。
 *
 * 迁移自 `frontend/src/views/cats/CatsView.vue`，用于一次性暴露核心平台风险。
 *
 * ## 相对 Web 端源码的改动（每一处都对应一个已确认的平台差异）
 *
 * | Web 端写法 | 本页写法 | 原因 |
 * |---|---|---|
 * | `<AppShell>` 包裹 | 移除 | 小程序 tabBar 由 pages.json 原生提供 |
 * | `<a :href="'#/cats/'+id">` | `<view @click>` + `uni.showToast` | `<navigator>` 需小程序页面路径，`#/xxx` 无效；详情页属步骤 3.4 |
 * | `<img :src>` | `<image mode="aspectFit">` | `<img>` 虽自动映射为 `<image>`，但**不补 mode**，默认 scaleToFill 会拉伸变形 |
 * | `<input type="date">` | `<picker mode="date">` | 小程序 input 无 date 类型 |
 * | `<span>` (112 处同款) | `<text>` | `<span>` 会被静默映射为 `<label>`（表单标签），语义错误 |
 * | `<p>` `<h2>` | `<view>` | 块级语义，映射正确，但显式写原生标签更清晰 |
 * | 浏览器全局键盘监听 | 删除 | 小程序无对应键盘事件，Esc 关闭不适用 |
 * | `services.getCats()` | `catApi.list()` 直调 | **本页刻意绕过 services 层**：services 依赖 pinia 与 mocks，属步骤 1.2 范围，本步只验证标签/网络/UI/图标四条链路 |
 * | `useCatStore()` | 移除 | 同上，pinia 未安装 |
 * | `onMounted` | `onShow` | 小程序页面生命周期；用 onShow 保证每次进入都刷新 |
 */
import { computed, ref } from 'vue'
import { onShow } from '@dcloudio/uni-app'

import { toApiError } from '../../api/adapter'
import { catApi, toCat } from '../../api/endpoints'
import { getStoredFamilyId } from '../../api/client'
import AppIcon from '../../components/app/AppIcon.vue'
import OfflineBanner from '../../components/app/OfflineBanner.vue'
import { guardOnShow } from '../../utils/guard'
import { useRouter } from '../../utils/navigation'
import { useCatStore } from '../../stores/cat'
import type { Cat } from '../../types'
import { usePageCapabilities } from '../../utils/page'

const cats = ref<Cat[]>([])
const router = useRouter()
const catStore = useCatStore()
const loading = ref(true)
const loadError = ref('')

// —— 添加猫咪表单 ——
const sheetOpen = ref(false)
const submitting = ref(false)
const error = ref('')
const nameFocus = ref(false)

const form = ref({
  name: '',
  gender: 'unknown' as 'female' | 'male' | 'unknown',
  breed: '',
  birthday: ''
})

const GENDERS = [
  { value: 'female', label: '母' },
  { value: 'male', label: '公' },
  { value: 'unknown', label: '未知' }
] as const

const canSubmit = computed(() => form.value.name.trim().length > 0 && !submitting.value)

function openDetail(cat: Cat) {
  catStore.setCat(cat.id)
  router.push(`/cats/${cat.id}`)
}

async function load() {
  const familyId = getStoredFamilyId()
  if (!familyId) {
    loading.value = false
    loadError.value = '尚未加入任何家庭，请先完成引导'
    return
  }
  try {
    const list = await catApi.list(familyId)
    cats.value = list.map(toCat)
    catStore.setCats(cats.value)
    loadError.value = ''
  } catch (e) {
    loadError.value = toApiError(e).message
  } finally {
    loading.value = false
  }
}

// 用 onShow 而非 onLoad：从详情/添加流程返回时也要刷新。
// 先过守卫（未登录 → 登录页；无家庭 → 引导页），通过后再取数。
onShow(async () => {
  if (!(await guardOnShow())) return
  await load()
})
usePageCapabilities(load, '猫宅 · 猫咪档案')

async function openSheet() {
  form.value = { name: '', gender: 'unknown', breed: '', birthday: '' }
  error.value = ''
  sheetOpen.value = true
  // 小程序 input 用 focus 属性控制聚焦，不能调 DOM 的 .focus()
  nameFocus.value = true
}

function closeSheet() {
  if (submitting.value) return // 提交中不允许关闭，避免半途中断
  sheetOpen.value = false
}

function onBirthdayChange(e: { detail: { value: string } }) {
  form.value.birthday = e.detail.value
}

async function submit() {
  if (!canSubmit.value) return

  const familyId = getStoredFamilyId()
  if (!familyId) {
    error.value = '尚未加入任何家庭'
    return
  }

  submitting.value = true
  error.value = ''
  try {
    await catApi.create(familyId, {
      name: form.value.name.trim(),
      gender: form.value.gender,
      breed: form.value.breed.trim(),
      birth_date: form.value.birthday || undefined
    })
    // 以服务端为准重新拉取
    await load()
    sheetOpen.value = false
  } catch (e) {
    error.value = toApiError(e).message
  } finally {
    submitting.value = false
  }
}
</script>

<template>
  <OfflineBanner />
  <view class="page">
    <view class="page-header">
      <view class="page-title"> 猫咪 </view>
    </view>

    <view class="page-content">
      <view v-if="loading" aria-busy="true">
        <view v-for="i in 3" :key="i" class="skeleton" style="height: 72px; margin-bottom: 12px" />
      </view>

      <template v-else>
        <view v-if="loadError" class="load-error">
          {{ loadError }}
        </view>

        <text v-else-if="cats.length === 0" class="empty-hint"> 还没有猫咪档案，先添加一只吧 </text>

        <view
          v-for="cat in cats"
          :key="cat.id"
          class="cat-list-item"
          :aria-label="'查看 ' + cat.name + ' 详情'"
          @click="openDetail(cat)"
        >
          <image v-if="cat.avatar" class="cat-avatar" :src="cat.avatar" mode="aspectFit" />
          <view v-else class="cat-avatar cat-avatar-empty">
            <AppIcon name="pawPrint" :size="22" />
          </view>

          <view class="cat-info">
            <view class="cat-name">
              {{ cat.name }}
            </view>
            <view class="cat-meta">
              {{ cat.age }}岁 · {{ cat.gender === 'female' ? '母' : cat.gender === 'male' ? '公' : '未知' }} ·
              {{ cat.breed }}
              <text v-if="cat.neutered"> · 已绝育</text>
            </view>
            <text class="cat-status normal">健康</text>
          </view>
          <view class="family-item-arrow">
            <AppIcon name="chevronRight" :size="20" />
          </view>
        </view>

        <button class="btn-primary" style="margin-top: 16px" @click="openSheet">添加猫咪</button>
      </template>
    </view>

    <!-- 添加猫咪：底部抽屉（复用全局 .sheet-overlay / .sheet-content，层级 200/201） -->
    <view class="sheet-overlay" :class="{ show: sheetOpen }" @click.self="closeSheet">
      <view class="sheet-content" role="dialog" aria-modal="true" aria-labelledby="add-cat-title">
        <view class="sheet-handle" />
        <view id="add-cat-title" class="sheet-title"> 添加猫咪 </view>

        <view class="sheet-body">
          <label class="field">
            <text class="field-label">名字 <text class="req">*</text></text>
            <input
              v-model="form.name"
              class="field-input"
              type="text"
              :focus="nameFocus"
              maxlength="64"
              placeholder="例如：小白"
              @blur="nameFocus = false"
            />
          </label>

          <view class="field">
            <text class="field-label">性别</text>
            <view class="segmented">
              <button
                v-for="g in GENDERS"
                :key="g.value"
                type="button"
                class="segment"
                :class="{ active: form.gender === g.value }"
                @click="form.gender = g.value"
              >
                {{ g.label }}
              </button>
            </view>
          </view>

          <label class="field">
            <text class="field-label">品种</text>
            <input
              v-model="form.breed"
              class="field-input"
              type="text"
              maxlength="120"
              placeholder="例如：中华田园猫"
            />
          </label>

          <label class="field">
            <text class="field-label">生日</text>
            <picker mode="date" :value="form.birthday" @change="onBirthdayChange">
              <view class="field-input picker-value" :class="{ placeholder: !form.birthday }">
                {{ form.birthday || '请选择生日' }}
              </view>
            </picker>
          </label>

          <text v-if="error" class="form-error">
            {{ error }}
          </text>
        </view>

        <!-- 操作区固定在抽屉底部，不随表单滚动 -->
        <view class="sheet-actions">
          <button type="button" class="btn-secondary" :disabled="submitting" @click="closeSheet">取消</button>
          <button type="button" class="btn-primary" :disabled="!canSubmit" @click="submit">
            {{ submitting ? '保存中…' : '保存' }}
          </button>
        </view>
      </view>
    </view>
  </view>
</template>

<style scoped>
.empty-hint {
  display: block;
  margin: 8px 0 0;
  font-size: 14px;
  color: var(--color-text-tertiary, #9b948c);
  text-align: center;
}

.load-error {
  margin: 16px 0;
  padding: 12px;
  font-size: 13px;
  color: var(--color-danger, #b5534b);
  background: var(--color-danger-soft, #f4e1df);
  border-radius: 12px;
  text-align: center;
}

.cat-avatar-empty {
  display: flex;
  align-items: center;
  justify-content: center;
  color: var(--color-text-tertiary, #9b948c);
  background: var(--color-bg-subtle, #f1ece5);
}

/* 复用全局 .sheet-content 的外观，只覆盖布局：
   外层纵向弹性 + 表单区单独滚动，操作区固定底部（Web 端同款处理） */
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

.picker-value.placeholder {
  color: #b3aca3;
}

/* 性别三选一 */
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

/* 覆盖全局的纵向堆叠：两个按钮并排，主操作占更大宽度 */
.sheet-actions {
  flex: none;
  flex-direction: row;
  gap: 10px;
  margin-top: 12px;
  padding-bottom: calc(56px + var(--safe-bottom));
}

.sheet-actions .btn-primary {
  flex: 2;
}

.sheet-actions .btn-secondary {
  flex: 1;
}
</style>
