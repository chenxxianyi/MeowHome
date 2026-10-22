<script setup lang="ts">
import { ref, computed } from 'vue'
import { useRouter } from '../../utils/navigation'
import AppIcon from '../../components/app/AppIcon.vue'
import CatSwitcher from '../../components/cat/CatSwitcher.vue'
import OfflineBanner from '../../components/app/OfflineBanner.vue'
import { useAppStore } from '../../stores/app'
import { useCatStore } from '../../stores/cat'
import { services } from '../../services'
import { recordTypes } from '../../data/recordTypes'
import { usePageCapabilities, useProtectedPage } from '../../utils/page'
import { deviceTimezone, formatZonedDateTime } from '../../utils/date'
import { buildMonthGrid, dateKeyInTimezone, monthRange, recordCalendarDate, shiftMonth } from '../../utils/recordCalendar'
import type { RecordDTO } from '../../api/endpoints'

const router = useRouter()
const app = useAppStore()
const catStore = useCatStore()

// 从 AI 确认页“返回修改”回来时，保留上次输入以便继续编辑
const aiInput = ref(app.aiInput || '')
const aiLoading = ref(false)
const aiError = ref('')
const viewMode = ref<'calendar' | 'list'>('calendar')
const timezone = ref(deviceTimezone())
const timezoneLoaded = ref(false)
const todayDate = ref(dateKeyInTimezone(new Date().toISOString(), timezone.value))
const currentMonth = ref(todayDate.value.slice(0, 7))
const selectedDate = ref(todayDate.value)
const selectedType = ref('all')
const calendarRecords = ref<RecordDTO[]>([])
const listRecords = ref<RecordDTO[]>([])
const recordsLoading = ref(true)
const recordsError = ref('')
let activeRequest = 0

const allRecordTypes = [...recordTypes.high, ...recordTypes.health, ...recordTypes.life]
const recordTypeOptions = [{ id: 'all', label: '全部类型', icon: 'record' }, ...allRecordTypes]
const selectedTypeIndex = computed(() => Math.max(0, recordTypeOptions.findIndex((item) => item.id === selectedType.value)))
const selectedTypeLabel = computed(() => recordTypeOptions[selectedTypeIndex.value]?.label || '全部类型')
const monthLabel = computed(() => {
  const [year, month] = currentMonth.value.split('-')
  return `${year}年${Number(month)}月`
})
const selectedDateLabel = computed(() => {
  const [year, month, day] = selectedDate.value.split('-').map(Number)
  return selectedDate.value === todayDate.value
    ? `今天 · ${month}月${day}日`
    : `${year}年${month}月${day}日`
})
const weekDays = ['一', '二', '三', '四', '五', '六', '日']

const recordGroups = computed(() => {
  const groups = new Map<string, RecordDTO[]>()
  for (const record of listRecords.value) {
    const date = recordCalendarDate(record, timezone.value) || '未知日期'
    const items = groups.get(date) || []
    items.push(record)
    groups.set(date, items)
  }
  return Array.from(groups.entries())
    .sort(([a], [b]) => b.localeCompare(a))
    .map(([date, items]) => ({ date, items }))
})

const calendarCells = computed(() => buildMonthGrid(currentMonth.value).map((cell) => {
  if (!cell) return null
  const items = calendarRecords.value.filter((record) => recordCalendarDate(record, timezone.value) === cell.date)
  const types = [...new Set(items.map((record) => record.type))].map((type) => {
    const option = allRecordTypes.find((item) => item.id === type)
    return {
      id: type,
      icon: option?.icon || 'record',
      label: option?.label || type,
      group: type === 'vaccine' ? 'vaccine' : [...recordTypes.health, ...recordTypes.high].some((item) => item.id === type) ? 'health' : 'life'
    }
  })
  return {
    ...cell,
    count: items.length,
    markers: types.slice(0, 3),
    extraTypes: Math.max(0, types.length - 3),
    ariaLabel: `${dateLabel(cell.date)}，${items.length}条记录${types.length ? `，${types.map((item) => item.label).join('、')}` : ''}`
  }
}))

