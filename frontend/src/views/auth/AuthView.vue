<script setup lang="ts">
// 登录 / 注册页。两个模式共用一个表单，避免重复维护样式。
import { computed, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'

import { toApiError } from '../../api/adapter'
import AppIcon from '../../components/app/AppIcon.vue'
import { useAuthStore } from '../../stores/auth'

const router = useRouter()
const route = useRoute()
const auth = useAuthStore()

const mode = ref<'login' | 'register'>('login')
const email = ref('')
const password = ref('')
const confirmPassword = ref('')
const showPassword = ref(false)
const userName = ref('')
const familyName = ref('')
const error = ref('')
const submitting = ref(false)

const isRegister = computed(() => mode.value === 'register')
const title = computed(() => (isRegister.value ? '创建账号' : '欢迎回来'))
const submitLabel = computed(() => (isRegister.value ? '注册并进入' : '登录'))

/** 仅当用户已开始输入确认密码、且两次不一致时才提示，避免刚进页面就报错。 */
const passwordMismatch = computed(
  () => isRegister.value && confirmPassword.value.length > 0 && confirmPassword.value !== password.value
)

/** 密码是否达到后端要求的最短长度（后端 Register 校验 >= 8）。 */
const passwordTooShort = computed(
  () => isRegister.value && password.value.length > 0 && password.value.length < 8
)

/**
 * 提交按钮可用性。
 * 登录模式只要求邮箱与密码非空；注册模式额外要求昵称、密码长度与两次一致。
 */
const canSubmit = computed(() => {
  if (!email.value || !password.value) return false
  if (!isRegister.value) return true
  if (!userName.value) return false
  if (password.value.length < 8) return false
  return confirmPassword.value.length > 0 && confirmPassword.value === password.value
})

function switchMode(next: 'login' | 'register') {
  mode.value = next
  error.value = ''
  // 切到登录时清掉仅注册相关的输入，避免把上一轮的密码带过去
  if (next === 'login') {
    confirmPassword.value = ''
    showPassword.value = false
  }
}

async function onSubmit() {
  error.value = ''
  if (!email.value || !password.value) {
    error.value = '请填写邮箱与密码'
    return
  }
  if (isRegister.value) {
    if (!userName.value) {
      error.value = '请填写昵称'
      return
    }
    if (password.value.length < 8) {
      error.value = '密码至少 8 位'
      return
    }
    if (!confirmPassword.value) {
      error.value = '请再次输入密码'
      return
    }
    if (confirmPassword.value !== password.value) {
      error.value = '两次输入的密码不一致'
      confirmPassword.value = ''
      return
    }
  }

  submitting.value = true
  try {
    const familyId = isRegister.value
      ? await auth.register(email.value, password.value, userName.value, familyName.value || `${userName.value}的猫宅`)
      : await auth.login(email.value, password.value)

    const redirect = (route.query.redirect as string) || '/today'
    await router.replace(familyId ? redirect : '/onboarding')
  } catch (e) {
    const apiError = toApiError(e)
    if (apiError.code === 'CONFLICT') {
      error.value = '该邮箱已注册，请直接登录'
      mode.value = 'login'
    } else if (apiError.code === 'AUTH_REQUIRED') {
      error.value = '邮箱或密码不正确'
    } else {
      error.value = apiError.message
    }
  } finally {
    submitting.value = false
  }
}
</script>

<template>
  <div class="auth-page">
    <div class="auth-card">
      <div class="brand">
        <div class="logo">🐾</div>
        <h1>猫宅</h1>
        <p class="subtitle">多猫家庭的生活与健康管理</p>
      </div>

      <h2>{{ title }}</h2>

      <form @submit.prevent="onSubmit">
        <label class="field">
          <span>邮箱</span>
          <input v-model.trim="email" type="email" autocomplete="email" placeholder="you@example.com" />
        </label>

        <label v-if="isRegister" class="field">
          <span>昵称</span>
          <input v-model.trim="userName" type="text" autocomplete="nickname" placeholder="家里怎么称呼你" />
        </label>

        <label class="field">
          <span>密码</span>
          <span class="input-wrap">
            <input
              v-model="password"
              :type="showPassword ? 'text' : 'password'"
              :autocomplete="isRegister ? 'new-password' : 'current-password'"
              :placeholder="isRegister ? '至少 8 位' : '请输入密码'"
              :aria-invalid="passwordTooShort"
              :aria-describedby="passwordTooShort ? 'password-hint' : undefined"
            />
            <button
              type="button"
              class="toggle"
              :aria-label="showPassword ? '隐藏密码' : '显示密码'"
              :aria-pressed="showPassword"
              @click="showPassword = !showPassword"
            >
              <AppIcon
                :name="showPassword ? 'eyeOff' : 'eye'"
                :size="18"
              />
            </button>
          </span>
        </label>
        <p
          v-if="passwordTooShort"
          id="password-hint"
          class="hint hint-error"
        >
          密码至少 8 位
        </p>

        <label
          v-if="isRegister"
          class="field"
        >
          <span>确认密码</span>
          <input
            v-model="confirmPassword"
            :type="showPassword ? 'text' : 'password'"
            autocomplete="new-password"
            placeholder="请再次输入密码"
            :aria-invalid="passwordMismatch"
            :aria-describedby="passwordMismatch ? 'confirm-hint' : undefined"
          />
        </label>
        <p
          v-if="passwordMismatch"
          id="confirm-hint"
          class="hint hint-error"
        >
          两次输入的密码不一致
        </p>

        <label v-if="isRegister" class="field">
          <span>家庭名称</span>
          <input v-model.trim="familyName" type="text" placeholder="例如：小家的猫宅" />
        </label>

        <p v-if="error" class="error">{{ error }}</p>

        <button
          type="submit"
          class="primary"
          :disabled="submitting || !canSubmit"
        >
          {{ submitting ? '处理中…' : submitLabel }}
        </button>
      </form>

      <p class="switch">
        <template v-if="isRegister">
          已有账号？<button type="button" class="link" @click="switchMode('login')">去登录</button>
        </template>
        <template v-else>
          还没有账号？<button type="button" class="link" @click="switchMode('register')">注册一个</button>
        </template>
      </p>
    </div>
  </div>
</template>

<style scoped>
.auth-page {
  min-height: 100vh;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 24px;
  background: linear-gradient(160deg, #fffcf7 0%, #f7f4ee 100%);
}

.auth-card {
  width: 100%;
  max-width: 380px;
  background: #fff;
  border-radius: 20px;
  padding: 32px 24px 24px;
  box-shadow: 0 12px 40px rgba(120, 100, 70, 0.12);
}

.brand {
  text-align: center;
  margin-bottom: 20px;
}

.logo {
  font-size: 40px;
  line-height: 1;
}

.brand h1 {
  margin: 8px 0 4px;
  font-size: 24px;
  color: #3d3426;
}

.subtitle {
  margin: 0;
  font-size: 13px;
  color: #9a8f7d;
}

h2 {
  margin: 0 0 16px;
  font-size: 17px;
  font-weight: 600;
  color: #3d3426;
}

.field {
  display: block;
  margin-bottom: 14px;
}

.field span:not(.input-wrap) {
  display: block;
  font-size: 13px;
  color: #7a6f5d;
  margin-bottom: 6px;
}

/* 输入框容器：仅作为定位上下文，让图标叠在输入框「内部」而不占布局宽度。
   此前用 flex 并排，密码框被按钮挤窄，比没有图标的「确认密码」框短一截。 */
.input-wrap {
  position: relative;
  display: block;
}

.field input {
  width: 100%;
  box-sizing: border-box;
  padding: 11px 14px;
  font-size: 15px;
  border: 1px solid #e8e1d5;
  border-radius: 12px;
  background: #fffdfa;
  color: #3d3426;
  outline: none;
}

/* 右侧留出图标的位置，避免长密码被图标压住 */
.input-wrap input {
  padding-right: 46px;
}

.field input:focus {
  border-color: #d9a441;
  background: #fff;
}

.field input[aria-invalid='true'] {
  border-color: #d98a80;
}

/* 显示/隐藏密码：叠在输入框内右侧，点击行为与原来的文字按钮完全一致 */
.toggle {
  position: absolute;
  top: 50%;
  right: 5px;
  transform: translateY(-50%);
  display: flex;
  align-items: center;
  justify-content: center;
  width: 36px;
  height: 36px;
  padding: 0;
  color: #9a8f7d;
  background: none;
  border: none;
  border-radius: 8px;
  cursor: pointer;
}

.toggle:hover {
  color: #7a6f5d;
}

.toggle:focus-visible {
  outline: 2px solid #d9a441;
  outline-offset: 1px;
}

/* 校验提示：上移抵消 .field 的下边距，避免字段间距被撑开 */
.hint {
  margin: -8px 0 12px;
  font-size: 12px;
  line-height: 1.4;
}

.hint-error {
  color: #b4453a;
}

.error {
  margin: 4px 0 12px;
  padding: 9px 12px;
  font-size: 13px;
  color: #b4453a;
  background: #fdf1ef;
  border-radius: 10px;
}

.primary {
  width: 100%;
  padding: 13px;
  font-size: 16px;
  font-weight: 600;
  color: #fff;
  background: #d9a441;
  border: none;
  border-radius: 12px;
  cursor: pointer;
}

.primary:disabled {
  opacity: 0.6;
  cursor: default;
}

.switch {
  margin: 18px 0 0;
  text-align: center;
  font-size: 13px;
  color: #9a8f7d;
}

.link {
  background: none;
  border: none;
  padding: 0;
  color: #c08a2a;
  font-size: 13px;
  cursor: pointer;
  text-decoration: underline;
}
</style>
