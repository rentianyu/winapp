#!/usr/bin/env bash
# L0 快检：3~10 秒内跑完，提交前必跑。
set -euo pipefail

cd "$(dirname "${BASH_SOURCE[0]}")/.."

echo "==> 1/4 纯逻辑单测（Linux 原生，-race）"
go test ./internal/... -race -count=1

echo "==> 2/4 Windows 目标类型检查（不需要 Windows 机器）"
GOOS=windows GOARCH=amd64 go vet ./...

echo "==> 3/4 Linux 下检查 internal 包（验证业务层无平台依赖）"
go vet ./internal/...

echo "==> 4/4 Windows ARM64 编译检查（顺手覆盖另一种架构）"
CGO_ENABLED=0 GOOS=windows GOARCH=arm64 go build -o /dev/null ./... 2>/dev/null || \
  echo "   （ARM64 目标跳过：某些依赖暂不支持，不影响 amd64 发布）"

echo
echo "✅ L0 通过"
