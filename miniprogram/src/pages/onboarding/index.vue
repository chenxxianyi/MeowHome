<script setup lang="ts">
import { ref, reactive } from 'vue'
import { useRouter } from '../../utils/navigation'
import AppIcon from '../../components/app/AppIcon.vue'
import { catApi, toCat } from '../../api/endpoints'
import { toApiError } from '../../api/adapter'
import { useAuthStore } from '../../stores/auth'
import { useCatStore } from '../../stores/cat'
import { greeting } from '../../utils/date'
import { useProtectedPage } from '../../utils/page'

const router = useRouter()
const auth = useAuthStore()
const catStore = useCatStore()
const step = ref(1)
const totalSteps = 5

const family = reactive({ name: '小家的猫宅' })
type CatGender = 'female' | 'male' | 'unknown'
const cat1 = reactive({
  name: '小白',
  gender: 'female' as CatGender,
  breed: '中华田园猫',
  birthday: '',
  neutered: true,
  diseases: '',
  allergies: ''
})
const cat2 = reactive({
  name: '小橘',
  gender: 'male' as CatGender,
  breed: '中华田园猫',
  birthday: '',
  neutered: true,
  diseases: '',
  allergies: ''
})
const genderLabels = ['母', '公', '未知']
const genderValues: CatGender[] = ['female', 'male', 'unknown']

const hasSecondCat = ref(false)
const submitting = ref(false)
const error = ref('')
const saved = ref(false)

function next() {
  error.value = ''
  if (step.value === 1 && !family.name.trim()) {
    error.value = '请填写家庭名称'
    uni.showToast({ title: error.value, icon: 'none' })
    return
  }
  if (step.value === 2 && !cat1.name.trim()) {
    error.value = '请填写第一只猫的名字'
    uni.showToast({ title: error.value, icon: 'none' })
    return
  }
  step.value = Math.min(step.value + 1, totalSteps)
}
function prev() {
  step.value = Math.max(step.value - 1, 1)
}

// 引导流程必须真正落库：家庭与猫咪都要创建，
// 否则路由守卫会因为 familyId 为空把用户反复弹回本页。
async function finish() {
  if (submitting.value || saved.value) return
  if (!cat1.name.trim()) {
    error.value = '至少需要添加一只猫咪'
    uni.showToast({ title: error.value, icon: 'none' })
    return
  }
  submitting.value = true
  error.value = ''
  try {
    const familyId = auth.familyId || (await auth.createFamily(family.name || '我的猫宅'))

    const drafts = [cat1, ...(hasSecondCat.value ? [cat2] : [])].filter((c) => c.name.trim())
    const created = []
    for (const d of drafts) {
      created.push(
        await catApi.create(familyId, {
          name: d.name.trim(),
          breed: d.breed,
          gender: d.gender,
          birth_date: d.birthday || undefined,
          neutered: d.neutered,
          diseases: splitHealthItems(d.diseases),
          allergies: splitHealthItems(d.allergies)
        })
      )
    }
    if (created.length) {
      catStore.setCats(created.map(toCat))
    }
    saved.value = true
    step.value = 5
  } catch (e) {
    error.value = toApiError(e).message
  } finally {
    submitting.value = false
  }
}

function selectGender(target: typeof cat1, event: unknown) {
  const value = (event as { detail: { value: string } }).detail.value
  target.gender = genderValues[Number(value)] || 'unknown'
}

function selectBirthday(event: unknown) {
  cat1.birthday = (event as { detail: { value: string } }).detail.value
}

function selectNeutered(event: unknown) {
  cat1.neutered = (event as { detail: { value: boolean } }).detail.value
}

function addSecondCat() {
  hasSecondCat.value = true
  next()
}

function splitHealthItems(value: string) {
  return value
    .split(/[，,]/)
    .map((item) => item.trim())
    .filter(Boolean)
}

function enterHome() {
  router.push('/today')
}

const stepTitles = ['欢迎与创建家庭', '添加第一只猫', '是否添加第二只猫', '健康基础信息', '完成']

useProtectedPage()
</script>

