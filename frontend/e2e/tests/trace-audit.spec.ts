/**
 * 测试层级: E2E
 * 覆盖流程: 高危端口扫描完成 → RunsPage 工具调用日志可见 → FindingsPage 调用溯源可见
 * 前置依赖: anchor-server / anchor-worker / rangefield(rf-redis 6379)
 * UI 断言点:
 *   - RunsPage 选中 run 后「工具调用日志」卡片非空，含 naabu/nuclei/httpx 等工具名
 *   - FindingsPage 打开 finding 详情后「调用溯源」区域显示 Run 与 tool 信息
 * API 仅用于:
 *   - cleanup(setup/teardown)
 *   - 目标注入(§3.3 例外: scope confirm 产品 bug)
 *   - 长扫描进度轮询(§3.3 例外条款)
 */
import { expect, test } from "@playwright/test";
import { cleanupTestData, addTarget } from "../fixtures/db-utils";
import { waitForPipeline, listFindings } from "../fixtures/api-helpers";
import { E2E_API_BASE, E2E_API_TOKEN } from "../fixtures/e2e-env";

const API_BASE = E2E_API_BASE;
const API_TOKEN = E2E_API_TOKEN;
const REDIS_IP = "172.31.0.13";

test.setTimeout(30 * 60 * 1000);

test.describe.serial("工具调用溯源 E2E", () => {
	test.beforeAll(async () => {
		await cleanupTestData();
	});

	test.afterAll(async () => {
		await cleanupTestData();
	});

	test("扫描完成后 RunsPage 工具调用日志 + FindingsPage 调用溯源", async ({
		page,
	}) => {
		const log = (msg: string) => {
			const ts = new Date().toISOString().slice(11, 19);
			console.log(`[${ts}] ${msg}`);
		};

		// ── Step 1: UI 创建项目 ──
		const projectName = `Trace-${Date.now()}`;
		log(`Step 1: Create project "${projectName}" via UI`);
		await page.goto("/projects");
		await expect(
			page.getByRole("heading", { name: /项目与授权边界|项目管理/ }),
		).toBeVisible({ timeout: 10_000 });

		await page
			.getByPlaceholder("例如：2024 Q2 外部红队评估")
			.fill(projectName);
		await page.getByPlaceholder("客户名称或部门").fill("E2E Trace");
		await page
			.getByPlaceholder("测试目的或项目背景")
			.fill("工具调用溯源 UI 验收");
		await page.getByRole("button", { name: "创建项目", exact: true }).click();

		const projectCard = page.getByRole("heading", { name: projectName });
		await expect(projectCard).toBeVisible({ timeout: 10_000 });
		await projectCard.click();
		const projectId = page.url().match(/\/projects\/([^/]+)\/targets/)![1];

		// ── Step 2: API 注入 scope + Redis 目标(§3.3 例外) ──
		log("Step 2: API inject scope + Redis target");
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
				reason: "E2E trace scope",
			},
		});
		expect([200, 201]).toContain(scopeRes.status());
		await addTarget(projectId, { type: "ip", value: REDIS_IP });
		await page.reload();
		await expect(
			page.getByRole("cell", { name: REDIS_IP }).first(),
		).toBeVisible({ timeout: 10_000 });

		// ── Step 3: UI 启动高危端口扫描 ──
		log("Step 3: Start high-risk port scan via ScanModal");
		await page.goto(`/projects/${projectId}/runs`);
		await page.getByRole("button", { name: /启动扫描/ }).first().click();
		await expect(
			page.getByRole("heading", { name: /新建扫描流水线/ }),
		).toBeVisible();
		await page.locator("button", { hasText: "内网扫描" }).first().click();
		await page.getByRole("button", { name: /高级配置/ }).click();
		await page.getByLabel("端口模式 -p 自定义").click();
		await expect(page.getByLabel("自定义端口列表")).toHaveValue(/6379/);
		await page.getByRole("button", { name: /^Ffuf/ }).click();
		await page.getByRole("button", { name: /立即启动扫描/ }).click();
		await expect(page.getByText("扫描任务已启动")).toBeVisible({
			timeout: 10_000,
		});

		// ── Step 4: 等待 pipeline 完成 ──
		log("Step 4: Wait for pipeline completion");
		const { status, runId } = await waitForPipeline(projectId);
		expect(status).toBe("completed");
		log(`Pipeline ${runId} completed`);

		// ── Step 5: UI 验证工具调用日志 ──
		log("Step 5: Verify tool call logs on RunsPage");
		await page.goto(`/projects/${projectId}/runs`);
		await page.locator('[class*="cursor-pointer"]').first().click();
		await expect(page.getByText("工具调用日志")).toBeVisible({
			timeout: 15_000,
		});
		// 至少有一条工具记录（naabu / httpx / nuclei 等）
		await expect(
			page.getByText(/naabu|nuclei|httpx|nmap|subfinder/i).first(),
		).toBeVisible({ timeout: 15_000 });

		// ── Step 6: UI 验证 Finding 调用溯源 ──
		log("Step 6: Verify finding trace on FindingsPage");
		const findings = await listFindings(projectId);
		if (findings.length === 0) {
			test.skip(true, "扫描未产出 finding，跳过溯源 UI 断言");
			return;
		}

		await page.goto(`/projects/${projectId}/findings`);
		await expect(
			page.locator("h1").filter({ hasText: /发现审核|漏洞发现|Finding/i }),
		).toBeVisible({ timeout: 10_000 });

		// 点击第一条 finding 卡片打开详情
		await page.locator(".group.cursor-pointer").first().click();
		await expect(page.getByText("漏洞详情报告")).toBeVisible({
			timeout: 10_000,
		});
		await expect(page.getByText("调用溯源")).toBeVisible();
		// Run 或 tool 信息至少出现一项
		await expect(
			page
				.getByText(/Run|nuclei|naabu|httpx|completed|running/i)
				.first(),
		).toBeVisible({ timeout: 15_000 });

		log("✓ Trace audit E2E completed");
	});
});
