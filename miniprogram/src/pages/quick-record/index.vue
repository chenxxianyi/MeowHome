<script setup lang="ts">
import { ref, computed } from 'vue'
import { useRoute, useRouter } from '../../utils/navigation'
import AppHeader from '../../components/app/AppHeader.vue'
import AppIcon from '../../components/app/AppIcon.vue'
import { useCatStore } from '../../stores/cat'
import { useRecordStore } from '../../stores/record'
import { services } from '../../services'
import { useProtectedPage } from '../../utils/page'
import { deviceTimezone, formatZonedDateTime, localDateTimeStr, todayStr } from '../../utils/date'
import { isValidDateKey } from '../../utils/recordCalendar'
import type { VaccineSuggestionDTO } from '../../api/endpoints'
import type { Reminder } from '../../types'

const catStore = useCatStore()
const recordStore = useRecordStore()
const route = useRoute()
const router = useRouter()

const catId = computed(() => (route.query.catId as string) || catStore.currentCatId || catStore.cats[0]?.id || '')
const recordType = computed(() => (route.params.type as string) || (route.query.type as string) || 'feeding')
const cat = computed(() => catStore.cats.find((c) => c.id === (formData.value.catId || catId.value)) || catStore.cats[0])

const formData = ref<Record<string, string>>({})
const collapsed = ref(true)
const saving = ref(false)
const saved = ref(false)
const validationError = ref('')
const vaccineRecordId = ref('')
const vaccineSuggestion = ref<VaccineSuggestionDTO | null>(null)
const vaccineSuggestionError = ref('')
const nextReminderDate = ref('')
const nextReminderDateSource = ref('rule_suggestion')
const addVaccineReminder = ref(true)
const reminderLinkError = ref('')
const matchingVaccineReminders = ref<Reminder[]>([])
const creatingVaccineReminder = ref(false)

const vaccineTypes = [
  { value: 'fvrhcp', label: '猫三联' },
  { value: 'felv', label: 'FeLV' },
  { value: 'rabies', label: '狂犬疫苗' },
  { value: 'other', label: '其他' },
  { value: 'unknown', label: '不确定' }
]
const doseOptions = [
  { value: '1', label: '第 1 针' },
  { value: '2', label: '第 2 针' },
  { value: '3', label: '第 3 针' },
  { value: 'booster', label: '加强针' },
  { value: 'unknown', label: '不确定' }
]
const seriesOptions = [
  { value: 'primary', label: '初免系列' },
  { value: 'booster', label: '加强针' },
  { value: 'unknown', label: '不确定' }
]
const dateSourceOptions = [
  { value: 'rule_suggestion', label: '系统规则建议' },
  { value: 'user', label: '我提供的日期' },
  { value: 'veterinarian', label: '兽医提供的日期' },
  { value: 'product_label', label: '产品说明提供的日期' }
]

const catNames = computed(() => catStore.cats.map((item) => item.name))
const catIndex = computed(() => Math.max(0, catStore.cats.findIndex((item) => item.id === (formData.value.catId || catId.value))))
const vaccineTypeNames = vaccineTypes.map((item) => item.label)
const doseNames = doseOptions.map((item) => item.label)
const seriesNames = seriesOptions.map((item) => item.label)
const dateSourceNames = dateSourceOptions.map((item) => item.label)
const vaccineName = computed(() => vaccineTypes.find((item) => item.value === formData.value.vaccine_code)?.label || '')
const selectedDoseLabel = computed(() => doseOptions.find((item) => item.value === formData.value.dose_number)?.label || '请选择')
const selectedSeriesLabel = computed(() => seriesOptions.find((item) => item.value === formData.value.series_stage)?.label || '请选择')
const selectedDateSourceLabel = computed(() => dateSourceOptions.find((item) => item.value === nextReminderDateSource.value)?.label || '请选择日期来源')
const suppliedDateSourceOptions = dateSourceOptions.filter((item) => item.value !== 'rule_suggestion')
const suppliedDateSourceNames = suppliedDateSourceOptions.map((item) => item.label)
const suppliedDateSourceIndex = computed(() => Math.max(0, suppliedDateSourceOptions.findIndex((item) => item.value === formData.value.next_due_source)))
const ageWeeksFromProfile = computed(() => {
  const birthday = cat.value?.birthday
  const date = formData.value.vaccination_date
  if (!birthday || !date) return null
  const birthParts = birthday.slice(0, 10).split('-').map(Number)
  const dateParts = date.split('-').map(Number)
  if (birthParts.length !== 3 || dateParts.length !== 3 || [...birthParts, ...dateParts].some(Number.isNaN)) return null
  const birthUTC = Date.UTC(birthParts[0], birthParts[1] - 1, birthParts[2])
  const vaccinationUTC = Date.UTC(dateParts[0], dateParts[1] - 1, dateParts[2])
  if (vaccinationUTC < birthUTC) return null
  return Math.floor((vaccinationUTC - birthUTC) / (7 * 24 * 60 * 60 * 1000))
})
const reminderId = computed(() => (route.query.reminderId as string) || '')

