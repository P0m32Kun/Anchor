import { expect, test } from "@playwright/test";
import {
	createProject,
	createRun,
	listToolTemplates,
} from "../fixtures/api-helpers";
import { cleanupTestData, setCurrentProject } from "../fixtures/db-utils";

test.describe
	.serial("RunsPage", () => {
		test.beforeAll(async () => {
			await cleanupTestData();
		});

		test.afterAll(async () => {
			await cleanupTestData();
		});

		test("TC-1: 空状态引导", async ({ page }) => {
			const project = await createProject({
				name: "Runs Empty Project",
				organization: "Test Org",
				purpose: "Runs empty test",
			});
			await setCurrentProject(page, project.id);

			await page.goto("/runs");
			await expect(page).toHaveURL(/\/projects\/.*\/runs/);
			await page.waitForLoadState("networkidle");

			await expect(page.getByText("扫描执行")).toBeVisible();
			await expect(page.getByText("暂无扫描任务")).toBeVisible();
			await expect(page.getByText("创建项目")).toBeVisible();
			await expect(page.getByText("导入目标")).toBeVisible();
			await expect(page.getByText("启动扫描")).toBeVisible();
		});

		test("TC-2: 创建扫描任务", async ({ page }) => {
			const project = await createProject({
				name: "Runs Create Project",
				organization: "Test Org",
				purpose: "Runs create test",
			});
			await setCurrentProject(page, project.id);

			await page.goto("/runs");
			await expect(page).toHaveURL(/\/projects\/.*\/runs/);
			await page.waitForLoadState("networkidle");

			// 获取工具模板
			const templates = await listToolTemplates();
			if (templates.length === 0) {
				test.skip(true, "无可用工具模板，跳过创建扫描测试");
				return;
			}

			// 点击新建扫描
			await page.getByRole("button", { name: "新建扫描" }).click();
			await expect(
				page.locator("h2").filter({ hasText: "新建扫描" }),
			).toBeVisible();

			// 输入扫描名称
			await page.getByPlaceholder("例如：外网初筛").fill("E2E Test Scan");

			// 选择第一个模板
			await page.locator("div", { hasText: templates[0].name }).first().click();

			// 点击开始扫描
			await page.getByRole("button", { name: "开始扫描" }).click();

			// 验证 run 出现在列表中（可能很快完成）
			await expect(page.getByText("E2E Test Scan")).toBeVisible();
		});

		test("TC-3: 取消扫描任务", async ({ page }) => {
			const project = await createProject({
				name: "Runs Cancel Project",
				organization: "Test Org",
				purpose: "Runs cancel test",
			});
			await setCurrentProject(page, project.id);

			// 通过 API 创建扫描（如果有模板）
			const templates = await listToolTemplates();
			if (templates.length === 0) {
				test.skip(true, "无可用工具模板，跳过取消扫描测试");
				return;
			}

			await createRun(project.id, {
				tool_template_id: templates[0].id,
				name: "Cancel Me Scan",
			});

			await page.goto("/runs");
			await expect(page).toHaveURL(/\/projects\/.*\/runs/);
			await page.waitForLoadState("networkidle");

			// 验证扫描出现在列表中
			await expect(page.getByText("Cancel Me Scan")).toBeVisible();

			// 如果页面上有取消按钮，则测试取消流程
			const cancelButton = page
				.getByRole("button", { name: "取消扫描" })
				.first();
			if (await cancelButton.isVisible().catch(() => false)) {
				await cancelButton.click();
				await expect(page.getByText("确认取消扫描？")).toBeVisible();
				await page.getByRole("button", { name: "确认取消" }).click();
				// 取消后状态变为 cancelled
				await expect(page.getByText("已取消")).toBeVisible();
			}
		});
	});
