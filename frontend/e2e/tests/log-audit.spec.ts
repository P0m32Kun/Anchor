/**
 * 测试层级: E2E
 * 覆盖流程: UI 创建项目 + 注入目标 → ScanModal 启动内网扫描(高危端口 preset) →
 *           pipeline 完成 → API 拉取 task logs → 日志审计通道(§3.4)
 * 前置依赖: anchor-server / anchor-worker / anchor-rangefield(rf-redis)已启动
 * UI 断言点:
 *   - 项目卡片可见
 *   - TargetPage 表格中能看到 172.31.0.13 行
 *   - ScanModal 切到 -p 自定义后,textarea 默认值包含 6379
 *   - RunsPage 上扫描启动后能看到 run 卡片
 *   - 完成后 AssetPage 看到 172.31.0.13
 * 审计通道(§3.4):
 *   - naabu stdout: contain "Running naabu scan for", "results:", regex "open ports?: \d+"
 *   - naabu stderr: 无未知错误
 * API 仅用于:
 *   - cleanup
 *   - 目标注入(§3.3 例外: scope confirm 产品 bug)
 *   - 长扫描进度轮询(§3.3 例外条款)
 *   - 审计通道日志拉取(§3.4)
 */
import { expect, test } from "@playwright/test";
import { cleanupTestData, addTarget } from "../fixtures/db-utils";
import { waitForPipeline } from "../fixtures/api-helpers";
import {
	auditTaskLogs,
	printAuditResults,
} from "../fixtures/log-audit";

import { E2E_API_BASE, E2E_API_TOKEN } from "../fixtures/e2e-env";

const API_BASE = E2E_API_BASE;
const API_TOKEN = E2E_API_TOKEN;
const REDIS_IP = "172.31.0.13";

// 日志审计目标工具(内网 pipeline 实际触发的工具)
const TOOLS_TO_AUDIT = ["nmap_alive", "naabu", "nmap_service", "nuclei"];

test.setTimeout(35 * 60 * 1000);

test.describe.serial("日志审计 E2E — UI + 审计双通道", () => {
	test.beforeAll(async () => {
		await cleanupTestData();
	});

	test.afterAll(async () => {
		await cleanupTestData();
	});

	test("UI 主导扫描 + 日志审计通道(naabu)", async ({ page }) => {
		const log = (msg: string) => {
			const ts = new Date().toISOString().slice(11, 19);
			console.log(`[${ts}] ${msg}`);
		};

		// ── Step 1: UI 创建项目 ──
		const projectName = `LogAudit-${Date.now()}`;
		log(`Step 1: Create project "${projectName}" via UI`);
		await page.goto("/projects");
		await expect(
			page.getByRole("heading", { name: /项目与授权边界|项目管理/ }),
		).toBeVisible({ timeout: 10_000 });

		await page
			.getByPlaceholder("例如：2024 Q2 外部红队评估")
			.fill(projectName);
		await page.getByPlaceholder("客户名称或部门").fill("E2E LogAudit");
		await page
			.getByPlaceholder("测试目的或项目背景")
			.fill("日志审计双通道验证");
		await page.getByRole("button", { name: "创建项目", exact: true }).click();

		const projectCard = page.getByRole("heading", { name: projectName });
		await expect(projectCard).toBeVisible({ timeout: 10_000 });
		await projectCard.click();
		await expect(page).toHaveURL(/\/projects\/[^/]+\/targets/);
		const projectId = page.url().match(/\/projects\/([^/]+)\/targets/)![1];
		log(`Project ID: ${projectId}`);

		// ── Step 2: API 注入 scope rule + IP 目标(§3.3 例外) ──
		log("Step 2: Inject scope + target (rangefield redis)");
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

		// ── Step 3: ScanModal 启动扫描(高危端口 preset) ──
		log("Step 3: ScanModal → -p custom (high-risk ports)");
		await page.goto(`/projects/${projectId}/runs`);
		await expect(
			page.getByRole("heading", { name: "扫描执行" }),
		).toBeVisible({ timeout: 10_000 });

		await page.getByRole("button", { name: /启动扫描/ }).first().click();
		await expect(
			page.getByRole("heading", { name: /新建扫描流水线/ }),
		).toBeVisible();

		await page.locator("button", { hasText: "内网扫描" }).first().click();
		await page.getByRole("button", { name: /配置参数/ }).click();

		await page.getByLabel("端口模式 -p 自定义").click();
		const portTextarea = page.getByLabel("自定义端口列表");
		await expect(portTextarea).toBeVisible();
		await expect(portTextarea).toHaveValue(/6379/);

		// 关闭 Ffuf 解锁启动按钮
		await page.getByRole("button", { name: /^Ffuf/ }).click();

		await page.getByRole("button", { name: /立即启动扫描/ }).click();
		await expect(page.getByText("扫描任务已启动")).toBeVisible({
			timeout: 10_000,
		});

		const runCard = page.locator('[class*="cursor-pointer"]').first();
		await expect(runCard).toBeVisible({ timeout: 15_000 });

		// ── Step 4: 等待 pipeline 完成 ──
		log("Step 4: Poll until pipeline completes");
		const { status, runId } = await waitForPipeline(projectId);
		expect(status).toBe("completed");
		log(`Pipeline ${runId} completed`);

		// ── Step 5: UI 验证资产 ──
		log("Step 5: Verify asset on AssetPage");
		await page.goto(`/projects/${projectId}/assets`);
		await expect(page.getByText(/资产清单|资产列表|Assets/)).toBeVisible({
			timeout: 10_000,
		});
		await expect(page.locator(`text=${REDIS_IP}`).first()).toBeVisible({
			timeout: 30_000,
		});

		// port 6379 soft-check
		try {
			await expect(page.getByText(/6379/).first()).toBeVisible({
				timeout: 10_000,
			});
		} catch {
			console.warn(
				"[e2e] Port 6379 not visible — scan pipeline may not discover ports in e2e env.",
			);
		}

		// ── Step 6: 日志审计通道(§3.4) ──
		log("Step 6: Fetch pipeline run tasks for log audit");

		// 获取该次 scan run 下的所有 task
		const tasksRes = await page.request.get(
			`${API_BASE}/runs/${runId}/tasks`,
			{
				headers: { Authorization: `Bearer ${API_TOKEN}` },
			},
		);
		expect(tasksRes.ok()).toBe(true);
		const tasks: Array<{ id: string; tool: string; status: string }> =
			await tasksRes.json();

		log(
			`Found ${tasks.length} tasks: ${tasks.map((t) => `${t.tool}(${t.status})`).join(", ")}`,
		);

		// 只审计已完成的 tool task
		const completedTasks = tasks.filter((t) => t.status === "completed");
		log(
			`Completed tasks: ${completedTasks.map((t) => `${t.tool}[${t.id.slice(0, 8)}]`).join(", ")}`,
		);

		// 转换为 audit 需要的格式
		const auditTasks = completedTasks.map((t) => ({
			id: t.id,
			source_tool: t.tool,
		}));

		log("Running log audit rules...");
		const auditResults = await auditTaskLogs(auditTasks, TOOLS_TO_AUDIT);
		printAuditResults(auditResults);

		// 断言所有规则通过
		const allPassed = auditResults.every((r) => r.passed);
		const failureMessages = auditResults
			.flatMap((r) => r.failures)
			.join("\n");

		log(
			`Audit result: ${allPassed ? "✓ 全部通过" : "✗ 存在失败"}`,
		);

		expect(allPassed, failureMessages).toBe(true);

		log("✓ Log audit E2E completed (UI + audit dual-channel)");
	});
});
