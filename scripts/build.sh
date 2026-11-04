#!/usr/bin/env bash
# 构建 Windows 单文件 exe。
#
#   bash scripts/build.sh                  # 版本取自 git tag，无 tag 则 0.0.0-dev
#   bash scripts/build.sh --version=0.1.0
#   bash scripts/build.sh --skip-frontend   # 前端已构建过
#
# 产物：dist/json-toolbox-windows-amd64.exe
#
# 资源（图标/清单/版本信息）优先用 wails3 generate syso（官方做法，CI 里可靠），
# 没有 wails3 时退回 go-winres。
set -euo pipefail

# Windows 上 Python 默认用 cp1252/gbk 输出，中文会直接抛 UnicodeEncodeError
export PYTHONIOENCODING=utf-8

ROOT=$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)
cd "$ROOT"

VERSION="${VERSION:-}"
SKIP_FE=0
for a in "$@"; do
  case "$a" in
    --skip-frontend) SKIP_FE=1 ;;
    --version=*) VERSION="${a#*=}" ;;
    -h|--help) sed -n '2,12p' "$0"; exit 0 ;;
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
  BUILD_DIR="${FE_BUILD_DIR:-/tmp/winapp-fe}"
  rm -rf "$BUILD_DIR"
  mkdir -p "$BUILD_DIR"
  cp -r frontend/. "$BUILD_DIR/"
  (
    cd "$BUILD_DIR"
    if command -v pnpm >/dev/null 2>&1; then
      pnpm install --no-frozen-lockfile
      pnpm run build
    else
      npm install --no-audit --no-fund
      npm run build
    fi
  )
  rm -rf frontend/dist
  cp -r "$BUILD_DIR/dist" frontend/dist
fi

[ -f frontend/dist/index.html ] || { echo "错误：缺少 frontend/dist/index.html（请先构建前端）" >&2; exit 1; }

echo "==> 生成 Windows 资源（图标 / 清单 / 版本信息）"
python3 - "$VERSION" <<'PY'
import json, pathlib, sys
v = sys.argv[1]
four = v if v.count('.') == 3 else (v + '.0' if v.count('.') == 2 else v)

# wails3 generate syso 读的 info.json
info_src = pathlib.Path('build/windows/info.json')
try:
    info = json.loads(info_src.read_text(encoding='utf-8'))
except Exception:
    info = {"fixed": {}, "info": {"0000": {}}}
info.setdefault('fixed', {})['file_version'] = four
info.setdefault('info', {}).setdefault('0000', {})
info['info']['0000'].update({
    'ProductVersion': v,
    'FileVersion': v,
    'CompanyName': 'rentianyu',
    'FileDescription': 'JSON Toolbox - JSON 格式化 / 压缩 / 结构分析',
    'ProductName': 'JSON Toolbox',
    'LegalCopyright': 'Copyright (c) 2026 rentianyu',
    'Comments': 'https://github.com/rentianyu/winapp',
})
pathlib.Path('build/windows/info.generated.json').write_text(
    json.dumps(info, ensure_ascii=False, indent=2), encoding='utf-8')

# go-winres 读的 winres.json
wr = json.loads(pathlib.Path('build/windows/winres.json').read_text(encoding='utf-8'))
fixed = wr['RT_VERSION']['#1']['0000']['fixed']
fixed['file_version'] = four
fixed['product_version'] = four
ri = wr['RT_VERSION']['#1']['0000']['info']['0409']
ri['FileVersion'] = v
ri['ProductVersion'] = v
pathlib.Path('build/windows/winres.generated.json').write_text(
    json.dumps(wr, ensure_ascii=False, indent=2), encoding='utf-8')
PY

rm -f rsrc*.syso wails_windows_*.syso

if command -v wails3 >/dev/null 2>&1; then
  echo "    使用 wails3 generate syso"
  (
    cd build
    wails3 generate syso \
      -arch amd64 \
      -icon windows/icon.ico \
      -manifest windows/wails.exe.manifest \
      -info windows/info.generated.json \
      -out ../wails_windows_amd64.syso
  )
  SYSO="wails_windows_amd64.syso"
elif command -v go-winres >/dev/null 2>&1; then
  echo "    使用 go-winres（fallback）"
  go-winres make --in build/windows/winres.generated.json --out rsrc --arch amd64
  SYSO="rsrc_windows_amd64.syso"
else
  echo "❌ 既没有 wails3 也没有 go-winres，无法生成图标与版本资源" >&2
  echo "   装一个即可：go install github.com/wailsapp/wails/v3/cmd/wails3@v3.0.0" >&2
  exit 1
fi

[ -f "$SYSO" ] || { echo "❌ 资源文件 $SYSO 未生成" >&2; exit 1; }
echo "    资源: $SYSO ($(wc -c < "$SYSO") B)"

echo "==> 编译（GOOS=windows GOARCH=amd64 CGO_ENABLED=0）"
mkdir -p dist
CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build \
  -trimpath -buildvcs=false -tags production \
  -ldflags "-w -s -H windowsgui -X main.version=${VERSION} -X main.commit=${COMMIT} -X main.builtAt=${BUILT_AT}" \
  -o "$OUT" .

rm -f rsrc*.syso wails_windows_*.syso build/windows/info.generated.json build/windows/winres.generated.json

echo "==> 产物验收"
# 版本信息校验依赖 pefile；CI 里通常已装，本地缺失时自动补一次
python3 -c "import pefile" 2>/dev/null || {
  echo "    未检测到 pefile，尝试安装"
  python3 -m pip install --quiet pefile 2>/dev/null || echo "    安装失败，版本信息校验将不可用"
}
python3 scripts/exe_peek.py "$OUT" --expect-version "$VERSION" || exit 1
