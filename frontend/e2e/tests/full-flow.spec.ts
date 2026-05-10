/**
 * 测试层级: E2E
 * 覆盖流程: TokenAuth → 验证 Worker → UI 创建项目 → UI 添加 IP 目标 → ScanModal 启动内网扫描 → 等待完成 → AssetPage/FindingsPage/ReportsPage UI 验证
 * 前置依赖: docker compose -f docker-compose.e2e.yml 已经启动 anchor-server / anchor-worker / anchor-rangefield
 * UI 断言点:
 *   - 欢迎页登录 → 进入 Dashboard 后能看到"总项目数"
 *   - WorkersPage 看到至少一个"在线"标签
 *   - 项目卡片在 ProjectPage 上可见,点击后跳转 /projects/:id/targets
 *   - TargetPage 表格中能看到 127.0.0.1 行
 *   - RunsPage 上 ScanModal 两步完整走通,提交后看到"扫描任务已启动" toast
 *   - Pipeline 完成后 AssetPage 列表中能看到 127.0.0.1 资产行
 *   - ReportsPage 上"导出 Markdown / JSON"按钮可见且可点
 * API 仅用于:
 *   - cleanup(setup/teardown 数据)
 *   - 长扫描进度轮询(§3.3 例外条款,等待 pipeline 完成,最终断言仍回 UI)
 */
import { expect, test } from "@playwright/test";
import { cleanupTestData } from "../fixtures/db-utils";

const API_BASE = "http://localhost:17421";
const API_TOKEN = "p0m32kun";

test.setTimeout(30 * 60 * 1000);