const selectedDayRecords = computed(() => calendarRecords.value
  .filter((record) => recordCalendarDate(record, timezone.value) === selectedDate.value)
  .sort((a, b) => new Date(a.occurred_at).getTime() - new Date(b.occurred_at).getTime()))

function recordLabel(type: string) {
  return allRecordTypes.find((item) => item.id === type)?.label || type
}

function recordIcon(type: string) {
  return allRecordTypes.find((item) => item.id === type)?.icon || 'record'
}

function recordTimeLabel(record: RecordDTO): string {
  if (record.type === 'vaccine') return '接种记录'
  return formatZonedDateTime(record.occurred_at, timezone.value).slice(11, 16)
}

function recordCatLabel(record: RecordDTO): string {
  const names = (record.cat_ids || []).map((id) => catStore.cats.find((cat) => cat.id === id)?.name).filter(Boolean)
  return names.join('、')
}

function dateLabel(date: string): string {
  const [year, month, day] = date.split('-').map(Number)
  return `${year}年${month}月${day}日`
}

async function ensureTimezone() {
  if (timezoneLoaded.value) return
  const family = await services.getFamily()
  timezone.value = family.success && family.data.timezone ? family.data.timezone : deviceTimezone()
  timezoneLoaded.value = family.success
  todayDate.value = dateKeyInTimezone(new Date().toISOString(), timezone.value)
  currentMonth.value = todayDate.value.slice(0, 7)
  selectedDate.value = todayDate.value
}

async function fetchRecords(options: { from?: string; to?: string; type?: string; limit: number }) {
  const records: RecordDTO[] = []
  let offset = 0
  const limit = options.limit
  while (true) {
    const result = await services.getRecords(catStore.currentCatId, { ...options, offset })
    if (!result.success) throw new Error('记录暂时无法加载')
    records.push(...result.data)
    if (result.data.length < limit) break
    offset += result.data.length
    if (offset > 100000) throw new Error('记录数量过多，暂时无法完整加载')
  }
  return records
}

async function loadCalendarRecords(selectLatestDate = false) {
  const requestId = ++activeRequest
  recordsLoading.value = true
  recordsError.value = ''
  try {
    const range = monthRange(currentMonth.value, timezone.value)
    // Records can be stored from a device timezone different from the family timezone.
    // Fetch a small buffer, then assign every row by its actual calendar date.
    const boundaryBuffer = 36 * 60 * 60 * 1000
    const items = await fetchRecords({
      from: new Date(Date.parse(range.from) - boundaryBuffer).toISOString(),
      to: new Date(Date.parse(range.to) + boundaryBuffer).toISOString(),
      type: selectedType.value === 'all' ? undefined : selectedType.value,
      limit: 500
    })
    if (requestId !== activeRequest) return
    const monthItems = items.filter((record) => recordCalendarDate(record, timezone.value).startsWith(`${currentMonth.value}-`))
    calendarRecords.value = monthItems
    if (selectLatestDate) {
      const dates = [...new Set(monthItems.map((record) => recordCalendarDate(record, timezone.value)))].sort().reverse()
      selectedDate.value = dates[0] || `${currentMonth.value}-01`
    }
  } catch {
    if (requestId !== activeRequest) return
    calendarRecords.value = []
    recordsError.value = '记录暂时无法加载，请检查网络后重试。'
  } finally {
    if (requestId === activeRequest) recordsLoading.value = false
  }
}

async function loadListRecords() {
  const requestId = ++activeRequest
  recordsLoading.value = true
  recordsError.value = ''
  try {
    const result = await services.getRecords(catStore.currentCatId, {
      type: selectedType.value === 'all' ? undefined : selectedType.value,
      limit: 500,
      offset: 0
    })
    if (!result.success) throw new Error('记录暂时无法加载')
    if (requestId !== activeRequest) return
    listRecords.value = result.data
  } catch {
    if (requestId !== activeRequest) return
    listRecords.value = []
    recordsError.value = '记录暂时无法加载，请检查网络后重试。'
  } finally {
    if (requestId === activeRequest) recordsLoading.value = false
  }
}

