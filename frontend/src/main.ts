import './style.css'
import { bridge } from './bridge'

const $ = <T extends HTMLElement>(id: string) => document.getElementById(id) as T

const input = $<HTMLTextAreaElement>('input')
const output = $<HTMLPreElement>('output')
const indentSel = $<HTMLSelectElement>('indent')
const statusBar = document.querySelector('.status') as HTMLElement
const statusText = $('status-text')
const statsEl = $('stats')
const verEl = $('ver')
const inSize = $('in-size')
const outSize = $('out-size')

const enc = new TextEncoder()
const bytes = (s: string) => enc.encode(s).length

/** 发布仓库，用于「检查更新」 */
const REPO = 'rentianyu/winapp'

function fmtSize(n: number): string {
  if (n < 1024) return `${n} B`
  if (n < 1024 * 1024) return `${(n / 1024).toFixed(1)} KB`
  return `${(n / 1024 / 1024).toFixed(2)} MB`
}

function setStatus(text: string, kind: 'ok' | 'err' | '' = '') {
  statusText.textContent = text
  statusBar.classList.remove('ok', 'err')
  if (kind) statusBar.classList.add(kind)
}

function refreshSizes() {
  inSize.textContent = fmtSize(bytes(input.value))
}

function errText(e: unknown): string {
  if (typeof e === 'string') return e
  if (e && typeof e === 'object') {
    const o = e as Record<string, unknown>
    if (typeof o.message === 'string') return o.message
    if (typeof o.msg === 'string') return String(o.msg)
    try {
      return JSON.stringify(e)
    } catch {
      /* ignore */
    }
  }
  return String(e)
}

/* --------------------------------- 动作 --------------------------------- */

async function doFormat() {
  const raw = input.value.trim()
  if (!raw) return setStatus('输入为空', 'err')
  const t0 = performance.now()
  try {
    const out = await bridge.formatJSON(raw, Number(indentSel.value) || 2)
    output.textContent = out
    outSize.textContent = fmtSize(bytes(out))
    const ms = (performance.now() - t0).toFixed(1)
    setStatus(`格式化完成 · ${ms} ms`, 'ok')
    await refreshStats(raw)
  } catch (e) {
    setStatus(`失败：${errText(e)}`, 'err')
  }
}

async function doMinify() {
  const raw = input.value.trim()
  if (!raw) return setStatus('输入为空', 'err')
  try {
    const out = await bridge.minifyJSON(raw)
    output.textContent = out
    outSize.textContent = fmtSize(bytes(out))
    const ratio = raw.length ? (100 - (out.length / raw.length) * 100).toFixed(1) : '0'
    setStatus(`压缩完成 · 体积减少 ${ratio}%`, 'ok')
    await refreshStats(raw)
  } catch (e) {
    setStatus(`失败：${errText(e)}`, 'err')
  }
}

async function doCheck() {
  const raw = input.value.trim()
  if (!raw) return setStatus('输入为空', 'err')
  try {
    const se = await bridge.check(raw)
    if (se) {
      const where = se.line > 0 ? `第 ${se.line} 行第 ${se.column} 列` : ''
      setStatus(`语法错误 ${where}：${se.msg}`, 'err')
    } else {
      setStatus('JSON 合法', 'ok')
    }
    await refreshStats(raw)
  } catch (e) {
    setStatus(`校验失败：${errText(e)}`, 'err')
  }
}

async function refreshStats(raw: string) {
  try {
    const s = await bridge.analyze(raw)
    const types = Object.entries(s.types)
      .map(([k, v]) => `${k}×${v}`)
      .join(' ')
    statsEl.textContent = `${fmtSize(s.bytes)} · 深度 ${s.depth} · 节点 ${s.nodes} · ${types}`
  } catch {
    statsEl.textContent = ''
  }
}

async function doCopy() {
  const text = output.textContent ?? ''
  if (!text) return setStatus('没有可复制的内容', 'err')
  try {
    await navigator.clipboard.writeText(text)
    setStatus('已复制到剪贴板', 'ok')
  } catch {
    const r = document.createRange()
    r.selectNodeContents(output)
    const sel = window.getSelection()
    sel?.removeAllRanges()
    sel?.addRange(r)
    setStatus('已选中，请按 Ctrl+C', 'ok')
  }
}

function doClear() {
  input.value = ''
  output.textContent = ''
  statsEl.textContent = ''
  outSize.textContent = '—'
  refreshSizes()
  setStatus('已清空')
  input.focus()
}

const SAMPLE = JSON.stringify({
  项目: 'winapp',
  版本: '0.1.0',
  特性: ['Wails v3', 'WebView2', '四层验证'],
  作者: { 昵称: '小贝塔', 主页: 'https://github.com/rentianyu' },
  启用: true,
  配额: 1024,
  备注: null,
})

function loadSample() {
  input.value = SAMPLE
  refreshSizes()
  setStatus('已填入示例，按 Ctrl+Enter 格式化')
  input.focus()
}

async function doCheckUpdate() {
  setStatus('正在查询最新版本…')
  try {
    const r = await bridge.checkUpdate(REPO)
    if (r.hasNewer) {
      setStatus(`发现新版本 v${r.version}，请到 ${r.url} 下载`, 'ok')
    } else {
      setStatus(`已是最新版本（v${r.version}）`, 'ok')
    }
  } catch (e) {
    setStatus(`检查更新失败：${errText(e)}`, 'err')
  }
}

/* --------------------------------- 绑定 --------------------------------- */

$('btn-format').addEventListener('click', doFormat)
$('btn-minify').addEventListener('click', doMinify)
$('btn-check').addEventListener('click', doCheck)
$('btn-copy').addEventListener('click', doCopy)
$('btn-clear').addEventListener('click', doClear)
$('btn-sample').addEventListener('click', loadSample)
$('btn-update').addEventListener('click', doCheckUpdate)

input.addEventListener('input', refreshSizes)
input.addEventListener('keydown', (ev) => {
  if ((ev.ctrlKey || ev.metaKey) && ev.key === 'Enter') {
    ev.preventDefault()
    void doFormat()
  }
  if (ev.key === 'Tab') {
    ev.preventDefault()
    const s = input.selectionStart
    const e = input.selectionEnd
    input.value = input.value.slice(0, s) + '  ' + input.value.slice(e)
    input.selectionStart = input.selectionEnd = s + 2
  }
})

/* --------------------------------- 启动 --------------------------------- */

void (async () => {
  refreshSizes()
  const info = await bridge.getInfo()
  verEl.textContent = `v${info.version}${info.commit && info.commit !== 'none' ? ` (${info.commit.slice(0, 7)})` : ''}`
  if (bridge.kind === 'mock') {
    setStatus('浏览器模拟模式（Go 后端未连接，仅用于前端调试）')
  } else {
    setStatus(`就绪 · ${info.os}/${info.arch}`)
  }
  // 首屏给个示例，降低“打开是空白”的困惑
  if (!input.value) {
    input.value = SAMPLE
    refreshSizes()
  }
})()
