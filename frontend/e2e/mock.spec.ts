import { expect, test } from '@playwright/test'

/**
 * 前端用例：跑在普通 Chromium 里，bridge 走 mock 实现。
 * 这一层覆盖 90% 的 UI 逻辑（事件绑定、状态更新、错误分支、布局），
 * 完全不需要 Windows。
 */
test.describe('UI（mock bridge）', () => {
  test.beforeEach(async ({ page }) => {
    await page.goto('/')
    await expect(page.locator('#btn-format')).toBeVisible()
  })

  test('首屏：工具条/输入框/状态栏都在，且提示模拟模式', async ({ page }) => {
    await expect(page.locator('.brand')).toHaveText('JSON Toolbox')
    await expect(page.locator('#input')).toBeVisible()
    await expect(page.locator('#output')).toBeVisible()
    await expect(page.locator('#status-text')).toContainText('模拟模式')
    // 首屏自动填示例，避免空白
    await expect(page.locator('#input')).not.toBeEmpty()
  })

  test('格式化：输出被重排且状态显示耗时', async ({ page }) => {
    await page.locator('#input').fill('{"a":1,"b":[1,2]}')
    await page.locator('#btn-format').click()
    await expect(page.locator('#output')).toHaveText(/{\n\s+"a": 1/)
    await expect(page.locator('#status-text')).toContainText('格式化完成')
    await expect(page.locator('#out-size')).not.toHaveText('—')
  })

  test('缩进切到 4 时输出跟着变', async ({ page }) => {
    await page.locator('#input').fill('{"a":1}')
    await page.locator('#indent').selectOption('4')
    await page.locator('#btn-format').click()
    await expect(page.locator('#output')).toHaveText(/{\n {4}"a": 1/)
  })

  test('压缩：结果无空白', async ({ page }) => {
    await page.locator('#input').fill('{\n  "a": 1\n}')
    await page.locator('#btn-minify').click()
    await expect(page.locator('#output')).toHaveText('{"a":1}')
    await expect(page.locator('#status-text')).toContainText('压缩完成')
  })

  test('校验合法 JSON', async ({ page }) => {
    await page.locator('#input').fill('{"ok":true}')
    await page.locator('#btn-check').click()
    await expect(page.locator('#status-text')).toContainText('JSON 合法')
  })

  test('校验非法 JSON：状态栏报错且变红', async ({ page }) => {
    await page.locator('#input').fill('{"a":1,}')
    await page.locator('#btn-check').click()
    await expect(page.locator('#status-text')).toContainText('语法错误')
    await expect(page.locator('.status')).toHaveClass(/err/)
  })

  test('统计栏：字节/深度/节点', async ({ page }) => {
    await page.locator('#input').fill('{"a":{"b":1}}')
    await page.locator('#btn-check').click()
    await expect(page.locator('#stats')).toContainText('深度 2')
    await expect(page.locator('#stats')).toContainText('节点')
  })

  test('清空：输入输出统计全部复位', async ({ page }) => {
    await page.locator('#btn-clear').click()
    await expect(page.locator('#input')).toHaveValue('')
    await expect(page.locator('#output')).toHaveText('')
    await expect(page.locator('#stats')).toHaveText('')
    await expect(page.locator('#in-size')).toHaveText('0 B')
  })

  test('Ctrl+Enter 触发格式化', async ({ page }) => {
    await page.locator('#input').fill('{"k":"v"}')
    await page.locator('#input').press('Control+Enter')
    await expect(page.locator('#output')).toContainText('"k"')
  })

  test('Tab 插入两个空格且不跳焦点', async ({ page }) => {
    await page.locator('#input').fill('')
    await page.locator('#input').click()
    await page.locator('#input').press('Tab')
    await expect(page.locator('#input')).toHaveValue('  ')
    await expect(page.locator('#input')).toBeFocused()
  })

  test('截图存档（人工复核用）', async ({ page }) => {
    await page.locator('#input').fill('{"名称":"样例","列表":[1,2,3],"flag":true}')
    await page.locator('#btn-format').click()
    await page.waitForTimeout(120)
    await page.screenshot({ path: 'e2e-report/ui-formatted.png', fullPage: true })
  })
})
