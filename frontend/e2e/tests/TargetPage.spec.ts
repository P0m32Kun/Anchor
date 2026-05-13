import { expect, test } from "@playwright/test";
import { createProject } from "../fixtures/api-helpers";
import { cleanupTestData, setCurrentProject } from "../fixtures/db-utils";

test.describe
	.serial("TargetPage", () => {
		test.beforeAll(async () => {
			await cleanupTestData();
		});

		test.afterAll(async () => {
			await cleanupTestData();
		});

		test("TC-1: 页面加载并显示目标管理", async ({ page }) => {
			const project = await createProject({
				name: "Target Test Project",
				organization: "Test Org",
				purpose: "Target testing",
			});
			await setCurrentProject(page, project.id);

			await page.goto("/targets");
			await expect(page).toHaveURL(/\/projects\/.*\/targets/);
			await expect(page.getByText("目标管理")).toBeVisible();
			await expect(page.getByText("暂无目标")).toBeVisible();
		});

		test("TC-2: 添加目标", async ({ page }) => {
			const project = await createProject({
				name: "Add Target Project",
				organization: "Test Org",
				purpose: "Add target testing",
			});
			await setCurrentProject(page, project.id);

			await page.goto("/targets");
			await expect(page).toHaveURL(/\/projects\/.*\/targets/);

			// 选择域名类型
			await page.locator("select").first().selectOption("domain");
			await page
				.getByPlaceholder(/example\.com 或 192\.168/)
				.first()
				.fill("e2e-target.example.com");
			await page.getByRole("button", { name: "添加" }).first().click();

			// 验证目标出现在列表中
			await expect(page.getByText("e2e-target.example.com")).toBeVisible();
		});
	});
