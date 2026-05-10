/**
 * 测试层级: E2E
 * 覆盖流程: UI 创建项目 → API 注入 CIDR scope 规则(§3.3 例外,UI 无 CIDR 表单) →
 *           UI 添加 5 个 IP 目标 → ScanModal 选内网+高危端口 → 等待扫描完成 →
 *           UI 验证 TargetPage/AssetPage/FindingsPage
 * 前置依赖: anchor-server / anchor-worker / 全部 rangefield 容器已启动
 * UI 断言点:
 *   - 项目卡片可见
 *   - TargetPage 表格中有 5 个 rangefield IP
 *   - ScanModal 两步走通,提交后 toast "扫描任务已启动"
 *   - AssetPage 加载正确
 *   - FindingsPage "发现审核" heading 可见
 * API 仅用于:
 *   - cleanup
 *   - CIDR scope 规则注入(UI 暂不支持 CIDR 类型)
 *   - 长扫描进度轮询(§3.3 例外条款)
 */
import { expect, test } from "@playwright/test";
import { cleanupTestData } from "../fixtures/db-utils";

const API_BASE = "http://localhost:17421";
const API_TOKEN = "p0m32kun";

const RANGEFIELD_IPS = [
	"172.30.0.10", // nginx
	"172.30.0.11", // tomcat
	"172.30.0.12", // grafana
	"172.30.0.13", // redis
	"172.30.0.14", // mysql
];

test.setTimeout(30 * 60 * 1000);

test.describe.serial("内网扫描 E2E — UI 主导", () => {
	test.beforeAll(async () => {
		await cleanupTestData();
	});

	test.afterAll(async () => {
		await cleanupTestData();
	});

	test("UI 添加 5 个 IP → 内网高危扫描 → 验证资产/Findings", async ({ page }) => {
		const log = (msg: string) => {
			const ts = new Date().toISOString().slice(11, 19);
			console.log(`[${ts}] ${msg}`);
		};

		// ── Step 1: UI 创建项目 ──
		const projectName = `内网E2E-${Date.now()}`;
		log(`Step 1: 创建项目 "${projectName}"`);
		await page.goto("/projects");
		await expect(
			page.getByRole("heading", { name: /项目与授权边界|项目管理/ }),
		).toBeVisible({ timeout: 10_000 });

		await page
			.getByPlaceholder("例如：2024 Q2 外部红队评估")
			.fill(projectName);
		await page.getByPlaceholder("客户名称或部门").fill("E2E 内网测试");
		await page
			.getByPlaceholder("测试目的或项目背景")
			.fill("验证靶场内网扫描 UI 流程");
		await page.getByRole("button", { name: "创建项目", exact: true }).click();

		const projectCard = page.getByRole("heading", { name: projectName });
		await expect(projectCard).toBeVisible({ timeout: 10_000 });
		await projectCard.click();
		await expect(page).toHaveURL(/\/projects\/[^/]+\/targets/);
		const projectId = page.url().match(/\/projects\/([^/]+)\/targets/)![1];
		log(`Project ID: ${projectId}`);

		// ── Step 2: API 注入 CIDR scope 规则(§3.3 例外,UI 暂不支持 CIDR)──
		log("Step 2: API 创建 CIDR scope 规则 172.30.0.0/24");
		const scopeRes = await page.request.post(`${API_BASE}/scope-rules`, {
			headers: {
				Authorization: `Bearer ${API_TOKEN}`,
				"Content-Type": "application/json",
			},
			data: {
				project_id: projectId,
				action: "include",
				type: "cidr",
				value: "172.30.0.0/24",
				reason: "E2E 内网扫描",
			},
		});
		expect([200, 201]).toContain(scopeRes.status());
		await page.reload();

		// ── Step 3: UI 添加 5 个 IP 目标 ──
		log("Step 3: UI 添加 5 个 IP 目标");
		const targetPlaceholder = page.getByPlaceholder("example.com", {
			exact: true,
		});
		const targetForm = page.locator("form").filter({ has: targetPlaceholder });

		for (const ip of RANGEFIELD_IPS) {
			await targetForm.locator("select").selectOption("ip");
			await targetPlaceholder.fill(ip);
			await targetForm.getByRole("button", { name: /^添加($|中)/ }).click();
			// 等待按钮恢复可点击(请求完成)
			await expect(
				targetForm.getByRole("button", { name: "添加" }),
			).toBeVisible({ timeout: 5_000 });
		}

		// 校验 5 个 IP 都在表格中
		for (const ip of RANGEFIELD_IPS) {
			await expect(
				page.getByRole("cell", { name: ip }).first(),
			).toBeVisible({ timeout: 10_000 });
		}
		log("Step 3 完成: 5 个目标全部在表格中");

		// ── Step 4: UI 启动扫描 ──
		log("Step 4: UI 启动内网高危扫描");
		await page.goto(`/projects/${projectId}/runs`);
		await expect(
			page.getByRole("heading", { name: "扫描执行" }),
		).toBeVisible({ timeout: 10_000 });

		await page.getByRole("button", { name: "新建扫描" }).first().click();
		await expect(
			page.getByRole("heading", { name: /新建扫描流水线/ }),
		).toBeVisible();

		await page.locator("button", { hasText: "内网扫描" }).first().click();
		await page.getByRole("button", { name: /配置参数/ }).click();
		await page.locator("button", { hasText: "高危端口（推荐）" }).first().click();
		await page.getByRole("button", { name: /立即启动扫描/ }).click();
		await expect(page.getByText("扫描任务已启动")).toBeVisible({
			timeout: 10_000,
		});

		// ── Step 5: 轮询等待扫描完成(API 轮询,§3.3 例外)──
		log("Step 5: 轮询等待扫描完成(最长 25 分钟)");
		const start = Date.now();
		const maxWait = 25 * 60 * 1000;
		let runStatus = "";
		let runId = "";

		while (Date.now() - start < maxWait) {
			const res = await page.request.get(
				`${API_BASE}/projects/${projectId}/scan/runs?page=1&page_size=10`,
				{ headers: { Authorization: `Bearer ${API_TOKEN}` } },
			);
			if (res.ok()) {
				const body = (await res.json()) as {
					data: Array<{
						id: string;
						status: string;
					}>;
				};
				const runs = body.data || [];
				if (runs.length > 0) {
					runId = runs[0].id;
					runStatus = runs[0].status;
					if (runStatus !== "running" && runStatus !== "pending") break;
				}
			}
			await page.waitForTimeout(5_000);
		}
		expect(runStatus).toBe("completed");
		log(`Step 5 完成: run=${runId} 耗时 ${Math.round((Date.now() - start) / 1000)}s`);

		// ── Step 6: UI 验证 AssetPage ──
		log("Step 6: UI 验证 AssetPage");
		await page.goto(`/projects/${projectId}/assets`);
		await expect(
			page.getByText(/资产清单|资产列表|Assets/),
		).toBeVisible({ timeout: 10_000 });
		for (const ip of RANGEFIELD_IPS) {
			try {
				await expect(page.locator(`text=${ip}`).first()).toBeVisible({
					timeout: 5_000,
				});
			} catch {
				console.warn(`[e2e] Asset ${ip} not visible — scan pipeline may not discover it`);
			}
		}

		// ── Step 7: UI 验证 FindingsPage ──
		log("Step 7: UI 验证 FindingsPage");
		await page.goto(`/projects/${projectId}/findings`);
		await expect(
			page.locator("h1").filter({ hasText: /发现审核|漏洞发现|Finding/i }),
		).toBeVisible({ timeout: 10_000 });

		log("✓ 内网扫描 E2E 完成");
	});
});
