<script setup lang="ts">
import { computed, ref, reactive } from 'vue'
import { onLoad } from '@dcloudio/uni-app'
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

const family = reactive({ name: '' })
type CatGender = 'female' | 'male' | 'unknown'
const cat1 = reactive({
  name: '',
  gender: 'unknown' as CatGender,
  breed: '',
  birthday: '',
  neutered: false,
  diseases: '',
  allergies: ''
})
const cat2 = reactive({
  name: '',
  gender: 'unknown' as CatGender,
  breed: '',
  birthday: '',
  neutered: false,
  diseases: '',
  allergies: ''
})
const genderLabels = ['母', '公', '未知']
const genderValues: CatGender[] = ['female', 'male', 'unknown']

const hasSecondCat = ref(false)
const stepTitles = computed(() =>
  hasSecondCat.value
    ? ['欢迎与创建家庭', '添加第一只猫', '是否添加第二只猫', '添加第二只猫', '健康基础信息', '完成']
    : ['欢迎与创建家庭', '添加第一只猫', '是否添加第二只猫', '健康基础信息', '完成']
)
const totalSteps = computed(() => stepTitles.value.length)
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
  if (step.value === 4 && hasSecondCat.value && !cat2.name.trim()) {
    error.value = '请填写第二只猫的名字'
    uni.showToast({ title: error.value, icon: 'none' })
    return
  }
  step.value = Math.min(step.value + 1, totalSteps.value)
}
function prev() {
  step.value = Math.max(step.value - 1, 1)
}

