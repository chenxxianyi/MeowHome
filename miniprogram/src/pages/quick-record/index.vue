<script setup lang="ts">
import { ref, computed } from 'vue'
import { useRoute, useRouter } from '../../utils/navigation'
import AppHeader from '../../components/app/AppHeader.vue'
import AppIcon from '../../components/app/AppIcon.vue'
import { useCatStore } from '../../stores/cat'
import { useRecordStore } from '../../stores/record'
import { services } from '../../services'
import { useProtectedPage } from '../../utils/page'

const catStore = useCatStore()
const recordStore = useRecordStore()
const route = useRoute()
const router = useRouter()

const catId = computed(() => (route.query.catId as string) || catStore.currentCatId || 'cat-whit')
const recordType = computed(() => (route.params.type as string) || 'feeding')
const cat = computed(() => catStore.cats.find((c) => c.id === catId.value) || catStore.cats[0])

const formData = ref<Record<string, string>>({})
const collapsed = ref(true)
const saving = ref(false)
const saved = ref(false)
const validationError = ref('')

// Initialize form fields based on type
const fieldLabels: Record<string, string> = {
  feeding: '喂食',
  vomit: '呕吐',
  elimination: '排便',
  weight: '体重',
  medication: '用药',
  custom: '自定义'
}

function init() {
  formData.value = { time: new Date().toISOString().slice(0, 16), catId: catId.value }
}

function toggleCollapse() {
  collapsed.value = !collapsed.value
}

async function save() {
  if (!validate()) return
  saving.value = true
  try {
    const record = {
      type: recordType.value,
      catId: catId.value,
      ...formData.value,
      createdAt: new Date().toISOString(),
      source: 'manual'
    }
    const res = await services.saveRecord(record)
    if (res.success) {
      saved.value = true
      setTimeout(() => router.push('/today'), 1000)
    } else {
      throw new Error('保存失败')
    }
  } catch {
    // Offline: save as draft
    recordStore.addDraft({
      id: 'draft-' + Date.now(),
      type: recordType.value,
      catId: catId.value,
      content: formData.value,
      createdAt: new Date().toISOString(),
      status: 'local'
    })
    saved.value = true
    setTimeout(() => router.push('/today'), 1000)
  } finally {
    saving.value = false
  }
}

interface FormField {
  key: string
  label: string
  type: 'text' | 'number' | 'date' | 'datetime-local' | 'select' | 'textarea'
  required?: boolean
  placeholder?: string
  options?: string[]
  min?: number
  step?: string
}

function getFormFields(): FormField[] {
  switch (recordType.value) {
    case 'feeding':
      return [
        { key: 'time', label: '发生时间', type: 'datetime-local', required: true },
        { key: 'meal', label: '餐次', type: 'select', options: ['早餐', '午餐', '晚餐', '零食'], required: true },
        { key: 'food', label: '食物', type: 'text', placeholder: '如：渴望室内猫粮', required: true },
        { key: 'provided', label: '提供量', type: 'number', placeholder: 'g' },
        { key: 'consumed', label: '实际食量', type: 'number', placeholder: 'g' },
        { key: 'appetite', label: '食欲状态', type: 'select', options: ['正常', '偏低', '很差', '拒食'] },
        { key: 'notes', label: '备注', type: 'textarea', placeholder: '选填' }
      ]
    case 'vomit':
      return [
        { key: 'time', label: '发生时间', type: 'datetime-local', required: true },
        { key: 'count', label: '次数', type: 'number', min: 1, required: true },
        {
          key: 'content',
          label: '内容物',
          type: 'select',
          options: ['黄色液体', '未消化食物', '白色泡沫', '带血', '毛球', '其他']
        },
        { key: 'beforeMeal', label: '发生在', type: 'select', options: ['进食前', '进食后', '不确定'] },
        { key: 'mentalState', label: '精神状态', type: 'select', options: ['正常', '偏低', '萎靡'] },
        { key: 'notes', label: '备注', type: 'textarea', placeholder: '选填' }
      ]
    case 'elimination':
      return [
        { key: 'time', label: '发生时间', type: 'datetime-local', required: true },
        { key: 'type', label: '类型', type: 'select', options: ['大便', '小便', '两者'] },
        { key: 'form', label: '粪便形态', type: 'select', options: ['成型', '偏干', '偏软', '稀便', '水样'] },
        { key: 'color', label: '颜色', type: 'select', options: ['棕色', '黄色', '绿色', '黑色', '带血'] },
        { key: 'notes', label: '备注', type: 'textarea', placeholder: '选填' }
      ]
    case 'weight':
      return [
        { key: 'time', label: '称重时间', type: 'datetime-local', required: true },
        { key: 'weight', label: '体重（kg）', type: 'number', step: '0.01', placeholder: '如：4.20', required: true },
        { key: 'notes', label: '备注', type: 'textarea', placeholder: '选填' }
      ]
    case 'medication':
      return [
        { key: 'time', label: '给药时间', type: 'datetime-local', required: true },
        { key: 'medication', label: '药物名称', type: 'text', placeholder: '如：肾上腺素抑制剂', required: true },
        { key: 'dose', label: '剂量', type: 'text', placeholder: '如：1片' },
        { key: 'notes', label: '备注', type: 'textarea', placeholder: '选填' }
      ]
    default:
      return [
        { key: 'time', label: '时间', type: 'datetime-local', required: true },
        { key: 'content', label: '记录内容', type: 'textarea', placeholder: '请描述猫咪的情况…', required: true },
        { key: 'notes', label: '备注', type: 'textarea', placeholder: '选填' }
      ]
  }
}

