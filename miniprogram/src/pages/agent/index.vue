<script setup lang="ts">
import { computed, onMounted, onUnmounted, ref } from 'vue'
import { useAgentStore } from '../../stores/agent'
import { useCatStore } from '../../stores/cat'
import { useAuthStore } from '../../stores/auth'
import { useProtectedPage } from '../../utils/page'
import { useRoute, useRouter } from '../../utils/navigation'
import type { AgentMessage, AgentReminderInput } from '../../types/agent'

const store = useAgentStore()
const cats = useCatStore()
const auth = useAuthStore()
const route = useRoute()
const router = useRouter()
const tab = ref<'patrol' | 'chat'>(route.query.sessionId ? 'chat' : 'patrol')
const input = ref('')
const chatCat = ref('')
const chatCatOptions = computed(() => [{ id: '', name: '选择猫咪或在问题中说明' }, ...cats.cats.map((cat) => ({ id: cat.id, name: cat.name }))])
const chatCatIndex = computed(() => Math.max(0, chatCatOptions.value.findIndex((cat) => cat.id === chatCat.value)))
const recommended = ['这两天有哪些进食记录？', '最近七天的体重趋势怎么样？', '帮我起草一条护理提醒']
const savedDraftFingerprint = ref('')
const draftDirty = computed(() => JSON.stringify({ form: form.value, date: date.value, time: clockTime.value }) !== savedDraftFingerprint.value)
const showPendingInput = computed(() => store.pendingChat && !store.chatMessages.some((m) => m.client_message_id === store.pendingChat?.client_message_id))
const messages = computed(() => route.query.catId ? store.important.filter((item) => item.cat_id === route.query.catId || item.evidence?.some((e) => e.cat_ids?.includes(route.query.catId))) : store.important)
const shownFamilyId = ref<string | null>(null)
const selected = ref<AgentMessage | null>(null)
const detailLoading = ref(false)
const busy = ref(false)
const error = ref('')
const form = ref<AgentReminderInput>({ cat_id: 'both', type: 'custom', title: '' })
const date = ref('')
const clockTime = ref('09:00')
const catOptions = computed(() => [{ id: 'both', name: '全家猫咪' }, ...cats.cats.map((cat) => ({ id: cat.id, name: cat.name }))])
const catIndex = computed(() => Math.max(0, catOptions.value.findIndex((cat) => cat.id === form.value.cat_id)))
const clockNow = ref(Date.now())
const draftExpired = computed(() => !!selected.value?.draft_expires_at && new Date(selected.value.draft_expires_at).getTime() <= clockNow.value)
let timer: ReturnType<typeof setInterval> | undefined
onMounted(() => { timer = setInterval(() => { clockNow.value = Date.now() }, 30_000) })
onUnmounted(() => { if (timer) clearInterval(timer) })

function localDateParts(at: Date) {
  const beijing = new Date(at.getTime() + 8 * 60 * 60 * 1000)
  const y = beijing.getUTCFullYear()
  const m = String(beijing.getUTCMonth() + 1).padStart(2, '0')
  const d = String(beijing.getUTCDate()).padStart(2, '0')
  return { date: `${y}-${m}-${d}`, time: `${String(beijing.getUTCHours()).padStart(2, '0')}:${String(beijing.getUTCMinutes()).padStart(2, '0')}` }
}

function fillForm(item: AgentMessage) {
  const source = item.draft_reminder
  form.value = {
    cat_id: source?.cat_id || item.cat_id || 'both',
    type: source?.type || 'custom',
    title: source?.title || item.title || '',
    subtitle: source?.subtitle,
    icon: source?.icon,
    timezone: 'Asia/Shanghai'
  }
  const fallback = new Date(Date.now() + 24 * 60 * 60 * 1000)
  fallback.setHours(9, 0, 0, 0)
  const parts = localDateParts(source?.scheduled_at ? new Date(source.scheduled_at) : fallback)
  date.value = parts.date
  clockTime.value = parts.time
  savedDraftFingerprint.value = JSON.stringify({ form: form.value, date: date.value, time: clockTime.value })
}

async function openMessage(id: string) {
  if (detailLoading.value) return
  detailLoading.value = true
  error.value = ''
  try {
    const item = await store.getMessage(id)
    if (item) { selected.value = item; fillForm(item) }
  } catch {
    error.value = '消息详情暂时无法读取，请重试'
  } finally {
    detailLoading.value = false
  }
}

async function load() {
  if (shownFamilyId.value !== auth.familyId) { selected.value = null; input.value = ''; chatCat.value = ''; error.value = '' }
  shownFamilyId.value = auth.familyId
  await store.load()
  await store.resumeChat(route.query.sessionId || '')
  if (route.query.catId && cats.cats.some((cat) => cat.id === route.query.catId)) chatCat.value = route.query.catId
  if (route.query.messageId) await openMessage(route.query.messageId)
}

