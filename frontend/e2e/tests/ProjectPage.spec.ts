import { expect, test } from "@playwright/test";
import { createProject } from "../fixtures/api-helpers";
import { cleanupTestData } from "../fixtures/db-utils";

test.describe
	.serial("ProjectPage", () => {
		test.beforeAll(async () => {
			await cleanupTestData();
		});

		test.afterAll(async () => {
			await cleanupTestData();
		});

		test("TC-1: 项目列表加载（空状态）", async ({ page }) => {
			await page.goto("/projects");
			await page.waitForLoadState("networkidle");

			await expect(page.getByText("项目管理")).toBeVisible();
			await expect(page.getByPlaceholder("项目名称 *")).toBeVisible();
			await expect(page.getByPlaceholder("组织/客户")).toBeVisible();
			await expect(page.getByPlaceholder("目的/描述")).toBeVisible();
		});

		test("TC-2: 创建项目表单", async ({ page }) => {
			await page.goto("/projects");
			await page.waitForLoadState("networkidle");

			await page.getByPlaceholder("项目名称 *").fill("E2E Test Project");
			await page.getByPlaceholder("组织/客户").fill("Test Org");
			await page.getByPlaceholder("目的/描述").fill("End-to-end test");

			await page.getByRole("button", { name: "创建项目", exact: true }).click();

			// 验证新项目出现在列表中
			await expect(page.getByText("E2E Test Project")).toBeVisible();
			await expect(page.getByText("Test Org")).toBeVisible();

			// 输入框应被清空
			await expect(page.getByPlaceholder("项目名称 *")).toHaveValue("");
		});

		test("TC-3: 删除项目", async ({ page }) => {
			await createProject({
				name: "Delete Me Project",
				organization: "Delete Org",
				purpose: "To be deleted",
			});

			await page.goto("/projects");
			await page.waitForLoadState("networkidle");
			await expect(page.getByText("Delete Me Project")).toBeVisible();

			// 点击删除按钮（在项目卡片内的垃圾桶图标）
			await page
				.locator(".card-dark", { hasText: "Delete Me Project" })
				.first()
				.getByTitle("删除项目")
				.click();

			// 确认对话框
			await expect(page.getByText("确认删除")).toBeVisible();
			await page.getByRole("button", { name: "删除", exact: true }).click();

			// 项目从列表中消失（使用精确选择器避免匹配对话框内的文本）
			await expect(
				page.locator(".card-dark", { hasText: "Delete Me Project" }),
			).not.toBeVisible();
		});
	});
