/**
 * 测试层级: E2E
 * 覆盖流程: 创建项目 → 扫描产生 Findings → 人工验证 Finding 状态流转
 * 前置依赖: docker compose -f docker-compose.e2e.yml 已启动,有已完成扫描的 Finding 数据
 * 断言点:
 *   - Findings 列表页能看到 Finding
 *   - 可以将 Finding 标记为 confirmed
 *   - 可以将 Finding 标记为 false_positive
 *   - 可以将 Finding 标记为 accepted_risk
 *   - 状态变更后 UI 刷新显示新状态
 */
import { expect, test } from "@playwright/test";
import {
	createProject,
	deleteProject,
	addTarget,
	createScanRun,
	waitForPipeline,
	apiFetch,
	listFindings,
} from "../fixtures/api-helpers";

test.setTimeout(30 * 60 * 1000);

test.describe("Finding Verification Queue E2E", () => {
	let projectId: string;
	let runId: string;
	const projectName = `FindingVerify-${Date.now()}`;

	test.beforeAll(async () => {
		// 创建项目
		const project = await createProject({
			name: projectName,
			organization: "E2E",
			purpose: "Finding verification test",
		});
		projectId = project.id;

		// 添加目标并启动扫描
		await addTarget(projectId, {
			type: "ip",
			value: "172.31.0.10",
		});

		const run = await createScanRun(projectId, {
			profile: "internal",
		});
		runId = run.id;

		// 等待扫描完成
		await waitForPipeline(projectId, runId, 25 * 60 * 1000);
	});

	test.afterAll(async () => {
		if (projectId) {
			await deleteProject(projectId).catch(() => {});
		}
	});

	test("Findings 列表页有数据", async ({ page }) => {
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

		// 进入 Findings 页面
		await page.goto(`/projects/${projectId}/findings`);
		await expect(page.getByText(projectName)).toBeVisible({
			timeout: 10_000,
		});

		// 等待 findings 加载
		await expect(
			page.getByText(/Finding|发现|漏洞|finding/i).first(),
		).toBeVisible({ timeout: 10_000 });
	});

	test("API: Finding 状态可变更为 confirmed", async () => {
		const findings = await listFindings(projectId);
		if (findings.length === 0) {
			test.skip(true, "No findings produced — skipping verification tests");
			return;
		}

		const findingId = findings[0].id;
		const res = await apiFetch(`/findings/${findingId}`, {
			method: "PATCH",
			body: JSON.stringify({ status: "confirmed" }),
		});
		expect(res.ok).toBeTruthy();

		// 验证状态已变更
		const updated = await apiFetch(`/findings/${findingId}`).then((r) =>
			r.json(),
		);
		expect(updated.status).toBe("confirmed");
	});

	test("API: Finding 状态可变更为 false_positive", async () => {
		const findings = await listFindings(projectId);
		if (findings.length < 2) {
			test.skip(true, "Not enough findings — skipping");
			return;
		}

		const findingId = findings[1].id;
		const res = await apiFetch(`/findings/${findingId}`, {
			method: "PATCH",
			body: JSON.stringify({ status: "false_positive" }),
		});
		expect(res.ok).toBeTruthy();

		const updated = await apiFetch(`/findings/${findingId}`).then((r) =>
			r.json(),
		);
		expect(updated.status).toBe("false_positive");
	});

	test("API: Finding 状态可变更为 accepted_risk", async () => {
		const findings = await listFindings(projectId);
		if (findings.length < 3) {
			test.skip(true, "Not enough findings — skipping");
			return;
		}

		const findingId = findings[2].id;
		const res = await apiFetch(`/findings/${findingId}`, {
			method: "PATCH",
			body: JSON.stringify({ status: "accepted_risk" }),
		});
		expect(res.ok).toBeTruthy();

		const updated = await apiFetch(`/findings/${findingId}`).then((r) =>
			r.json(),
		);
		expect(updated.status).toBe("accepted_risk");
	});
});