<template>
  <view class="page">
    <!-- Progress -->
    <view
      class="onboarding-progress"
      role="progressbar"
      :aria-valuenow="step"
      :aria-valuemin="1"
      :aria-valuemax="totalSteps"
    >
      <view class="progress-steps">
        <view v-for="i in totalSteps" :key="i" class="progress-step" :class="{ active: i <= step, done: i < step }">
          <view class="step-num">
            {{ i < step ? '✓' : i }}
          </view>
          <view class="step-label">
            {{ stepTitles[i - 1] }}
          </view>
        </view>
      </view>
      <view class="progress-bar">
        <view class="progress-fill" :style="{ width: (step / totalSteps) * 100 + '%' }" />
      </view>
    </view>

    <!-- Step 1 -->
    <view v-if="step === 1" class="onboarding-step">
      <view>欢迎来到猫宅</view>
      <view>为你的多猫家庭建立专属的生活与健康档案。</view>
      <view class="form-group">
        <label class="form-label" for="fam-name">家庭名称</label>
        <input id="fam-name" v-model="family.name" class="form-input" placeholder="如：小家的猫宅" />
      </view>
      <button class="btn-primary" @click="next">下一步</button>
      <view v-if="error" class="form-error">{{ error }}</view>
    </view>

    <!-- Step 2 -->
    <view v-if="step === 2" class="onboarding-step">
      <view>添加第一只猫</view>
      <view class="form-group">
        <label class="form-label" for="c1-name">猫咪名称 *</label>
        <input id="c1-name" v-model="cat1.name" class="form-input" placeholder="如：小白" required />
      </view>
      <view class="form-row">
        <view class="form-group">
          <label class="form-label" for="c1-gender">性别</label>
          <picker
            mode="selector"
            :range="genderLabels"
            :value="genderValues.indexOf(cat1.gender)"
            @change="selectGender(cat1, $event)"
          >
            <view class="form-select">{{ genderLabels[genderValues.indexOf(cat1.gender)] }}</view>
          </picker>
        </view>
        <view class="form-group">
          <label class="form-label" for="c1-breed">品种</label>
          <input id="c1-breed" v-model="cat1.breed" class="form-input" placeholder="中华田园猫" />
        </view>
      </view>
      <view class="form-group">
        <label class="form-label" for="c1-birthday">生日</label>
        <picker mode="date" :value="cat1.birthday" @change="selectBirthday">
          <view class="form-input">{{ cat1.birthday || '请选择生日' }}</view>
        </picker>
      </view>
      <view class="form-group">
        <label class="form-label" for="c1-neutered">
          <switch :checked="cat1.neutered" color="#C87345" @change="selectNeutered" /> 已绝育
        </label>
      </view>
      <view class="form-actions-onboarding">
        <button class="btn-secondary" @click="prev">上一步</button>
        <button class="btn-primary" @click="next">下一步</button>
      </view>
      <view v-if="error" class="form-error">{{ error }}</view>
    </view>

    <!-- Step 3 -->
    <view v-if="step === 3" class="onboarding-step">
      <view>是否添加第二只猫？</view>
      <view>多猫家庭可以更好地追踪每只猫的健康数据。</view>
      <view class="form-actions-onboarding" style="flex-direction: column; gap: 12px">
        <button class="btn-primary" @click="addSecondCat">添加第二只猫</button>
        <button class="btn-secondary" @click="next()">暂时只有一只</button>
      </view>
    </view>

    <!-- Step 4 -->
    <view v-if="step === 4" class="onboarding-step">
      <view>健康基础信息</view>
      <view>填写已知疾病与过敏，帮助 AI 管家提供更好的建议。</view>
      <view class="form-group">
        <label class="form-label" for="c1-diseases">已知疾病（逗号分隔）</label>
        <input id="c1-diseases" v-model="cat1.diseases" class="form-input" placeholder="如：慢性肾病" />
      </view>
      <view class="form-group">
        <label class="form-label" for="c1-allergies">过敏</label>
        <input id="c1-allergies" v-model="cat1.allergies" class="form-input" placeholder="如：鸡肉" />
      </view>
      <view v-if="hasSecondCat">
        <view style="margin-top: 16px"> 第二只猫 </view>
        <view class="form-group">
          <label class="form-label" for="c2-name">猫咪名称 *</label>
          <input id="c2-name" v-model="cat2.name" class="form-input" placeholder="如：小橘" />
        </view>
        <view class="form-row">
          <view class="form-group">
            <label class="form-label" for="c2-gender">性别</label>
            <picker
              mode="selector"
              :range="genderLabels"
              :value="genderValues.indexOf(cat2.gender)"
              @change="selectGender(cat2, $event)"
            >
              <view class="form-select">{{ genderLabels[genderValues.indexOf(cat2.gender)] }}</view>
            </picker>
          </view>
          <view class="form-group">
            <label class="form-label" for="c2-breed">品种</label>
            <input id="c2-breed" v-model="cat2.breed" class="form-input" placeholder="中华田园猫" />
          </view>
        </view>
      </view>
      <view class="form-actions-onboarding">
        <button class="btn-secondary" @click="prev">上一步</button>
        <button class="btn-primary" :disabled="submitting" @click="finish">
          {{ submitting ? '保存中…' : '完成' }}
        </button>
      </view>
    </view>

    <!-- Step 5 -->
    <view v-if="step === 5" class="onboarding-step">
      <view class="finish-icon">
        <AppIcon name="success" :size="48" />
      </view>
      <view>设置完成</view>
      <view>{{ family.name }} · {{ cat1.name }}</view>
      <view v-if="hasSecondCat" style="color: var(--color-text-secondary)"> + {{ cat2.name }} </view>
      <view style="color: var(--color-text-tertiary); font-size: var(--font-size-assist)">
        欢迎使用猫宅 · 开启你的猫咪健康管理之旅
      </view>
      <button class="btn-primary" :disabled="submitting" @click="enterHome">
        {{ submitting ? '保存中…' : '进入今日首页' }}
      </button>
      <view v-if="error" style="color: #b4453a; font-size: var(--font-size-assist); margin-top: var(--space-12)">
        {{ error }}
      </view>
    </view>
  </view>
