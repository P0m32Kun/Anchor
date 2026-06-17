/**
 * 测试层级: E2E
 * 场景 ID: E2E-WRK-02
 * 覆盖流程:
 *   双 worker 环境 → 60 domain 外网扫描 → 任务按 worker_id 分散到 ≥2 节点
 * 前置依赖:
 *   docker-compose.e2e.yml + docker-compose.e2e-multi-worker.override.yml
 *   （make test-e2e-multi-worker-scan 或 ANCHOR_E2E_MULTI_WORKER=1 global-setup）
 */

import { expect, test } from "@playwright/test";
import {
	createProject,
	addTarget,
	startScan,
	waitForPipelineRun,
	listWorkers,
	listRunTasks,
	groupTasksByWorker,
} from "../fixtures/api-helpers";
import { cleanupTestData } from "../fixtures/db-utils";

/** 双项目并行扫描，提高并发 dispatch 机会 */
const PARALLEL_PROJECTS = 2;
const TARGETS_PER_PROJECT = 60;
/** 双项目并行扫描（各 60 domain）；batch 模式下约 2 task/项目 → 合计 ≥4 */
const MIN_DISPATCHED_TASKS = 4;
const MIN_SHARE_PER_WORKER = 0.12;
/** task 总数较少时只要求每个 worker 至少 1 个 */
const SMALL_BATCH_THRESHOLD = 10;

test.setTimeout(30 * 60 * 1000);

test.describe.serial("Multi-worker task dispatch — E2E-WRK-02", () => {
	const projectIds: string[] = [];

	test.beforeAll(async () => {
		await cleanupTestData();

		const workers = await listWorkers();
		const online = workers.filter(
			(w) => w.status === "online" || w.status === "busy",
		);
		test.skip(
			online.length < 2,
			"need ≥2 online workers — run: make test-e2e-multi-worker-up",
		);

		for (let p = 0; p < PARALLEL_PROJECTS; p++) {
			const project = await createProject({
				name: `MultiWorker-${p}-${Date.now()}`,
				organization: "E2E Worker",
				purpose: "multi-worker dispatch smoke",
			});
			projectIds.push(project.id);

			for (let i = 0; i < TARGETS_PER_PROJECT; i++) {
				await addTarget(project.id, {
					type: "domain",
					value: `mw${p}-${i}.multi-worker-e2e.test`,
				});
			}
		}
	});

	test.afterAll(async () => {
		await cleanupTestData();
	});

	test("扫描任务分散到多个 worker", async () => {
		const log = (msg: string) => {
			console.log(`[${new Date().toISOString().slice(11, 19)}] ${msg}`);
		};

		const onlineWorkers = (await listWorkers()).filter(
			(w) => w.status === "online" || w.status === "busy",
		);
		log(`Online workers: ${onlineWorkers.length}`);
		expect(onlineWorkers.length).toBeGreaterThanOrEqual(2);

		log(`Step 1: start ${PARALLEL_PROJECTS} parallel scans`);
		const runIds: string[] = [];
		for (const projectId of projectIds) {
			const { run_id: runId } = await startScan(projectId, {
				mode: "external",
				config: {
					noise_level: "low",
					enable_dnsx: true,
					enable_cdn_filter: true,
					enable_httpx: true,
					enable_passive_search: false,
					enable_subfinder: false,
					enable_ffuf: false,
					enable_katana: false,
					enable_nuclei: false,
					port_range: "80,443",
				},
			});
			expect(runId).toBeTruthy();
			runIds.push(runId);
		}

		log("Step 2: wait for all pipelines");
		for (let i = 0; i < projectIds.length; i++) {
			const { status } = await waitForPipelineRun(
				projectIds[i],
				runIds[i],
				25 * 60 * 1000,
			);
			expect(status).toBe("completed");
		}

		log("Step 3: assert task distribution across workers");
		const allTasks = (
			await Promise.all(runIds.map((runId) => listRunTasks(runId)))
		).flat();
		const withWorker = allTasks.filter((t) => t.worker_id);
		log(`Total scan_tasks=${allTasks.length} with worker_id=${withWorker.length}`);

		expect(
			withWorker.length,
			`need ≥${MIN_DISPATCHED_TASKS} dispatched tasks — run make build-fast before test-e2e-multi-worker-scan`,
		).toBeGreaterThanOrEqual(MIN_DISPATCHED_TASKS);

		const byWorker = groupTasksByWorker(withWorker);
		log(
			`distinct workers=${byWorker.size} distribution=${JSON.stringify(Object.fromEntries(byWorker))}`,
		);

		expect(byWorker.size).toBeGreaterThanOrEqual(2);

		const minTasksPerWorker =
			withWorker.length >= SMALL_BATCH_THRESHOLD
				? Math.max(2, Math.floor(withWorker.length * MIN_SHARE_PER_WORKER))
				: 1;
		for (const [workerId, count] of byWorker) {
			expect(
				count,
				`worker ${workerId} should have ≥${minTasksPerWorker} tasks`,
			).toBeGreaterThanOrEqual(minTasksPerWorker);
		}

		log("E2E-WRK-02 passed");
	});
});
