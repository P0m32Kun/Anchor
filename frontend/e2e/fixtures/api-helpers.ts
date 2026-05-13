const API_BASE = "http://localhost:17421";
const API_TOKEN = "p0m32kun";

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

export async function addTarget(
	projectId: string,
	data: { type: string; value: string },
): Promise<Target> {
	const res = await apiFetch(`/projects/${projectId}/targets`, {
		method: "POST",
		body: JSON.stringify(data),
	});
	return res.json();
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
	return fetchList<Worker>("/workers");
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
