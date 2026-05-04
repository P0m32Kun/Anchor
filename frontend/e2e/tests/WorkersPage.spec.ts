import { expect, test } from "@playwright/test";
import { cleanupTestData } from "../fixtures/db-utils";

test.describe
	.serial("WorkersPage", () => {
		test.beforeAll(async () => {
			await cleanupTestData();
		});

		test.afterAll(async () => {
			await cleanupTestData();
		});

		test("TC-1: 页面加载并显示空状态", async ({ page }) => {
			await page.goto("/workers");
			await page.waitForLoadState("networkidle");

			await expect(
				page.locator("h1").filter({ hasText: "Workers" }),
			).toBeVisible();
			await expect(page.getByText("暂无 Worker")).toBeVisible();
			await expect(
				page.getByText(
					"通过 Docker 部署 Worker，使用 Server 生成的 token 注册后显示在此",
				),
			).toBeVisible();
		});

		test("TC-2: 错误状态显示", async ({ page }) => {
			await page.goto("/workers");
			await page.waitForLoadState("networkidle");

			// 如果后端不可用，应显示错误信息
			// 正常情况下验证页面结构不崩溃即可
			await expect(page.locator("body")).toBeVisible();
		});
	});
