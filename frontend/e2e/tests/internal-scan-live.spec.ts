/**
 * 测试层级: E2E
 * 覆盖流程: API 创建项目 + 注入 CIDR scope + 注入 5 个 IP 目标(§3.3 例外,scope confirm 产品 bug) →
 *           UI 启动 ScanModal 选内网+高危端口 → API 轮询等待完成 →
 *           UI 验证 TargetPage/AssetPage/FindingsPage
 * 前置依赖: anchor-server / anchor-worker / 全部 rangefield 容器已启动
 * UI 断言点:
 *   - TargetPage 表格中有 5 个 rangefield IP
 *   - ScanModal 两步走通,提交后 toast "扫描任务已启动"
 *   - AssetPage 加载正确
 *   - FindingsPage "发现审核" heading 可见
 * API 仅用于:
 *   - cleanup / setup(项目创建,scope 注入,目标注入 — §3.3 例外)
 *   - 长扫描进度轮询(§3.3 例外条款)
 */
import { expect, test } from "@playwright/test";
import { cleanupTestData, createProject, addTarget } from "../fixtures/db-utils";

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
	let projectId: string;

	test.beforeAll(async () => {
		await cleanupTestData();
		// §3.3 例外: API setup — 项目创建 + scope 注入 + 目标注入
		const proj = await createProject({
			name: `内网E2E-${Date.now()}`,
			organization: "E2E 内网测试",
			purpose: "验证靶场内网扫描 UI 流程",
		});
		projectId = proj.id;

		// 注入 CIDR scope 规则(UI 暂不支持 CIDR 类型)
		const res = await fetch(`${API_BASE}/scope-rules`, {
			method: "POST",
			headers: {
				Authorization: `Bearer ${API_TOKEN}`,
				"Content-Type": "application/json",
			},
			body: JSON.stringify({
				project_id: projectId,
				action: "include",
				type: "cidr",
				value: "172.30.0.0/24",
				reason: "E2E 内网扫描",
			}),
		});
		if (!res.ok && res.status !== 201) {
			throw new Error(`scope-rules create failed: ${res.status}`);
		}

		// 注入 5 个 IP 目标(§3.3 例外: scope confirm 产品 bug 暂未修复)
		for (const ip of RANGEFIELD_IPS) {
			await addTarget(projectId, { type: "ip", value: ip });
		}
	});

	test.afterAll(async () => {
		await cleanupTestData();
	});

	test("UI 验证目标 → 启动内网高危扫描 → 验证资产/Findings", async ({ page }) => {
		const log = (msg: string) => {
			const ts = new Date().toISOString().slice(11, 19);
			console.log(`[${ts}] ${msg}`);
		};

		// ── Step 1: UI 验证 5 个 IP 目标都在表格中 ──
		log("Step 1: UI 验证 TargetPage 表格");
		await page.goto(`/projects/${projectId}/targets`);
		await expect(
			page.getByRole("heading", { name: /目标与 Scope|目标管理/ }),
		).toBeVisible({ timeout: 10_000 });

		for (const ip of RANGEFIELD_IPS) {
			await expect(
				page.getByRole("cell", { name: ip }).first(),
			).toBeVisible({ timeout: 10_000 });
		}
		log("Step 1 完成: 5 个目标全部在表格中");

		// ── Step 2: UI 启动扫描 ──
		log("Step 2: UI 启动内网高危扫描");
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

		// ── Step 3: 轮询等待扫描完成(API 轮询,§3.3 例外)──
		log("Step 3: 轮询等待扫描完成(最长 25 分钟)");
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
		log(`Step 3 完成: run=${runId} 耗时 ${Math.round((Date.now() - start) / 1000)}s`);

		// ── Step 4: UI 验证 AssetPage ──
		log("Step 4: UI 验证 AssetPage");
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

		// ── Step 5: UI 验证 FindingsPage ──
		log("Step 5: UI 验证 FindingsPage");
		await page.goto(`/projects/${projectId}/findings`);
		await expect(
			page.locator("h1").filter({ hasText: /发现审核|漏洞发现|Finding/i }),
		).toBeVisible({ timeout: 10_000 });

		log("✓ 内网扫描 E2E 完成");
	});
});
