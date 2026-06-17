import { E2E_API_BASE, E2E_API_TOKEN } from "./e2e-env";

const API_BASE = E2E_API_BASE;
const API_TOKEN = E2E_API_TOKEN;

async function apiFetch(
	path: string,
	options?: RequestInit,
): Promise<Response> {
	const url = `${API_BASE}${path}`;
	const headers: Record<string, string> = {
		Authorization: `Bearer ${API_TOKEN}`,
		"Content-Type": "application/json",
		...((options?.headers as Record<string, string>) || {}),
	};

	const res = await fetch(url, {
		...options,
		headers,
	});

	if (!res.ok) {
		const text = await res.text().catch(() => "");
		throw new Error(
			`API ${options?.method || "GET"} ${path} failed: ${res.status} ${res.statusText} - ${text}`,
		);
	}

	return res;
}

// 后端 list 端点统一返回 { data, total, page, page_size }(见 internal/api/pagination.go),
// 兼容历史"直接返回数组"的 mock,统一在此 unwrap 一次,避免每个 helper 重复判断。
async function fetchList<T>(path: string): Promise<T[]> {
	const res = await apiFetch(path);
	const body = await res.json();
	if (Array.isArray(body)) return body as T[];
	if (body && Array.isArray(body.data)) return body.data as T[];
	return [];
}

// --- Types ---

export interface Project {
	id: string;
	name: string;
	organization?: string;
	purpose?: string;
	start_time?: string;
	end_time?: string;
	rate_limit?: number;
	default_profile?: string;
	port_range?: string;
	created_at: string;
}

export interface Target {
	id: string;
	project_id: string;
	type: string;
	value: string;
	created_at: string;
}

export interface Run {
	id: string;
	project_id: string;
	tool_template_id?: string;
	name: string;
	status: string;
	started_at?: string;
	finished_at?: string;
	created_at: string;
}

export interface Finding {
	id: string;
	project_id: string;
	asset_id?: string;
	service_id?: string;
	web_endpoint_id?: string;
	source_tool: string;
	source_rule_id?: string;
	dedup_key: string;
	title: string;
	severity: string;
	confidence: number;
	priority: number;
	status: string;
	summary?: string;
	remediation?: string;
	created_at: string;
	updated_at: string;
}

export interface Worker {
	id: string;
	name: string;
	mode: string;
	status: string;
	endpoint?: string;
	running_tasks?: number;
	max_concurrency?: number;
}

export interface ScanTask {
	id: string;
	tool: string;
	status: string;
	worker_id?: string | null;
}

export interface ToolTemplate {
	id: string;
	name: string;
	description: string;
}

// --- Projects ---

export async function createProject(data: {
	name: string;
	organization?: string;
	purpose?: string;
}): Promise<Project> {
	const res = await apiFetch("/projects", {
		method: "POST",
		body: JSON.stringify(data),
	});
	return res.json();
}

export async function getProject(id: string): Promise<Project> {
	const res = await apiFetch(`/projects/${id}`);
	return res.json();
}

export async function listProjects(): Promise<Project[]> {
	return fetchList<Project>("/projects");
}

export async function updateProject(
	id: string,
	data: Partial<Omit<Project, "id" | "created_at">>,
): Promise<Project> {
	const res = await apiFetch(`/projects/${id}`, {
		method: "PATCH",
		body: JSON.stringify(data),
	});
	return res.json();
}

export async function deleteProject(id: string): Promise<void> {
	await apiFetch(`/projects/${id}`, { method: "DELETE" });
}

// --- Targets ---

type ScopeRuleSuggestion = {
	action: string;
	type: string;
	value: string;
};

type CreateTargetResponse = Target & {
	needs_scope_confirmation?: boolean;
	suggested_rule?: ScopeRuleSuggestion;
};

export async function addScopeRule(
	projectId: string,
	rule: { action: string; type: string; value: string; reason?: string },
): Promise<void> {
	await apiFetch("/scope-rules", {
		method: "POST",
		body: JSON.stringify({ project_id: projectId, ...rule }),
	});
}

