<script setup lang="ts">
import { ref, reactive } from 'vue'
import { useRouter } from 'vue-router'
import AppIcon from '../../components/app/AppIcon.vue'
import { greeting } from '../../utils/date'

const router = useRouter()
const step = ref(1)
const totalSteps = 5

const family = reactive({ name: '小家的猫宅' })
const cat1 = reactive({ name: '小白', gender: 'female' as const, breed: '中华田园猫', birthday: '', neutered: true, diseases: '', allergies: '' })
const cat2 = reactive({ name: '小橘', gender: 'male' as const, breed: '中华田园猫', birthday: '', neutered: true, diseases: '', allergies: '' })

const hasSecondCat = ref(false)

function next() { step.value = Math.min(step.value + 1, totalSteps) }
function prev() { step.value = Math.max(step.value - 1, 1) }

function finish() {
  router.push('/today')
}

const stepTitles = ['欢迎与创建家庭', '添加第一只猫', '是否添加第二只猫', '健康基础信息', '完成']
</script>

<template>
  <div class="page">
    <!-- Progress -->
    <div
      class="onboarding-progress"
      role="progressbar"
      :aria-valuenow="step"
      :aria-valuemin="1"
      :aria-valuemax="totalSteps"
    >
      <div class="progress-steps">
        <div
          v-for="i in totalSteps"
          :key="i"
          class="progress-step"
          :class="{ active: i <= step, done: i < step }"
        >
          <div class="step-num">
            {{ i < step ? '✓' : i }}
          </div>
          <div class="step-label">
            {{ stepTitles[i - 1] }}
          </div>
        </div>
      </div>
      <div class="progress-bar">
        <div
          class="progress-fill"
          :style="{ width: (step / totalSteps * 100) + '%' }"
        />
      </div>
    </div>

    <!-- Step 1 -->
    <div
      v-if="step === 1"
      class="onboarding-step"
    >
      <h2>欢迎来到猫宅</h2>
      <p>为你的多猫家庭建立专属的生活与健康档案。</p>
      <div class="form-group">
        <label
          class="form-label"
          for="fam-name"
        >家庭名称</label>
        <input
          id="fam-name"
          v-model="family.name"
          class="form-input"
          placeholder="如：小家的猫宅"
        />
      </div>
      <button
        class="btn-primary"
        @click="next"
      >
        下一步
      </button>
    </div>

    <!-- Step 2 -->
    <div
      v-if="step === 2"
      class="onboarding-step"
    >
      <h2>添加第一只猫</h2>
      <div class="form-group">
        <label
          class="form-label"
          for="c1-name"
        >猫咪名称 *</label>
        <input
          id="c1-name"
          v-model="cat1.name"
          class="form-input"
          placeholder="如：小白"
          required
        />
      </div>
      <div class="form-row">
        <div class="form-group">
          <label
            class="form-label"
            for="c1-gender"
          >性别</label>
          <select
            id="c1-gender"
            v-model="cat1.gender"
            class="form-select"
          >
            <option value="female">
              母
            </option><option value="male">
              公
            </option>
          </select>
        </div>
        <div class="form-group">
          <label
            class="form-label"
            for="c1-breed"
          >品种</label>
          <input
            id="c1-breed"
            v-model="cat1.breed"
            class="form-input"
            placeholder="中华田园猫"
          />
        </div>
      </div>
      <div class="form-group">
        <label
          class="form-label"
          for="c1-birthday"
        >生日</label>
        <input
          id="c1-birthday"
          v-model="cat1.birthday"
          type="date"
          class="form-input"
        />
      </div>
      <div class="form-group">
        <label
          class="form-label"
          for="c1-neutered"
        >
          <input
            id="c1-neutered"
            v-model="cat1.neutered"
            type="checkbox"
          /> 已绝育
        </label>
      </div>
      <div class="form-actions-onboarding">
        <button
          class="btn-secondary"
          @click="prev"
        >
          上一步
        </button>
        <button
          class="btn-primary"
          @click="next"
        >
          下一步
        </button>
      </div>
    </div>

    <!-- Step 3 -->
    <div
      v-if="step === 3"
      class="onboarding-step"
    >
      <h2>是否添加第二只猫？</h2>
      <p>多猫家庭可以更好地追踪每只猫的健康数据。</p>
      <div
        class="form-actions-onboarding"
        style="flex-direction:column;gap:12px;"
      >
        <button
          class="btn-primary"
          @click="hasSecondCat = true; next()"
        >
          添加第二只猫
        </button>
        <button
          class="btn-secondary"
          @click="next()"
        >
          暂时只有一只
        </button>
      </div>
    </div>

    <!-- Step 4 -->
    <div
      v-if="step === 4"
      class="onboarding-step"
    >
      <h2>健康基础信息</h2>
      <p>填写已知疾病、过敏和禁忌，帮助 AI 管家提供更好的建议。</p>
      <div class="form-group">
        <label
          class="form-label"
          for="c1-diseases"
        >已知疾病（逗号分隔）</label>
        <input
          id="c1-diseases"
          v-model="cat1.diseases"
          class="form-input"
          placeholder="如：慢性肾病"
        />
      </div>
      <div class="form-group">
        <label
          class="form-label"
          for="c1-allergies"
        >过敏</label>
        <input
          id="c1-allergies"
          v-model="cat1.allergies"
          class="form-input"
          placeholder="如：鸡肉"
        />
      </div>
      <div v-if="hasSecondCat">
        <h3 style="margin-top:16px;">
          第二只猫
        </h3>
        <div class="form-group">
          <label
            class="form-label"
            for="c2-name"
          >猫咪名称 *</label>
          <input
            id="c2-name"
            v-model="cat2.name"
            class="form-input"
            placeholder="如：小橘"
          />
        </div>
        <div class="form-row">
          <div class="form-group">
            <label
              class="form-label"
              for="c2-gender"
            >性别</label>
            <select
              id="c2-gender"
              v-model="cat2.gender"
              class="form-select"
            >
              <option value="female">
                母
              </option><option value="male">
                公
              </option>
            </select>
          </div>
          <div class="form-group">
            <label
              class="form-label"
              for="c2-breed"
            >品种</label>
            <input
              id="c2-breed"
              v-model="cat2.breed"
              class="form-input"
              placeholder="中华田园猫"
            />
          </div>
        </div>
      </div>
      <div class="form-actions-onboarding">
        <button
          class="btn-secondary"
          @click="prev"
        >
          上一步
        </button>
        <button
          class="btn-primary"
          @click="finish"
        >
          完成
        </button>
      </div>
    </div>

    <!-- Step 5 -->
    <div
      v-if="step === 5"
      class="onboarding-step"
    >
      <div class="finish-icon">
        <AppIcon
          name="success"
          :size="48"
        />
      </div>
      <h2>设置完成</h2>
      <p>{{ family.name }} · {{ cat1.name }}</p>
      <p
        v-if="hasSecondCat"
        style="color:var(--color-text-secondary);"
      >
        + {{ cat2.name }}
      </p>
      <p style="color:var(--color-text-tertiary);font-size:var(--font-size-assist);">
        欢迎使用猫宅 · 开启你的猫咪健康管理之旅
      </p>
      <button
        class="btn-primary"
        @click="finish"
      >
        进入今日首页
      </button>
    </div>
  </div>