async function loadRecords() {
  await ensureTimezone()
  if (viewMode.value === 'calendar') await loadCalendarRecords()
  else await loadListRecords()
}

async function onCatSelect() {
  await loadRecords()
}

async function setView(mode: 'calendar' | 'list') {
  if (viewMode.value === mode) return
  viewMode.value = mode
  await loadRecords()
}

async function moveMonth(amount: number) {
  currentMonth.value = shiftMonth(currentMonth.value, amount)
  await loadCalendarRecords(true)
}

async function goToToday() {
  todayDate.value = dateKeyInTimezone(new Date().toISOString(), timezone.value)
  const month = todayDate.value.slice(0, 7)
  const changedMonth = currentMonth.value !== month
  currentMonth.value = month
  selectedDate.value = todayDate.value
  if (changedMonth || viewMode.value === 'calendar') await loadCalendarRecords()
}

function selectRecordType(event: unknown) {
  const index = Number((event as { detail: { value: string } }).detail.value)
  selectedType.value = recordTypeOptions[index]?.id || 'all'
  loadRecords()
}

const showRecordTypeSheet = ref(false)
function openRecordTypeSheet() {
  showRecordTypeSheet.value = true
}

function createRecord(type: string) {
  showRecordTypeSheet.value = false
  const selectedCatId = catStore.currentCatId === 'all' ? catStore.cats[0]?.id : catStore.currentCatId
  const query = [`date=${encodeURIComponent(selectedDate.value)}`]
  if (selectedCatId) query.push(`catId=${encodeURIComponent(selectedCatId)}`)
  router.push(`/records/quick/${encodeURIComponent(type)}?${query.join('&')}`)
}

const aiExamples = ['今天没怎么吃饭', '晚上吐了一次黄色的水', '精神状态很好，玩了一整天']

function useAIExample(example: string) {
  aiInput.value = example
  aiError.value = ''
}

const recording = ref(false)
const micError = ref('')

function toggleVoiceInput() {
  recording.value = false
  micError.value = '当前版本尚未接入语音转文字，请先使用文字输入'
  uni.showToast({ title: micError.value, icon: 'none' })
}

// 常用（高频）记录：主路径，首屏大卡片
const commonTypes = computed(() => recordTypes.high)

// 其余分类弱化为两个分组
const moreGroups = computed(() => [
  { key: 'health', title: '健康', items: recordTypes.health },
  { key: 'life', title: '生活', items: recordTypes.life }
])

function selectType(type: string) {
  router.push(`/records/quick/${type}`)
}

async function parseAI() {
  if (!aiInput.value.trim() || aiLoading.value) return
  aiLoading.value = true
  aiError.value = ''
  app.setAIState(aiInput.value, null, true)
  try {
    const res = await services.parseAI(aiInput.value)
    if (res.success) {
      app.setAIState(aiInput.value, res.data, false)
      aiLoading.value = false
      router.push('/records/ai/confirm')
    } else {
      aiError.value = '没有识别到有效内容，请补充描述后重试'
      app.setAIState(aiInput.value, null, false)
      aiLoading.value = false
    }
  } catch {
    aiError.value = 'AI 服务暂时不可用，可先使用下方分类手动记录'
    app.setAIState(aiInput.value, null, false)
    aiLoading.value = false
  }
}

useProtectedPage(loadRecords)
usePageCapabilities(loadRecords, '猫宅 · 记录猫咪的每一天')
</script>

