#!/usr/bin/env bash
# 构建 Windows 单文件 exe（不依赖 wails3 CLI）。
#
#   bash scripts/build.sh                  # 版本取自 git tag，无 tag 则 0.0.0-dev
#   bash scripts/build.sh --version=0.1.0
#   bash scripts/build.sh --skip-frontend   # 前端已构建过
#
# 产物：dist/json-toolbox-windows-amd64.exe
set -euo pipefail

ROOT=$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)
cd "$ROOT"

VERSION="${VERSION:-}"
SKIP_FE=0
for a in "$@"; do
  case "$a" in
    --skip-frontend) SKIP_FE=1 ;;
    --version=*) VERSION="${a#*=}" ;;
    -h|--help) sed -n '2,10p' "$0"; exit 0 ;;
    *) echo "未知参数: $a" >&2; exit 2 ;;
  esac
done

if [ -z "$VERSION" ]; then
  if git describe --tags --abbrev=0 >/dev/null 2>&1; then
    VERSION=$(git describe --tags --abbrev=0 | sed 's/^v//')
  else
    VERSION="0.0.0-dev"
  fi
fi
COMMIT=$(git rev-parse --short HEAD 2>/dev/null || echo none)
BUILT_AT=$(date -u +%Y-%m-%dT%H:%M:%SZ)
OUT="dist/json-toolbox-windows-amd64.exe"

echo "==> 版本 ${VERSION} (commit ${COMMIT})"

if [ "$SKIP_FE" = "0" ]; then
  echo "==> 构建前端"
  # 在本地盘构建：/sandbox/workspace 是 virtiofs，装 node_modules 会慢一个数量级
  BUILD_DIR="${FE_BUILD_DIR:-/tmp/winapp-fe}"
  rm -rf "$BUILD_DIR"
  mkdir -p "$BUILD_DIR"
  cp -r frontend/. "$BUILD_DIR/"
  (
    cd "$BUILD_DIR"
    if [ -f package-lock.json ]; then
      npm ci --no-audit --no-fund
    else
      npm install --no-audit --no-fund
    fi
    npm run build
  )
  rm -rf frontend/dist
  cp -r "$BUILD_DIR/dist" frontend/dist
fi

[ -f frontend/dist/index.html ] || { echo "错误：缺少 frontend/dist/index.html" >&2; exit 1; }

echo "==> 生成 Windows 资源（图标 / 清单 / 版本信息）"
python3 - "$VERSION" <<'PY'
import json, pathlib, sys
v = sys.argv[1]
src = pathlib.Path('build/windows/winres.json')
d = json.loads(src.read_text(encoding='utf-8'))
four = v if v.count('.') == 3 else (v + '.0' if v.count('.') == 2 else v)
fixed = d['RT_VERSION']['#1']['0000']['fixed']
fixed['file_version'] = four
fixed['product_version'] = four
info = d['RT_VERSION']['#1']['0000']['info']['0409']
info['FileVersion'] = v
info['ProductVersion'] = v
out = pathlib.Path('build/windows/winres.generated.json')
out.write_text(json.dumps(d, ensure_ascii=False, indent=2), encoding='utf-8')
PY
go-winres make --in build/windows/winres.generated.json --out rsrc --arch amd64

echo "==> 编译（GOOS=windows GOARCH=amd64 CGO_ENABLED=0）"
mkdir -p dist
CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build \
  -trimpath -buildvcs=false -tags production \
  -ldflags "-w -s -H windowsgui -X main.version=${VERSION} -X main.commit=${COMMIT} -X main.builtAt=${BUILT_AT}" \
  -o "$OUT" .

rm -f rsrc_windows_amd64.syso build/windows/winres.generated.json

echo "==> 产物验收"
python3 scripts/exe_peek.py "$OUT" --expect-version "$VERSION" || exit 1
