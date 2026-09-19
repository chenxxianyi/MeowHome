// Vitest setup — load Pinia testing helpers
import { config } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'

// 复用同一个 Pinia 实例：此前 setActivePinia 与 global.plugins 各建了一个，
// 组件内注入的与测试里取到的不是同一份，Vue 会警告 “App already provides
// property with key Symbol(pinia)”，状态也容易对不上。
const pinia = createPinia()

setActivePinia(pinia)

config.global.plugins = [pinia]