// 引导流程必须真正落库：家庭与猫咪都要创建，
// 否则路由守卫会因为 familyId 为空把用户反复弹回本页。
async function finish() {
  if (submitting.value || saved.value) return
  if (!cat1.name.trim() || (hasSecondCat.value && !cat2.name.trim())) {
    error.value = cat1.name.trim() ? '请填写第二只猫的名字' : '至少需要添加一只猫咪'
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
    step.value = totalSteps.value
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

function selectBirthday(target: typeof cat1, event: unknown) {
  target.birthday = (event as { detail: { value: string } }).detail.value
}

function selectNeutered(target: typeof cat1, event: unknown) {
  target.neutered = (event as { detail: { value: boolean } }).detail.value
}

function addSecondCat() {
  hasSecondCat.value = true
  error.value = ''
  step.value = 4
}

function skipSecondCat() {
  hasSecondCat.value = false
  error.value = ''
  step.value = 4
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

onLoad((options) => {
  if (options && typeof options.familyName === 'string') {
    try {
      family.name = decodeURIComponent(options.familyName)
    } catch {
      family.name = options.familyName
    }
  }
})

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
        <input id="c1-name" v-model="cat1.name" class="form-input" placeholder="如：猫咪名字" required />
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
        <picker mode="date" :value="cat1.birthday" @change="selectBirthday(cat1, $event)">
          <view class="form-input">{{ cat1.birthday || '请选择生日' }}</view>
        </picker>
      </view>
      <view class="form-group">
        <label class="form-label" for="c1-neutered">
          <switch :checked="cat1.neutered" color="#C87345" @change="selectNeutered(cat1, $event)" /> 已绝育
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
        <button class="btn-secondary" @click="skipSecondCat">暂时只有一只</button>
      </view>
    </view>

    <!-- Step 4: second cat details (only when requested) -->
    <view v-if="step === 4 && hasSecondCat" class="onboarding-step">
      <view>添加第二只猫</view>
      <view>填写第二只猫的基本资料，健康信息可以稍后补充。</view>
      <view class="form-group">
        <label class="form-label" for="c2-name">猫咪名称 *</label>
        <input id="c2-name" v-model="cat2.name" class="form-input" placeholder="如：第二只猫咪" required />
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
      <view class="form-group">
        <label class="form-label" for="c2-birthday">生日</label>
        <picker mode="date" :value="cat2.birthday" @change="selectBirthday(cat2, $event)">
          <view class="form-input">{{ cat2.birthday || '请选择生日' }}</view>
        </picker>
      </view>
      <view class="form-group">
        <label class="form-label" for="c2-neutered">
          <switch :checked="cat2.neutered" color="#C87345" @change="selectNeutered(cat2, $event)" /> 已绝育
        </label>
      </view>
      <view class="form-actions-onboarding">
        <button class="btn-secondary" @click="prev">上一步</button>
        <button class="btn-primary" @click="next">下一步</button>
      </view>
      <view v-if="error" class="form-error">{{ error }}</view>
    </view>

    <!-- Health basics -->
    <view v-if="(step === 4 && !hasSecondCat) || (step === 5 && hasSecondCat)" class="onboarding-step">
      <view>健康基础信息</view>
      <view>填写已知疾病与过敏，帮助 AI 管家提供更好的建议。</view>
      <view class="cat-health-section">
        <view class="cat-health-heading">第一只猫 · {{ cat1.name }}</view>
        <view class="cat-profile-summary">
          {{ genderLabels[genderValues.indexOf(cat1.gender)] }} · {{ cat1.breed || '品种未填写' }} ·
          {{ cat1.birthday || '生日未填写' }} · {{ cat1.neutered ? '已绝育' : '未绝育' }}
        </view>
        <view class="form-group">
          <label class="form-label" for="c1-diseases">已知疾病（逗号分隔）</label>
          <input id="c1-diseases" v-model="cat1.diseases" class="form-input" placeholder="如：慢性肾病" />
        </view>
        <view class="form-group">
          <label class="form-label" for="c1-allergies">过敏</label>
          <input id="c1-allergies" v-model="cat1.allergies" class="form-input" placeholder="如：鸡肉" />
        </view>
      </view>
      <view v-if="hasSecondCat" class="cat-health-section">
        <view class="cat-health-heading">第二只猫 · {{ cat2.name }}</view>
        <view class="cat-profile-summary">
          {{ genderLabels[genderValues.indexOf(cat2.gender)] }} · {{ cat2.breed || '品种未填写' }} ·
          {{ cat2.birthday || '生日未填写' }} · {{ cat2.neutered ? '已绝育' : '未绝育' }}
        </view>
        <view class="form-group">
          <label class="form-label" for="c2-diseases">已知疾病（逗号分隔）</label>
          <input id="c2-diseases" v-model="cat2.diseases" class="form-input" placeholder="如：慢性肾病" />
        </view>
        <view class="form-group">
          <label class="form-label" for="c2-allergies">过敏</label>
          <input id="c2-allergies" v-model="cat2.allergies" class="form-input" placeholder="如：鸡肉" />
        </view>
      </view>
      <view class="form-actions-onboarding">
        <button class="btn-secondary" @click="prev">上一步</button>
        <button class="btn-primary" :disabled="submitting" @click="finish">
          {{ submitting ? '保存中…' : '完成' }}
        </button>
      </view>
      <view v-if="error" class="form-error">{{ error }}</view>
    </view>

    <!-- Final step -->
    <view v-if="step === totalSteps" class="onboarding-step">
      <view class="finish-icon">
        <AppIcon name="success" :size="48" />
      </view>
      <view class="finish-title">猫宅准备就绪</view>
      <view class="finish-family">家庭：{{ family.name }}</view>
      <view class="finish-cats">
        猫咪：{{ cat1.name }}<text v-if="hasSecondCat">、{{ cat2.name }}</text>
      </view>
      <view class="finish-description">
        喂食、护理和健康变化，都可以在这里持续记录。
      </view>
      <button class="btn-primary" :disabled="submitting" @click="enterHome">
        {{ submitting ? '正在进入…' : '开始使用' }}
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
.cat-health-section + .cat-health-section {
  margin-top: var(--space-8);
  padding-top: var(--space-12);
  border-top: 1px solid var(--color-divider);
}
.cat-health-heading {
  margin-bottom: 4px;
  font-size: var(--font-size-body);
  font-weight: 600;
  color: var(--color-text-primary);
}
.cat-profile-summary {
  margin-bottom: var(--space-12);
  font-size: var(--font-size-caption);
  line-height: 1.5;
  color: var(--color-text-tertiary);
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
.onboarding-step > view.finish-title {
  margin-bottom: var(--space-12);
  font-size: var(--font-size-title);
  font-weight: 600;
  color: var(--color-text-primary);
  text-align: center;
}
.finish-family,
.finish-cats {
  margin-bottom: var(--space-8);
  color: var(--color-text-secondary);
}
.finish-description {
  margin: var(--space-8) 0 var(--space-16);
  color: var(--color-text-tertiary);
  font-size: var(--font-size-assist);
  line-height: 1.6;
}
</style>
