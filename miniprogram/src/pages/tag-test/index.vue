<script setup lang="ts">
/**
 * HTML 标签容错实测页（步骤 0.6 第一步，非正式页面）。
 *
 * ## 为什么要做这个实验
 *
 * Web 端 17 个视图共使用约 560 个 HTML 标签（实测：`<div>` 324、
 * `<span>` 112、`<button>` 73、`<label>` 24、`<input>` 20、`<h2>` 17、
 * `<img>` 7、`<p>` 7、`<h1>` 4）。
 *
 * 小程序只认 `<view>` / `<text>` / `<image>` 等内置组件。问题是：
 * **uni-app 编译器对 HTML 标签是报错、警告，还是静默映射？**
 *
 * 结论直接决定工作量：
 *   - 若报错/警告   → 必须全量替换（约 560 处）
 *   - 若静默映射    → 可保留，只改必需的（工作量大幅下降）
 *   - 若静默吞掉    → 最危险：编译通过但运行时白屏，必须全量替换
 *
 * ## 本页内容
 *
 * 故意混用 HTML 标签与小程序组件，观察三件事：
 *   1. `npm run build:mp-weixin` 是否输出警告
 *   2. 产物 WXML 里这些标签变成了什么
 *   3. 运行时能否渲染（需在开发者工具里目视）
 */
import { ref } from 'vue'

const tapped = ref('（尚未点击）')

function onTap() {
  tapped.value = `已点击 · ${new Date().toLocaleTimeString()}`
}
</script>

<template>
  <view class="page">
    <view class="section">
      <view class="section-title">
        A · HTML 结构标签
      </view>
      <view class="section-hint">
        编译产物 WXML 中应能看出它们被映射成了什么。
      </view>

      <div class="box">
        <span>span 内联文本</span>
        <p>p 段落文本</p>
        <h1>h1 标题</h1>
        <h2>h2 标题</h2>
        <div class="nested">
          div 嵌套 div
        </div>
      </div>
    </view>

    <view class="section">
      <view class="section-title">
        B · 表单与交互标签
      </view>
      <label class="row">
        <span>label + input：</span>
        <input
          class="inp"
          placeholder="输入点什么"
        />
      </label>
      <button
        class="btn"
        @click="onTap"
      >
        button 点击
      </button>
      <view class="result">
        {{ tapped }}
      </view>
    </view>

    <view class="section">
      <view class="section-title">
        C · 小程序原生组件（对照组）
      </view>
      <view class="section-hint">
        这些是小程序确定支持的写法，用它对照 A / B 的渲染结果。
      </view>
      <view class="box">
        <text>text 内联文本</text>
        <view>view 块级</view>
      </view>
      <button
        class="btn"
        @click="onTap"
      >
        button（原生同样写法）
      </button>
      <image
        class="img"
        src="/static/logo.png"
        mode="aspectFit"
      />
    </view>

    <view class="section">
      <view class="section-title">
        D · 媒体、链接与嵌套限制
      </view>
      <view class="section-hint">
        重点看编译产物：img / a 会被映射成什么；text 内嵌 view 是否报错。
      </view>
      <img
        class="img"
        src="/static/logo.png"
        alt="img 标签"
      >
      <a href="/pages/index/index">a 标签链接</a>
      <text>
        <view>非法嵌套测试：text 内放 view</view>
      </text>
    </view>

    <view class="section">
      <view class="section-title">
        E · 语义标签（Web 端实际用到的其余标签）
      </view>
      <view class="section-hint">
        这些在 Web 端共约 59 处，需确认映射目标是否安全。
      </view>
      <section>section 区块</section>
      <header>header 页头</header>
      <main>main 主体</main>
      <nav>nav 导航</nav>
      <h3>h3 标题</h3>
      <strong>strong 强调</strong>
      <em>em 强调</em>
      <i>i 斜体</i>
      <b>b 粗体</b>
    </view>

    <view class="footer">
      步骤 0.6 标签容错实测页
    </view>
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
  margin-bottom: 24px;
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
  margin-bottom: 12px;
}

.box {
  padding: 10px;
  border: 1px dashed var(--color-border, #e7e1d9);
  border-radius: 8px;
  font-size: 13px;
  color: var(--color-text-primary, #2d2925);
}

.nested {
  margin-top: 6px;
  padding: 6px;
  background: var(--color-bg-subtle, #f1ece5);
}

.row {
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 13px;
  color: var(--color-text-secondary, #706a64);
}

.inp {
  flex: 1;
  padding: 8px 10px;
  font-size: 14px;
  background: #fffdfa;
  border: 1px solid var(--color-border, #e7e1d9);
  border-radius: 8px;
}

.btn {
  margin-top: 12px;
  font-size: 14px;
}

.result {
  margin-top: 8px;
  font-size: 12px;
  color: var(--color-success, #5d846d);
}

.img {
  width: 40px;
  height: 40px;
  margin-top: 10px;
}

.footer {
  text-align: center;
  font-size: 12px;
  color: var(--color-text-tertiary, #9b948c);
}
</style>
