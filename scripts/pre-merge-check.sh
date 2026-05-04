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
echo "[6/7] Tauri check..."
cd ../src-tauri
if command -v cargo >/dev/null; then
	cargo check
fi

# 7. E2E 测试（可选，需要 Docker + Chromium）
echo "[7/7] E2E tests (optional)..."
cd ../frontend
if command -v docker >/dev/null 2>&1 && docker info >/dev/null 2>&1; then
	if [ -d "node_modules/.bin/playwright" ] || [ -f "node_modules/@playwright/test/package.json" ]; then
		if npx playwright install --with-deps chromium --dry-run >/dev/null 2>&1; then
			npx playwright test --reporter=line || {
				echo "⚠️  E2E tests failed. Fix before merging if the change affects UI."
				exit 1
			}
		else
			echo "⚠️  Chromium not installed. Skipping E2E tests. Run: npx playwright install chromium"
		fi
	else
		echo "⚠️  Playwright not installed. Skipping E2E tests."
	fi
else
	echo "⚠️  Docker not running. Skipping E2E tests."
fi

echo "=== All checks passed ==="
