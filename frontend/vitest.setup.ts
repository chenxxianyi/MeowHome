// Vitest setup — load Pinia testing helpers
import { config } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'

// Ensure a Pinia instance is available globally
setActivePinia(createPinia())

// Ensure a Pinia instance is available
config.global.plugins = [createPinia()]
