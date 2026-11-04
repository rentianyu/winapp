import { chromium, expect, test } from '@playwright/test'

/**
 * 真机用例：连到已经在运行的 Windows 应用（Wails + WebView2）。
 *
 * 前置：
 *   1. 应用以 WINAPP_DEBUG_PORT=9222 启动（main.go 会把该端口传给 WebView2）
 *   2. npx playwright test --project=webview2
 *
 * 这层能覆盖 mock 层覆盖不了的东西：Go 侧真实调用、IPC 序列化、
 * WebView2 的渲染差异、单实例锁、窗口创建。
 */

const PORT = process.env.CDP_PORT ?? '9222'

async function connect() {
  const browser = await chromium.connectOverCDP(`http://127.0.0.1:${PORT}`)
  const ctx = browser.contexts()[0]
  if (!ctx) throw new Error('CDP 连上了但没有 context')

  // 应用里可能已有页面；挑出含我们 UI 的那个
  let page = ctx.pages()[0]
  for (const p of ctx.pages()) {
    const has = await p.locator('#btn-format').count().catch(() => 0)
    if (has > 0) {
      page = p
      break
    }
  }
  if (!page) {
    page = await ctx.waitForEvent('page', { timeout: 20_000 })
  }
  await page.waitForSelector('#btn-format', { timeout: 20_000 })
  return { browser, page }
}

test('真实应用冒烟：Go 后端可用、格式化/校验/统计正确', async () => {
  const { page } = await connect()

  // 1. 真实 bridge 已连上（不是浏览器模拟模式）
  await expect(page.locator('#status-text')).not.toContainText('模拟模式', { timeout: 15_000 })

  // 2. 格式化走真实 Go 调用
  await page.locator('#input').fill('{"x":1,"y":[true,null]}')
  await page.locator('#btn-format').click()
  await expect(page.locator('#output')).toContainText('"x": 1')
  await expect(page.locator('#status-text')).toContainText('格式化完成')

  // 3. 统计由 Go 侧计算（含类型计数，能区分 mock 与真机实现）
  await expect(page.locator('#stats')).toContainText('节点')
  await expect(page.locator('#stats')).toContainText('boolean')

  // 4. 错误分支：Go 侧返回带行号的语法错误
  await page.locator('#input').fill('{\n  "a": 1,\n  "b": oops\n}')
  await page.locator('#btn-check').click()
  await expect(page.locator('#status-text')).toContainText('第 3 行')

  // 5. 版本号来自 -ldflags 注入（证明构建参数生效）
  await expect(page.locator('#ver')).toContainText(/v\d+\.\d+\.\d+/)

  // 6. 截图存档：CI 里作为 artifact 上传，人工一眼看真机长什么样
  await page.locator('#input').fill('{"名称":"真机测试","来源":"WebView2"}')
  await page.locator('#btn-format').click()
  await page.waitForTimeout(150)
  await page.screenshot({ path: 'e2e-report/webview-real.png', fullPage: true })

  // 不调用 browser.close()：那会关掉宿主应用，测试进程退出即可
})
