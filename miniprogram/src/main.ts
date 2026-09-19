import { createSSRApp } from 'vue'
import { createPinia } from 'pinia'

import App from './App.vue'

/**
 * 小程序入口。
 *
 * `createSSRApp` 是 uni-app 的运行时约定（并非真做 SSR），与 Web 端的
 * `createApp` 不同；Pinia 的接入方式与 Web 端一致。
 */
export function createApp() {
  const app = createSSRApp(App)
  app.use(createPinia())
  return {
    app
  }
}
