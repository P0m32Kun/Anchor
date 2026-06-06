# Anchor Frontend E2E Tests

基于 **Playwright** 的自动化端到端测试，覆盖全部页面和核心用户流程。

> 旧的 `.e2e.md` 手工测试计划文档仍保留在 `tests/` 目录中作为参考，但不再维护。
> 所有新的测试用例应编写为 `.spec.ts` 文件。

## Prerequisites

```bash
# 1. 安装 Playwright（已包含在 package.json devDependencies 中）
npm install

# 2. 安装 Chromium 浏览器（首次需要，约 100MB）
npx playwright install chromium

# 3. 确保 Docker 正在运行（后端通过 docker-compose 启动）
docker info
```

## Running Tests

### 快速套件（页面 + smoke，约数分钟）

```bash
cd frontend
npm run test:e2e          # --project=chromium
# 或从仓库根目录
make test-e2e-smoke
```

### 长扫描套件（单 spec 最长 30 分钟，勿与 globalTimeout 混用）

```bash
make test-e2e-scan        # chromium-scan + chromium-auth
npm run test:e2e:scan

# 单条 pipeline
make test-e2e-full        # 仅 full-flow
npx playwright test e2e/tests/sse-realtime.spec.ts --project=chromium-scan
```

Playwright 会自动：

1. 通过 `globalSetup` 启动 Docker 后端（如果未运行）
2. 生成本地 `storage-state.json`，注入 API Base 和测试 token
3. 通过 `webServer` 启动前端 dev server
4. 按 project 执行 `.spec.ts`（`chromium` 与 `chromium-scan` 分离）
5. 通过 `globalTeardown` 停止 Docker 后端并删除 `storage-state.json`

### 交互式调试（UI 模式）

```bash
npm run test:e2e:ui
```

### 调试单个测试

```bash
npx playwright test tests/DashboardPage.spec.ts --debug
```

### 生成 HTML 报告

```bash
npm run test:e2e
npx playwright show-report
```

## Test Coverage

| 模块           | 测试文件                      | 覆盖内容                                                        |
| -------------- | ----------------------------- | --------------------------------------------------------------- |
| **Dashboard**  | `tests/DashboardPage.spec.ts` | 空状态、统计卡片、最近活动、待处理 Findings、快速操作、错误重试 |
| **Smoke Test** | `tests/smoke.spec.ts`         | 完整主流程：Dashboard → 创建项目 → 导入目标 → 查看 Runs         |

## 测试基础设施

```
e2e/
├── global-setup.ts          # 测试前：启动 Docker 后端，等待 healthcheck
├── global-teardown.ts       # 测试后：停止 Docker 后端，清理 storage-state
├── fixtures/
│   ├── api-helpers.ts       # 直接调用后端 API 的工具函数
│   ├── db-utils.ts          # 数据库清理/重置工具
│   └── test-data.ts         # 测试数据工厂
├── tests/
│   ├── DashboardPage.spec.ts
│   ├── smoke.spec.ts
│   └── *.e2e.md             # 旧的手工测试计划（保留参考）
└── README.md
```

`storage-state.json` 是运行时生成文件，包含本地测试 token，不应提交到 Git。

## Fixtures 使用示例

```typescript
import { test, expect } from "@playwright/test";
import { createProject, addTarget } from "../fixtures/api-helpers";
import { cleanupTestData } from "../fixtures/db-utils";

test.beforeEach(async () => {
  await cleanupTestData(); // 每个测试前清理数据
});

test("my test", async ({ page }) => {
  await createProject({ name: "Test Project" });
  await page.goto("/");
  await expect(page.getByText("Test Project")).toBeVisible();
});
```

## 添加新测试

1. 在 `e2e/tests/` 下创建 `<FeatureName>.spec.ts`
2. 使用 `test.beforeEach` 调用 `cleanupTestData()` 确保数据隔离
3. 使用 `page.goto()` + `expect()` 进行断言
4. 使用 fixtures 中的 API helpers 准备测试数据
5. 运行 `npx playwright test <file> --list` 确认测试被发现

## Troubleshooting

| 问题                                       | 解决方案                                               |
| ------------------------------------------ | ------------------------------------------------------ |
| `Executable doesn't exist at ... chromium` | 运行 `npx playwright install chromium`                 |
| `Docker daemon is not running`             | 启动 Docker Desktop 或 Docker daemon                   |
| `Backend did not become healthy`           | 检查 `docker-compose logs server`                      |
| 测试间数据污染                             | 确保每个 `test` 或 `describe` 都有 `cleanupTestData()` |
