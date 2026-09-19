<script setup lang="ts">
/**
 * 登录 / 注册页（小程序版）。
 *
 * 迁移自 `frontend/src/views/auth/AuthView.vue`（389 行），两个模式共用一个表单。
 *
 * ## 相对 Web 端源码的改动
 *
 * | Web 端写法 | 本页写法 | 原因 |
 * |---|---|---|
 * | `<form @submit.prevent>` | `<view>` + 按钮 `@click` | 小程序 `<form>` 的 submit 语义不同，且无法 preventDefault |
 * | `<input type="email">` | `<input type="text">` | 小程序 input 的 type 只支持 text/number/idcard/digit/nickname/safe-password，**无 email** |
 * | `<input type="password">` | `<input :password="!showPassword">` | 小程序用 **password 属性** 而非 type 控制掩码 |
 * | `autocomplete="..."` | 移除 | 小程序 input 不支持 autocomplete |
 * | `useRoute().query.redirect` | `onLoad(options)` 的 `options.redirect` | 无 vue-router |
 * | `router.replace('/today')` | `uni.switchTab` / `uni.reLaunch` | 页面栈模型；**tab 页必须用 switchTab**，用 reLaunch 会静默失败 |
 * | `<h1>` `<h2>` `<p>` `<span>` | `<view>` `<text>` | WXSS 不支持 HTML 元素选择器，且 `<span>` 会被静默映射成 `<label>` |
 * | `::placeholder` | `placeholder-class` | 小程序不支持该伪元素 |
 *
 * ## 登录方式
 *
 * 沿用 Web 端的**邮箱 + 密码**，不引入微信一键登录——迁移目标是「一模一样」，
 * 微信登录属新增能力（且需后端加 `openid` 字段与 `code2Session` 接口）。
 * 见《MeowHome-小程序迁移步骤.md》§11 阻塞项 #1。
 */
import { computed, ref } from 'vue'
import { onLoad, onShow } from '@dcloudio/uni-app'

import { toApiError } from '../../api/adapter'
import AppIcon from '../../components/app/AppIcon.vue'
import { useAuthStore } from '../../stores/auth'
import { guardOnShow, navigate } from '../../utils/guard'

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
const redirect = ref('')

const isRegister = computed(() => mode.value === 'register')
const title = computed(() => (isRegister.value ? '创建账号' : '欢迎回来'))
const submitLabel = computed(() => (isRegister.value ? '注册并进入' : '登录'))

/** 仅当用户已开始输入确认密码、且两次不一致时才提示，避免刚进页面就报错。 */
const passwordMismatch = computed(
  () => isRegister.value && confirmPassword.value.length > 0 && confirmPassword.value !== password.value
)

/** 密码是否达到后端要求的最短长度（后端 `app/auth.go` 校验 >= 8）。 */
const passwordTooShort = computed(() => isRegister.value && password.value.length > 0 && password.value.length < 8)

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

onLoad((options) => {
  // 路由守卫跳转过来时携带的原目标（小程序页面路径）
  if (options && typeof options.redirect === 'string') redirect.value = options.redirect
})

