/**
 * bridge.ts — 前端与 Go 的唯一出入口。
 *
 * 为什么要有这一层：
 *  1. 前端永远不直接 import @wailsio/runtime 的 Call，改一处就能全局替换
 *  2. 在纯浏览器（vite dev / Playwright / CI）里自动降级为本地 mock，
 *     于是 90% 的 UI 逻辑不需要 Windows 也能跑测试
 *  3. Go 侧方法全名（PkgPath.Type.Method）只在这里出现一次，改模块路径时只改这里
 */

import { Call } from '@wailsio/runtime'

export interface Stats {
  bytes: number
  chars: number
  depth: number
  nodes: number
  types: Record<string, number>
  topLevel: string
}

export interface SyntaxErrorInfo {
  offset: number
  line: number
  column: number
  msg: string
}

export interface AppInfo {
  version: string
  commit: string
  builtAt: string
  goVersion: string
  os: string
  arch: string
  now: string
}

export interface ReleaseInfo {
  tag: string
  version: string
  url: string
  hasNewer: boolean
}

export interface ToolboxBridge {
  formatJSON(raw: string, indent: number): Promise<string>
  minifyJSON(raw: string): Promise<string>
  analyze(raw: string): Promise<Stats>
  check(raw: string): Promise<SyntaxErrorInfo | null>
  getInfo(): Promise<AppInfo>
  checkUpdate(repo: string): Promise<ReleaseInfo>
  readonly kind: 'wails' | 'mock'
}

const SVC = 'github.com/rentianyu/winapp/internal/services.ToolboxService'

/* ------------------------------- 真实实现 ------------------------------- */

const realBridge: ToolboxBridge = {
  kind: 'wails',
  formatJSON: (raw, indent) => Call.ByName(`${SVC}.FormatJSON`, raw, indent),
  minifyJSON: (raw) => Call.ByName(`${SVC}.MinifyJSON`, raw),
  analyze: (raw) => Call.ByName(`${SVC}.Analyze`, raw),
  check: (raw) => Call.ByName(`${SVC}.Check`, raw),
  getInfo: () => Call.ByName(`${SVC}.GetInfo`),
  checkUpdate: (repo) => Call.ByName(`${SVC}.CheckUpdate`, repo),
}

/* -------------------------------- mock -------------------------------- */

const mockBridge: ToolboxBridge = {
  kind: 'mock',

  async formatJSON(raw, indent) {
    const v = JSON.parse(raw) // 故意让浏览器抛错，走统一 catch
    return JSON.stringify(v, null, indent)
  },

  async minifyJSON(raw) {
    return JSON.stringify(JSON.parse(raw))
  },

  async analyze(raw) {
    const v = JSON.parse(raw)
    const types: Record<string, number> = {}
    let nodes = 0
    let depth = 0
    const walk = (x: unknown, d: number) => {
      nodes++
      depth = Math.max(depth, d)
      if (Array.isArray(x)) {
        types.array = (types.array ?? 0) + 1
        x.forEach((c) => walk(c, d + 1))
      } else if (x === null) {
        types.null = (types.null ?? 0) + 1
      } else if (typeof x === 'object') {
        types.object = (types.object ?? 0) + 1
        Object.values(x as Record<string, unknown>).forEach((c) => walk(c, d + 1))
      } else {
        const t = typeof x
        types[t] = (types[t] ?? 0) + 1
      }
    }
    walk(v, 0)
    return {
      bytes: new TextEncoder().encode(raw).length,
      chars: [...raw].length,
      depth,
      nodes,
      types,
      topLevel: Array.isArray(v) ? 'array' : v === null ? 'null' : typeof v,
    }
  },

  async check(raw) {
    try {
      JSON.parse(raw)
      return null
    } catch (e) {
      const msg = (e as Error).message
      // Chrome 的报错形如 "Unexpected token o in JSON at position 17"
      const m = /position (\d+)/.exec(msg)
      const offset = m ? Number(m[1]) : 0
      const before = raw.slice(0, offset).split('\n')
      return {
        offset,
        line: before.length,
        column: before[before.length - 1].length + 1,
        msg,
      }
    }
  },

  async getInfo() {
    return {
      version: 'mock',
      commit: 'mock',
      builtAt: 'mock',
      goVersion: 'n/a',
      os: 'browser',
      arch: navigator.platform,
      now: new Date().toLocaleString('zh-CN'),
    }
  },

  async checkUpdate() {
    // 浏览器里不真的去请求 GitHub（CORS + 限流），返回一个固定值保证 UI 可测
    return { tag: 'v0.0.0-mock', version: '0.0.0-mock', url: 'https://github.com/rentianyu/winapp/releases', hasNewer: false }
  },
}

/**
 * 当前生效的实现。默认 mock，initBridge() 探测成功后切到 realBridge。
 *
 * 为什么不在模块加载时同步判断：
 *  Wails v3 注入的 window._wails 字段在不同版本/平台下不一定有 environment，
 *  按字段猜会误判（真机里也走 mock）。直接试调一次后端最可靠。
 */
export let bridge: ToolboxBridge = mockBridge

/** 在应用启动时调用一次，确定用真实后端还是 mock。 */
export async function initBridge(): Promise<ToolboxBridge> {
  try {
    const info = await Call.ByName(`${SVC}.GetInfo`)
    if (info && typeof info === 'object' && 'version' in (info as object)) {
      bridge = realBridge
    }
  } catch {
    // 纯浏览器里没有 Wails runtime，保持 mock
  }
  return bridge
}
