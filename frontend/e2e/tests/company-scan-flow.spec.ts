/**
 * 测试层级: E2E
 * 场景 ID: E2E-COMPANY-01, E2E-LINEAGE-01
 * 关联 REQ: REQ-1, REQ-2, REQ-3
 * 覆盖流程:
 *   UI 配置 FOFA 凭证 → UI 添加企业名目标 → UI 启动外网扫描 →
 *   Runs 出现 run → AssetPage 可见 FOFA mock 展开的子域
 * 前置依赖: docker-compose.e2e.yml（含 fofa-mock、FOFA_BASE_URL）
 * UI 断言点:
 *   - EngineKeysPage 保存 FOFA Key 成功 toast
 *   - TargetPage 企业名目标 TestCorp 可见
 *   - RunsPage 扫描启动 toast + run 卡片
 *   - AssetPage 出现 sub1.testcorp.example（FOFA mock 展开产物）
 * API 仅用于:
 *   - cleanupTestData（setup/teardown）
 */

import { expect, test } from "@playwright/test";
import {
	getAssetLineage,
	listAssets,
	listScanRuns,
} from "../fixtures/api-helpers";
import { cleanupTestData } from "../fixtures/db-utils";

const COMPANY_NAME = "TestCorp";
const FOFA_MOCK_DOMAIN = "sub1.testcorp.example";

test.setTimeout(10 * 60 * 1000);

