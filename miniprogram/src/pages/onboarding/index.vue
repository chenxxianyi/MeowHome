<script setup lang="ts">
/**
 * 占位页 —— 步骤 1.1 注册路由用，**尚未迁移**。
 *
 * 迁移时用 Web 端对应视图替换本文件全部内容：
 *   frontend/src/views/onboarding/OnboardingView.vue
 *
 * 迁移步骤：3.1
 *
 * 注意迁移时的标签映射（实测结论，见《MeowHome-小程序迁移步骤.md》§记录 0.6）：
 *   <span> <strong> <i> <em> <b> <small>  →  <text>
 *   <div> <p> <h1>-<h3> <section> ...      →  <view>
 *   <img>                                  →  <image mode="aspectFit">
 *   <a href="#/...">                       →  <navigator> 或 @click + uni.navigateTo
 *   <input type="date">                    →  <picker mode="date">
 */
</script>

<template>
  <view class="page">
    <view class="placeholder">
      <text class="placeholder-title">开始使用</text>
      <text class="placeholder-hint">页面待迁移（步骤 3.1）</text>
    </view>
  </view>
</template>

<style scoped>
.placeholder {
  padding: 48px 24px;
  text-align: center;
}

.placeholder-title {
  display: block;
  margin-bottom: 8px;
  font-size: 20px;
  font-weight: 600;
  color: var(--color-text-primary, #2d2925);
}

.placeholder-hint {
  display: block;
  font-size: 13px;
  color: var(--color-text-tertiary, #9b948c);
}
</style>