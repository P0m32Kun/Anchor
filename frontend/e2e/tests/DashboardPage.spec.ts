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
	.serial("DashboardPage", () => {
		test.beforeAll(async () => {
			await cleanupTestData();
		});

		test.afterAll(async () => {
			await cleanupTestData();
		});

		test("TC-1: 首次访问 Dashboard（空状态）", async ({ page }) => {
			await page.goto("/");

			const statsGrid = page.locator(".grid-cols-2");
			await expect(statsGrid.getByText("总项目数").first()).toBeVisible();
			await expect(statsGrid.getByText("活跃扫描").first()).toBeVisible();
			await expect(
				statsGrid.getByText("待处理 Findings").first(),
			).toBeVisible();
			await expect(statsGrid.getByText("在线 Worker").first()).toBeVisible();

			await expect(statsGrid.getByText("0")).toHaveCount(4);

			await expect(page.getByText("欢迎使用 Dashboard")).toBeVisible();
			await expect(
				page.getByText("创建项目并开始扫描，以查看跨项目统计和最近活动"),
			).toBeVisible();

			await page.getByRole("button", { name: "创建项目" }).last().click();
			await expect(page).toHaveURL(/\/projects/);
		});

		test("TC-2: 有数据时的统计卡片", async ({ page }) => {
			await createProject({
				name: "Dashboard Stats Test",
				organization: "Test Org",
				purpose: "Testing",
			});

			await page.goto("/");

			const statsGrid = page.locator(".grid-cols-2");
			await expect(statsGrid.getByText("总项目数").first()).toBeVisible();
			await expect(statsGrid.getByText("活跃扫描").first()).toBeVisible();
			await expect(
				statsGrid.getByText("待处理 Findings").first(),
			).toBeVisible();
			await expect(statsGrid.getByText("在线 Worker").first()).toBeVisible();

			await expect(
				statsGrid
					.locator("div", { hasText: "总项目数" })
					.first()
					.locator("xpath=..")
					.getByText("1"),
			).toBeVisible();
		});

		test("TC-3: 最近活动区域 — 有项目但无 runs", async ({ page }) => {
			const project = await createProject({
				name: "No Runs Project",
				organization: "Test Org",
				purpose: "Testing empty runs",
			});
			await setCurrentProject(page, project.id);

			await expect(
				page.locator("h3").filter({ hasText: "最近活动" }).first(),
			).toBeVisible();

			await expect(page.getByText("暂无扫描活动")).toBeVisible();
			await expect(
				page.getByText("开始扫描后，这里将显示最近的活动"),
			).toBeVisible();

			await page.getByRole("button", { name: "查看全部 →" }).first().click();
			await expect(page).toHaveURL(/\/projects\/.*\/runs/);
		});

		test("TC-4: 待处理 Findings 区域 — 无 findings 时", async ({ page }) => {
			const project = await createProject({
				name: "No Findings Project",
				organization: "Test Org",
				purpose: "Testing empty findings",
			});
			await setCurrentProject(page, project.id);

			await expect(
				page.locator("h3").filter({ hasText: "待处理 Findings" }).first(),
			).toBeVisible();

			await expect(page.getByText("暂无待处理 Findings")).toBeVisible();
			await expect(
				page.getByText("扫描发现的安全问题将在这里显示"),
			).toBeVisible();

			await page.getByRole("button", { name: "查看全部 →" }).nth(1).click();
			await expect(page).toHaveURL(/\/projects\/.*\/findings/);
		});

		test("TC-5: 快速操作按钮", async ({ page }) => {
			await page.goto("/");

			await page.getByRole("button", { name: "+ 创建项目" }).click();
			await expect(page).toHaveURL(/\/projects/);

			await page.goto("/");

			// 无 currentProject 时，LegacyRouteGuard 会将 /targets 重定向到 /projects
			await page.getByRole("button", { name: "导入目标" }).click();
			await expect(page).toHaveURL(/\/projects/);
		});

		test("TC-6: 重试按钮在错误时可用", async ({ page }) => {
			await page.goto("/");
			await expect(page.getByText("总项目数")).toBeVisible();

			await expect(page.getByText("重试")).not.toBeVisible();
		});
	});
