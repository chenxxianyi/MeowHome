<script setup lang="ts">
/**
 * 图标组件（小程序版）。
 *
 * 与 Web 端 `frontend/src/components/app/AppIcon.vue` 的**对外接口完全一致**
 * （`name` / `size` 两个 props），因此所有调用点无需改动。
 *
 * ## 实现差异（必须理解，否则会误改）
 *
 * Web 端：`<svg stroke="currentColor" v-html="content">`
 * 小程序：**不支持内联 `<svg>`，也不支持 `v-html`**，因此改为——
 *
 *   把同一份 path 数据内联成 `<svg>` 后 base64 编码为 data URI，
 *   作为 CSS 遮罩（`-webkit-mask-image`）贴在 `<view>` 上；
 *   颜色由 `background-color: currentColor` 提供。
 *
 * 遮罩**只取 alpha 通道**，SVG 里的 stroke 颜色（`#000`）不影响显示，
 * 真正上色的是 `currentColor`——这与 Web 端 `stroke="currentColor"` 语义等价，
 * 所以图标依旧会跟随父元素的文字颜色变化。
 *
 * ## 为什么不转图标字体
 *
 * 这 57 个图标是描边式（`fill="none"` + `stroke`，含 `polyline` 与无填充
 * `circle`）。字体字形是填充形状，描边需先做 outline 展开才能成为字形。
 * 遮罩直接复用原始 path，几何零损失，也不需要字体生成工具链。
 *
 * 图标数据由 `scripts/gen-icons.mjs` 从 Web 端 `iconPaths.ts` 生成，
 * **Web 端图标有变更时必须重跑该脚本**。
 */
import { computed } from 'vue'

import { iconDataUris, type AppIconName } from './iconData'

const props = withDefaults(defineProps<{ name: AppIconName | string; size?: number }>(), {
  size: 20
})

const iconStyle = computed(() => {
  const uri = iconDataUris[props.name as AppIconName] || iconDataUris.info
  return {
    width: `${props.size}px`,
    height: `${props.size}px`,
    '-webkit-mask-image': `url("${uri}")`,
    'mask-image': `url("${uri}")`
  }
})
</script>

<template>
  <view class="app-icon" :style="iconStyle" aria-hidden="true" />
</template>

<style scoped>
.app-icon {
  display: inline-block;
  flex: none;
  vertical-align: middle;
  /* 颜色跟随父元素文字色，等价于 Web 端的 stroke="currentColor" */
  background-color: currentColor;
  -webkit-mask-repeat: no-repeat;
  mask-repeat: no-repeat;
  -webkit-mask-position: center;
  mask-position: center;
  -webkit-mask-size: contain;
  mask-size: contain;
}
</style>
