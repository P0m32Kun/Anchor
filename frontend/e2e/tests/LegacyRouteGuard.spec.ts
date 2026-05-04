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
	.serial("LegacyRouteGuard", () => {
		test.beforeAll(async () => {
			await cleanupTestData();
		});

		test.afterAll(async () => {
			await cleanupTestData();
		});

		test("TC-1: 无项目时 legacy 路由重定向到 /projects", async ({ page }) => {
			await page.goto("/targets");
			await page.waitForLoadState("networkidle");
			await expect(page).toHaveURL(/\/projects/);
		});

		test("TC-2: 有项目时 legacy 路由重定向到嵌套路由", async ({ page }) => {
			const project = await createProject({
				name: "Legacy Route Project",
				organization: "Test Org",
				purpose: "Legacy route test",
			});
			await setCurrentProject(page, project.id);

			await page.goto("/targets");
			await expect(page).toHaveURL(/\/projects\/.*\/targets/);
			await expect(page.getByText("目标管理")).toBeVisible();
		});
	});
