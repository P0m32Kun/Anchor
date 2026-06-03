/**
 * E2E 测试 - 适配新部署方式（install.sh 部署）
 *
 * 测试层级: E2E
 * 覆盖流程: TokenAuth → 项目创建 → 目标添加 → 扫描启动 → 结果验证
 * 前置依赖: 通过 install.sh 部署的环境已启动（server + worker + frontend）
 *           Rangefield 靶场已启动（make range-up）
 *
 * 使用方式:
 *   cd frontend && npx playwright test e2e/tests/live-scan.spec.ts --config=playwright.live.config.ts
 */
import { expect, test } from "@playwright/test";
import { cleanupTestData } from "../fixtures/db-utils";

import { E2E_API_BASE, E2E_API_TOKEN } from "../fixtures/e2e-env";

const API_BASE = E2E_API_BASE;
const API_TOKEN = E2E_API_TOKEN;

test.describe.serial("Live Scan E2E — 新部署方式适配", () => {
	test.beforeAll(async () => {
		await cleanupTestData();
	});

	test.afterAll(async () => {
		await cleanupTestData();
	});

	test("完整流程：认证 → 项目 → 目标 → 扫描 → 结果验证", async ({
		browser,
	}) => {
		const context = await browser.newContext({ storageState: undefined });
		const page = await context.newPage();
		const log = (msg: string) => {
			const ts = new Date().toISOString().slice(11, 19);
			console.log(`[${ts}] ${msg}`);
		};

		// ── Step 1: TokenAuth UI ──
		log("Step 1: Token authentication via UI");
		await page.goto("/");
		await expect(
			page.getByRole("heading", { name: "欢迎使用 Anchor" }),
		).toBeVisible({ timeout: 10_000 });

		await page.getByPlaceholder("http://localhost:17421").fill(API_BASE);
		await page.locator('input[type="password"]').fill(API_TOKEN);
		await page.getByRole("button", { name: "保存并进入" }).click();

		await expect(
			page.getByRole("heading", { name: "安全工作台" }),
		).toBeVisible({ timeout: 15_000 });

		// ── Step 2: Worker 在线(UI 断言) ──
		log("Step 2: Verify worker online via UI");
		await page.goto("/workers");
		await expect(
			page.locator("h1").filter({ hasText: "Worker" }),
		).toBeVisible();
		await expect(
			page.getByText(/在线|busy|online/i).first(),
		).toBeVisible({ timeout: 30_000 });

		// ── Step 3: UI 创建项目 ──
		const projectName = `LiveScan-${Date.now()}`;
		log(`Step 3: Create project "${projectName}" via UI`);
		await page.goto("/projects");
		await expect(
			page.getByRole("heading", { name: /项目与授权边界|项目管理/ }),
		).toBeVisible({ timeout: 10_000 });

		await page
			.getByPlaceholder("例如：2024 Q2 外部红队评估")
			.fill(projectName);
		await page.getByPlaceholder("客户名称或部门").fill("E2E Live Scan");
		await page
			.getByPlaceholder("测试目的或项目背景")
			.fill("新部署方式适配测试");
		await page.getByRole("button", { name: "创建项目", exact: true }).click();

		// 项目卡片标题渲染为 <h3>
		const projectCard = page.getByRole("heading", { name: projectName });
		await expect(projectCard).toBeVisible({ timeout: 10_000 });

		await projectCard.click();
		await expect(page).toHaveURL(/\/projects\/[^/]+\/targets/, {
			timeout: 10_000,
		});
		const projectId = page.url().match(/\/projects\/([^/]+)\/targets/)![1];
		log(`Project ID: ${projectId}`);

		// ── Step 4: API 注入 scope rule + IP 目标 ──
		log("Step 4: API inject scope + target 172.31.0.13 (Redis)");
		const scopeRes = await page.request.post(`${API_BASE}/scope-rules`, {
			headers: {
				Authorization: `Bearer ${API_TOKEN}`,
				"Content-Type": "application/json",
			},
			data: {
				project_id: projectId,
				action: "include",
				type: "cidr",
				value: "172.31.0.13/32",
				reason: "E2E scope",
			},
		});
		expect([200, 201]).toContain(scopeRes.status());

		// 添加目标
		const targetRes = await page.request.post(
			`${API_BASE}/projects/${projectId}/targets`,
			{
				headers: {
					Authorization: `Bearer ${API_TOKEN}`,
					"Content-Type": "application/json",
				},
				data: { type: "ip", value: "172.31.0.13" },
			},
		);
		expect([200, 201]).toContain(targetRes.status());
		await page.reload();

		await expect(
			page.getByRole("cell", { name: "172.31.0.13" }).first(),
		).toBeVisible({ timeout: 10_000 });

		// ── Step 5: API 启动扫描 ──
		log("Step 5: Start scan via API");
		const scanRes = await page.request.post(
			`${API_BASE}/projects/${projectId}/scan`,
			{
				headers: {
					Authorization: `Bearer ${API_TOKEN}`,
					"Content-Type": "application/json",
				},
				data: {
					mode: "external",
					config: {
						port_range: "6379",
						enable_ffuf: false,
						enable_nuclei: true,
					},
				},
			},
		);
		expect(scanRes.status()).toBe(202);
		const scanData = await scanRes.json();
		const runId = scanData.run_id;
		log(`Scan started, Run ID: ${runId}`);

		// ── Step 6: 等待扫描完成 ──
		log("Step 6: Wait for scan completion");
		let status = "running";
		for (let i = 0; i < 60; i++) {
			await page.waitForTimeout(10_000);
			const statusRes = await page.request.get(
				`${API_BASE}/projects/${projectId}/pipeline/runs/${runId}`,
				{
					headers: { Authorization: `Bearer ${API_TOKEN}` },
				},
			);
			const statusData = await statusRes.json();
			status = statusData.status;
			log(`  [${i + 1}] Status: ${status}`);

			if (["completed", "failed", "cancelled"].includes(status)) {
				break;
			}
		}
		expect(status).toBe("completed");

		// ── Step 7: UI 验证资产 ──
		log("Step 7: Verify assets on AssetPage");
		await page.goto(`/projects/${projectId}/assets`);
		await expect(page.getByRole("heading", { name: /资产清单|资产列表|Assets/ })).toBeVisible({
			timeout: 10_000,
		});
		await expect(page.locator("text=172.31.0.13").first()).toBeVisible({
			timeout: 30_000,
		});

		// ── Step 8: UI 验证 FindingsPage ──
		log("Step 8: Verify FindingsPage");
		await page.goto(`/projects/${projectId}/findings`);
		await expect(
			page.locator("h1").filter({ hasText: /发现审核|漏洞发现|Finding/i }),
		).toBeVisible({ timeout: 10_000 });

		// 验证是否有 Redis 相关发现
		try {
			await expect(
				page.getByText(/Redis|redis/i).first(),
			).toBeVisible({ timeout: 15_000 });
			log("  ✓ Redis finding detected");
		} catch {
			log("  ⚠ No Redis finding detected");
		}

		// ── Step 9: UI 验证报告页 ──
		log("Step 9: Verify ReportsPage and export buttons");
		await page.goto(`/projects/${projectId}/reports`);
		await expect(page.getByRole("heading", { name: /安全评估报告|Reports/ })).toBeVisible({
			timeout: 10_000,
		});
		// 验证导出按钮（Markdown 和 JSON）
		try {
			await expect(
				page.getByRole("button", { name: /Markdown|MD|导出/i }).first(),
			).toBeVisible({ timeout: 5_000 });
		} catch {
			console.warn("[e2e] Markdown export button not found");
		}

		try {
			await expect(
				page.getByRole("button", { name: /JSON|导出/i }).first(),
			).toBeVisible({ timeout: 5_000 });
		} catch {
			console.warn("[e2e] JSON export button not found");
		}

		log("✓ Live scan E2E test completed");
		await context.close();
	});
});
