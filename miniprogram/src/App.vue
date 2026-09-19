<script setup lang="ts">
import { onLaunch } from '@dcloudio/uni-app'

import { useAppStore } from './stores/app'

onLaunch(() => {
  console.info('[app] MeowHome 小程序启动')

  // 小程序没有 navigator.onLine，网络状态必须主动查询 + 注册监听。
  // Web 端是在 store 的 state 初始化时同步读取的，这里改为异步校准。
  const app = useAppStore()
  app.syncNetworkStatus()
  app.listenNetworkStatus()
})
</script>

<!--
  小程序没有 index.html，App.vue 不渲染任何 DOM，仅承载全局样式与生命周期。

  全局样式必须在 App.vue 的**非 scoped** style 中引入：
  uni-app 会把它编译成 app.wxss，对全部页面生效。

  ⚠️ pages.css 里的类名（.page-header / .cat-list-item / .btn-primary /
  .sheet-overlay 等）是 Web 端各视图直接使用的全局类，**必须在此全局引入**，
  否则页面样式会整体丢失。

  顺序不可调换：tokens（变量）→ reset（归零）→ global（基础）→ pages（业务类）。
-->
<style>
@import './styles/tokens.css';
@import './styles/reset.css';
@import './styles/global.css';
@import './styles/pages.css';
</style>