// Initialize form fields based on type
const fieldLabels: Record<string, string> = {
  feeding: '喂食',
  vomit: '呕吐',
  elimination: '排便',
  weight: '体重',
  medication: '用药',
  vaccine: '疫苗',
  custom: '自定义'
}

function init() {
  const requestedDate = route.query.date
  const initialDate = isValidDateKey(requestedDate) ? requestedDate : todayStr()
  const currentTime = localDateTimeStr().slice(10)
  formData.value = {
    time: `${initialDate}${currentTime}`, catId: catId.value,
    vaccination_date: initialDate, vaccine_code: '', vaccine_name: '', dose_number: 'unknown',
    series_stage: 'unknown', timezone: deviceTimezone(), next_due_date: '', next_due_source: '',
    previous_dose_date: '', age_weeks: '', product_name: '', manufacturer: '', lot_number: '', clinic: '', notes: ''
  }
}

function toggleCollapse() {
  collapsed.value = !collapsed.value
}

async function save() {
  if (recordType.value === 'vaccine') {
    await saveVaccine()
    return
  }
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

function selectVaccineType(event: unknown) {
  const index = Number((event as { detail: { value: string } }).detail.value)
  const option = vaccineTypes[index]
  if (!option) return
  formData.value.vaccine_code = option.value
  formData.value.vaccine_name = option.value === 'other' ? '' : option.label
}

function selectDose(event: unknown) {
  const index = Number((event as { detail: { value: string } }).detail.value)
  const option = doseOptions[index]
  if (!option) return
  formData.value.dose_number = option.value
  if (option.value === 'booster') formData.value.series_stage = 'booster'
  else if (option.value !== 'unknown' && formData.value.series_stage === 'unknown') formData.value.series_stage = 'primary'
}

function selectSeries(event: unknown) {
  const index = Number((event as { detail: { value: string } }).detail.value)
  formData.value.series_stage = seriesOptions[index]?.value || 'unknown'
}

function selectSuppliedDateSource(event: unknown) {
  const index = Number((event as { detail: { value: string } }).detail.value)
  formData.value.next_due_source = suppliedDateSourceOptions[index]?.value || 'user'
}

function selectCat(event: unknown) {
  const index = Number((event as { detail: { value: string } }).detail.value)
  formData.value.catId = catStore.cats[index]?.id || ''
}

function selectReminderDate(event: unknown) {
  const value = (event as { detail: { value: string } }).detail.value
  if (typeof value !== 'string') return
  nextReminderDate.value = value
  if (vaccineSuggestion.value?.default_reminder_date !== value) nextReminderDateSource.value = 'user'
}

function selectVaccinationDate(event: unknown) {
  const value = (event as { detail: { value: string } }).detail.value
  if (typeof value === 'string') formData.value.vaccination_date = value
}

function selectProvidedDueDate(event: unknown) {
  const value = (event as { detail: { value: string } }).detail.value
  if (typeof value === 'string') formData.value.next_due_date = value
}

function selectPreviousDoseDate(event: unknown) {
  const value = (event as { detail: { value: string } }).detail.value
  if (typeof value === 'string') formData.value.previous_dose_date = value
}

function selectManualAge(event: unknown) {
  const value = (event as { detail: { value: string } }).detail.value
  if (typeof value === 'string') formData.value.age_weeks = value
}

function selectScheduleSource(event: unknown) {
  const index = Number((event as { detail: { value: string } }).detail.value)
  nextReminderDateSource.value = dateSourceOptions[index]?.value || 'user'
}

function copyVaccineSource() {
  const url = vaccineSuggestion.value?.source_url
  if (!url) return
  uni.setClipboardData({ data: url, success: () => uni.showToast({ title: '来源链接已复制', icon: 'success' }) })
}

function vaccineDateTime(date: string): string {
  const [year, month, day] = date.split('-').map(Number)
  return new Date(year, month - 1, day, 9, 0, 0, 0).toISOString()
}

function validateVaccine(): boolean {
  validationError.value = ''
  if (!formData.value.catId || !cat.value) validationError.value = '请选择猫咪'
  else if (!formData.value.vaccination_date) validationError.value = '请选择接种日期'
  else if (!formData.value.vaccine_code) validationError.value = '请选择疫苗类型'
  else if (!formData.value.vaccine_name.trim()) validationError.value = '请填写疫苗名称'
  else if (!formData.value.dose_number) validationError.value = '请选择针次；不确定时可选择“不确定”'
  else if (formData.value.vaccine_code === 'other' && !formData.value.vaccine_name.trim()) validationError.value = '请填写疫苗名称'
  if (validationError.value) {
    uni.showToast({ title: validationError.value, icon: 'none' })
    return false
  }
  return true
}

async function saveVaccine() {
  if (!validateVaccine() || saving.value || vaccineRecordId.value) return
  saving.value = true
  saved.value = false
  vaccineSuggestion.value = null
  vaccineSuggestionError.value = ''
  reminderLinkError.value = ''
  matchingVaccineReminders.value = []
  try {
    const ageWeeks = ageWeeksFromProfile.value
    const vaccineRecord = {
      type: 'vaccine',
      catIds: [formData.value.catId],
      occurredAt: vaccineDateTime(formData.value.vaccination_date),
      vaccination_date: formData.value.vaccination_date,
      timezone: formData.value.timezone || deviceTimezone(),
      vaccine_code: formData.value.vaccine_code,
      vaccine_name: formData.value.vaccine_name.trim(),
      dose_number: formData.value.dose_number,
      series_stage: formData.value.series_stage,
      previous_dose_date: formData.value.previous_dose_date,
      age_weeks: ageWeeks ?? formData.value.age_weeks,
      product_name: formData.value.product_name,
      manufacturer: formData.value.manufacturer,
      lot_number: formData.value.lot_number,
      clinic: formData.value.clinic,
      next_due_date: formData.value.next_due_date,
      next_due_source: formData.value.next_due_date ? formData.value.next_due_source : '',
      title: `${formData.value.vaccine_name.trim()} · ${selectedDoseLabel.value}`,
      note: formData.value.notes.trim(),
      source: 'manual'
    }
    const result = await services.saveRecord(vaccineRecord)
    if (!result.success) throw new Error('记录未能保存。请检查网络后重试，避免重复提交。')
    const savedRecord = result.data as { id?: string } | null
    if (!savedRecord?.id) throw new Error('记录已提交，但没有收到记录编号；请先检查记录列表，再决定是否重试。')
    vaccineRecordId.value = savedRecord.id
    saved.value = true

    if (reminderId.value) {
      const linked = await services.completeVaccineReminder(reminderId.value, savedRecord.id)
      if (!linked.success) reminderLinkError.value = '接种记录已保存，但原待办尚未完成关联。可稍后在提醒中心重试。'
    }

    const recommendation = await services.getVaccineSuggestion(savedRecord.id)
    if (recommendation.success) {
      vaccineSuggestion.value = recommendation.data
      nextReminderDate.value = recommendation.data.default_reminder_date || ''
      nextReminderDateSource.value = recommendation.data.date_source || (recommendation.data.status === 'suggested' ? 'rule_suggestion' : 'user')
    } else {
      vaccineSuggestionError.value = '接种记录已保存，但日期建议暂不可用。你仍可按兽医意见或产品说明手动选择日期。'
      nextReminderDate.value = ''
      nextReminderDateSource.value = 'user'
    }
    const todo = await services.getReminders('todo')
    if (todo.success) {
      matchingVaccineReminders.value = todo.data.filter((item) =>
        item.type === 'vaccine' && item.state === 'todo' && item.catId === formData.value.catId &&
        item.metadata?.vaccine_code === formData.value.vaccine_code && item.source_record_id !== savedRecord.id
      )
    }
    uni.showToast({ title: '接种记录已保存', icon: 'success' })
    if (!addVaccineReminder.value && matchingVaccineReminders.value.length === 0) {
      setTimeout(() => router.push('/today'), 700)
    }
  } catch (error) {
    validationError.value = error instanceof Error ? error.message : '保存失败，请重试'
  } finally {
    saving.value = false
  }
}

async function createVaccineReminder() {
  if (!vaccineRecordId.value || !nextReminderDate.value || creatingVaccineReminder.value) return
  creatingVaccineReminder.value = true
  reminderLinkError.value = ''
  try {
    const result = await services.createVaccineReminder(
      vaccineRecordId.value, nextReminderDate.value, nextReminderDateSource.value, formData.value.timezone || deviceTimezone()
    )
    if (!result.success) throw new Error('待办创建失败，请确认日期是未来日期后重试。接种记录仍已保存。')
    uni.showToast({ title: '疫苗待办已添加', icon: 'success' })
    setTimeout(() => router.push('/reminders'), 700)
  } catch (error) {
    reminderLinkError.value = error instanceof Error ? error.message : '待办创建失败，接种记录仍已保存。'
  } finally {
    creatingVaccineReminder.value = false
  }
}

async function linkExistingReminder(reminder: Reminder) {
  if (!vaccineRecordId.value) return
  const result = await services.completeVaccineReminder(reminder.id, vaccineRecordId.value)
  if (!result.success) {
    reminderLinkError.value = '接种记录已保存，但关联待办失败。请稍后重试。'
    return
  }
  matchingVaccineReminders.value = matchingVaccineReminders.value.filter((item) => item.id !== reminder.id)
  uni.showToast({ title: '已关联并完成原待办', icon: 'success' })
}

function dismissVaccineReminder() {
  addVaccineReminder.value = false
  if (saved.value) setTimeout(() => router.push('/today'), 200)
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
      <template v-if="recordType === 'vaccine'">
        <view class="form-group">
          <label class="form-label">猫咪 *</label>
          <picker mode="selector" :range="catNames" :value="catIndex" @change="selectCat">
            <view class="form-select">{{ cat?.name || '请选择猫咪' }}</view>
          </picker>
        </view>
        <view class="form-group">
          <label class="form-label">接种日期 *</label>
          <picker mode="date" :value="formData.vaccination_date" @change="selectVaccinationDate">
            <view class="form-select">{{ formData.vaccination_date || '请选择日期' }}</view>
          </picker>
        </view>
        <view class="form-group">
          <label class="form-label">疫苗类型 *</label>
          <picker mode="selector" :range="vaccineTypeNames" :value="Math.max(0, vaccineTypes.findIndex((item) => item.value === formData.vaccine_code))" @change="selectVaccineType">
            <view class="form-select">{{ vaccineName || '请选择疫苗类型' }}</view>
          </picker>
        </view>
        <view class="form-group">
          <label class="form-label">疫苗名称 *</label>
          <input v-model="formData.vaccine_name" class="form-input" placeholder="例如：猫三联、FeLV 或产品名称" />
        </view>
        <view class="form-group">
          <label class="form-label">针次/阶段 *</label>
          <picker mode="selector" :range="doseNames" :value="Math.max(0, doseOptions.findIndex((item) => item.value === formData.dose_number))" @change="selectDose">
            <view class="form-select">{{ selectedDoseLabel }}</view>
          </picker>
        </view>
        <view class="form-group">
          <label class="form-label">系列阶段</label>
          <picker mode="selector" :range="seriesNames" :value="Math.max(0, seriesOptions.findIndex((item) => item.value === formData.series_stage))" @change="selectSeries">
            <view class="form-select">{{ selectedSeriesLabel }}</view>
          </picker>
        </view>
        <view class="form-group">
          <label class="form-label">接种时年龄</label>
          <view v-if="ageWeeksFromProfile !== null" class="form-select">根据猫咪生日计算：{{ ageWeeksFromProfile }} 周龄</view>
          <input v-else v-model="formData.age_weeks" class="form-input" type="number" placeholder="档案生日未知时填写周龄；不确定可留空" @input="selectManualAge" />
          <text v-if="cat?.birthday" class="vaccine-hint">档案生日：{{ cat.birthday }}。如果生日不准确，请先更新猫咪档案。</text>
        </view>
        <view v-if="formData.dose_number === '2' || formData.dose_number === '3'" class="form-group">
          <label class="form-label">上一针日期（未在本应用记录时填写）</label>
          <picker mode="date" :value="formData.previous_dose_date || formData.vaccination_date" @change="selectPreviousDoseDate">
            <view class="form-select">{{ formData.previous_dose_date || '请选择上一针日期' }}</view>
          </picker>
        </view>
        <view class="form-group">
          <label class="form-label">品牌/产品</label>
          <input v-model="formData.product_name" class="form-input" placeholder="用于核对产品说明，可选" />
        </view>
        <view class="form-group">
          <label class="form-label">生产商</label>
          <input v-model="formData.manufacturer" class="form-input" placeholder="选填" />
        </view>
        <view class="form-group">
          <label class="form-label">批号</label>
          <input v-model="formData.lot_number" class="form-input" placeholder="选填" />
        </view>
        <view class="form-group">
          <label class="form-label">诊所</label>
          <input v-model="formData.clinic" class="form-input" placeholder="选填" />
        </view>
        <view class="form-group">
          <label class="form-label">兽医/产品说明给出的下一次日期</label>
          <picker mode="date" :value="formData.next_due_date || formData.vaccination_date" @change="selectProvidedDueDate">
            <view class="form-select">{{ formData.next_due_date || '可选；未提供时由规则判断或手动设置' }}</view>
          </picker>
        </view>
        <view v-if="formData.next_due_date" class="form-group">
          <label class="form-label">该日期的来源 *</label>
          <picker mode="selector" :range="suppliedDateSourceNames" :value="suppliedDateSourceIndex" @change="selectSuppliedDateSource">
            <view class="form-select">{{ suppliedDateSourceOptions[suppliedDateSourceIndex]?.label || '请选择日期来源' }}</view>
          </picker>
        </view>
        <view class="form-group">
          <label class="form-label">备注</label>
          <textarea v-model="formData.notes" class="form-input" rows="3" placeholder="记录接种情况或兽医说明" />
        </view>
        <view v-if="reminderId" class="vaccine-disclaimer">从疫苗待办进入：保存接种记录后会尝试将本记录关联到原待办。</view>
        <button type="button" class="reminder-btn" @click="addVaccineReminder = !addVaccineReminder">
          {{ addVaccineReminder ? '✓ 保存后查看下一针建议' : '保存后不创建下一次待办' }}
        </button>
      </template>

      <template v-else>
        <view v-for="field in getFormFields()" :key="field.key" class="form-group">
          <label class="form-label" :for="'qf-' + field.key"> {{ field.label }}{{ field.required ? ' *' : '' }} </label>
          <template v-if="field.type === 'textarea'">
            <textarea :id="'qf-' + field.key" v-model="formData[field.key]" class="form-input" rows="3" :placeholder="field.placeholder" />
          </template>
          <template v-else-if="field.type === 'select'">
            <picker mode="selector" :range="field.options || []" @change="selectOption(field, $event)">
              <view class="form-select">{{ formData[field.key] || '请选择' }}</view>
            </picker>
          </template>
          <template v-else>
            <input :id="'qf-' + field.key" v-model="formData[field.key]" :type="field.type === 'number' ? 'digit' : 'text'" class="form-input" :placeholder="field.placeholder" />
          </template>
        </view>
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
      </template>

      <view v-if="recordType === 'vaccine' && vaccineRecordId" class="vaccine-suggestion-card">
        <view class="vaccine-card-title">下一次接种日期建议</view>
        <view v-if="vaccineSuggestionError" class="vaccine-hint">{{ vaccineSuggestionError }}</view>
        <template v-if="vaccineSuggestion">
          <view class="vaccine-status">{{ vaccineSuggestion.status === 'suggested' ? '可核对的建议窗口' : vaccineSuggestion.status === 'unsupported' ? '当前类型由你手动设置' : '需要补充信息' }}</view>
          <view v-if="vaccineSuggestion.age_weeks !== undefined" class="vaccine-hint">判断时年龄：{{ vaccineSuggestion.age_weeks }} 周 · 已记录同类型针次：{{ vaccineSuggestion.prior_doses_recorded }}</view>
          <view v-if="vaccineSuggestion.date_window_start && vaccineSuggestion.date_window_end" class="vaccine-hint">
            建议窗口：{{ vaccineSuggestion.date_window_start }} 至 {{ vaccineSuggestion.date_window_end }}
          </view>
          <view class="vaccine-hint">{{ vaccineSuggestion.reason }}</view>
          <view v-if="vaccineSuggestion.rule_id" class="vaccine-hint">规则 {{ vaccineSuggestion.rule_id }} · 版本 {{ vaccineSuggestion.rule_version }}</view>
          <button v-if="vaccineSuggestion.source_url" type="button" class="vaccine-source" @click="copyVaccineSource">复制指南来源链接</button>
        </template>
        <view class="vaccine-disclaimer">此日期用于预约或核对，不是强制接种安排。实际接种时间以兽医意见、当地规定和产品说明为准。</view>
        <template v-if="addVaccineReminder">
          <view class="form-group">
            <label class="form-label">确认的待办日期</label>
            <picker mode="date" :value="nextReminderDate || todayStr()" @change="selectReminderDate">
              <view class="form-select">{{ nextReminderDate || '选择日期' }}</view>
            </picker>
          </view>
          <view class="form-group">
            <label class="form-label">日期来源</label>
            <picker mode="selector" :range="dateSourceNames" :value="Math.max(0, dateSourceOptions.findIndex((item) => item.value === nextReminderDateSource))" @change="selectScheduleSource">
              <view class="form-select">{{ selectedDateSourceLabel }}</view>
            </picker>
          </view>
          <button type="button" class="btn-primary" :disabled="!nextReminderDate || creatingVaccineReminder" @click="createVaccineReminder">
            {{ creatingVaccineReminder ? '正在创建…' : '核对后创建疫苗待办' }}
          </button>
        </template>
        <button type="button" class="reminder-btn" @click="dismissVaccineReminder">暂不提醒</button>
        <view v-if="matchingVaccineReminders.length" class="vaccine-existing-reminders">
          <view class="vaccine-card-title">发现同类待办</view>
          <view v-for="item in matchingVaccineReminders" :key="item.id" class="vaccine-existing-row">
            <text>{{ item.title }} · {{ item.scheduled_at ? formatZonedDateTime(item.scheduled_at, item.timezone) : item.time }}</text>
            <button type="button" class="reminder-btn" @click="linkExistingReminder(item)">关联本次接种并完成</button>
          </view>
        </view>
        <view v-if="reminderLinkError" class="form-error">{{ reminderLinkError }}</view>
      </view>

      <view class="form-actions">
        <view v-if="validationError" class="form-error">{{ validationError }}</view>
        <button type="button" class="btn-primary" :disabled="saving || (recordType === 'vaccine' && !!vaccineRecordId)" @click="save">
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
.vaccine-form-hint,
.vaccine-hint,
.vaccine-disclaimer {
  display: block;
  color: var(--color-text-secondary);
  font-size: var(--font-size-assist);
  line-height: 1.6;
  margin-top: 6px;
}
.vaccine-disclaimer {
  padding: var(--space-12);
  background: var(--color-bg-surface);
  border-radius: 8px;
}
.vaccine-suggestion-card {
  display: flex;
  flex-direction: column;
  gap: 10px;
  padding: var(--space-16);
  margin: var(--space-16) 0;
  border: 1px solid var(--color-divider);
  border-radius: 12px;
  background: var(--color-bg-surface);
}
.vaccine-card-title { font-weight: 600; }
.vaccine-status { color: var(--color-primary); font-weight: 600; }
.vaccine-source { text-align: left; color: var(--color-primary); background: transparent; padding: 0; }
.vaccine-existing-reminders { display: flex; flex-direction: column; gap: 8px; }
.vaccine-existing-row { display: flex; flex-direction: column; gap: 8px; padding: 10px 0; border-top: 1px solid var(--color-divider); }
</style>