onShow(async () => {
  // 本页属公共页：未登录时守卫不跳转；已登录时守卫会把用户送回首页
  await guardOnShow()
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

    if (!familyId) {
      navigate('/pages/onboarding/index')
      return
    }
    navigate(redirect.value.startsWith('/pages/') ? redirect.value : '/pages/today/index')
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
  <view class="auth-page">
    <view class="auth-card">
      <view class="brand">
        <view class="logo"> 🐾 </view>
        <view class="brand-name"> 猫宅 </view>
        <text class="subtitle">多猫家庭的生活与健康管理</text>
      </view>

      <view class="form-title">
        {{ title }}
      </view>

      <view class="form">
        <label class="field">
          <text class="field-label">邮箱</text>
          <input
            v-model="email"
            class="field-input"
            type="text"
            placeholder="you@example.com"
            placeholder-class="input-placeholder"
          />
        </label>

        <label v-if="isRegister" class="field">
          <text class="field-label">昵称</text>
          <input
            v-model="userName"
            class="field-input"
            type="text"
            placeholder="家里怎么称呼你"
            placeholder-class="input-placeholder"
          />
        </label>

        <label class="field">
          <text class="field-label">密码</text>
          <view class="input-wrap">
            <input
              v-model="password"
              class="field-input has-affix"
              type="text"
              :password="!showPassword"
              :placeholder="isRegister ? '至少 8 位' : '请输入密码'"
              placeholder-class="input-placeholder"
            />
            <button
              type="button"
              class="toggle"
              :aria-label="showPassword ? '隐藏密码' : '显示密码'"
              @click="showPassword = !showPassword"
            >
              <AppIcon :name="showPassword ? 'eyeOff' : 'eye'" :size="18" />
            </button>
          </view>
        </label>
        <text v-if="passwordTooShort" class="hint hint-error"> 密码至少 8 位 </text>

        <label v-if="isRegister" class="field">
          <text class="field-label">确认密码</text>
          <input
            v-model="confirmPassword"
            class="field-input"
            type="text"
            :password="!showPassword"
            placeholder="请再次输入密码"
            placeholder-class="input-placeholder"
          />
        </label>
        <text v-if="passwordMismatch" class="hint hint-error"> 两次输入的密码不一致 </text>

        <label v-if="isRegister" class="field">
          <text class="field-label">家庭名称</text>
          <input
            v-model="familyName"
            class="field-input"
            type="text"
            placeholder="例如：小家的猫宅"
            placeholder-class="input-placeholder"
          />
        </label>

        <text v-if="error" class="error">
          {{ error }}
        </text>

        <button class="primary" :disabled="submitting || !canSubmit" @click="onSubmit">
          {{ submitting ? '处理中…' : submitLabel }}
        </button>
      </view>

      <view class="switch">
        <template v-if="isRegister">
          <text class="switch-text">已有账号？</text>
          <text class="link" @click="switchMode('login')">去登录</text>
        </template>
        <template v-else>
          <text class="switch-text">还没有账号？</text>
          <text class="link" @click="switchMode('register')">注册一个</text>
        </template>
      </view>
    </view>
  </view>
</template>

<!--
  非 scoped：小程序不支持 ::placeholder，占位符样式必须通过 placeholder-class 指定，
  而该类名由组件内部元素引用，无法命中 scoped 的属性选择器。
  小程序页面的 WXSS 本身是页级隔离的，不会污染其它页面。
-->
<style>
.input-placeholder {
  color: #b3aca3;
}
</style>

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
  padding: 32px 24px 24px;
  background: #fff;
  border-radius: 20px;
  box-shadow: 0 12px 40px rgba(120, 100, 70, 0.12);
}

.brand {
  margin-bottom: 20px;
  text-align: center;
}

.logo {
  font-size: 40px;
  line-height: 1;
}

.brand-name {
  margin: 8px 0 4px;
  font-size: 24px;
  color: #3d3426;
}

.subtitle {
  display: block;
  font-size: 13px;
  color: #9a8f7d;
}

.form-title {
  margin-bottom: 16px;
  font-size: 17px;
  font-weight: 600;
  color: #3d3426;
}

.field {
  display: block;
  margin-bottom: 14px;
}

.field-label {
  display: block;
  margin-bottom: 6px;
  font-size: 13px;
  color: #7a6f5d;
}

.field-input {
  width: 100%;
  box-sizing: border-box;
  padding: 11px 14px;
  font-size: 15px;
  color: #3d3426;
  background: #fffdfa;
  border: 1px solid #e8e1d5;
  border-radius: 12px;
}

/* 密码框右侧给图标留位（与 Web 端一致） */
.input-wrap {
  position: relative;
  display: block;
}

.has-affix {
  padding-right: 46px;
}

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
}

/* 校验提示：上移抵消 .field 的下边距 */
.hint {
  display: block;
  margin: -8px 0 12px;
  font-size: 12px;
  line-height: 1.4;
}

.hint-error {
  color: #b4453a;
}

.error {
  display: block;
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
  line-height: 1.2;
  color: #fff;
  background: #d9a441;
  border: none;
  border-radius: 12px;
}

.primary[disabled] {
  opacity: 0.6;
}

.switch {
  margin-top: 18px;
  text-align: center;
}

.switch-text {
  font-size: 13px;
  color: #9a8f7d;
}

.link {
  font-size: 13px;
  color: #c08a2a;
  text-decoration: underline;
}
</style>