function evidenceCatNames(ids?: string[]) {
  return (ids || []).map((id) => cats.cats.find((cat) => cat.id === id)?.name || id).join('、')
}

function setCat(e: any) { form.value.cat_id = catOptions.value[Number(e.detail.value)]?.id || 'both' }
function setDate(e: any) { date.value = String(e.detail.value || '') }
function setTime(e: any) { clockTime.value = String(e.detail.value || '') }

async function saveDraft() {
  const item = selected.value
  if (!item || busy.value) return
  const title = form.value.title.trim()
  const local = new Date(`${date.value}T${clockTime.value}:00+08:00`)
  if (!title || title.length > 120 || Number.isNaN(local.getTime()) || local.getTime() <= Date.now()) {
    error.value = '请填写 1–120 字标题和未来的具体时间'
    return
  }
  busy.value = true
  error.value = ''
  try {
    const draft = await store.editDraft(item.id, {
      expected_version: item.draft_version || 0,
      reminder: { ...form.value, title, scheduled_at: local.toISOString(), timezone: 'Asia/Shanghai' }
    })
    if (!draft) return
    selected.value = await store.getMessage(item.id)
    if (selected.value) fillForm(selected.value)
    uni.showToast({ title: '草稿已保存', icon: 'success' })
  } catch {
    error.value = '保存失败或草稿已变化，请重新打开'
  } finally {
    busy.value = false
  }
}

async function confirm() {
  const item = selected.value
  if (!item || busy.value || item.action_status !== 'pending' || !item.draft_version) return
  if (draftDirty.value) { error.value = '请先保存修改，再确认当前草稿'; return }
  busy.value = true
  error.value = ''
  try {
    const result = await store.confirm(item.id, item.draft_version)
    if (!result) return
    selected.value = await store.getMessage(item.id)
    uni.showToast({ title: '提醒已创建', icon: 'success' })
  } catch {
    error.value = '确认失败，请检查草稿版本与计划时间'
  } finally {
    busy.value = false
  }
}

async function dismiss() {
  const item = selected.value
  if (!item || busy.value) return
  busy.value = true
  error.value = ''
  try {
    await store.dismiss(item.id)
    selected.value = null
    uni.showToast({ title: '已忽略', icon: 'success' })
  } catch {
    error.value = '忽略失败，请稍后重试'
  } finally {
    busy.value = false
  }
}

function openEvidence(sourceType: string, sourceId: string, catId?: string) {
  if (sourceType === 'record' || sourceType === 'weight') {
    router.push(`/records/detail/${encodeURIComponent(sourceId)}`)
  } else if (sourceType === 'cat_profile' && sourceId) {
    router.push(`/cats/${encodeURIComponent(sourceId)}`)
  } else if (sourceType === 'reminder') {
    router.push('/reminders')
  } else if (sourceType === 'trend' && catId) {
    router.push(`/cats/${encodeURIComponent(catId)}/trends`)
  }
}

function openAction(action: NonNullable<AgentMessage['action_suggestions']>[number]) {
  if (action.type === 'view_trend' && action.cat_id && action.cat_id !== 'both') {
    router.push(`/cats/${encodeURIComponent(action.cat_id)}/trends`)
  } else if (action.type === 'view_records') {
    const evidence = selected.value?.evidence?.find((item) => item.source_type === 'record' || item.source_type === 'weight')
    if (evidence) openEvidence(evidence.source_type, evidence.source_id, action.cat_id)
    else router.push('/records')
  } else if (action.type === 'create_reminder' && selected.value) {
    if (typeof action.payload?.message_id === 'string') { void openMessage(action.payload.message_id); return }
    form.value.cat_id = action.cat_id || selected.value.cat_id || 'both'
  } else if (action.type === 'view_trend') {
    const catId = chatCat.value || selected.value?.evidence?.find((e) => e.cat_ids?.length)?.cat_ids?.[0]
    if (catId) router.push(`/cats/${encodeURIComponent(catId)}/trends`)
    else error.value = '请先选择猫咪，再查看趋势'
  }
}

async function send() {
  const text = input.value.trim()
  if (!text || store.chatSending || store.pendingChat) return
  const cat = cats.cats.find((item) => item.id === chatCat.value)
  const message = cat ? `猫咪：${cat.name}（ID：${cat.id}）。${text}` : text
  if ([...message].length > 2000) { error.value = '问题最多 2000 字'; return }
  input.value = ''
  await store.sendChat(message)
}

