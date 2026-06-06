/**
 * 测试层级: E2E
 * 覆盖流程: RunsPage SSE 连接 → UI 显示「SSE 实时连接」→ UI 启动内网扫描 → 扫描完成后 UI 状态更新
 * 前置依赖: docker-compose.e2e.yml（anchor-server / anchor-worker / rangefield）
 * UI 断言点:
 *   - RunsPage 右上角显示「SSE 实时连接」（非「轮询模式」）
 *   - ScanModal 提交后出现「扫描任务已启动」toast
 *   - pipeline 完成后 run 卡片显示 completed（无需手动刷新页面）
 * API 仅用于:
 *   - cleanup(setup/teardown)
 *   - 目标注入(§3.3 例外: scope confirm 产品 bug)
 *   - 长扫描进度轮询(§3.3 例外条款)
 */
import { expect, test } from "@playwright/test";
import { cleanupTestData, addTarget } from "../fixtures/db-utils";
import { waitForPipeline } from "../fixtures/api-helpers";
import { E2E_API_BASE, E2E_API_TOKEN } from "../fixtures/e2e-env";

const API_BASE = E2E_API_BASE;
const API_TOKEN = E2E_API_TOKEN;
const TARGET_IP = "172.31.0.10";

test.setTimeout(30 * 60 * 1000);

test.describe.serial("SSE 实时连接 E2E", () => {
	test.beforeAll(async () => {
		await cleanupTestData();
	});

	test.afterAll(async () => {
		await cleanupTestData();
	});

	test("RunsPage SSE 连接 + 扫描完成后 UI 自动更新", async ({ page }) => {
		const log = (msg: string) => {
			const ts = new Date().toISOString().slice(11, 19);
			console.log(`[${ts}] ${msg}`);
		};

		// ── Step 1: UI 创建项目 ──
		const projectName = `SSE-${Date.now()}`;
		log(`Step 1: Create project "${projectName}" via UI`);
		await page.goto("/projects");
		await expect(
			page.getByRole("heading", { name: /项目与授权边界|项目管理/ }),
		).toBeVisible({ timeout: 10_000 });

		await page
			.getByPlaceholder("例如：2024 Q2 外部红队评估")
			.fill(projectName);
		await page.getByPlaceholder("客户名称或部门").fill("E2E SSE");
		await page
			.getByPlaceholder("测试目的或项目背景")
			.fill("SSE 实时连接验收");
		await page.getByRole("button", { name: "创建项目", exact: true }).click();

		const projectCard = page.getByRole("heading", { name: projectName });
		await expect(projectCard).toBeVisible({ timeout: 10_000 });
		await projectCard.click();
		await expect(page).toHaveURL(/\/projects\/[^/]+\/targets/);
		const projectId = page.url().match(/\/projects\/([^/]+)\/targets/)![1];

		// ── Step 2: API 注入 scope + 目标(§3.3 例外) ──
		log(`Step 2: API inject scope + target ${TARGET_IP}`);
		const scopeRes = await page.request.post(`${API_BASE}/scope-rules`, {
			headers: {
				Authorization: `Bearer ${API_TOKEN}`,
				"Content-Type": "application/json",
			},
			data: {
				project_id: projectId,
				action: "include",
				type: "cidr",
				value: `${TARGET_IP}/32`,
				reason: "E2E SSE scope",
			},
		});
		expect([200, 201]).toContain(scopeRes.status());
		await addTarget(projectId, { type: "ip", value: TARGET_IP });

		// ── Step 3: RunsPage — SSE 连接 UI 断言 ──
		log("Step 3: Verify SSE live badge on RunsPage");
		await page.goto(`/projects/${projectId}/runs`);
		await expect(
			page.getByRole("heading", { name: "扫描执行" }),
		).toBeVisible({ timeout: 10_000 });
		await expect(page.getByText("SSE 实时连接")).toBeVisible({
			timeout: 30_000,
		});
		await expect(page.getByText("轮询模式")).not.toBeVisible();

		// ── Step 4: UI 启动内网扫描 ──
		log("Step 4: Start internal scan via ScanModal");
		await page.getByRole("button", { name: /启动扫描/ }).first().click();
		await expect(
			page.getByRole("heading", { name: /新建扫描流水线/ }),
		).toBeVisible();
		await page.locator("button", { hasText: "内网扫描" }).first().click();
		await page.getByRole("button", { name: /配置参数/ }).click();
		await page.getByLabel("端口模式 -tp 预设").click();
		await page.getByLabel("Top-N 端口预设").selectOption("top100");
		const ffufLabel = page.getByText("Ffuf", { exact: true });
		await expect(ffufLabel).toBeVisible();
		await ffufLabel.locator("..").click();
		await page.getByRole("button", { name: /立即启动扫描/ }).click();
		await expect(page.getByText("扫描任务已启动")).toBeVisible({
			timeout: 10_000,
		});

		// ── Step 5: 轮询等待完成(§3.3 例外) ──
		log("Step 5: Wait for pipeline completion");
		const { status } = await waitForPipeline(projectId, 25 * 60 * 1000);
		expect(status).toBe("completed");

		// ── Step 6: UI 断言 run 状态已更新（SSE/polling 触发 loadRuns）──
		log("Step 6: Verify completed status visible on RunsPage without reload");
		await expect(page.getByText("completed").first()).toBeVisible({
			timeout: 30_000,
		});
		// SSE 连接在扫描结束后仍应保持
		await expect(page.getByText("SSE 实时连接")).toBeVisible({
			timeout: 10_000,
		});

		log("✓ SSE realtime E2E completed");
	});
});