<template>
  <OfflineBanner />
  <view class="page-header records-header">
    <view class="page-title"> 记录 </view>
    <CatSwitcher @select="onCatSelect" />
  </view>

  <view class="page-content records-page-content">
    <!-- 常用记录：主路径，首屏直达 -->
    <view class="records-hero" aria-label="常用记录">
      <view class="records-section-title"> 常用 </view>
      <view class="record-hero-grid">
        <button v-for="item in commonTypes" :key="item.id" class="record-hero-btn" @click="selectType(item.id)">
          <AppIcon :name="item.icon" :size="26" />
          <text>{{ item.label }}</text>
        </button>
      </view>
    </view>

    <!-- AI 管家：自然语言优先的记录工作台 -->
    <view class="ai-assistant" aria-label="AI 管家">
      <view class="ai-assistant-head">
        <view class="ai-assistant-mark">
          <AppIcon name="ai" :size="22" />
        </view>
        <view class="ai-assistant-heading">
          <text class="ai-assistant-kicker">AI 智能记录</text>
          <view>说一说，管家帮你记</view>
        </view>
      </view>

      <view class="ai-assistant-subtitle"> 不用先选分类，直接描述发生了什么，AI 会提取时间、事件和数值。 </view>

      <view class="ai-example-list" aria-label="输入示例">
        <text class="ai-example-label">试试这样说</text>
        <button
          v-for="example in aiExamples"
          :key="example"
          type="button"
          class="ai-example-chip"
          @click="useAIExample(example)"
        >
          {{ example }}
        </button>
      </view>

      <view class="ai-composer" :class="{ 'is-recording': recording }">
        <textarea
          v-model="aiInput"
          class="ai-composer-input"
          rows="3"
          maxlength="300"
          :placeholder="recording ? '正在聆听，请说话…' : '例如：今天早上没怎么吃，下午吐了一次黄色的水…'"
          aria-label="AI 自然语言输入"
          @keydown.ctrl.enter.prevent="parseAI"
          @keydown.meta.enter.prevent="parseAI"
        />
        <view class="ai-composer-footer">
          <text class="ai-composer-hint">Ctrl / ⌘ + Enter 发送</text>
          <view class="ai-composer-actions">
            <button
              type="button"
              class="ai-voice-btn"
              :class="{ recording }"
              :aria-label="recording ? '停止语音输入' : '语音输入'"
              :aria-pressed="recording"
              @click="toggleVoiceInput"
            >
              <AppIcon name="mic" :size="18" />
              <text>{{ recording ? '停止' : '语音' }}</text>
            </button>
            <button
              type="button"
              class="ai-submit-btn"
              :class="{ 'is-loading': aiLoading }"
              :disabled="!aiInput.trim() || aiLoading"
              @click="parseAI"
            >
              <AppIcon :name="aiLoading ? 'loading' : 'ai'" :size="17" />
              {{ aiLoading ? '正在整理…' : '智能整理' }}
            </button>
          </view>
        </view>
      </view>
      <view v-if="micError" class="ai-assistant-error" aria-live="polite">
        {{ micError }}
      </view>
      <view v-if="aiError" class="ai-assistant-error" aria-live="polite">
        {{ aiError }}
      </view>
      <button type="button" class="ai-medical-entry" @click="router.push('/medical/upload')">
        <view class="ai-medical-icon">
          <AppIcon name="medical" :size="19" />
        </view>
        <view class="ai-medical-copy">
          <text>导入病历或检查单</text>
          <view>拍照扫描，自动提取检查信息</view>
        </view>
        <AppIcon class="ai-medical-arrow" name="chevronRight" :size="18" />
      </button>
    </view>

    <!-- 健康 / 生活：其余分类 -->
    <view v-for="group in moreGroups" :key="group.key" class="record-type-section">
      <view class="record-type-group-title">
        {{ group.title }}
      </view>
      <view class="record-type-grid">
        <button v-for="item in group.items" :key="item.id" class="record-type-btn" @click="selectType(item.id)">
          <AppIcon :name="item.icon" :size="20" />
          <text>{{ item.label }}</text>
        </button>
      </view>
    </view>

    <view class="records-history">
      <view class="record-browser-heading">
        <view class="records-section-title">记录浏览</view>
        <view class="record-browser-actions">
          <picker mode="selector" :range="recordTypeOptions.map((item) => item.label)" :value="selectedTypeIndex" @change="selectRecordType">
            <view class="calendar-filter" aria-label="筛选记录类型">
              <text>{{ selectedTypeLabel }}</text>
              <AppIcon name="chevronDown" :size="14" />
            </view>
          </picker>
          <view class="record-view-toggle" role="tablist" aria-label="记录浏览方式">
            <button type="button" :class="{ active: viewMode === 'calendar' }" :aria-pressed="viewMode === 'calendar'" @click="setView('calendar')">日历</button>
            <button type="button" :class="{ active: viewMode === 'list' }" :aria-pressed="viewMode === 'list'" @click="setView('list')">列表</button>
          </view>
        </view>
      </view>

      <template v-if="viewMode === 'calendar'">
        <view class="calendar-card">
          <view class="calendar-toolbar">
            <view class="calendar-month-title">{{ monthLabel }}</view>
            <view class="calendar-toolbar-actions">
              <button type="button" class="calendar-today-btn" @click="goToToday">今天</button>
              <button type="button" class="calendar-nav-btn" aria-label="上个月" @click="moveMonth(-1)">‹</button>
              <button type="button" class="calendar-nav-btn" aria-label="下个月" @click="moveMonth(1)">›</button>
            </view>
          </view>

          <view class="calendar-grid calendar-weekdays" aria-hidden="true">
            <text v-for="day in weekDays" :key="day">{{ day }}</text>
          </view>
          <view v-if="recordsLoading" class="calendar-grid calendar-days calendar-skeleton-grid" aria-label="正在加载月历" aria-busy="true">
            <view v-for="index in 42" :key="index" class="calendar-day-skeleton" />
          </view>
          <view v-else class="calendar-grid calendar-days">
            <template v-for="(cell, index) in calendarCells" :key="cell?.date || `blank-${index}`">
              <button
                v-if="cell"
                type="button"
                class="calendar-day"
                :class="{ 'is-today': cell.date === todayDate, 'is-selected': cell.date === selectedDate, 'has-records': cell.count > 0 }"
                :aria-label="cell.ariaLabel"
                :aria-pressed="cell.date === selectedDate"
                @click="selectedDate = cell.date"
              >
                <view class="calendar-day-top">
                  <text class="calendar-day-number">{{ cell.day }}</text>
                </view>
                <view class="calendar-markers">
                  <AppIcon
                    v-for="marker in cell.markers"
                    :key="marker.id"
                    :name="marker.icon"
                    :size="10"
                    class="calendar-marker"
                    :class="`calendar-marker--${marker.group}`"
                    :aria-label="marker.label"
                  />
                  <text v-if="cell.extraTypes" class="calendar-extra-types">+{{ cell.extraTypes }}</text>
                </view>
              </button>
              <view v-else class="calendar-day calendar-day--blank" aria-hidden="true" />
            </template>
          </view>

          <view class="selected-day-panel">
            <view class="selected-day-heading">
              <view class="selected-day-title">{{ selectedDateLabel }}</view>
              <view class="selected-day-heading-actions">
                <text class="selected-day-count">{{ selectedDayRecords.length }} 条记录</text>
                <button type="button" class="add-selected-day-btn" aria-label="记录这一天" @click="openRecordTypeSheet">
                  <AppIcon name="plus" :size="16" />
                </button>
              </view>
            </view>

            <view v-if="recordsLoading" class="skeleton calendar-detail-skeleton" aria-label="正在加载当天记录" />
            <view v-else-if="recordsError" class="calendar-feedback" role="alert">
              <text>{{ recordsError }}</text>
              <button type="button" class="calendar-retry-btn" @click="loadRecords">重新加载</button>
            </view>
            <view v-else-if="!selectedDayRecords.length" class="calendar-empty-day">
              <AppIcon name="calendar" :size="19" />
              <text>{{ calendarRecords.length ? '这一天还没有符合条件的记录' : selectedType === 'all' ? '暂无记录，新的记录会显示在这里' : '这一天没有符合筛选条件的记录' }}</text>
            </view>
            <view v-else class="calendar-record-list">
              <view
                v-for="item in selectedDayRecords"
                :key="item.id"
                class="records-history-item calendar-record-item"
                @click="router.push(`/records/detail/${encodeURIComponent(item.id)}`)"
              >
                <AppIcon :name="recordIcon(item.type)" :size="18" :class="{ 'calendar-record-vaccine': item.type === 'vaccine' }" />
                <view class="records-history-copy">
                  <text>{{ item.title || recordLabel(item.type) }}</text>
                  <view>{{ recordLabel(item.type) }} · {{ recordTimeLabel(item) }}<text v-if="recordCatLabel(item)"> · {{ recordCatLabel(item) }}</text></view>
                </view>
                <AppIcon name="chevronRight" :size="16" class="calendar-record-arrow" />
              </view>
            </view>
          </view>

          <view class="calendar-legend">
            <view><AppIcon name="vaccine" :size="11" class="calendar-marker--vaccine" /> 疫苗</view>
            <view><AppIcon name="medical" :size="11" class="calendar-marker--health" /> 健康</view>
            <view><AppIcon name="calendar" :size="11" class="calendar-marker--life" /> 生活</view>
          </view>
        </view>
      </template>

      <view v-else class="records-list-panel">
        <view v-if="recordsLoading" class="skeleton" style="height: 96px" />
        <view v-else-if="recordsError" class="calendar-feedback" role="alert">
          <text>{{ recordsError }}</text>
          <button type="button" class="calendar-retry-btn" @click="loadRecords">重新加载</button>
        </view>
        <view v-else-if="!recordGroups.length" class="empty-state">还没有记录，从上方选择一种开始吧。</view>
        <view v-for="group in recordGroups" v-else :key="group.date" class="records-day-group">
          <view class="records-day-title">{{ group.date }}</view>
          <view
            v-for="item in group.items"
            :key="item.id"
            class="records-history-item"
            @click="router.push(`/records/detail/${encodeURIComponent(item.id)}`)"
          >
            <AppIcon :name="recordIcon(item.type)" :size="18" :class="{ 'calendar-record-vaccine': item.type === 'vaccine' }" />
            <view class="records-history-copy">
              <text>{{ item.title || recordLabel(item.type) }}</text>
              <view>{{ item.note || recordLabel(item.type) }} · {{ recordTimeLabel(item) }}<text v-if="recordCatLabel(item)"> · {{ recordCatLabel(item) }}</text></view>
            </view>
          </view>
        </view>
      </view>
    </view>

    <view v-if="showRecordTypeSheet" class="record-type-sheet-mask" @click="showRecordTypeSheet = false">
      <view class="record-type-sheet" @click.stop>
        <view class="record-type-sheet-heading">
          <view>
            <text>记录这一天</text>
            <view>{{ dateLabel(selectedDate) }}</view>
          </view>
          <button type="button" aria-label="关闭" @click="showRecordTypeSheet = false">×</button>
        </view>
        <view class="record-type-sheet-grid">
          <button v-for="item in allRecordTypes" :key="item.id" type="button" @click="createRecord(item.id)">
            <AppIcon :name="item.icon" :size="19" />
            <text>{{ item.label }}</text>
          </button>
        </view>
      </view>
    </view>
  </view>
