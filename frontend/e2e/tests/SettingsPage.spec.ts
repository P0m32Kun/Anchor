import { expect, test } from "@playwright/test";
import { cleanupTestData } from "../fixtures/db-utils";

test.describe
	.serial("SettingsPage", () => {
		test.beforeAll(async () => {
			await cleanupTestData();
		});

		test.afterAll(async () => {
			await cleanupTestData();
		});

		test("TC-1: 页面渲染并显示所有设置区域", async ({ page }) => {
			await page.goto("/settings");
			await page.waitForLoadState("networkidle");

			await expect(
				page.locator("h1").filter({ hasText: "Settings" }),
			).toBeVisible();
			await expect(page.getByText("应用配置和偏好设置")).toBeVisible();

			// Server 地址
			await expect(page.getByText("Server 地址")).toBeVisible();
			await expect(
				page.getByRole("button", { name: "保存并刷新" }),
			).toBeVisible();
			await expect(page.getByRole("button", { name: "重置" })).toBeVisible();

			// API Token
			await expect(page.getByText("API Token")).toBeVisible();
			await expect(page.getByPlaceholder("输入新的 API Token")).toBeVisible();

			// Scan Config 已移至项目内页面，SettingsPage 不再包含扫描配置
		});

		test("TC-2: Server URL 保存和重置", async ({ page }) => {
			await page.goto("/settings");
			await page.waitForLoadState("networkidle");

			// 修改 URL
			await page.locator("input").first().fill("http://localhost:9999");

			// 保存按钮可点击
			await page.getByRole("button", { name: "保存并刷新" }).click();

			// 页面会刷新（window.location.reload），等待稳定
			await page.waitForLoadState("networkidle");

			// 重置
			await page.getByRole("button", { name: "重置" }).click();
			await expect(page.locator("input").first()).not.toHaveValue(
				"http://localhost:9999",
			);
		});

	});
