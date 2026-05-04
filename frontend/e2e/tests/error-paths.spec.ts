import { expect, test } from "@playwright/test";
import { cleanupTestData } from "../fixtures/db-utils";

test.describe
	.serial("Error Paths", () => {
		test.beforeAll(async () => {
			await cleanupTestData();
		});

		test.afterAll(async () => {
			await cleanupTestData();
		});

		test("TC-1: 访问不存在的项目路径", async ({ page }) => {
			await page.goto("/projects/nonexistent-uuid-1234/targets");
			await page.waitForLoadState("networkidle");

			// 项目不存在时应显示 EmptyState 或 Toast 提示
			await expect(
				page
					.getByRole("heading", { name: "项目不存在" })
					.first()
					.or(page.getByText("请先选择一个项目").first()),
			).toBeVisible();
		});

		test("TC-2: Dashboard 页面加载不崩溃", async ({ page }) => {
			await page.goto("/");
			await page.waitForLoadState("networkidle");

			// 无论是否有数据，页面都应正常渲染
			await expect(page.locator("nav")).toBeVisible();
			await expect(page.getByText("Dashboard").first()).toBeVisible();
		});

		test("TC-3: 后端未运行时的错误提示 @skip-ci", async () => {
			// 此测试需要手动停止后端，CI 环境中跳过
			test.skip(true, "需要手动控制后端生命周期");
		});

		test("TC-4: 网络断开恢复 @skip-ci", async () => {
			// 此测试需要模拟网络断开，CI 环境中跳过
			test.skip(true, "需要网络模拟工具支持");
		});
	});