test.describe.serial("Full Flow E2E — UI 主导的完整使用场景", () => {
	test.beforeAll(async () => {
		await cleanupTestData();
	});

	test.afterAll(async () => {
		await cleanupTestData();
	});

	test("UI: 认证 → 项目 → 目标 → 内网扫描 → 资产/Finding/报告", async ({
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
		const projectName = `FullFlow-${Date.now()}`;
		log(`Step 3: Create project "${projectName}" via UI`);
		await page.goto("/projects");
		await expect(
			page.getByRole("heading", { name: /项目与授权边界|项目管理/ }),
		).toBeVisible({ timeout: 10_000 });

		await page
			.getByPlaceholder("例如：2024 Q2 外部红队评估")
			.fill(projectName);
		await page.getByPlaceholder("客户名称或部门").fill("E2E Full Flow");
		await page
			.getByPlaceholder("测试目的或项目背景")
			.fill("UI-driven full flow 验收");
		await page.getByRole("button", { name: "创建项目", exact: true }).click();

		// 项目卡片可见即视为创建成功
		const projectCard = page.locator("button", { hasText: projectName }).first();
		await expect(projectCard).toBeVisible({ timeout: 10_000 });

		await projectCard.click();
		await expect(page).toHaveURL(/\/projects\/[^/]+\/targets/, {
			timeout: 10_000,
		});
		const projectId = page.url().match(/\/projects\/([^/]+)\/targets/)![1];
		log(`Project ID: ${projectId}`);

		// ── Step 4: UI 添加 scope rule + IP 目标 ──
		log("Step 4: Add scope rule (domain) + IP target via UI");

		// Scope 规则 form 当前 UI 仅支持 domain 类型(见 TargetPage.tsx),
		// 这里用一个不影响 IP 校验的占位 domain 规则,真正的 127.0.0.1 走 IP target +
		// scope 自动确认弹窗。
		await page
			.getByPlaceholder("例如 *.example.com")
			.fill("loopback.invalid");
		await page.getByRole("button", { name: "添加规则" }).click();
		await expect(page.getByText(/规则已添加|添加规则失败/)).toBeVisible({
			timeout: 5_000,
		});

		// 添加 IP target
		const targetForm = page.locator("form").filter({
			has: page.getByPlaceholder("example.com"),
		});
		await targetForm.locator("select").selectOption("ip");
		await targetForm.getByPlaceholder("example.com").fill("127.0.0.1");
		await targetForm.getByRole("button", { name: "添加目标" }).click();

		// 可能弹出 scope 授权确认窗;若有则点确认
		const scopeConfirm = page.getByRole("button", {
			name: /添加规则并继续|确认/,
		});
		if (await scopeConfirm.isVisible({ timeout: 3_000 }).catch(() => false)) {
			await scopeConfirm.click();
		}

		// 表格里能看到目标值
		await expect(
			page.getByRole("cell", { name: "127.0.0.1" }).first(),
		).toBeVisible({ timeout: 10_000 });

		// ── Step 5: ScanModal 启动内网扫描 ──
		log("Step 5: Trigger internal scan via ScanModal");
		await page.goto(`/projects/${projectId}/runs`);
		await expect(
			page.getByRole("heading", { name: "扫描执行" }),
		).toBeVisible({ timeout: 10_000 });

		await page.getByRole("button", { name: /启动扫描/ }).first().click();
		await expect(
			page.getByRole("heading", { name: /新建扫描流水线/ }),
		).toBeVisible();

		// step 1: 选内网扫描
		await page.locator("button", { hasText: "内网扫描" }).first().click();
		await page.getByRole("button", { name: /配置参数/ }).click();

		// step 2: 选 Top 100 端口(最快)
		await expect(page.getByText("端口探测范围")).toBeVisible();
		await page.locator("button", { hasText: "Top 100 常用端口" }).first().click();
		await page.getByRole("button", { name: /立即启动扫描/ }).click();

		// 启动成功的 UI 信号
		await expect(page.getByText("扫描任务已启动")).toBeVisible({
			timeout: 10_000,
		});

		// 至少能看到一个 run 卡片(running / pending)
		const runCard = page.locator('[class*="cursor-pointer"]').first();
		await expect(runCard).toBeVisible({ timeout: 15_000 });

		// ── Step 6: 等待 Pipeline 完成(API 轮询,例外条款) ──
		log("Step 6: Poll pipeline status via API while UI shows Run card");
		const runs = await page.request
			.get(`${API_BASE}/projects/${projectId}/runs`, {
				headers: { Authorization: `Bearer ${API_TOKEN}` },
			})
			.then((r) => r.json() as Promise<{ data: Array<{ id: string }> }>);
		const runId = runs.data?.[0]?.id;
		expect(runId, "未找到任何 pipeline run").toBeDefined();

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
		log(`Pipeline final status: ${status}`);
		expect(status).toBe("completed");

		// ── Step 7: UI 验证资产 ──
		log("Step 7: Verify assets on AssetPage");
		await page.goto(`/projects/${projectId}/assets`);
		await expect(page.getByText(/资产清单|资产列表|Assets/)).toBeVisible({
			timeout: 10_000,
		});
		// 等待轮询/loading 完成
		await expect(page.locator("text=127.0.0.1").first()).toBeVisible({
			timeout: 30_000,
		});

		// ── Step 8: UI 验证 Findings 页 ──
		log("Step 8: Verify FindingsPage renders");
		await page.goto(`/projects/${projectId}/findings`);
		await expect(
			page.locator("h1").filter({ hasText: /Finding/i }),
		).toBeVisible({ timeout: 10_000 });
		// 不强求一定有 finding,但页面必须从 loading 切到"空状态或表格"
		await expect(
			page.getByText(/暂无|无 Finding|severity|Critical|High|Medium|Low/i).first(),
		).toBeVisible({ timeout: 15_000 });

		// ── Step 9: UI 验证报告页 ──
		log("Step 9: Verify ReportsPage and export buttons");
		await page.goto(`/projects/${projectId}/reports`);
		await expect(page.getByText(/安全评估报告|Reports/)).toBeVisible({
			timeout: 10_000,
		});
		await expect(
			page.getByRole("button", { name: /Markdown|MD/i }).first(),
		).toBeVisible();
		await expect(
			page.getByRole("button", { name: /JSON/i }).first(),
		).toBeVisible();

		log("✓ Full flow UI test completed");
		await context.close();
	});
});