useProtectedPage(load)
</script>

<template>
  <view class="page-header">
    <view class="page-title">猫管家</view>
    <view class="page-subtitle">基于当前家庭记录的巡检与问答</view>
  </view>
  <view class="page-content">
    <view v-if="selected" class="agent-detail">
      <button class="reminder-btn" @click="selected = null">返回消息列表</button>
      <view class="agent-card" :class="`agent-${selected.severity}`">
        <view class="agent-card-header"><text class="agent-card-title">{{ selected.title }}</text><text class="ai-badge">{{ selected.severity }}</text></view>
        <view class="agent-card-body">{{ selected.body }}</view>
        <view v-if="selected.disclaimer" class="agent-evidence">{{ selected.disclaimer }}</view>
        <view v-if="selected.evidence?.length" class="agent-evidence">
          <view v-for="evidence in selected.evidence" :key="`${selected.id}-${evidence.source_id}`" class="agent-evidence-row">
            <text>{{ evidenceCatNames(evidence.cat_ids) }} · {{ evidence.excerpt || evidence.source_type }} · {{ evidence.occurred_at || '时间未记录' }}</text>
            <button v-if="['record', 'weight', 'cat_profile', 'reminder'].includes(evidence.source_type)" class="reminder-btn" @click="openEvidence(evidence.source_type, evidence.source_id, evidence.cat_ids?.[0])">查看来源</button>
          </view>
        </view>
        <view v-if="selected.action_suggestions?.length" class="reminder-actions">
          <button v-for="action in selected.action_suggestions" :key="action.title" class="reminder-btn" @click="openAction(action)">{{ action.title }}</button>
        </view>
      </view>
      <view v-if="(selected.visibility === 'family' || selected.type === 'reminder_draft') && selected.action_status !== 'confirmed' && selected.action_status !== 'dismissed' && selected.action_status !== 'expired' && !draftExpired" class="agent-draft">
        <view class="records-section-title">{{ selected.draft_version ? '编辑提醒草稿' : '创建提醒草稿' }}</view>
        <view class="page-subtitle">请核对猫咪、标题和具体时间；以下按北京时间（Asia/Shanghai，UTC+8）选择。</view>
        <picker :range="catOptions.map((cat) => cat.name)" :value="catIndex" @change="setCat"><view class="reminder-btn">{{ catOptions[catIndex]?.name || '选择猫咪' }}</view></picker>
        <input v-model="form.title" maxlength="120" placeholder="提醒标题" />
        <picker mode="date" :value="date" @change="setDate"><view class="reminder-btn">日期：{{ date }}</view></picker>
        <picker mode="time" :value="clockTime" @change="setTime"><view class="reminder-btn">时间：{{ clockTime }}</view></picker>
        <view class="reminder-actions">
          <button class="reminder-btn" :disabled="busy" @click="saveDraft">{{ busy ? '处理中…' : '保存草稿' }}</button>
          <button v-if="selected.action_status === 'pending' && selected.draft_version" class="reminder-btn primary" :disabled="busy || draftDirty || !selected.draft_reminder?.scheduled_at" @click="confirm">确认创建提醒</button>
        </view>
      </view>
      <view v-else-if="selected.action_status === 'confirmed'" class="page-subtitle">提醒已创建，可在提醒中心查看。</view>
      <view v-else-if="selected.type === 'chat_answer' && selected.visibility === 'private'" class="page-subtitle">需要提醒时，请在对话中请求生成草稿，再核对并确认。</view>
      <view v-else class="page-subtitle">这条草稿已关闭或过期。</view>
      <button v-if="selected.display_status !== 'dismissed'" class="reminder-btn" :disabled="busy" @click="dismiss">忽略此消息</button>
    </view>
    <view v-else>
      <view class="reminder-actions">
        <button class="reminder-btn" :class="{ primary: tab === 'patrol' }" @click="tab = 'patrol'">家庭巡检</button>
        <button class="reminder-btn" :class="{ primary: tab === 'chat' }" @click="tab = 'chat'">我的对话</button>
      </view>
      <view v-if="tab === 'chat'" class="agent-chat">
        <view class="page-subtitle">仅本人可见 · 当前家庭 · {{ chatCatOptions[chatCatIndex]?.name }}</view>
        <picker :range="chatCatOptions.map((cat) => cat.name)" :value="chatCatIndex" @change="chatCat = chatCatOptions[Number($event.detail.value)]?.id || ''"><view class="reminder-btn">{{ chatCatOptions[chatCatIndex]?.name }}</view></picker>
        <view class="reminder-actions">
          <button class="reminder-btn" :disabled="store.chatSending" @click="store.newChat()">新对话</button>
          <picker v-if="store.sessions.length" :range="store.sessions.map((s) => s.title || '历史对话')" @change="store.openSession(store.sessions[Number($event.detail.value)].id)"><view class="reminder-btn">恢复历史对话</view></picker>
          <button v-if="store.sessionsCursor" class="reminder-btn" @click="store.loadSessions(store.sessionsCursor)">更多会话</button>
        </view>
        <button v-if="store.chatCursor" class="reminder-btn" :disabled="store.chatLoading" @click="store.loadChatMessages(store.chatCursor)">加载更早对话</button>
        <view v-if="store.chatLoading" class="page-subtitle">正在恢复对话…</view>
        <view v-for="message in store.chatMessages" :key="message.id" class="agent-card" :class="{ 'chat-user': message.role === 'user' }">
          <view class="agent-card-header"><text>{{ message.role === 'user' ? '我' : '猫管家' }}</text><text v-if="message.degraded" class="ai-badge">降级回复</text></view>
          <view class="agent-card-body">{{ message.body }}</view>
          <view v-if="message.type === 'reminder_draft'" class="page-subtitle">{{ message.action_status || '草稿' }} · {{ message.draft_reminder?.title }}</view>
          <button v-if="message.role === 'assistant'" class="reminder-btn" @click="openMessage(message.id)">查看证据{{ message.type === 'reminder_draft' ? '与提醒草稿' : '与操作' }}</button>
        </view>
        <view v-if="showPendingInput" class="agent-card chat-user"><view class="agent-card-body">{{ store.pendingChat?.message }}</view></view>
        <view v-if="store.chatSending" class="page-subtitle">猫管家正在查询，请稍候…</view>
        <view v-if="store.chatError" class="empty-state-desc" role="alert">{{ store.chatError }}</view>
        <button v-if="store.pendingChat && !store.chatSending" class="reminder-btn primary" @click="store.retryChat()">恢复结果 / 重试本轮</button>
        <view class="reminder-actions"><button v-for="question in recommended" :key="question" class="reminder-btn" :disabled="store.chatSending || !!store.pendingChat" @click="input = question">{{ question }}</button></view>
        <textarea v-model="input" maxlength="2000" placeholder="选择猫咪并描述问题；有歧义时猫管家会进一步确认" :disabled="store.chatSending || !!store.pendingChat" />
        <button class="reminder-btn primary" :disabled="!input.trim() || store.chatSending || !!store.pendingChat" @click="send">发送</button>
        <view class="page-subtitle">回答以已记录数据为依据，不提供诊断或药量。没有记录不代表没有发生。</view>
      </view>
      <view v-else>
      <view v-if="store.loading || detailLoading" class="empty-state">正在读取巡检消息…</view>
      <view v-if="store.error" class="empty-state"><view class="empty-state-title">暂时无法读取</view><view class="empty-state-desc">{{ store.error }}</view><button class="reminder-btn primary" @click="store.load()">重试</button></view>
      <view v-else-if="messages.length === 0" class="empty-state"><view class="empty-state-title">暂无新的巡检提示</view><view class="empty-state-desc">有新的记录后，猫管家会在这里展示可核对的提醒。</view></view>
      <view v-else class="agent-list">
        <button v-for="item in messages" :key="item.id" class="agent-card" :class="`agent-${item.severity}`" @click="openMessage(item.id)">
          <view class="agent-card-header"><text class="agent-card-title">{{ item.title }}</text><text class="ai-badge">{{ item.severity }}</text></view>
          <view class="agent-card-body">{{ item.body }}</view>
          <view class="page-subtitle">查看证据与操作</view>
        </button>
      </view>
      <button v-if="store.nextCursor" class="reminder-btn" @click="store.load({ before: store.nextCursor })">加载更早消息</button>
      </view>
    </view>
    <view v-if="error" class="empty-state-desc" role="alert">{{ error }}</view>
  </view>
</template>

<style scoped>
.agent-card { width: 100%; text-align: left; }
.agent-chat { display: flex; flex-direction: column; gap: 12px; }
.agent-chat textarea { width: 100%; box-sizing: border-box; background: var(--color-bg-surface); border: 1px solid var(--color-divider); border-radius: 8px; padding: 12px; }
.chat-user { background: var(--color-bg-surface); }
.agent-detail, .agent-draft { display: flex; flex-direction: column; gap: 12px; }
.agent-evidence-row { display: flex; align-items: center; justify-content: space-between; gap: 8px; padding: 8px 0; }
.agent-draft input { background: var(--color-bg-surface); border: 1px solid var(--color-divider); border-radius: 8px; padding: 12px; }
</style>
