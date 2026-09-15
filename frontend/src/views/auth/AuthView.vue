<script setup lang="ts">
// 登录 / 注册页。两个模式共用一个表单，避免重复维护样式。
import { computed, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'

import { toApiError } from '../../api/adapter'
import { useAuthStore } from '../../stores/auth'

const router = useRouter()
const route = useRoute()
const auth = useAuthStore()

const mode = ref<'login' | 'register'>('login')
const email = ref('')
const password = ref('')
const userName = ref('')
const familyName = ref('')
const error = ref('')
const submitting = ref(false)

const isRegister = computed(() => mode.value === 'register')
const title = computed(() => (isRegister.value ? '创建账号' : '欢迎回来'))
const submitLabel = computed(() => (isRegister.value ? '注册并进入' : '登录'))

function switchMode(next: 'login' | 'register') {
  mode.value = next
  error.value = ''
}

async function onSubmit() {
  error.value = ''
  if (!email.value || !password.value) {
    error.value = '请填写邮箱与密码'
    return
  }
  if (isRegister.value && !userName.value) {
    error.value = '请填写昵称'
    return
  }
  if (isRegister.value && password.value.length < 8) {
    error.value = '密码至少 8 位'
    return
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
          <input
            v-model="password"
            type="password"
            :autocomplete="isRegister ? 'new-password' : 'current-password'"
            :placeholder="isRegister ? '至少 8 位' : '请输入密码'"
          />
        </label>

        <label v-if="isRegister" class="field">
          <span>家庭名称</span>
          <input v-model.trim="familyName" type="text" placeholder="例如：小家的猫宅" />
        </label>

        <p v-if="error" class="error">{{ error }}</p>

        <button type="submit" class="primary" :disabled="submitting">
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

.field span {
  display: block;
  font-size: 13px;
  color: #7a6f5d;
  margin-bottom: 6px;
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

.field input:focus {
  border-color: #d9a441;
  background: #fff;
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
