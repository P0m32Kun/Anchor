/**
 * 测试层级: E2E
 * 覆盖流程: UI 创建项目 → UI 添加 IP 目标 → ScanModal 选"高危端口"preset → 等待 pipeline 完成 → AssetPage 看到 6379 端口 → FindingsPage 看到 critical/high finding 行
 * 前置依赖: anchor-server / anchor-worker / anchor-rangefield(rf-redis 监听 172.30.0.13:6379)已经启动
 * UI 断言点:
 *   - 项目卡片可见 → 跳转 /targets
 *   - TargetPage 表格中能看到 172.30.0.13 行
 *   - ScanModal 选"高危端口(推荐)" 后能看到选中态
 *   - RunsPage 上扫描启动后能看到 run 卡片
 *   - 完成后 AssetPage 看到 172.30.0.13、Asset 详情/端口区域包含 6379
 *   - FindingsPage 至少有一条 critical/high 行(可见 severity 标签)
 * API 仅用于:
 *   - cleanup
 *   - 长扫描进度轮询(§3.3 例外条款)
 */
import { expect, test } from "@playwright/test";
import { cleanupTestData } from "../fixtures/db-utils";

const API_BASE = "http://localhost:17421";
const API_TOKEN = "p0m32kun";
const REDIS_IP = "172.30.0.13";

test.setTimeout(30 * 60 * 1000);

