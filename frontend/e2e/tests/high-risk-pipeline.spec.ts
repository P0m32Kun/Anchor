/**
 * 测试层级: E2E
 * 覆盖流程: UI 创建项目 → API 注入 IP 目标(§3.3 例外,scope confirm 产品 bug) →
 *           ScanModal 切到 -p 自定义模式(默认填充高危端口) → 等待 pipeline 完成 →
 *           AssetPage 看到 6379 端口 → FindingsPage 看到 critical/high finding 行
 * 前置依赖: anchor-server / anchor-worker / anchor-rangefield(rf-redis 监听 172.31.0.13:6379)已经启动
 * UI 断言点:
 *   - 项目卡片可见 → 跳转 /targets
 *   - TargetPage 表格中能看到 172.31.0.13 行
 *   - ScanModal 切到 -p 自定义后,textarea 默认值包含 6379
 *   - RunsPage 上扫描启动后能看到 run 卡片
 *   - 完成后 AssetPage 看到 172.31.0.13、Asset 详情/端口区域包含 6379
 *   - FindingsPage 至少有一条 critical/high 行(可见 severity 标签)
 * API 仅用于:
 *   - cleanup
 *   - 目标注入(§3.3 例外: scope confirm 产品 bug 暂未修复)
 *   - 长扫描进度轮询(§3.3 例外条款)
 */
import { expect, test } from "@playwright/test";
import { cleanupTestData, addTarget } from "../fixtures/db-utils";
import { waitForPipeline } from "../fixtures/api-helpers";
import { E2E_API_BASE, E2E_API_TOKEN } from "../fixtures/e2e-env";

const API_BASE = E2E_API_BASE;
const API_TOKEN = E2E_API_TOKEN;
const REDIS_IP = "172.31.0.13";

test.setTimeout(30 * 60 * 1000);

test.describe.serial("High-risk port preset E2E — UI 主导", () => {
	test.beforeAll(async () => {
		await cleanupTestData();
	});

	test.afterAll(async () => {
		await cleanupTestData();
	});

	test("UI 切 -p 自定义(默认高危端口) → 扫到 Redis 6379 → 至少 1 条 critical/high finding", async ({
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

		await page
			.getByPlaceholder("例如：2024 Q2 外部红队评估")
			.fill(projectName);
		await page.getByPlaceholder("客户名称或部门").fill("E2E HighRisk");
		await page
			.getByPlaceholder("测试目的或项目背景")
			.fill("-p 自定义端口 UI 验收");
		await page.getByRole("button", { name: "创建项目", exact: true }).click();

		// 项目卡片标题渲染为 <h3>,匹配 heading role
		const projectCard = page.getByRole("heading", { name: projectName });
		await expect(projectCard).toBeVisible({ timeout: 10_000 });
		await projectCard.click();
		await expect(page).toHaveURL(/\/projects\/[^/]+\/targets/);
		const projectId = page.url().match(/\/projects\/([^/]+)\/targets/)![1];
		log(`Project ID: ${projectId}`);

		// ── Step 2: API 注入 scope rule + IP 目标 + UI 验证(§3.3 例外: scope confirm 产品 bug)──
		log("Step 2: API inject scope + target (rangefield redis)");
		const scopeRes = await page.request.post(`${API_BASE}/scope-rules`, {
			headers: {
				Authorization: `Bearer ${API_TOKEN}`,
				"Content-Type": "application/json",
			},
			data: {
				project_id: projectId,
				action: "include",
				type: "cidr",
				value: `${REDIS_IP}/32`,
				reason: "E2E scope",
			},
		});
		expect([200, 201]).toContain(scopeRes.status());
		await addTarget(projectId, { type: "ip", value: REDIS_IP });
		await page.reload();

		await expect(
			page.getByRole("cell", { name: REDIS_IP }).first(),
		).toBeVisible({ timeout: 10_000 });

		// ── Step 3: 通过 ScanModal 切到 -p 自定义模式(默认填充高危端口列表) ──
		log("Step 3: Open ScanModal and switch to -p custom mode (default high-risk ports)");
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
		await page.getByRole("button", { name: /高级配置/ }).click();

		// step 2: 切到 -p 自定义模式（textarea 自动填充高危端口列表，
		// 后端 BuildNaabuCommand 收到 -p <ports> 与旧的 "high-risk" 别名等价）
		await page.getByLabel("端口模式 -p 自定义").click();
		const portTextarea = page.getByLabel("自定义端口列表");
		await expect(portTextarea).toBeVisible();
		// 默认填充的高危端口列表里必须包含 6379（Redis）
		await expect(portTextarea).toHaveValue(/6379/);

		// Ffuf 默认开启但无字典时 disable 立即启动扫描;关掉 Ffuf 以解锁按钮。
		await page.getByRole("button", { name: /^Ffuf/ }).click();

		await page.getByRole("button", { name: /立即启动扫描/ }).click();
		await expect(page.getByText("扫描任务已启动")).toBeVisible({
			timeout: 10_000,
		});

		// run 卡片出现
		const runCard = page.locator('[class*="cursor-pointer"]').first();
		await expect(runCard).toBeVisible({ timeout: 15_000 });

		// ── Step 4: 等待 pipeline 完成(waitForPipeline) ──
		log("Step 4: Poll until pipeline completes");
		const { status, runId } = await waitForPipeline(projectId);
		expect(status).toBe("completed");
		log(`Pipeline ${runId} completed`);

		// ── Step 5: UI 验证资产 + Redis 6379 ──
		log("Step 5: Verify asset on AssetPage (Redis 6379 required)");
		await page.goto(`/projects/${projectId}/assets`);
		await expect(page.getByText(/资产清单|资产列表|Assets/)).toBeVisible({
			timeout: 10_000,
		});
		await expect(page.locator(`text=${REDIS_IP}`).first()).toBeVisible({
			timeout: 30_000,
		});
		await expect(page.getByText(/6379/).first()).toBeVisible({
			timeout: 30_000,
		});

		// ── Step 6: UI 验证 FindingsPage（至少 1 条 critical/high）──
		log("Step 6: Verify FindingsPage has critical/high finding");
		await page.goto(`/projects/${projectId}/findings`);
		await expect(
			page.locator("h1").filter({ hasText: /发现审核|漏洞发现|Finding/i }),
		).toBeVisible({ timeout: 10_000 });
		await expect(
			page.getByText(/critical|high|严重|高危/i).first(),
		).toBeVisible({ timeout: 30_000 });

		log("✓ High-risk pipeline UI test completed");
	});
});
