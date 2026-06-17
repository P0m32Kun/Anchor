/**
 * 测试层级: E2E
 * 场景 ID: E2E-SCAN-CONV-01
 * 关联 REQ: REQ-5, REQ-6
 * 覆盖流程:
 *   UI 创建项目 → API 注入 rangefield Redis IP → ScanModal 外网默认 Profile 启动扫描 →
 *   等待 pipeline 完成 → 断言 work 总数收敛且无 katana/ffuf → AssetPage 可见 6379
 * 前置依赖: anchor-server / anchor-worker / rangefield (172.31.0.13:6379)
 */

import { expect, test } from "@playwright/test";
import {
	getScanRunMetrics,
	listScanRunWorks,
	waitForPipeline,
} from "../fixtures/api-helpers";
import { cleanupTestData, addTarget } from "../fixtures/db-utils";
import { E2E_API_BASE, E2E_API_TOKEN } from "../fixtures/e2e-env";

const API_BASE = E2E_API_BASE;
const API_TOKEN = E2E_API_TOKEN;
const RANGEFIELD_IP = "172.31.0.13";

/** 单 IP + 外网 Profile + 高危端口（katana/ffuf 关）时的 work 上限 */
const MAX_TOTAL_WORKS = 200;

test.setTimeout(30 * 60 * 1000);

test.describe.serial("External scan profile convergence — E2E-SCAN-CONV-01", () => {
	test.beforeAll(async () => {
		await cleanupTestData();
	});

	test.afterAll(async () => {
		await cleanupTestData();
	});

	test("外网默认 Profile：work 收敛、无 katana/ffuf、6379 端口可见", async ({
		page,
	}) => {
		const log = (msg: string) => {
			const ts = new Date().toISOString().slice(11, 19);
			console.log(`[${ts}] ${msg}`);
		};

		const projectName = `ExtConv-${Date.now()}`;
		log(`Step 1: Create project "${projectName}"`);
		await page.goto("/projects");
		await expect(
			page.getByRole("heading", { name: /项目与授权边界|项目管理/ }),
		).toBeVisible({ timeout: 10_000 });

		await page
			.getByPlaceholder("例如：2024 Q2 外部红队评估")
			.fill(projectName);
		await page.getByPlaceholder("客户名称或部门").fill("E2E ExtConv");
		await page
			.getByPlaceholder("测试目的或项目背景")
			.fill("external profile work convergence");
		await page.getByRole("button", { name: "创建项目", exact: true }).click();

		const projectCard = page.getByRole("heading", { name: projectName });
		await expect(projectCard).toBeVisible({ timeout: 10_000 });
		await projectCard.click();
		await expect(page).toHaveURL(/\/projects\/[^/]+\/targets/);
		const projectId = page.url().match(/\/projects\/([^/]+)\/targets/)![1];
		log(`Project ID: ${projectId}`);

		log("Step 2: API inject scope + IP target (rangefield redis)");
		const scopeRes = await page.request.post(`${API_BASE}/scope-rules`, {
			headers: {
				Authorization: `Bearer ${API_TOKEN}`,
				"Content-Type": "application/json",
			},
			data: {
				project_id: projectId,
				action: "include",
				type: "cidr",
				value: `${RANGEFIELD_IP}/32`,
				reason: "E2E scope",
			},
		});
		expect([200, 201]).toContain(scopeRes.status());
		await addTarget(projectId, { type: "ip", value: RANGEFIELD_IP });

		log("Step 3: Start external scan with factory defaults (katana/ffuf off)");
		await page.evaluate(() => {
			localStorage.removeItem("anchor.scanModal.config");
			localStorage.removeItem("anchor.scanModal.mode");
		});
		await page.goto(`/projects/${projectId}/runs`);
		await expect(
			page.getByRole("heading", { name: "扫描执行" }),
		).toBeVisible({ timeout: 10_000 });

		await page.getByRole("button", { name: /启动扫描/ }).first().click();
		await expect(
			page.getByRole("heading", { name: /新建扫描流水线/ }),
		).toBeVisible();

		await page.locator("button", { hasText: "外网扫描" }).first().click();
		await page.getByRole("button", { name: /高级配置/ }).click();
		await page.getByRole("button", { name: "恢复出厂默认值" }).click();

		// 外网 top100 不含 6379；切到 -p 自定义（默认高危端口含 Redis）以产生可验证 finding
		await page.getByLabel("端口模式 -p 自定义").click();
		const portTextarea = page.getByLabel("自定义端口列表");
		await expect(portTextarea).toBeVisible({ timeout: 10_000 });
		await expect(portTextarea).toHaveValue(/6379/);

		for (const tool of ["Ffuf", "Katana"]) {
			const toggle = page.getByRole("button", { name: new RegExp(`^${tool}`) });
			if (await toggle.isVisible().catch(() => false)) {
				const cls = await toggle.getAttribute("class");
				if (cls?.includes("border-primary") || cls?.includes("bg-primary")) {
					await toggle.click();
				}
			}
		}

		const startBtn = page.getByRole("button", { name: /立即启动扫描/ });
		await expect(startBtn).toBeEnabled({ timeout: 15_000 });
		await startBtn.click();
		await expect(page.getByText("扫描任务已启动")).toBeVisible({
			timeout: 15_000,
		});

		log("Step 4: Wait for pipeline completion");
		const { runId, status } = await waitForPipeline(projectId, 25 * 60 * 1000);
		log(`Pipeline run ${runId} finished with status=${status}`);
		expect(["completed", "failed", "cancelled"]).toContain(status);
		expect(status).toBe("completed");

		log("Step 5: Assert work count convergence and no katana/ffuf works");
		const works = await listScanRunWorks(projectId, runId);
		const metrics = await getScanRunMetrics(projectId, runId);
		log(
			`Works total=${works.length} done=${metrics.works_done} skipped=${metrics.works_skipped} engine=${metrics.engine_state}`,
		);

		expect(works.length).toBeGreaterThan(0);
		expect(works.length).toBeLessThanOrEqual(MAX_TOTAL_WORKS);

		const katanaWorks = works.filter((w) => w.action === "KATANA_CRAWL");
		const ffufWorks = works.filter((w) => w.action === "FFUF_BRUTE");
		expect(katanaWorks, "katana should be off in external default").toHaveLength(
			0,
		);
		expect(ffufWorks, "ffuf should be off in external default").toHaveLength(0);

		log("Step 6: Verify AssetPage shows Redis 6379 (scan produced open port)");
		await page.goto(`/projects/${projectId}/assets`);
		await expect(page.getByText(/资产清单|资产列表|Assets/)).toBeVisible({
			timeout: 10_000,
		});
		await expect(page.locator(`text=${RANGEFIELD_IP}`).first()).toBeVisible({
			timeout: 120_000,
		});
		await expect(page.getByText(/6379/).first()).toBeVisible({
			timeout: 120_000,
		});

		log("E2E-SCAN-CONV-01 passed");
	});
});
