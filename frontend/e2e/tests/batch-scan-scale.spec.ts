/**
 * 测试层级: E2E
 * 场景 ID: E2E-BATCH-01
 * 关联 REQ: REQ-B2, REQ-B10
 * 覆盖流程:
 *   API 注入 120 个 domain 目标 → 外网 low-noise 扫描 → pipeline 完成 →
 *   断言 work 总数与 tool call 总数低于批量调度阈值，且存在 batch_mode work
 * 前置依赖: docker-compose.e2e.yml（anchor-server + worker）
 * API 仅用于:
 *   - seed 120 targets、启动扫描、进度轮询（§3.3 例外）
 *   - 收敛指标断言（work/tool-call 计数，对标 external-scan-conv）
 */

import { expect, test } from "@playwright/test";
import {
	createProject,
	addTarget,
	startScan,
	waitForPipelineRun,
	listScanRunWorks,
	listToolCallLogs,
	getScanRunMetrics,
} from "../fixtures/api-helpers";
import { cleanupTestData } from "../fixtures/db-utils";

/** 120 domain × 旧 per-asset 模型可达 600+ work；批量调度后应显著低于此值 */
const TARGET_COUNT = 120;
const MAX_TOTAL_WORKS = 400;
const MAX_TOOL_CALLS = 200;

test.setTimeout(30 * 60 * 1000);

test.describe.serial("Batch scan scale — E2E-BATCH-01", () => {
	let projectId: string;

	test.beforeAll(async () => {
		await cleanupTestData();
		const project = await createProject({
			name: `BatchScale-${Date.now()}`,
			organization: "E2E Batch",
			purpose: "batch scheduling scale smoke",
		});
		projectId = project.id;

		for (let i = 0; i < TARGET_COUNT; i++) {
			await addTarget(projectId, {
				type: "domain",
				value: `scale${i}.batch-e2e.test`,
			});
		}
	});

	test.afterAll(async () => {
		await cleanupTestData();
	});

	test("120 domain 外网扫描：work/tool-call 收敛且含 batch work", async () => {
		const log = (msg: string) => {
			const ts = new Date().toISOString().slice(11, 19);
			console.log(`[${ts}] ${msg}`);
		};

		log(`Step 1: Start external low-noise scan (${TARGET_COUNT} domains)`);
		const { run_id: runId } = await startScan(projectId, {
			mode: "external",
			config: {
				noise_level: "low",
				enable_dnsx: true,
				enable_cdn_filter: true,
				enable_httpx: true,
				enable_nmap_service: true,
				enable_passive_search: false,
				enable_passive_cert: false,
				enable_passive_url: false,
				enable_subfinder: false,
				enable_ffuf: false,
				enable_katana: false,
				enable_nuclei: false,
				port_range: "80,443",
			},
		});
		expect(runId).toBeTruthy();

		log("Step 2: Wait for pipeline completion");
		const { status } = await waitForPipelineRun(projectId, runId, 25 * 60 * 1000);
		log(`Run ${runId} finished status=${status}`);
		expect(["completed", "failed", "cancelled"]).toContain(status);
		expect(status).toBe("completed");

		log("Step 3: Assert work and tool-call convergence");
		const { items: works, total: workTotal } = await listScanRunWorks(
			projectId,
			runId,
		);
		const { total: toolCallTotal } = await listToolCallLogs(projectId, runId);
		const metrics = await getScanRunMetrics(projectId, runId);

		log(
			`works total=${workTotal} fetched=${works.length} tool_calls=${toolCallTotal} engine=${metrics.engine_state}`,
		);

		expect(workTotal).toBeGreaterThan(0);
		expect(workTotal).toBeLessThanOrEqual(MAX_TOTAL_WORKS);
		expect(toolCallTotal).toBeLessThanOrEqual(MAX_TOOL_CALLS);

		const batchWorks = works.filter((w) => w.batch_mode);
		expect(
			batchWorks.length,
			"expected at least one pooled batch work item",
		).toBeGreaterThan(0);

		const dnsBatch = works.filter(
			(w) => w.action === "DNS_RESOLVE" && w.batch_mode,
		);
		expect(
			dnsBatch.length,
			"120 domains should merge DNS into batch works",
		).toBeGreaterThan(0);
		expect(dnsBatch.length).toBeLessThan(TARGET_COUNT / 10);

		log("E2E-BATCH-01 passed");
	});
});