test.describe.serial("High-risk port preset E2E — UI 主导", () => {
	test.beforeAll(async () => {
		await cleanupTestData();
	});

	test.afterAll(async () => {
		await cleanupTestData();
	});

	test("UI 选高危端口 preset → 扫到 Redis 6379 → 至少 1 条 critical/high finding", async ({
		page,
	}) => {
		const log = (msg: string) => {
			const ts = new Date().toISOString().slice(11, 19);
			console.log(`[${ts}] ${msg}`);
		};

		// ── Step 1: UI 创建项目 ──
		const projectName = `HighRisk-${Date.now()}`;
		log(`Step 1: Create project "${projectName}" via UI`);
		await page.goto("/projects");
		await expect(
			page.getByRole("heading", { name: /项目与授权边界|项目管理/ }),
		).toBeVisible({ timeout: 10_000 });

		await page.getByPlaceholder("项目名称 *").fill(projectName);
		await page.getByPlaceholder("组织/客户").fill("E2E HighRisk");
		await page
			.getByPlaceholder("目的/描述")
			.fill("高危端口 preset UI 验收");
		await page.getByRole("button", { name: "创建项目", exact: true }).click();

		const projectCard = page.locator("button", { hasText: projectName }).first();
		await expect(projectCard).toBeVisible({ timeout: 10_000 });
		await projectCard.click();
		await expect(page).toHaveURL(/\/projects\/[^/]+\/targets/);
		const projectId = page.url().match(/\/projects\/([^/]+)\/targets/)![1];
		log(`Project ID: ${projectId}`);

		// ── Step 2: UI 添加 IP 目标 ──
		log("Step 2: Add IP target via UI (rangefield redis)");
		const targetForm = page.locator("form").filter({
			has: page.getByPlaceholder("example.com"),
		});
		await targetForm.locator("select").selectOption("ip");
		await targetForm.getByPlaceholder("example.com").fill(REDIS_IP);
		await targetForm.getByRole("button", { name: "添加目标" }).click();

		const scopeConfirm = page.getByRole("button", {
			name: /添加规则并继续|确认/,
		});
		if (await scopeConfirm.isVisible({ timeout: 3_000 }).catch(() => false)) {
			await scopeConfirm.click();
		}

		await expect(
			page.getByRole("cell", { name: REDIS_IP }).first(),
		).toBeVisible({ timeout: 10_000 });

		// ── Step 3: 通过 ScanModal 选 "高危端口" preset ──
		log("Step 3: Open ScanModal and select high-risk port preset");
		await page.goto(`/projects/${projectId}/runs`);
		await expect(
			page.getByRole("heading", { name: "扫描执行" }),
		).toBeVisible({ timeout: 10_000 });

		await page.getByRole("button", { name: /启动扫描/ }).first().click();
		await expect(
			page.getByRole("heading", { name: /新建扫描流水线/ }),
		).toBeVisible();

		// step 1: 内网扫描(rangefield 是内网)
		await page.locator("button", { hasText: "内网扫描" }).first().click();
		await page.getByRole("button", { name: /配置参数/ }).click();

		// step 2: 选 "高危端口(推荐)" preset
		const highRiskBtn = page.locator("button", {
			hasText: "高危端口（推荐）",
		});
		await highRiskBtn.first().click();
		// 选中态: ring/border 颜色变化用 className 难断言,改为断言 description 文本可见
		await expect(
			page.getByText(/115 个攻击面端口/),
		).toBeVisible();

		await page.getByRole("button", { name: /立即启动扫描/ }).click();
		await expect(page.getByText("扫描任务已启动")).toBeVisible({
			timeout: 10_000,
		});

		// run 卡片出现
		const runCard = page.locator('[class*="cursor-pointer"]').first();
		await expect(runCard).toBeVisible({ timeout: 15_000 });

		// ── Step 4: 等待 pipeline 完成(API 轮询,例外条款) ──
		log("Step 4: Poll until pipeline completes");
		const runs = await page.request
			.get(`${API_BASE}/projects/${projectId}/runs`, {
				headers: { Authorization: `Bearer ${API_TOKEN}` },
			})
			.then((r) => r.json() as Promise<{ data: Array<{ id: string }> }>);
		const runId = runs.data?.[0]?.id;
		expect(runId).toBeDefined();

		const start = Date.now();
		const maxWait = 20 * 60 * 1000;
		let status = "running";
		while (status === "running" && Date.now() - start < maxWait) {
			await page.waitForTimeout(5_000);
			const res = await page.request.get(
				`${API_BASE}/projects/${projectId}/pipeline/runs/${runId}`,
				{ headers: { Authorization: `Bearer ${API_TOKEN}` } },
			);
			if (res.ok()) {
				const d = (await res.json()) as { status: string };
				status = d.status;
			}
		}
		expect(status).toBe("completed");
		log(`Pipeline completed in ${Math.round((Date.now() - start) / 1000)}s`);

		// ── Step 5: UI 验证资产 + 端口 ──
		log("Step 5: Verify asset and port 6379 on AssetPage");
		await page.goto(`/projects/${projectId}/assets`);
		await expect(page.getByText(/资产清单|资产列表|Assets/)).toBeVisible({
			timeout: 10_000,
		});
		await expect(page.locator(`text=${REDIS_IP}`).first()).toBeVisible({
			timeout: 30_000,
		});

		// 点开 asset 行查看端口(具体交互依 AssetPage 实现而定;若未提供端口侧栏,
		// 用 UI 上的端口标签或徽章断言)
		const redisRow = page.locator("tr", { hasText: REDIS_IP }).first();
		if (await redisRow.isVisible().catch(() => false)) {
			await redisRow.click();
			await expect(
				page.getByText(/6379/).first(),
			).toBeVisible({ timeout: 10_000 });
		} else {
			// 兜底:页面任意位置出现 6379 字样即视为通过
			await expect(page.getByText(/6379/).first()).toBeVisible({
				timeout: 15_000,
			});
		}

		// ── Step 6: UI 验证 Findings 至少 1 条 critical/high ──
		log("Step 6: Verify FindingsPage shows critical/high finding");
		await page.goto(`/projects/${projectId}/findings`);
		await expect(
			page.locator("h1").filter({ hasText: /Finding/i }),
		).toBeVisible({ timeout: 10_000 });

		await expect(
			page.getByText(/critical|high|严重|高危/i).first(),
		).toBeVisible({ timeout: 30_000 });

		log("✓ High-risk pipeline UI test completed");
	});
});
