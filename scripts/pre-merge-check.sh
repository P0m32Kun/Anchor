#!/bin/bash
set -e

echo "=== Pre-Merge Check ==="

# 1. Go 编译
echo "[1/6] Go build..."
go build ./...

# 2. Go 测试
echo "[2/6] Go test..."
go test ./...

# 3. Go vet
echo "[3/6] Go vet..."
go vet ./...

# 4. 前端类型检查
echo "[4/6] Frontend typecheck..."
cd frontend
npm run typecheck

# 5. 前端构建
echo "[5/6] Frontend build..."
npm run build

# 6. Tauri 构建（可选，如果有 cargo）
echo "[6/6] Tauri check..."
cd ../src-tauri
if command -v cargo >/dev/null; then
    cargo check
fi

echo "=== All checks passed ==="
