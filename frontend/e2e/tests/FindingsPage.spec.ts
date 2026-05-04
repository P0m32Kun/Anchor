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
	.serial("FindingsPage", () => {
		test.beforeAll(async () => {
			await cleanupTestData();
		});

		test.afterAll(async () => {
			await cleanupTestData();
		});

		test("TC-1: 页面加载并显示空状态", async ({ page }) => {
			const project = await createProject({
				name: "Findings Test Project",
				organization: "Test Org",
				purpose: "Findings testing",
			});
			await setCurrentProject(page, project.id);

			await page.goto("/findings");
			await expect(page).toHaveURL(/\/projects\/.*\/findings/);
			await expect(
				page.locator("h1").filter({ hasText: "Findings" }),
			).toBeVisible();
			await expect(page.getByText("暂无 Finding")).toBeVisible();
		});

		test("TC-2: 搜索框可用", async ({ page }) => {
			const project = await createProject({
				name: "Findings Search Project",
				organization: "Test Org",
				purpose: "Findings search testing",
			});
			await setCurrentProject(page, project.id);

			await page.goto("/findings");
			await expect(page).toHaveURL(/\/projects\/.*\/findings/);

			await page.getByPlaceholder("搜索标题或描述...").fill("test query");
			await expect(page.getByPlaceholder("搜索标题或描述...")).toHaveValue(
				"test query",
			);
		});
	});