export async function addTarget(
	projectId: string,
	data: { type: string; value: string },
): Promise<Target> {
	const res = await apiFetch(`/projects/${projectId}/targets`, {
		method: "POST",
		body: JSON.stringify(data),
	});
	const body = (await res.json()) as CreateTargetResponse;
	if (body.needs_scope_confirmation && body.suggested_rule) {
		await addScopeRule(projectId, {
			action: body.suggested_rule.action,
			type: body.suggested_rule.type,
			value: body.suggested_rule.value,
			reason: "E2E auto scope",
		});
		const retry = await apiFetch(`/projects/${projectId}/targets`, {
			method: "POST",
			body: JSON.stringify(data),
		});
		return retry.json();
	}
	return body;
}

export async function listTargets(projectId: string): Promise<Target[]> {
	return fetchList<Target>(`/projects/${projectId}/targets`);
}

// --- Runs ---

export async function listRuns(projectId: string): Promise<Run[]> {
	return fetchList<Run>(`/projects/${projectId}/runs`);
}

export async function createRun(
	projectId: string,
	data: { tool_template_id: string; name: string },
): Promise<Run> {
	const res = await apiFetch(`/projects/${projectId}/runs`, {
		method: "POST",
		body: JSON.stringify(data),
	});
	return res.json();
}

export async function getRun(id: string): Promise<Run> {
	const res = await apiFetch(`/runs/${id}`);
	return res.json();
}

export async function cancelRun(id: string): Promise<{ status: string }> {
	const res = await apiFetch(`/runs/${id}/cancel`, { method: "POST" });
	return res.json();
}

// --- Findings ---

export async function listFindings(
	projectId: string,
	status?: string,
): Promise<Finding[]> {
	const query = status ? `?status=${encodeURIComponent(status)}` : "";
	return fetchList<Finding>(`/projects/${projectId}/findings${query}`);
}

export async function getFinding(
	id: string,
): Promise<{ finding: Finding; evidence: unknown[] }> {
	const res = await apiFetch(`/findings/${id}`);
	return res.json();
}

export async function updateFindingStatus(
	id: string,
	status: string,
): Promise<{ status: string }> {
	const res = await apiFetch(`/findings/${id}/status`, {
		method: "PATCH",
		body: JSON.stringify({ status }),
	});
	return res.json();
}

// --- Workers ---

export async function listWorkers(): Promise<Worker[]> {
	const res = await apiFetch("/workers");
	return res.json();
}

export async function listRunTasks(runId: string): Promise<ScanTask[]> {
	const res = await apiFetch(`/runs/${runId}/tasks`);
	return res.json();
}

export function groupTasksByWorker(
	tasks: ScanTask[],
): Map<string, number> {
	const counts = new Map<string, number>();
	for (const t of tasks) {
		if (!t.worker_id) continue;
		counts.set(t.worker_id, (counts.get(t.worker_id) ?? 0) + 1);
	}
	return counts;
}

// --- Tool Templates ---

export async function listToolTemplates(): Promise<ToolTemplate[]> {
	return fetchList<ToolTemplate>("/tool-templates");
}

// --- Health ---

export async function healthCheck(): Promise<{ status: string }> {
	const res = await apiFetch("/health");
	return res.json();
}

// --- Pipeline / Scan ---

/**
 * Wait for a scan pipeline to complete by polling the pipeline status endpoint.
 *
 * 1. Fetches the list of scan runs for the project to get the most recent run ID
 * 2. Polls `/projects/:id/pipeline/runs/:runId` every 5 seconds until status
 *    transitions away from "running" or "pending"
 * 3. Returns the final status
 *
 * @throws if no scan run is found for the project
 */
export async function waitForPipeline(
	projectId: string,
	timeoutMs: number = 20 * 60 * 1000,
): Promise<{ runId: string; status: string }> {
	// 1. Get the most recent scan run ID
	const listRes = await apiFetch(`/projects/${projectId}/scan/runs`);
	const listBody = await listRes.json();
	const runs: Array<{ id: string }> = Array.isArray(listBody)
		? listBody
		: (listBody.data ?? []);
	const runId = runs[0]?.id;
	if (!runId) throw new Error(`No scan runs found for project ${projectId}`);

	// 2. Poll until completion
	const start = Date.now();
	let status = "running";
	while ((status === "running" || status === "pending") && Date.now() - start < timeoutMs) {
		await sleep(5_000);
		const res = await apiFetch(`/projects/${projectId}/pipeline/runs/${runId}`);
		const d = (await res.json()) as { status: string };
		status = d.status;
	}
	return { runId, status };
}

function sleep(ms: number): Promise<void> {
	return new Promise((resolve) => setTimeout(resolve, ms));
}

