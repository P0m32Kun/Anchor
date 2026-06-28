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

# 6. E2E 测试（可选，需要 Docker + Chromium + running dev server）
echo "[6/6] E2E tests (optional)..."
# already in frontend/

# 检查 localhost:1420 是否有 dev server 在运行（否则 Playwright webServer 会因没有 Go backend 而超时）
if ! curl -sf http://localhost:1420 > /dev/null 2>&1; then
	echo "⚠️  Local dev server (localhost:1420) not detected. Skipping E2E tests."
elif command -v docker >/dev/null 2>&1 && docker info >/dev/null 2>&1; then
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
