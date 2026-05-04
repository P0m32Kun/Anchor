import { expect, test } from "@playwright/test";
import { createProject } from "../fixtures/api-helpers";
import { cleanupTestData } from "../fixtures/db-utils";

test.describe
	.serial("ProjectLayout", () => {
		test.beforeAll(async () => {
			await cleanupTestData();
		});

		test.afterAll(async () => {
			await cleanupTestData();
		});

		test("TC-1: 嵌套路由正常加载", async ({ page }) => {
			const project = await createProject({
				name: "Layout Test Project",
				organization: "Test Org",
				purpose: "Layout test",
			});

			await page.goto(`/projects/${project.id}/targets`);
			await page.waitForLoadState("networkidle");

			await expect(page.getByText("目标管理")).toBeVisible();
		});

		test("TC-2: 项目不存在时显示错误并跳转", async ({ page }) => {
			await page.goto("/projects/nonexistent-uuid/targets");
			await page.waitForLoadState("networkidle");

			await expect(
				page.getByRole("heading", { name: "项目不存在" }).first(),
			).toBeVisible();
		});
	});
