#!/bin/bash
set -e

echo "=== Pre-Merge Check ==="

# 1. 文档一致性
echo "[1/7] Documentation..."
node --test scripts/check-docs.test.mjs
node scripts/check-docs.mjs

# 2. Go 编译
echo "[2/7] Go build..."
go build ./...

# 3. Go 测试
echo "[3/7] Go test..."
go test ./...

# 4. Go vet
echo "[4/7] Go vet..."
go vet ./...

# 5. 前端类型检查
echo "[5/7] Frontend typecheck..."
cd frontend
npm run typecheck

# 6. 前端构建
echo "[6/7] Frontend build..."
npm run build

# 7. E2E 测试（可选，需要 Docker + Chromium + running dev server）
echo "[7/7] E2E tests (optional)..."
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
