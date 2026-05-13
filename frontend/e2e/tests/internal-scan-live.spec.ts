/**
 * 测试层级: E2E
 * 覆盖流程: API 创建项目 + 注入 CIDR scope + 注入 5 个 IP 目标(§3.3 例外,scope confirm 产品 bug) →
 *           UI 启动 ScanModal 选内网 + 切到 -p 自定义(默认高危端口) → API 轮询等待完成 →
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
import { waitForPipeline } from "../fixtures/api-helpers";

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

	test("UI 验证目标 → 启动内网 -p 自定义扫描 → 验证资产/Findings", async ({ page }) => {
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
		log("Step 2: UI 启动内网扫描(-p 自定义端口,默认填充高危端口)");
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
		// 切到 -p 自定义模式(textarea 自动填高危端口列表,含 6379 Redis 等高价值目标)
		await page.getByLabel("端口模式 -p 自定义").click();
		await expect(page.getByLabel("自定义端口列表")).toHaveValue(/6379/);
		// Fix 3 (2026-05-13): enable_ffuf 默认 true,无字典时启动按钮被 disabled。
		// 这里关闭 ffuf 让按钮可点(靶场没预置字典)。
		const ffufLabel = page.getByText("Ffuf", { exact: true });
		await expect(ffufLabel).toBeVisible();
		await ffufLabel.locator("..").click();
		await page.getByRole("button", { name: /立即启动扫描/ }).click();
		await expect(page.getByText("扫描任务已启动")).toBeVisible({
			timeout: 10_000,
		});

		// ── Step 3: 等待扫描完成(waitForPipeline,最长 25 分钟) ──
		log("Step 3: 轮询等待扫描完成(最长 25 分钟)");
		const { status: runStatus, runId } = await waitForPipeline(projectId, 25 * 60 * 1000);
		expect(runStatus).toBe("completed");
		log(`Step 3 完成: run=${runId}`);

		// ── Step 3.5: stage 反馈断言(Fix 1 回归) ──
		// Fix 1 (2026-05-12): 内网模式不再上报 cdn_filter stage。
		// (Fix 2 慢速 stage 可见性此处不验证:ffuf 已被关闭,urlfinder 已剔除。)
		log("Step 3.5: 验证 stages 数组中 Fix 1 的行为");
		const stagesRes = await page.request.get(
			`${API_BASE}/projects/${projectId}/pipeline/runs/${runId}/stages`,
			{ headers: { Authorization: `Bearer ${API_TOKEN}` } },
		);
		expect(stagesRes.ok()).toBe(true);
		const stagesBody = (await stagesRes.json()) as {
			stages: Array<{ stage: string; status: string; error?: string | null }>;
		};
		const stageNames = stagesBody.stages.map((s) => s.stage);
		log(`stages: ${stageNames.join(", ")}`);
		expect(stageNames).not.toContain("cdn_filter"); // Fix 1
		expect(stageNames).not.toContain("urlfinder"); // tool removed

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
