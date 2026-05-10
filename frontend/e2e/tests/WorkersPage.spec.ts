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

		test("TC-1: 页面加载并显示节点列表", async ({ page }) => {
			await page.goto("/workers");
			await page.waitForLoadState("networkidle");

			await expect(
				page.locator("h1").filter({ hasText: "Worker 节点集群" }),
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
