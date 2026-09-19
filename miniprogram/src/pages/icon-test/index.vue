<script setup lang="ts">
/**
 * 图标方案验证页（步骤 0.5 专用，非正式页面）。
 *
 * 目的：在微信开发者工具里一屏确认三件事——
 *   1. 遮罩方案能否正常渲染 57 个图标（不是空白、也不是整块实心色）
 *   2. 颜色是否跟随父元素文字色（等价于 Web 端的 stroke="currentColor"）
 *   3. size prop 是否生效
 *
 * 第四屏是**对照组**：用 `<image>` 直贴同一份 data URI。
 * 若遮罩不受支持（图标显示为实心色块），而 `<image>` 能显示图标，
 * 说明需要改走 image 方案（代价：失去 currentColor 变色能力）。
 */
import AppIcon from '../../components/app/AppIcon.vue'
import { iconDataUris, iconNames } from '../../components/app/iconData'

/** 颜色继承验证用的色值，取自 tokens.css 的实际用色。 */
const colors = [
  { label: '主文字 #2D2925', value: '#2D2925' },
  { label: '次要 #706A64', value: '#706A64' },
  { label: '品牌 #C87345', value: '#C87345' },
  { label: '成功 #5D846D', value: '#5D846D' },
  { label: '危险 #B5534B', value: '#B5534B' }
]

const sizes = [16, 20, 24, 32, 48]

/** 对照组：只取前 6 个图标，够判断可行性即可。 */
const sampleNames = iconNames.slice(0, 6)
</script>

<template>
  <view class="page">
    <view class="section">
      <view class="section-title"> 1 · 全部 57 个图标（遮罩方案） </view>
      <view class="section-hint"> 全部应为清晰的描边线条图标。若出现整块实心色块 → 遮罩未生效。 </view>
      <view class="grid">
        <view v-for="name in iconNames" :key="name" class="grid-cell">
          <AppIcon :name="name" :size="24" />
          <text class="grid-label">{{ name }}</text>
        </view>
      </view>
    </view>

    <view class="section">
      <view class="section-title"> 2 · 颜色继承（关键验证） </view>
      <view class="section-hint"> 图标颜色应随外层文字色变化，且与左侧文字颜色一致。 </view>
      <view v-for="c in colors" :key="c.value" class="row" :style="{ color: c.value }">
        <text class="color-label">{{ c.label }}</text>
        <AppIcon name="cat" :size="28" />
        <AppIcon name="bell" :size="28" />
        <AppIcon name="pawPrint" :size="28" />
        <AppIcon name="warning" :size="28" />
      </view>
    </view>

    <view class="section">
      <view class="section-title"> 3 · 尺寸控制 </view>
      <view class="section-hint"> 16 / 20 / 24 / 32 / 48 px，应由小到大且不变形。 </view>
      <view class="row baseline">
        <AppIcon v-for="s in sizes" :key="s" name="cat" :size="s" />
      </view>
    </view>

    <view class="section">
      <view class="section-title"> 4 · 对照组：image 直贴 </view>
      <view class="section-hint">
        若第 1 屏是实心色块、而这里能显示图标 → 遮罩不受支持，需改走 image 方案。 注意 image 方案会固定为 SVG
        内嵌的黑色，<text>不跟随文字色</text>。
      </view>
      <view class="row baseline">
        <image v-for="name in sampleNames" :key="name" class="cmp-img" :src="iconDataUris[name]" mode="aspectFit" />
      </view>
    </view>

    <view class="footer"> 步骤 0.5 验证页 · 确认后由步骤 1.1 移除 </view>
  </view>
</template>

<style scoped>
.page {
  padding: 16px;
  padding-bottom: 48px;
  background: var(--color-bg-page, #f7f4ee);
  min-height: 100vh;
}

.section {
  margin-bottom: 28px;
  padding: 16px;
  background: var(--color-bg-surface, #fffcf7);
  border-radius: 12px;
}

.section-title {
  font-size: 16px;
  font-weight: 600;
  color: var(--color-text-primary, #2d2925);
  margin-bottom: 6px;
}

.section-hint {
  font-size: 12px;
  line-height: 1.5;
  color: var(--color-text-tertiary, #9b948c);
  margin-bottom: 14px;
}

.grid {
  display: flex;
  flex-wrap: wrap;
}

.grid-cell {
  width: 25%;
  display: flex;
  flex-direction: column;
  align-items: center;
  margin-bottom: 14px;
  color: var(--color-text-primary, #2d2925);
}

.grid-label {
  font-size: 10px;
  color: var(--color-text-tertiary, #9b948c);
  margin-top: 4px;
  text-align: center;
  word-break: break-all;
}

.row {
  display: flex;
  align-items: center;
  gap: 14px;
  padding: 8px 0;
}

.baseline {
  align-items: baseline;
}

.color-label {
  flex: 1;
  font-size: 12px;
}

.cmp-img {
  width: 24px;
  height: 24px;
}

.footer {
  text-align: center;
  font-size: 12px;
  color: var(--color-text-tertiary, #9b948c);
}
</style>
