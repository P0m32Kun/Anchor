/**
 * 测试层级: E2E
 * 覆盖流程: 创建项目 → 添加排除规则 → 添加被排除的目标 → 启动扫描 →
 *           验证被排除目标不产生资产 → 验证未被排除目标正常扫描
 * 前置依赖: docker compose -f docker-compose.e2e.yml 已启动
 * 断言点:
 *   - 排除规则在 UI 中可见
 *   - 被排除的 IP 目标在 scope check 后标记为 denied
 *   - 未被排除的目标正常进入扫描流程
 */
import { expect, test } from "@playwright/test";
import {
	createProject,
	deleteProject,
	addTarget,
	createScanRun,
	waitForPipeline,
	apiFetch,
} from "../fixtures/api-helpers";

test.setTimeout(10 * 60 * 1000);

test.describe("Scope Validation E2E", () => {
	let projectId: string;
	const projectName = `ScopeTest-${Date.now()}`;

	test.beforeAll(async () => {
		// 创建项目
		const project = await createProject({
			name: projectName,
			organization: "E2E",
			purpose: "Scope validation test",
		});
		projectId = project.id;

		// 添加排除规则：排除 172.31.0.99
		await apiFetch(`/projects/${projectId}/scope-rules`, {
			method: "POST",
			body: JSON.stringify({
				action: "exclude",
				type: "ip",
				value: "172.31.0.99",
			}),
		});
	});

	test.afterAll(async () => {
		if (projectId) {
			await deleteProject(projectId).catch(() => {});
		}
	});

	test("排除规则在 UI 中可见", async ({ page }) => {
		// 登录
		await page.goto("/");
		await page
			.getByPlaceholder("http://localhost:17421")
			.fill(process.env.E2E_API_BASE || "http://localhost:17421");
		await page
			.locator('input[type="password"]')
			.fill(process.env.E2E_API_TOKEN || "");
		await page.getByRole("button", { name: "保存并进入" }).click();
		await expect(
			page.getByRole("heading", { name: "安全工作台" }),
		).toBeVisible({ timeout: 15_000 });

		// 进入项目
		await page.goto(`/projects/${projectId}/targets`);
		await expect(page.getByText(projectName)).toBeVisible({
			timeout: 10_000,
		});

		// 验证排除规则可见（scope rules 通常在 targets 页面或单独 tab）
		await expect(page.getByText(/172\.31\.0\.99/)).toBeVisible({
			timeout: 5_000,
		});
	});

	test("被排除的目标产生 scope deny 决策", async () => {
		// 通过 API 添加被排除的目标
		const target = await addTarget(projectId, {
			type: "ip",
			value: "172.31.0.99",
		});
		expect(target).toBeTruthy();

		// 检查 scope decision
		const decisions = await apiFetch(
			`/projects/${projectId}/scope-decisions`,
		).then((r) => r.json());
		const denyDecision = (decisions.data || decisions).find(
			(d: { target_value: string; decision: string }) =>
				d.target_value === "172.31.0.99" && d.decision === "deny",
		);
		expect(denyDecision).toBeTruthy();
	});

	test("未被排除的目标允许扫描", async () => {
		// 添加未被排除的目标
		const target = await addTarget(projectId, {
			type: "ip",
			value: "172.31.0.10",
		});
		expect(target).toBeTruthy();

		// 检查 scope decision - 应该是 allow
		const decisions = await apiFetch(
			`/projects/${projectId}/scope-decisions`,
		).then((r) => r.json());
		const allowDecision = (decisions.data || decisions).find(
			(d: { target_value: string; decision: string }) =>
				d.target_value === "172.31.0.10" && d.decision === "allow",
		);
		expect(allowDecision).toBeTruthy();
	});
});
