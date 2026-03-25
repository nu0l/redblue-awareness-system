import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'
import path from 'node:path'

export default defineConfig({
  plugins: [vue()],
  // 多入口：大屏 + 后台
  build: {
    rollupOptions: {
      input: {
        screen: path.resolve(__dirname, 'screen-vite.html'),
        admin: path.resolve(__dirname, 'admin-vite.html'),
        leaderboard: path.resolve(__dirname, 'leaderboard-vite.html')
      }
    }
  }
})