</template>

<style scoped>
.page { min-height: 100dvh; padding: var(--space-24) var(--space-16); padding-bottom: calc(64px + var(--safe-bottom)); display: flex; flex-direction: column; }
.onboarding-progress { margin-bottom: var(--space-24); }
.progress-steps { display: flex; justify-content: space-between; margin-bottom: var(--space-8); }
.progress-step { display: flex; flex-direction: column; align-items: center; gap: 4px; flex: 1; }
.step-num { width: 28px; height: 28px; border-radius: 50%; background: var(--color-bg-subtle); display: flex; align-items: center; justify-content: center; font-size: var(--font-size-caption); font-weight: 600; color: var(--color-text-tertiary); }
.progress-step.done .step-num { background: var(--color-success); color: white; }
.progress-step.active .step-num { background: var(--color-brand); color: white; }
.step-label { font-size: 10px; color: var(--color-text-tertiary); text-align: center; }
.progress-bar { height: 4px; background: var(--color-divider); border-radius: 2px; }
.progress-fill { height: 100%; background: var(--color-brand); border-radius: 2px; transition: width 0.3s ease; }
.onboarding-step { flex: 1; display: flex; flex-direction: column; }
.onboarding-step h2 { font-size: var(--font-size-title); margin-bottom: var(--space-16); }
.onboarding-step p { color: var(--color-text-secondary); margin-bottom: var(--space-16); line-height: 1.6; }
.form-actions-onboarding { display: flex; gap: var(--space-12); margin-top: auto; padding-top: var(--space-24); }
.form-actions-onboarding .btn-primary { flex: 2; }
.form-actions-onboarding .btn-secondary { flex: 1; }
.finish-icon { text-align: center; margin-bottom: var(--space-16); }
</style>
