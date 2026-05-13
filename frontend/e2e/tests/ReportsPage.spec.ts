import { expect, test } from "@playwright/test";
import { createProject } from "../fixtures/api-helpers";
import { cleanupTestData, setCurrentProject } from "../fixtures/db-utils";

test.describe
	.serial("ReportsPage", () => {
		test.beforeAll(async () => {
			await cleanupTestData();
		});

		test.afterAll(async () => {
			await cleanupTestData();
		});

		test("TC-1: 无项目时 LegacyRouteGuard 重定向到项目列表", async ({
			page,
		}) => {
			await page.goto("/reports");
			await page.waitForLoadState("networkidle");

			// LegacyRouteGuard 会在无 currentProjectId 时重定向到 /projects
			await expect(page).toHaveURL(/\/projects/);
			await expect(page.getByText("项目管理")).toBeVisible();
		});

		test("TC-2: 有项目时显示报告页面", async ({ page }) => {
			const project = await createProject({
				name: "Report Test Project",
				organization: "Test Org",
				purpose: "Report testing",
			});
			await setCurrentProject(page, project.id);

			await page.goto("/reports");
			await expect(page).toHaveURL(/\/projects\/.*\/reports/);
			await page.waitForLoadState("networkidle");

			await expect(page.getByText("安全评估报告")).toBeVisible();
			await expect(page.getByText("← 返回项目")).toBeVisible();
		});
	});
