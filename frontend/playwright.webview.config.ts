import { defineConfig, devices } from '@playwright/test'

/**
 * 真机用例：连到已经在跑的 Windows 应用（Wails + WebView2）。
 *
 * 前置：
 *   应用启动时设置 WINAPP_DEBUG_PORT=9222
 *   （main.go 会把它作为 --remote-debugging-port 传给 WebView2）
 *
 *   npm run e2e:win
 *
 * 注意：本配置不启动 webServer，也不下载浏览器 —— 它只是借 Playwright 的
 * CDP 客户端去驱动真实 WebView2，所以跑得很快。
 */
export default defineConfig({
  testDir: './e2e',
  testMatch: /webview\.spec\.ts/,
  timeout: 60_000,
  expect: { timeout: 15_000 },
  retries: process.env.CI ? 1 : 0,
  workers: 1,
  reporter: [['list'], ['html', { open: 'never', outputFolder: 'e2e-report/webview-html' }]],
  use: {
    ...devices['Desktop Chrome'],
    screenshot: 'only-on-failure',
  },
})
