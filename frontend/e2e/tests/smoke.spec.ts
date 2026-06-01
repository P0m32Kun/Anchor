import { expect, test } from "@playwright/test";
import { cleanupTestData } from "../fixtures/db-utils";

test.describe
	.serial("Smoke Test — 核心主流程", () => {
		test.beforeAll(async () => {
			await cleanupTestData();
		});

		test.afterAll(async () => {
			await cleanupTestData();
		});

		test("完整流程：Dashboard → 创建项目 → 导入目标 → 查看 Runs", async ({
			page,
		}) => {
			// ── Step 1: Dashboard ──
			await page.goto("/");
			await expect(page.getByText("安全工作台")).toBeVisible();

			// 点击"新建项目"导航到项目列表
			await page.getByRole("button", { name: "新建项目" }).click();
			await expect(page).toHaveURL(/\/projects/);

			// ── Step 2: 创建项目 ──
			await page.getByPlaceholder("例如：2024 Q2 外部红队评估").fill("Smoke Test Project");
			await page.getByPlaceholder("客户名称或部门").fill("Smoke Test Org");
			await page.getByPlaceholder("测试目的或项目背景").fill("End-to-end smoke test");

			await page.getByRole("button", { name: "创建项目", exact: true }).click();

			// 验证新项目出现在列表中
			await expect(page.getByRole("heading", { name: "Smoke Test Project" })).toBeVisible();
			await expect(page.getByText("Smoke Test Org")).toBeVisible();

			// ── Step 3: 进入目标管理 ──
			// 点击项目卡片进入项目详情（会自动设置 currentProject）
			await page.getByRole("heading", { name: "Smoke Test Project" }).click();

			// 导航到目标管理页（LegacyRouteGuard 会重定向到 /projects/:id/targets）
			await page.goto("/targets");
			await expect(page).toHaveURL(/\/projects\/.*\/targets/);
			await expect(page.getByRole("heading", { name: "目标与 Scope" })).toBeVisible({ timeout: 10000 });

			// ── Step 4: 导入目标 ──
			// 选择目标类型为域名（使用更精确的选择器）
			await page.locator("select").first().selectOption("domain");
			await page
				.getByPlaceholder("example.com")
				.first()
				.fill("smoke-test.example.com");
			await page.getByRole("button", { name: "添加" }).first().click();

			// 处理可能的 Scope 确认弹窗
			const scopeDialog = page.getByText("添加规则并继续");
			if (await scopeDialog.isVisible().catch(() => false)) {
				await scopeDialog.click();
				await page.waitForLoadState("networkidle");
			}

			// 验证目标出现在列表中
			await expect(page.getByText("smoke-test.example.com")).toBeVisible();

			// ── Step 5: 查看 Runs ──
			await page.goto("/runs");
			await expect(page).toHaveURL(/\/projects\/.*\/runs/);
			await expect(page.getByText("扫描执行")).toBeVisible();
		});
	});
