# winapp — Windows 桌面应用骨架（Wails v3 + Go + TypeScript）

一个**能在 Linux/CI 里开发和验证**的 Windows 桌面应用模板。核心理念：把「需要 Windows 才能验的东西」压到最小，剩下的全部在快循环里跑完。

- 产物：单个 exe，约 10–12 MB，无运行时依赖（Win10 1803+ / Win11 自带 WebView2）
- 构建：`bash scripts/build.sh` —— 在 Linux 上 7 秒出 exe，不用装 Windows SDK
- 验证：四层（3 秒 → 8 分钟），其中前三层都不需要 Windows
- 发布：GitHub Actions 出 exe + 绿色版 zip + Inno Setup 安装包

---

## 一、为什么是这个组合

| 方案 | 包体 | 无 Windows 可验证度 | CI 出包 |
|---|---|---|---|
| **Wails v3（本方案）** | 10–15 MB | ★★★★★ | 3–4 min |
| Tauri 2（Rust） | 3–8 MB | ★★★ | 8–15 min（Rust 冷编译） |
| Python + PySide6 | 40–120 MB | ★★★☆ | 2–3 min |
| Electron | 120 MB+ | ★★★★★ | 4 min |

选 Wails 的决定性因素不是体积，而是**「没有 Windows 机器时还能不能高效开发」**：UI 是网页，所以能在浏览器里跑完整交互测试；Go 逻辑是纯代码，所以能在 Linux 上单测。壳层只剩窗口和托盘，Bug 面极小。

---

## 二、目录结构（三层减法）

```
winapp/
├── frontend/                    前端（Vite + TS）
│   ├── src/bridge.ts            ★ 唯一 IPC 出入口；浏览器里自动切 mock
│   ├── src/main.ts              UI 逻辑
│   ├── e2e/mock.spec.ts         前端用例（Chromium，不需要 exe）
│   └── e2e/webview.spec.ts      真机用例（CDP 连 WebView2）
├── internal/
│   ├── core/                    ★ 纯 Go，零平台依赖 → Linux 100% 可测
│   ├── services/                薄封装层（只做参数校正 + 调 core）
│   └── platform/                //go:build 隔离的平台代码
│       ├── msgbox/
│       └── singleinstance/
├── build/windows/               图标、清单、版本资源、Inno Setup 脚本
├── scripts/
│   ├── build.sh                 一条命令出 exe
│   ├── check.sh                 L0 快检
│   └── exe_peek.py              PE 产物验收（版本号/架构/子系统）
├── main.go                      壳，约 150 行
└── .github/workflows/           ci.yml + release.yml
```

**三条硬纪律：**

1. `internal/core` 不许 import 任何平台包 —— 这样 `go test ./internal/...` 在 Linux 上就能跑全量单测。
2. 前端不直接调 `@wailsio/runtime`，一律走 `bridge.ts` —— 于是 UI 能在纯浏览器里跑。
3. `main.go` 不写业务 —— 出 Bug 一定在 `core` 或 `frontend`，两个都能秒级复现。

---

## 三、四层验证

| 层 | 在哪跑 | 覆盖 | 耗时 | 命令 |
|---|---|---|---|---|
| **L0** | 任意 Linux/macOS | 纯逻辑单测、四平台类型检查 | 3–10 s | `bash scripts/check.sh` |
| **L1** | 任意机器（浏览器） | 真实 UI 交互（mock bridge）、前端打包 | 30–60 s | `pnpm run e2e:mock` |
| **L2** | GitHub `windows-latest` | 真实 WebView2、Go 后端、截图 | 4–8 min | push 即触发 |
| **L3** | 任意机器 | 产物完整性、版本号、架构 | 20 s | `python3 scripts/exe_peek.py dist/*.exe --expect-version 0.1.0` |

### L0：秒级挡住低级错误

```bash
bash scripts/check.sh
# ✅ 单测（-race）
# ✅ GOOS=windows go vet ./...      ← 不用 Windows 就能发现 Windows 侧类型错误
# ✅ go vet ./internal/...          ← 反证业务层没有平台依赖
```

### L1：在浏览器里验 UI

```bash
cd frontend
pnpm install
pnpm run build          # tsc 类型检查 + vite 打包，约 1.6 秒
pnpm run e2e:mock       # 11 条用例：格式化/压缩/校验/统计/快捷键/清空…
```

`bridge.ts` 用 `window._wails.environment` 判断环境，浏览器里自动走 mock 实现，所以**不需要 exe 就能把 UI 逻辑全测完**。

### L2：真机（CI 自动做）

