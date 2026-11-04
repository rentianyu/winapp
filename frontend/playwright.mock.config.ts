import { defineConfig, devices } from '@playwright/test'

/**
 * 前端用例：起一个 vite preview 静态站，bridge 自动降级为 mock。
 * 任何一台机器（Linux / macOS / Windows）都能跑，不需要 exe，秒级反馈。
 *
 *   npm run build
 *   npx playwright install chromium      # 首次
 *   npm run e2e:mock
 */
export default defineConfig({
  testDir: './e2e',
  testMatch: /mock\.spec\.ts/,
  timeout: 30_000,
  expect: { timeout: 8_000 },
  retries: process.env.CI ? 1 : 0,
  workers: 1,
  reporter: [['list'], ['html', { open: 'never', outputFolder: 'e2e-report/mock-html' }]],
  use: {
    ...devices['Desktop Chrome'],
    baseURL: 'http://127.0.0.1:4173',
    screenshot: 'only-on-failure',
  },
  webServer: {
    // 必须显式 --host 127.0.0.1：vite 默认监听 localhost，在 IPv6 优先的环境里
    // 会只绑 ::1，导致 Playwright 连 127.0.0.1 一直超时
    command: 'pnpm exec vite preview --host 127.0.0.1 --port 4173 --strictPort',
    url: 'http://127.0.0.1:4173',
    reuseExistingServer: !process.env.CI,
    timeout: 90_000,
    stdout: 'pipe',
    stderr: 'pipe',
  },
})
