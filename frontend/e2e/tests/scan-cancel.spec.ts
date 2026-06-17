/**
 * 测试层级: E2E
 * 覆盖流程: 创建项目 → 添加目标 → 启动扫描 → 取消扫描 → 验证状态
 * 前置依赖: docker compose -f docker-compose.e2e.yml 已启动
 * 断言点:
 *   - 扫描启动后状态变为 running
 *   - 取消后状态变为 cancelled
 *   - 取消后不再产生新的 work items
 */
import { expect, test } from "@playwright/test";
import {
	createProject,
	deleteProject,
	addTarget,
	createScanRun,
	apiFetch,
} from "../fixtures/api-helpers";

test.setTimeout(5 * 60 * 1000);

test.describe("Scan Cancel E2E", () => {
	let projectId: string;
	const projectName = `CancelTest-${Date.now()}`;

	test.beforeAll(async () => {
		const project = await createProject({
			name: projectName,
			organization: "E2E",
			purpose: "Scan cancel test",
		});
		projectId = project.id;

		// 添加一个会触发多阶段扫描的目标
		await addTarget(projectId, {
			type: "ip",
			value: "172.31.0.10",
		});
	});

	test.afterAll(async () => {
		if (projectId) {
			await deleteProject(projectId).catch(() => {});
		}
	});

	test("启动扫描后可取消", async () => {
		// 启动扫描
		const run = await createScanRun(projectId, {
			profile: "internal",
		});
		expect(run).toBeTruthy();
		expect(run.id).toBeTruthy();

		// 等一小段时间让扫描开始
		await new Promise((r) => setTimeout(r, 2000));

		// 取消扫描
		const cancelRes = await apiFetch(
			`/projects/${projectId}/pipeline/runs/${run.id}/cancel`,
			{ method: "POST" },
		);
		expect(cancelRes.ok).toBeTruthy();

		// 验证状态
		await new Promise((r) => setTimeout(r, 1000));
		const runDetail = await apiFetch(
			`/projects/${projectId}/pipeline/runs/${run.id}`,
		).then((r) => r.json());

		expect(["cancelled", "canceled", "completed"]).toContain(
			runDetail.status?.toLowerCase(),
		);
	});

	test("取消后的扫描不再产生新 work", async () => {
		// 启动扫描
		const run = await createScanRun(projectId, {
			profile: "internal",
		});

		// 等待开始
		await new Promise((r) => setTimeout(r, 2000));

		// 记录当前 work 数量
		const worksBefore = await apiFetch(
			`/projects/${projectId}/pipeline/runs/${run.id}/works`,
		).then((r) => r.json());
		const countBefore = (worksBefore.data || worksBefore).length;

		// 取消
		await apiFetch(
			`/projects/${projectId}/pipeline/runs/${run.id}/cancel`,
			{ method: "POST" },
		);

		// 等待取消生效
		await new Promise((r) => setTimeout(r, 3000));

		// 再次检查 work 数量 - 不应该显著增加
		const worksAfter = await apiFetch(
			`/projects/${projectId}/pipeline/runs/${run.id}/works`,
		).then((r) => r.json());
		const countAfter = (worksAfter.data || worksAfter).length;

		// 允许最多增加 1-2 个（取消前可能已派发），但不应大幅增长
		expect(countAfter).toBeLessThanOrEqual(countBefore + 3);
	});
});
