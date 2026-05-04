import { expect, test } from "@playwright/test";
import { cleanupTestData } from "../fixtures/db-utils";

test.describe
	.serial("App Navigation", () => {
		test.beforeAll(async () => {
			await cleanupTestData();
		});

		test.afterAll(async () => {
			await cleanupTestData();
		});

		test("TC-1: 导航栏链接都能正常加载", async ({ page }) => {
			await page.goto("/");
			await page.waitForLoadState("networkidle");

			// Projects
			await page.getByRole("link", { name: "Projects" }).click();
			await expect(page).toHaveURL(/\/projects/);
			await expect(page.getByText("项目管理")).toBeVisible();

			// Workers
			await page.getByRole("link", { name: "Workers" }).click();
			await expect(page).toHaveURL(/\/workers/);
			await expect(
				page.locator("h1").filter({ hasText: "Workers" }),
			).toBeVisible();

			// Settings
			await page.getByRole("link", { name: "Settings" }).click();
			await expect(page).toHaveURL(/\/settings/);
			await expect(
				page.locator("h1").filter({ hasText: "Settings" }),
			).toBeVisible();

			// Dashboard
			await page.getByRole("link", { name: "Dashboard" }).click();
			await expect(page).toHaveURL(/\/$/);
			await expect(page.locator("nav")).toBeVisible();
		});
	});
