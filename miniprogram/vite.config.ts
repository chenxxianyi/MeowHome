import { defineConfig } from 'vite'
import uni from '@dcloudio/vite-plugin-uni'

// https://vitejs.dev/config/
export default defineConfig({
  define: {
    __MEOWHOME_BUILD_TIME__: JSON.stringify(new Date().toISOString())
  },
  plugins: [uni()]
})
