<script setup lang="ts">
import { computed, nextTick, onBeforeUnmount, onMounted, reactive, ref } from 'vue'

import { toApiError } from '../../api/adapter'
import AppIcon from '../../components/app/AppIcon.vue'
import AppShell from '../../components/app/AppShell.vue'
import { services } from '../../services'
import { useCatStore } from '../../stores/cat'
import type { Cat } from '../../types'

const store = useCatStore()
const cats = ref<Cat[]>(store.cats)
const loading = ref(true)

// —— 添加猫咪表单 ——
const sheetOpen = ref(false)
const submitting = ref(false)
const error = ref('')

const form = reactive({
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

const canSubmit = computed(() => form.name.trim().length > 0 && !submitting.value)

async function load() {
  const res = await services.getCats()
  cats.value = res.data
  loading.value = false
}

onMounted(async () => {
  await load()
  window.addEventListener('keydown', onKeydown)
})

onBeforeUnmount(() => window.removeEventListener('keydown', onKeydown))

const nameInput = ref<HTMLInputElement | null>(null)

async function openSheet() {
  form.name = ''
  form.gender = 'unknown'
  form.breed = ''
  form.birthday = ''
  error.value = ''
  sheetOpen.value = true
  // 抽屉常驻 DOM（靠 .show 切换显隐以保留过渡动画），
  // 因此不能用 autofocus，改为打开后显式聚焦
  await nextTick()
  nameInput.value?.focus()
}

function closeSheet() {
  if (submitting.value) return // 提交中不允许关闭，避免半途中断
  sheetOpen.value = false
}

function onKeydown(e: KeyboardEvent) {
  if (e.key === 'Escape' && sheetOpen.value) closeSheet()
}

async function submit() {
  if (!canSubmit.value) return
  submitting.value = true
  error.value = ''
  try {
    await services.createCat({
      name: form.name.trim(),
      gender: form.gender,
      breed: form.breed.trim(),
      birthday: form.birthday
    })
    // 以服务端为准重新拉取，避免依赖 service 写 store 的副作用
    await load()
    sheetOpen.value = false
  } catch (e) {
    // createCat 是唯一会抛异常的方法，这里把后端校验信息展示出来
    error.value = toApiError(e).message
  } finally {
    submitting.value = false
  }
}
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
        <p
          v-if="cats.length === 0"
          class="empty-hint"
        >
          还没有猫咪档案，先添加一只吧
        </p>

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
              {{ cat.age }}岁 · {{ cat.gender === 'female' ? '母' : cat.gender === 'male' ? '公' : '未知' }} · {{ cat.breed }}
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
          @click="openSheet"
        >
          添加猫咪
        </button>
      </template>
    </div>

    <!-- 添加猫咪：底部抽屉。
         复用全局 .sheet-overlay/.sheet-content（z-index 200/201）——
         自建样式曾用 z-index 100，与 .bottom-nav 同级，底栏会盖住操作按钮。 -->
    <div
      class="sheet-overlay"
      :class="{ show: sheetOpen }"
      @click.self="closeSheet"
    >
      <div
        class="sheet-content"
        role="dialog"
        aria-modal="true"
        aria-labelledby="add-cat-title"
      >
        <div class="sheet-handle" />
        <h2
          id="add-cat-title"
          class="sheet-title"
        >
          添加猫咪
        </h2>

        <div class="sheet-body">
          <label class="field">
            <span>名字 <em>*</em></span>
            <input
              ref="nameInput"
              v-model.trim="form.name"
              type="text"
              maxlength="64"
              placeholder="例如：小白"
            />
          </label>

          <div class="field">
            <span>性别</span>
            <div class="segmented">
              <button
                v-for="g in GENDERS"
                :key="g.value"
                type="button"
                class="segment"
                :class="{ active: form.gender === g.value }"
                :aria-pressed="form.gender === g.value"
                @click="form.gender = g.value"
              >
                {{ g.label }}
              </button>
            </div>
          </div>

          <label class="field">
            <span>品种</span>
            <input
              v-model.trim="form.breed"
              type="text"
              maxlength="120"
              placeholder="例如：中华田园猫"
            />
          </label>

          <label class="field">
            <span>生日</span>
            <input
              v-model="form.birthday"
              type="date"
            />
          </label>

          <p
            v-if="error"
            class="form-error"
          >
            {{ error }}
          </p>
        </div>

        <!-- 操作区固定在抽屉底部，不随表单滚动 -->
        <div class="sheet-actions">
          <button
            type="button"
            class="btn-secondary"
            :disabled="submitting"
            @click="closeSheet"
          >
            取消
          </button>
          <button
            type="button"
            class="btn-primary"
            :disabled="!canSubmit"
            @click="submit"
          >
            {{ submitting ? '保存中…' : '保存' }}
          </button>
        </div>
      </div>
    </div>
  </AppShell>
</template>

<style scoped>
.empty-hint {
  margin: 8px 0 0;
  font-size: 14px;
  color: var(--color-text-tertiary, #9a8f7d);
  text-align: center;
}

/* 复用全局 .sheet-content 的外观（层级 201、安全区、圆角、滑入过渡均由全局提供），
   这里只覆盖布局：外层纵向弹性 + 表单区单独滚动，操作区固定在底部。 */
.sheet-content {
  display: flex;
  flex-direction: column;
  max-height: 86dvh;
  overflow: hidden;
}

.sheet-body {
  flex: 1;
  min-height: 0;
  overflow-y: auto;
  overscroll-behavior: contain;
}

.field {
  display: block;
  margin-bottom: 14px;
}

.field span {
  display: block;
  margin-bottom: 6px;
  font-size: 13px;
  color: #7a6f5d;
}

.field em {
  color: #b4453a;
  font-style: normal;
}

.field input {
  width: 100%;
  box-sizing: border-box;
  padding: 11px 14px;
  font-size: 15px;
  color: #3d3426;
  background: #fffdfa;
  border: 1px solid #e8e1d5;
  border-radius: 12px;
  outline: none;
}

.field input:focus {
  border-color: #d9a441;
  background: #fff;
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
  color: #7a6f5d;
  background: #fffdfa;
  border: 1px solid #e8e1d5;
  border-radius: 12px;
  cursor: pointer;
}

.segment.active {
  color: #fff;
  background: #d9a441;
  border-color: #d9a441;
}

.form-error {
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
  margin-top: var(--space-12);
}

.sheet-actions .btn-primary {
  flex: 2;
}

.sheet-actions .btn-secondary {
  flex: 1;
}
</style>
