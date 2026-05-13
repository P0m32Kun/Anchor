import { expect, test } from "@playwright/test";
import { createProject } from "../fixtures/api-helpers";
import { cleanupTestData, setCurrentProject } from "../fixtures/db-utils";

test.describe
	.serial("FindingsPage", () => {
		test.beforeAll(async () => {
			await cleanupTestData();
		});

		test.afterAll(async () => {
			await cleanupTestData();
		});

		test("空状态：无 Finding 时显示引导文案", async ({ page }) => {
			const project = await createProject({
				name: "Findings Test Project",
				organization: "Test Org",
				purpose: "Findings testing",
			});
			await setCurrentProject(page, project.id);

			await page.goto("/findings");
			await expect(page).toHaveURL(/\/projects\/.*\/findings/);
			// 标题可见
			await expect(
				page.locator("h1").filter({ hasText: "Findings" }),
			).toBeVisible();
			// 空状态文案存在（EmptyState 渲染为 <h3>）
			await expect(
				page.getByRole("heading", { name: "未找到漏洞发现" }),
			).toBeVisible();
		});

		test("搜索框 + 严重度筛选器可交互", async ({ page }) => {
			const project = await createProject({
				name: "Findings Search Project",
				organization: "Test Org",
				purpose: "Findings filters",
			});
			await setCurrentProject(page, project.id);

			await page.goto("/findings");
			await expect(page).toHaveURL(/\/projects\/.*\/findings/);

			// 搜索框
			await page
				.getByPlaceholder("搜索标题或描述...")
				.fill("test query");
			await expect(
				page.getByPlaceholder("搜索标题或描述..."),
			).toHaveValue("test query");

			// 严重度筛选下拉（Select 组件）
			const severitySelect = page.locator("select").first();
			await expect(severitySelect).toBeVisible();
			await severitySelect.selectOption("critical");
			await expect(severitySelect).toHaveValue("critical");

			// 状态筛选下拉
			const statusSelect = page.locator("select").nth(1);
			await expect(statusSelect).toBeVisible();
			await statusSelect.selectOption("pending_review");
			await expect(statusSelect).toHaveValue("pending_review");
		});

		test("后端暂无数据时筛选组合保持页面不崩溃", async ({ page }) => {
			const project = await createProject({
				name: "Findings Empty Filter",
				organization: "Test Org",
				purpose: "Filter resilience",
			});
			await setCurrentProject(page, project.id);

			await page.goto("/findings");
			// 同时应用筛选 + 搜索，页面不应崩溃
			await page.locator("select").first().selectOption("critical");
			await page.locator("select").nth(1).selectOption("pending_review");
			await page
				.getByPlaceholder("搜索标题或描述...")
				.fill("nonexistent-finding");
			await page.waitForTimeout(500);

			// 页面主体仍在
			await expect(page.locator("body")).toBeVisible();
		});
	});