</template>

<style scoped>
.page {
  min-height: 100vh;
  padding: var(--space-24) var(--space-16);
  padding-bottom: var(--safe-bottom);
  display: flex;
  flex-direction: column;
}
.onboarding-progress {
  margin-bottom: var(--space-24);
}
.progress-steps {
  display: flex;
  justify-content: space-between;
  margin-bottom: var(--space-8);
}
.progress-step {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 4px;
  flex: 1;
}
.step-num {
  width: 28px;
  height: 28px;
  border-radius: 50%;
  background: var(--color-bg-subtle);
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: var(--font-size-caption);
  font-weight: 600;
  color: var(--color-text-tertiary);
}
.progress-step.done .step-num {
  background: var(--color-success);
  color: white;
}
.progress-step.active .step-num {
  background: var(--color-brand);
  color: white;
}
.step-label {
  font-size: 10px;
  color: var(--color-text-tertiary);
  text-align: center;
}
.progress-bar {
  height: 4px;
  background: var(--color-divider);
  border-radius: 2px;
}
.progress-fill {
  height: 100%;
  background: var(--color-brand);
  border-radius: 2px;
  transition: width 0.3s ease;
}
.onboarding-step {
  flex: 1;
  display: flex;
  flex-direction: column;
}
.onboarding-step > view:first-child {
  font-size: var(--font-size-title);
  font-weight: 600;
  margin-bottom: var(--space-16);
}
.onboarding-step > view:nth-child(2) {
  color: var(--color-text-secondary);
  margin-bottom: var(--space-16);
  line-height: 1.6;
}
.form-actions-onboarding {
  display: flex;
  gap: var(--space-12);
  margin-top: auto;
  padding-top: var(--space-24);
}
.form-actions-onboarding .btn-primary {
  flex: 2;
}
.form-actions-onboarding .btn-secondary {
  flex: 1;
}
.finish-icon {
  text-align: center;
  margin-bottom: var(--space-16);
}
</style>
