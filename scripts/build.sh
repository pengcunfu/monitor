#!/usr/bin/env bash
# 熔岩网络安全事件应急处置系统 一键构建脚本
# 用法：bash scripts/build.sh [版本号]
# 产出：bin/monitor-linux-amd64、monitor-windows-amd64.exe 与 tar.gz
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
VERSION="${1:-$(date +%Y%m%d)}"
OUT_DIR="$ROOT/bin"

echo "==> 1/3 构建前端"
cd "$ROOT/frontend"
npm ci
npm run build

echo "==> 2/3 交叉编译后端（CGO_ENABLED=0，静态链接）"
mkdir -p "$OUT_DIR"
cd "$ROOT/backend"
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -ldflags="-s -w" -o "$OUT_DIR/monitor-linux-amd64" ./cmd/server
CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -trimpath -ldflags="-s -w" -o "$OUT_DIR/monitor-windows-amd64.exe" ./cmd/server

echo "==> 3/3 打包"
cd "$ROOT"
tar czf "monitor-linux-amd64-${VERSION}.tar.gz" \
  -C "$ROOT/bin" monitor-linux-amd64 \
  -C "$ROOT" deploy/monitor.yaml deploy/monitor.service README.md

echo "==> 完成：bin/monitor-linux-amd64、monitor-windows-amd64.exe 与 monitor-linux-amd64-${VERSION}.tar.gz"
