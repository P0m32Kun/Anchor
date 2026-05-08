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
	.serial("ScanConfigPage", () => {
		test.beforeAll(async () => {
			await cleanupTestData();
		});

		test.afterAll(async () => {
			await cleanupTestData();
		});

		test("TC-1: 页面渲染和默认加载", async ({ page }) => {
			const project = await createProject({
				name: "Scan Config Render",
				organization: "Test Org",
				purpose: "Scan config render test",
			});
			await setCurrentProject(page, project.id);

			await page.goto("/projects/" + project.id + "/scan-config");
			await page.waitForLoadState("networkidle");

			await expect(page.locator("h1").filter({ hasText: "扫描配置" })).toBeVisible();
			await expect(page.getByText("调整流水线各阶段的开关、端口范围、并发与超时")).toBeVisible();

			// 端口范围 select
			await expect(page.locator("label").filter({ hasText: "预设" })).toBeVisible();

			// 流水线阶段 toggles
			await expect(page.getByText("FOFA 资产搜集")).toBeVisible();
			await expect(page.getByText("Subfinder 子域名爆破")).toBeVisible();
			await expect(page.getByText("CDN 过滤")).toBeVisible();
			await expect(page.getByText("nmap -sV 服务指纹")).toBeVisible();
			await expect(page.getByText("Nuclei 漏洞扫描")).toBeVisible();

			// 默认端口范围应为 top1000
			const portSelect = page.locator("select").first();
			await expect(portSelect).toHaveValue("top1000");

			// 未修改时保存按钮显示 "已保存" 且 disabled
			const saveBtn = page.locator("button").filter({ hasText: /^已保存$/ });
			await expect(saveBtn).toBeVisible();
			await expect(saveBtn).toBeDisabled();
		});

		test("TC-2: 切换端口预设为高危端口并保存", async ({ page }) => {
			const project = await createProject({
				name: "High Risk Test",
				organization: "Test Org",
				purpose: "High-risk preset test",
			});
			await setCurrentProject(page, project.id);

			await page.goto("/projects/" + project.id + "/scan-config");
			await page.waitForLoadState("networkidle");

			// 切换到 high-risk
			const portSelect = page.locator("select").first();
			await portSelect.selectOption("high-risk");
			await expect(portSelect).toHaveValue("high-risk");

			// 保存按钮变为可点击（顶部按钮）
			await page.locator("button", { hasText: "保存配置" }).first().click();
			await expect(page.getByText("扫描配置已保存")).toBeVisible();

			// 刷新页面，验证持久化
			await page.reload();
			await page.waitForLoadState("networkidle");
			await expect(portSelect).toHaveValue("high-risk");
		});

		test("TC-3: 自定义端口并验证", async ({ page }) => {
			const project = await createProject({
				name: "Custom Port Test",
				organization: "Test Org",
				purpose: "Custom port test",
			});
			await setCurrentProject(page, project.id);

			await page.goto("/projects/" + project.id + "/scan-config");
			await page.waitForLoadState("networkidle");

			const portSelect = page.locator("select").first();
			await portSelect.selectOption("custom");
			await expect(portSelect).toHaveValue("custom");

			// 自定义输入框出现
			await expect(page.getByPlaceholder("例如 80,443,8080 或 1-1000")).toBeVisible();

			// 输入自定义端口
			await page.getByPlaceholder("例如 80,443,8080 或 1-1000").fill("22,443,3306,6379,9200");
			await page.locator("button", { hasText: "保存配置" }).first().click();
			await expect(page.getByText("扫描配置已保存")).toBeVisible();

			// 刷新后验证
			await page.reload();
			await page.waitForLoadState("networkidle");
			await expect(portSelect).toHaveValue("custom");
			await expect(page.getByPlaceholder("例如 80,443,8080 或 1-1000")).toHaveValue("22,443,3306,6379,9200");
		});

		test("TC-4: 重置默认配置", async ({ page }) => {
			const project = await createProject({
				name: "Reset Test",
				organization: "Test Org",
				purpose: "Reset test",
			});
			await setCurrentProject(page, project.id);

			await page.goto("/projects/" + project.id + "/scan-config");
			await page.waitForLoadState("networkidle");

			// 先切换到 high-risk
			const portSelect = page.locator("select").first();
			await portSelect.selectOption("high-risk");
			await expect(portSelect).toHaveValue("high-risk");

			// 点击重置
			await page.getByRole("button", { name: "重置默认" }).click();
			await expect(page.getByText("已重置为默认配置")).toBeVisible();

			// 端口应恢复到 top1000
			await expect(portSelect).toHaveValue("top1000");
		});

		test("TC-5: 阶段开关切换并保存", async ({ page }) => {
			const project = await createProject({
				name: "Toggle Test",
				organization: "Test Org",
				purpose: "Toggle test",
			});
			await setCurrentProject(page, project.id);

			await page.goto("/projects/" + project.id + "/scan-config");
			await page.waitForLoadState("networkidle");

			// 关闭 Nuclei
			await page.getByRole("switch", { name: /Nuclei/ }).click();
			// 关闭 Nerva
			await page.getByRole("switch", { name: /Nerva/ }).click();

			await page.locator("button", { hasText: "保存配置" }).first().click();
			await expect(page.getByText("扫描配置已保存")).toBeVisible();

			// 刷新验证 Nuclei 调优区域应消失（因为关闭了）
			await page.reload();
			await page.waitForLoadState("networkidle");
			await expect(page.getByText("Nuclei 调优")).not.toBeVisible();
			await expect(page.getByText("Nerva 调优")).not.toBeVisible();
		});
	});
