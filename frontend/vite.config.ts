import { defineConfig } from 'vite'

// wails3 dev 会以 --port 9245 启动 vite，这里保持一致便于直接 npm run dev
export default defineConfig({
  build: {
    outDir: 'dist',
    emptyOutDir: true,
    target: 'chrome110', // WebView2 基于 Chromium，可以放心用新语法
  },
  server: {
    port: 9245,
    strictPort: true,
  },
})