</template>

<style scoped>
.records-history {
  margin-top: var(--space-24);
}
.record-browser-heading,
.calendar-toolbar,
.selected-day-heading,
.record-type-sheet-heading {
  display: flex;
  align-items: center;
  justify-content: space-between;
}
.record-browser-heading {
  margin-bottom: var(--space-12);
}
.record-browser-actions {
  display: flex;
  align-items: center;
  gap: 8px;
}
.record-view-toggle {
  display: flex;
  gap: 3px;
  padding: 3px;
  border-radius: 999px;
  background: var(--color-bg-muted, #f2efeb);
}
.record-view-toggle button {
  padding: 6px 14px;
  border-radius: 999px;
  color: var(--color-text-secondary);
  background: transparent;
  font-size: var(--font-size-caption);
}
.record-view-toggle button.active {
  color: var(--color-text-primary);
  background: var(--color-bg-surface);
  box-shadow: 0 1px 4px rgb(43 38 32 / 10%);
}
.calendar-card,
.records-list-panel {
  padding: var(--space-16);
  border: 1px solid var(--color-divider);
  border-radius: 18px;
  background: var(--color-bg-surface);
}
.calendar-toolbar {
  margin-bottom: 10px;
}
.calendar-month-title,
.selected-day-title {
  color: var(--color-text-primary);
  font-size: var(--font-size-body-lg);
  font-weight: 700;
}
.calendar-toolbar-actions {
  display: flex;
  align-items: center;
  gap: 4px;
}
.calendar-filter {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  border: 1px solid var(--color-divider);
  color: var(--color-text-secondary);
  background: var(--color-bg-surface);
}
.calendar-today-btn,
.calendar-nav-btn {
  display: flex;
  align-items: center;
  justify-content: center;
  box-sizing: border-box;
  height: 38px;
  min-height: 38px;
  padding: 0;
  border: 0;
  color: var(--color-text-secondary);
  background: transparent;
  box-shadow: none;
}
.calendar-today-btn {
  min-width: 40px;
  padding: 0 4px;
  color: var(--color-brand);
  font-size: 12px;
  font-weight: 700;
  line-height: 1;
  white-space: nowrap;
}
.calendar-nav-btn {
  width: 36px;
  min-width: 36px;
  font-size: 22px;
  line-height: 1;
}
.calendar-filter {
  gap: 5px;
  min-height: 34px;
  padding: 0 9px;
  border-radius: 999px;
  font-size: var(--font-size-caption);
  white-space: nowrap;
}
.calendar-grid {
  display: grid;
  grid-template-columns: repeat(7, minmax(0, 1fr));
  gap: 4px;
}
.calendar-weekdays {
  margin-bottom: 3px;
  color: var(--color-text-tertiary);
  text-align: center;
  font-size: var(--font-size-caption);
}
.calendar-weekdays text {
  padding: 5px 0;
}
.calendar-day {
  display: flex;
  min-width: 0;
  width: 100%;
  height: 40px;
  min-height: 40px;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 0;
  padding: 0;
  border: 1px solid transparent;
  border-radius: 10px;
  color: var(--color-text-primary);
  background: transparent;
}
.calendar-day-skeleton {
  height: 40px;
  min-height: 40px;
  border-radius: 10px;
  background: #f4f1ed;
  animation: calendar-pulse 1.2s ease-in-out infinite alternate;
}
@keyframes calendar-pulse {
  from { opacity: 0.6; }
  to { opacity: 1; }
}
.calendar-day--blank {
  pointer-events: none;
}
.calendar-day.is-today .calendar-day-number {
  color: var(--color-brand);
  font-weight: 700;
}
.calendar-day.is-selected {
  border-color: var(--color-brand);
  color: #fff;
  background: var(--color-brand);
  font-weight: 700;
}
.calendar-day.is-selected.is-today .calendar-day-number {
  color: #fff;
}
.calendar-day-top {
  display: flex;
  width: 100%;
  min-height: 17px;
  align-items: center;
  justify-content: center;
}
.calendar-day-number {
  font-size: var(--font-size-caption);
  line-height: 17px;
}
.calendar-markers {
  display: flex;
  width: 100%;
  height: 10px;
  min-height: 10px;
  align-items: center;
  justify-content: center;
  gap: 2px;
  overflow: hidden;
}
.calendar-marker--vaccine {
  color: #b75d78;
}
.calendar-marker--health {
  color: #6b8da2;
}
.calendar-marker--life {
  color: #83a27d;
}
.calendar-extra-types {
  color: var(--color-text-tertiary);
  font-size: 8px;
  line-height: 10px;
}
.calendar-legend {
  display: flex;
  justify-content: center;
  gap: var(--space-16);
  margin-top: var(--space-12);
  padding-top: var(--space-12);
  border-top: 1px solid var(--color-divider);
  color: var(--color-text-tertiary);
  font-size: var(--font-size-caption);
}
.calendar-legend view {
  display: flex;
  align-items: center;
  gap: 4px;
}
.selected-day-panel {
  margin-top: 10px;
  padding-top: 10px;
  border-top: 1px solid var(--color-divider);
}
.selected-day-heading {
  gap: 8px;
  margin-bottom: 6px;
}
.selected-day-heading-actions {
  display: flex;
  align-items: center;
  gap: 8px;
}
.selected-day-count {
  color: var(--color-text-tertiary);
  font-size: var(--font-size-caption);
}
.add-selected-day-btn {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 32px;
  height: 32px;
  min-width: 32px;
  min-height: 32px;
  padding: 0;
  border: 0;
  border-color: transparent;
  border-radius: 50%;
  color: var(--color-brand);
  background: #f7f0eb;
  box-shadow: none;
}
.calendar-detail-skeleton {
  height: 70px;
  margin-top: var(--space-12);
}
.calendar-empty-day {
  display: flex;
  min-height: 40px;
  align-items: center;
  gap: 8px;
  color: var(--color-text-tertiary);
  font-size: var(--font-size-caption);
}
.calendar-empty-day text {
  flex: 1;
  min-width: 0;
}
.calendar-feedback {
  display: flex;
  min-height: 108px;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 8px;
  color: var(--color-text-tertiary);
  text-align: center;
  font-size: var(--font-size-caption);
}
.calendar-feedback {
  color: var(--color-text-secondary);
}
.calendar-retry-btn {
  padding: 5px 10px;
  border-radius: 999px;
  color: var(--color-brand);
  background: transparent;
  font-size: var(--font-size-caption);
}
.calendar-record-item {
  min-height: 60px;
  padding-right: 0;
  padding-left: 0;
}
.calendar-record-vaccine {
  color: #b75d78;
}
.calendar-record-arrow {
  color: var(--color-text-tertiary);
}
.records-list-panel {
  padding-top: 0;
}
.record-type-sheet-mask {
  position: fixed;
  z-index: 30;
  inset: 0;
  display: flex;
  align-items: flex-end;
  background: rgb(23 21 19 / 40%);
}
.record-type-sheet {
  width: 100%;
  max-height: 72vh;
  overflow-y: auto;
  padding: var(--space-20) var(--space-16) calc(var(--space-24) + env(safe-area-inset-bottom));
  border-radius: 22px 22px 0 0;
  background: var(--color-bg-surface);
}
.record-type-sheet-heading {
  margin-bottom: var(--space-16);
  color: var(--color-text-primary);
  font-weight: 700;
}
.record-type-sheet-heading view view {
  margin-top: 4px;
  color: var(--color-text-tertiary);
  font-size: var(--font-size-caption);
  font-weight: 400;
}
.record-type-sheet-heading button {
  width: 32px;
  height: 32px;
  border-radius: 50%;
  color: var(--color-text-secondary);
  background: var(--color-bg-muted, #f2efeb);
  font-size: 22px;
}
.record-type-sheet-grid {
  display: grid;
  grid-template-columns: repeat(4, minmax(0, 1fr));
  gap: 10px;
}
.record-type-sheet-grid button {
  display: flex;
  min-height: 70px;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 6px;
  border-radius: 14px;
  color: var(--color-text-secondary);
  background: var(--color-bg-muted, #f6f3ef);
  font-size: var(--font-size-caption);
}
.records-day-title {
  margin: var(--space-16) 0 var(--space-8);
  color: var(--color-text-tertiary);
  font-size: var(--font-size-assist);
}
.records-history-item {
  display: flex;
  gap: var(--space-12);
  align-items: center;
  padding: var(--space-12);
  border-bottom: 1px solid var(--color-divider);
  background: var(--color-bg-surface);
}
.records-history-copy {
  flex: 1;
}
.records-history-copy > text {
  color: var(--color-text-primary);
  font-weight: 600;
}
.records-history-copy > view {
  margin-top: 3px;
  color: var(--color-text-tertiary);
  font-size: var(--font-size-caption);
}
.empty-state {
  padding: var(--space-24);
  text-align: center;
  color: var(--color-text-tertiary);
}
</style>