// --- Paginated list helpers ---

export interface PaginatedResponse<T> {
	items: T[];
	total: number;
	page: number;
	page_size: number;
}

async function fetchPaginated<T>(path: string): Promise<PaginatedResponse<T>> {
	const res = await apiFetch(path);
	const body = await res.json();
	if (body && Array.isArray(body.items)) {
		return {
			items: body.items as T[],
			total: typeof body.total === "number" ? body.total : body.items.length,
			page: body.page ?? 1,
			page_size: body.page_size ?? body.items.length,
		};
	}
	if (Array.isArray(body)) {
		return { items: body as T[], total: body.length, page: 1, page_size: body.length };
	}
	if (body && Array.isArray(body.data)) {
		return {
			items: body.data as T[],
			total: body.total ?? body.data.length,
			page: body.page ?? 1,
			page_size: body.page_size ?? body.data.length,
		};
	}
	return { items: [], total: 0, page: 1, page_size: 0 };
}

export interface ScanWorkItem {
	id: string;
	run_id: string;
	asset_id: string;
	action: string;
	status: string;
	batch_mode?: boolean;
	stage?: string;
}

export interface ScanRunMetrics {
	engine_state: string;
	works_pending: number;
	works_done: number;
	works_skipped: number;
	works_running: number;
	works_failed: number;
}

export interface ToolCallLog {
	id: string;
	tool: string;
	action: string;
	status: string;
}

export interface Asset {
	id: string;
	project_id: string;
	type: string;
	value: string;
}

export interface ScanRun {
	id: string;
	project_id: string;
	status: string;
	engine_state?: string;
}

export interface LineageNode {
	node_type: string;
	value: string;
	relation?: string;
	source_engine?: string;
}

export interface AssetLineage {
	chain: LineageNode[];
}

export async function listScanRuns(projectId: string): Promise<ScanRun[]> {
	return fetchList<ScanRun>(`/projects/${projectId}/scan/runs`);
}

export async function listAssets(projectId: string): Promise<Asset[]> {
	return fetchList<Asset>(`/projects/${projectId}/assets`);
}

export async function getAssetLineage(
	assetId: string,
	runId: string,
): Promise<AssetLineage> {
	const res = await apiFetch(
		`/assets/${assetId}/lineage?run_id=${encodeURIComponent(runId)}`,
	);
	return res.json();
}

export async function listScanRunWorks(
	projectId: string,
	runId: string,
	pageSize = 500,
): Promise<{ items: ScanWorkItem[]; total: number }> {
	const data = await fetchPaginated<ScanWorkItem>(
		`/projects/${projectId}/pipeline/runs/${runId}/works?page=1&page_size=${pageSize}`,
	);
	return { items: data.items, total: data.total };
}

export async function listToolCallLogs(
	projectId: string,
	runId: string,
	pageSize = 500,
): Promise<{ items: ToolCallLog[]; total: number }> {
	const data = await fetchPaginated<ToolCallLog>(
		`/projects/${projectId}/pipeline/runs/${runId}/tool-calls?page=1&page_size=${pageSize}`,
	);
	return { items: data.items, total: data.total };
}

export async function getScanRunMetrics(
	projectId: string,
	runId: string,
): Promise<ScanRunMetrics> {
	const res = await apiFetch(
		`/projects/${projectId}/pipeline/runs/${runId}/metrics`,
	);
	return res.json();
}

export async function startScan(
	projectId: string,
	body: { mode: string; config?: Record<string, unknown> },
): Promise<{ run_id: string }> {
	const res = await apiFetch(`/projects/${projectId}/scan`, {
		method: "POST",
		body: JSON.stringify(body),
	});
	return res.json();
}

export async function getPipelineRun(
	projectId: string,
	runId: string,
): Promise<{ status: string; engine_state?: string }> {
	const res = await apiFetch(`/projects/${projectId}/pipeline/runs/${runId}`);
	return res.json();
}

export async function waitForPipelineRun(
	projectId: string,
	runId: string,
	timeoutMs = 20 * 60 * 1000,
): Promise<{ status: string }> {
	const start = Date.now();
	let status = "running";
	while ((status === "running" || status === "pending") && Date.now() - start < timeoutMs) {
		await sleep(5_000);
		const run = await getPipelineRun(projectId, runId);
		status = run.status;
	}
	return { status };
}