CI 在 `windows-latest` 上：编译 → 以 `WINAPP_DEBUG_PORT=9222` 启动 → 等 CDP 就绪 → Playwright `connectOverCDP` 连进真实 WebView2 → 点按钮、断言行号错误、截图 → 上传 artifact。

```
frontend/e2e-report/webview-real.png   ← 真机截图，5 分钟后可下载
dist/json-toolbox-windows-amd64.exe    ← 可执行产物
```

### L3：产物验收

```bash
python3 scripts/exe_peek.py dist/*.exe --expect-version 0.1.0 --expect-arch amd64
```

检查 PE 魔数、子系统必须是 `WINDOWS_GUI`（否则双击弹黑窗）、`FileVersion` 是否等于构建版本、架构、SHA256、有无签名。

---

## 四、本地开发（Windows）

三种模式，按需要选：

### 1) 只改 UI —— 最快（不需要 Go 环境）

```powershell
cd frontend
pnpm install
pnpm dev            # http://localhost:9245，bridge 走 mock
```

改 CSS/TS 即时生效，F12 正常用。

### 2) 前后端联调 —— 需要装 Go + Wails CLI

```powershell
go install github.com/wailsapp/wails/v3/cmd/wails3@v3.0.0
wails3 dev          # 前端 HMR + Go 改动自动重编译 + 内置 DevTools
```

### 3) 出包

```bash
bash scripts/build.sh                    # 版本取 git tag
bash scripts/build.sh --version=0.1.0
```

产物：`dist/json-toolbox-windows-amd64.exe`

> Windows 上跑 `build.sh` 需要 Git Bash（Git for Windows 自带）和 `go-winres`：
> `go install github.com/tc-hib/go-winres@latest`

---

## 五、CI / CD

| 触发 | 做什么 |
|---|---|
| push / PR | `lint-test`（单测 + vet + 前端 E2E）+ `build-windows`（出 exe + 真机 E2E + 截图） |
| 打 tag `v0.1.0` | `release`：出 exe、绿色版 zip、Inno Setup 安装包、SHA256SUMS，上传 Release |

**关键点：**

- runner 预装 Go 只有 1.24，Wails v3 要求 1.26+，所以 `setup-go` 必须显式指定版本
- `go-winres` 的 `--no-suffix` 参数有坑：加了它**不会生成任何文件**且退出码为 0（本项目踩过）
- 图标必须是 ≤256×256 的 PNG/ICO，1024 会报 `image size too big`
- 未签名 exe 在公开分发时会被 SmartScreen 拦，首次下载有「更多信息 → 仍要运行」提示

---

## 六、常见改动指引

| 想做什么 | 改哪里 |
|---|---|
| 加一个后端能力 | `internal/core/xxx.go`（纯逻辑）+ `internal/services/toolbox.go` 加一个方法 + `frontend/src/bridge.ts` 加一行 |
| 加一个按钮 | `frontend/index.html` + `frontend/src/main.ts` |
| 改应用名/图标/版本信息 | `build/windows/winres.json` + `build/windows/installer.iss` |
| 改包名/输出名 | `scripts/build.sh` 的 `OUT` + `installer.iss` 的 `MyAppExeName` |
| 换前端框架（React/Vue/Svelte） | 只动 `frontend/`，Go 侧完全不用改 |
| 开真机调试端口 | 启动前设 `WINAPP_DEBUG_PORT=9222`，然后浏览器打开 `http://127.0.0.1:9222/json` |

---

## 七、踩过的坑（省你时间）

1. **`wails3` 在 Linux 上编译要 GTK4**（`gtk4` / `webkitgtk-6.0` pkg-config）。所以本项目**构建不依赖 wails3**，只有本地 `wails3 dev` 需要。
2. **新版 WebView2 Runtime 不再从 `WEBVIEW2_ADDITIONAL_BROWSER_ARGUMENTS` 读 `--remote-debugging-port`**，必须在代码里通过 `AdditionalBrowserArgs` 传（本项目的做法）。
3. **`go-winres --no-suffix` 静默失败**：不生成文件、退出码 0。
4. **图标 >256px 直接报错**，需要先缩放。
5. **Windows 专属代码不隔离**，`go vet` 在 Linux 上会报 `undefined: syscall.NewLazyDLL`（本项目 `internal/platform` 就是这么拆出来的）。
6. **`/sandbox/workspace` 是 virtiofs**，在里面装 `node_modules` 比本地盘慢一个数量级 —— `build.sh` 会先复制到 `/tmp` 再装。

---

## 八、License

MIT
