import { expect, test } from "@playwright/test";
import { createProject } from "../fixtures/api-helpers";
import { cleanupTestData } from "../fixtures/db-utils";

async function setCurrentProject(page: any, projectId: string) {
	await page.goto("/");
	await page.evaluate((id: string) => {
		localStorage.setItem(
			"app-store",
			JSON.stringify({ state: { currentProjectId: id }, version: 0 }),
		);
	}, projectId);
	await page.reload();
}

test.describe
	.serial("AssetPage", () => {
		test.beforeAll(async () => {
			await cleanupTestData();
		});

		test.afterAll(async () => {
			await cleanupTestData();
		});

		test("TC-1: 页面加载并显示资产清单", async ({ page }) => {
			const project = await createProject({
				name: "Asset Test Project",
				organization: "Test Org",
				purpose: "Asset testing",
			});
			await setCurrentProject(page, project.id);

			await page.goto("/assets");
			await expect(page).toHaveURL(/\/projects\/.*\/assets/);
			await expect(page.getByText("资产清单")).toBeVisible();
		});

		test("TC-2: 筛选输入框可用", async ({ page }) => {
			const project = await createProject({
				name: "Asset Filter Project",
				organization: "Test Org",
				purpose: "Asset filter testing",
			});
			await setCurrentProject(page, project.id);

			await page.goto("/assets");
			await expect(page).toHaveURL(/\/projects\/.*\/assets/);

			await page.getByPlaceholder("筛选资产值...").fill("test-asset");
			await expect(page.getByPlaceholder("筛选资产值...")).toHaveValue(
				"test-asset",
			);
		});
	});
