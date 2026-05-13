import { expect, test } from "@playwright/test";
import { createProject } from "../fixtures/api-helpers";
import { cleanupTestData, setCurrentProject } from "../fixtures/db-utils";

test.describe
	.serial("AssetPage", () => {
		test.beforeAll(async () => {
			await cleanupTestData();
		});

		test.afterAll(async () => {
			await cleanupTestData();
		});

		test("空状态：无资产时显示引导文案", async ({ page }) => {
			const project = await createProject({
				name: "Asset Empty Project",
				organization: "Test Org",
				purpose: "Asset empty state",
			});
			await setCurrentProject(page, project.id);

			await page.goto("/assets");
			await expect(page).toHaveURL(/\/projects\/.*\/assets/);
			// 标题可见
			await expect(page.getByText("资产清单")).toBeVisible();
			// 空状态文案存在（EmptyState 渲染为 <h3>）
			await expect(
				page.getByRole("heading", { name: "未找到资产" }),
			).toBeVisible();
		});

		test("切换标签页：资产 / Web / 端口三个 tab 均可点击", async ({
			page,
		}) => {
			const project = await createProject({
				name: "Asset Tabs Project",
				organization: "Test Org",
				purpose: "Asset tab switching",
			});
			await setCurrentProject(page, project.id);

			await page.goto("/assets");
			await expect(page).toHaveURL(/\/projects\/.*\/assets/);

			// 三个 tab 默认显示
			const tabs = ["资产", "Web", "端口"];
			for (const name of tabs) {
				await expect(
					page.getByRole("button", { name }).first(),
				).toBeVisible();
			}

			// 切换到 Web tab
			await page.getByRole("button", { name: "Web" }).first().click();
			await expect(
				page.getByRole("heading", { name: "未找到 Web 端点" }),
			).toBeVisible();

			// 切换到端口 tab
			await page.getByRole("button", { name: "端口" }).first().click();
			await expect(
				page.getByRole("heading", { name: "未探测到端口" }),
			).toBeVisible();
		});
	});