function validate(): boolean {
  validationError.value = ''
  const fields = getFormFields()
  for (const f of fields) {
    if (f.required && !formData.value[f.key]) {
      validationError.value = `请填写${f.label}`
      uni.showToast({ title: validationError.value, icon: 'none' })
      return false
    }
  }
  return true
}

function selectOption(field: FormField, event: unknown) {
  const index = Number((event as { detail: { value: string } }).detail.value)
  formData.value[field.key] = field.options?.[index] || ''
}

function choosePhoto() {
  uni.showToast({ title: '图片上传接口尚未接入，可先保存文字记录', icon: 'none' })
}

useProtectedPage(init)
</script>

<template>
  <AppHeader :title="fieldLabels[recordType] || '记录'" :subtitle="cat?.name || '猫咪'" />

  <view class="page-content">
    <view style="font-size: var(--font-size-assist); color: var(--color-text-tertiary); margin-bottom: var(--space-16)">
      为 <text>{{ cat?.name || '猫咪' }}</text> 记录
    </view>

    <view class="record-form">
      <view v-for="field in getFormFields()" :key="field.key" class="form-group">
        <label class="form-label" :for="'qf-' + field.key"> {{ field.label }}{{ field.required ? ' *' : '' }} </label>
        <template v-if="field.type === 'textarea'">
          <textarea
            :id="'qf-' + field.key"
            v-model="formData[field.key]"
            class="form-input"
            rows="3"
            :placeholder="field.placeholder"
          />
        </template>
        <template v-else-if="field.type === 'select'">
          <picker mode="selector" :range="field.options || []" @change="selectOption(field, $event)">
            <view class="form-select">{{ formData[field.key] || '请选择' }}</view>
          </picker>
        </template>
        <template v-else>
          <input
            :id="'qf-' + field.key"
            v-model="formData[field.key]"
            :type="field.type === 'number' ? 'digit' : 'text'"
            class="form-input"
            :placeholder="field.placeholder"
          />
        </template>
      </view>

      <!-- Collapsed extra fields -->
      <view v-if="!collapsed" class="record-form-extra">
        <view class="form-group">
          <label class="form-label">图片</label>
          <button type="button" class="btn-secondary" @click="choosePhoto">选择图片（暂未开放上传）</button>
        </view>
        <view class="form-group">
          <label class="form-label">额外备注</label>
          <textarea class="form-input" rows="2" placeholder="选填" />
        </view>
      </view>

      <button v-if="collapsed" type="button" class="collapse-toggle" @click="toggleCollapse">
        展开附加选项 <AppIcon name="chevronDown" :size="16" />
      </button>

      <view class="form-actions">
        <view v-if="validationError" class="form-error">{{ validationError }}</view>
        <button type="button" class="btn-primary" :disabled="saving" @click="save">
          {{ saving ? '保存中…' : saved ? '✓ 已保存' : '保存记录' }}
        </button>
      </view>
    </view>
  </view>
</template>

<style scoped>
.record-form {
  padding-bottom: 80px;
}
.collapse-toggle {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: var(--space-12) 0;
  color: var(--color-text-tertiary);
  font-size: var(--font-size-assist);
  cursor: pointer;
  border: none;
  background: none;
}
.record-form-extra {
  padding: var(--space-8) 0;
  border-top: 1px solid var(--color-divider);
}
.form-actions {
  position: sticky;
  bottom: var(--safe-bottom);
  left: 0;
  right: 0;
  padding: var(--space-12) var(--space-16);
  background: var(--color-bg-surface);
  border-top: 1px solid var(--color-divider);
}
</style>