test.describe.serial("Company scan flow — E2E-COMPANY-01", () => {
	test.beforeAll(async () => {
		await cleanupTestData();
	});

	test.afterAll(async () => {
		await cleanupTestData();
	});

	test("UI: FOFA 凭证 → 企业名目标 → 外网扫描 → 资产展开", async ({ page }) => {
		const log = (msg: string) => {
			const ts = new Date().toISOString().slice(11, 19);
			console.log(`[${ts}] ${msg}`);
		};

		// ── Step 1: UI 配置 FOFA 凭证 ──
		log("Step 1: Save FOFA API key via EngineKeysPage");
		await page.goto("/engines/keys");
		await expect(
			page.getByRole("heading", { name: "API 凭证管理" }),
		).toBeVisible({ timeout: 15_000 });

		const fofaCard = page
			.locator(".relative.overflow-hidden")
			.filter({ has: page.getByRole("heading", { name: "FOFA" }) });
		await expect(fofaCard).toBeVisible({ timeout: 10_000 });

		const fofaKeyInput = fofaCard.getByPlaceholder("Paste your key here...");
		await fofaKeyInput.click();
		await fofaKeyInput.fill("e2e-fofa-key");
		await fofaCard.getByRole("button", { name: /Save Key|Update/ }).click();
		await expect(page.getByText("保存成功")).toBeVisible({ timeout: 10_000 });

		// ── Step 2: UI 创建项目 ──
		const projectName = `CompanyFlow-${Date.now()}`;
		log(`Step 2: Create project "${projectName}"`);
		await page.goto("/projects");
		await expect(
			page.getByRole("heading", { name: /项目与授权边界|项目管理/ }),
		).toBeVisible({ timeout: 10_000 });

		await page
			.getByPlaceholder("例如：2024 Q2 外部红队评估")
			.fill(projectName);
		await page.getByPlaceholder("客户名称或部门").fill("E2E Company");
		await page
			.getByPlaceholder("测试目的或项目背景")
			.fill("company → FOFA expand E2E");
		await page.getByRole("button", { name: "创建项目", exact: true }).click();

		const projectCard = page.getByRole("heading", { name: projectName });
		await expect(projectCard).toBeVisible({ timeout: 10_000 });
		await projectCard.click();
		await expect(page).toHaveURL(/\/projects\/[^/]+\/targets/);
		const projectId = page.url().match(/\/projects\/([^/]+)\/targets/)![1];
		log(`Project ID: ${projectId}`);

		// ── Step 3: UI 添加企业名目标 ──
		log(`Step 3: Add company target "${COMPANY_NAME}"`);
		await page.locator("main select").first().selectOption("company");
		await page
			.getByPlaceholder(/某某科技有限公司/)
			.first()
			.fill(COMPANY_NAME);
		await page.getByRole("button", { name: "添加" }).first().click();

		await expect(page.getByText("目标已添加")).toBeVisible({ timeout: 15_000 });
		await expect(page.getByRole("cell", { name: COMPANY_NAME })).toBeVisible({
			timeout: 15_000,
		});

		// ── Step 4: UI 启动外网扫描 ──
		log("Step 4: Start external scan via ScanModal");
		await page.evaluate(() => {
			localStorage.removeItem("anchor.scanModal.config");
			localStorage.removeItem("anchor.scanModal.mode");
		});
		await page.goto(`/projects/${projectId}/runs`);
		await expect(
			page.getByRole("heading", { name: "扫描执行" }),
		).toBeVisible({ timeout: 10_000 });

		const scanBtn = page.getByRole("button", { name: /启动扫描/ }).first();
		await expect(scanBtn).toBeEnabled({ timeout: 30_000 });
		await scanBtn.click();
		await expect(
			page.getByRole("heading", { name: /新建扫描流水线/ }),
		).toBeVisible();

		await page.locator("button", { hasText: "外网扫描" }).first().click();
		await page.getByRole("button", { name: /高级配置/ }).click();
		await page.getByRole("button", { name: "恢复出厂默认值" }).click();

		// 外网默认已关闭 ffuf/katana；若 localStorage 残留则显式关掉
		for (const tool of ["Ffuf", "Katana"]) {
			const toggle = page.getByRole("button", { name: new RegExp(`^${tool}`) });
			if (await toggle.isVisible().catch(() => false)) {
				const cls = await toggle.getAttribute("class");
				if (cls?.includes("border-primary") || cls?.includes("bg-primary")) {
					await toggle.click();
				}
			}
		}

		const startScanBtn = page.getByRole("button", { name: /立即启动扫描/ });
		await expect(startScanBtn).toBeEnabled({ timeout: 15_000 });
		await startScanBtn.click();
		await expect(page.getByText("扫描任务已启动")).toBeVisible({
			timeout: 15_000,
		});

		await expect(page.locator('[class*="cursor-pointer"]').first()).toBeVisible({
			timeout: 15_000,
		});

		// ── Step 5: UI 验证 FOFA 展开资产（不等待完整 pipeline）──
		log(`Step 5: Wait for expanded asset "${FOFA_MOCK_DOMAIN}" on AssetPage`);
		const assetDeadline = Date.now() + 120_000;
		let found = false;
		while (Date.now() < assetDeadline) {
			await page.goto(`/projects/${projectId}/assets`);
			await expect(
				page.getByRole("heading", { name: "资产清单" }),
			).toBeVisible({ timeout: 10_000 });
			if (
				await page
					.getByText(FOFA_MOCK_DOMAIN)
					.first()
					.isVisible()
					.catch(() => false)
			) {
				found = true;
				break;
			}
			await page.waitForTimeout(3_000);
		}

		expect(found, `expected FOFA-expanded domain ${FOFA_MOCK_DOMAIN} on AssetPage`).toBe(
			true,
		);

		// ── Step 6: API 验证血缘（E2E-LINEAGE-01）──
		log("Step 6: Verify lineage API traces back to company target");
		const assets = await listAssets(projectId);
		const domainAsset = assets.find((a) => a.value === FOFA_MOCK_DOMAIN);
		expect(domainAsset, `asset ${FOFA_MOCK_DOMAIN} in API list`).toBeTruthy();

		const runs = await listScanRuns(projectId);
		expect(runs.length).toBeGreaterThan(0);

		const lineage = await getAssetLineage(domainAsset!.id, runs[0].id);
		expect(lineage.chain.length).toBeGreaterThanOrEqual(2);
		expect(lineage.chain[0].node_type).toBe("target");
		expect(lineage.chain[0].value).toBe(COMPANY_NAME);
		expect(lineage.chain[1].relation).toBe("expanded_by");
		expect(lineage.chain[1].source_engine?.toLowerCase()).toContain("fofa");

		log("E2E-COMPANY-01 + E2E-LINEAGE-01 passed");
	});
});
