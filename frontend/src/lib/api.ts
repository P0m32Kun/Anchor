import { getApiBase, getApiToken } from "./config";
export const API_BASE = getApiBase();

export class APIError extends Error {
  code: "TIMEOUT" | "NETWORK_ERROR" | "HTTP_5xx" | "HTTP_4xx" | "NON_JSON_RESPONSE" | "UNKNOWN";
  retryable: boolean;
  constructor(message: string, code: APIError["code"] = "UNKNOWN") {
    super(message);
    this.code = code;
    this.retryable = code === "TIMEOUT" || code === "NETWORK_ERROR" || code === "HTTP_5xx";
  }
}

export interface PaginatedResponse<T> {
  data: T[];
  total: number;
  page: number;
  page_size: number;
}

export interface PaginationParams {
  page?: number;
  page_size?: number;
}

/** 无分页 UI 时使用，一次性获取全量数据 */
export const PAGE_ALL: PaginationParams = { page_size: 10000 };

function buildQueryString(base: string, params: Record<string, string | number | undefined>): string {
  const qs = new URLSearchParams();
  for (const [key, value] of Object.entries(params)) {
    if (value !== undefined && value !== "") {
      qs.append(key, String(value));
    }
  }
  const q = qs.toString();
  return q ? `${base}?${q}` : base;
}

// Global error handler — registered by App.tsx
let globalErrorHandler: ((err: APIError) => void) | null = null;
export function setGlobalErrorHandler(handler: (err: APIError) => void) {
  globalErrorHandler = handler;
}
export function clearGlobalErrorHandler() {
  globalErrorHandler = null;
}

let consecutiveErrors = 0;
let consecutiveErrorCallback: (() => void) | null = null;

export function setConsecutiveErrorCallback(cb: () => void) {
  consecutiveErrorCallback = cb;
}

export function resetConsecutiveErrors() {
  consecutiveErrors = 0;
}

function classifyError(err: unknown): APIError {
  if (err instanceof APIError) return err;
  if (err instanceof TypeError) {
    return new APIError("网络连接失败，请检查后端服务是否运行", "NETWORK_ERROR");
  }
  return new APIError((err as any)?.message || "请求失败", "UNKNOWN");
}

export async function request(
  path: string,
  opts?: RequestInit & { timeout?: number }
): Promise<Response> {
  const controller = new AbortController();
  const timeoutId = setTimeout(() => controller.abort(), opts?.timeout ?? 30000);

  if (opts?.signal) {
    opts.signal.addEventListener("abort", () => controller.abort(), { once: true });
  }

  const token = getApiToken();
  const headers: Record<string, string> = {
    ...(opts?.headers as Record<string, string> || {}),
    ...(token ? { Authorization: `Bearer ${token}` } : {}),
  };

  try {
    const res = await fetch(`${API_BASE}${path}`, {
      ...opts,
      headers,
      signal: controller.signal,
    });

    if (!res.ok) {
      let message: string;
      try {
        const data = await res.json();
        message = data?.error?.message || `${res.status}: ${res.statusText}`;
      } catch {
        message = `${res.status}: ${res.statusText}`;
      }
      if (res.status === 401) {
        message = "认证失败，请检查 API Token";
      }
      throw new APIError(message, res.status >= 500 ? "HTTP_5xx" : "HTTP_4xx");
    }

    resetConsecutiveErrors();
    return res;
  } catch (err) {
    if (err instanceof DOMException && err.name === "AbortError") {
      throw err;
    }
    const apiErr = classifyError(err);

    if (globalErrorHandler) globalErrorHandler(apiErr);

    consecutiveErrors++;
    if (consecutiveErrors >= 3 && consecutiveErrorCallback) {
      consecutiveErrorCallback();
    }

    throw apiErr;
  } finally {
    clearTimeout(timeoutId);
  }
}

async function fetchAPI<T>(path: string, opts?: RequestInit & { timeout?: number }): Promise<T> {
  const isFormData = opts?.body instanceof FormData;
  const headers: Record<string, string> = {
    ...(isFormData ? {} : { "Content-Type": "application/json" }),
    ...(opts?.headers as Record<string, string>),
  };

  const res = await request(path, { ...opts, headers });

  if (res.status === 204) return null as T;

  const contentType = res.headers.get("content-type") || "";
  if (!contentType.includes("application/json")) {
    // Some proxies (e.g. Vite dev proxy, system HTTP proxy) may strip or
    // rewrite the Content-Type header. If the body looks like JSON, parse it
    // anyway rather than failing the health check.
    const clone = res.clone();
    const text = await clone.text();
    const trimmed = text.trim();
    if (trimmed.startsWith("{") || trimmed.startsWith("[")) {
      try {
        return JSON.parse(trimmed) as T;
      } catch {
        throw new APIError("服务器返回了非 JSON 响应", "NON_JSON_RESPONSE");
      }
    }
    throw new APIError("服务器返回了非 JSON 响应", "NON_JSON_RESPONSE");
  }

  return res.json();
}

export async function fetchBlob(path: string, opts?: RequestInit & { timeout?: number }): Promise<Blob> {
  const res = await request(path, { ...opts, timeout: opts?.timeout ?? 60000 });
  return res.blob();
}

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
  pipeline_config?: string;
  created_at: string;
}

export interface Target {
  id: string;
  project_id: string;
  type: string;
  value: string;
  created_at: string;
}

export interface ScopeConfirmationResponse {
  message: string;
  needs_scope_confirmation: boolean;
  suggested_rule: {
    action: "include" | "exclude";
    type: string;
    value: string;
  };
}

export interface ImportResult {
  imported: number;
  duplicates: number;
  denied: number;
  errors: number;
  expanded?: number;
  targets: Target[];
  denied_targets: { value: string; reason: string }[];
}

export interface DryRunResult {
  project_id: string;
  mode: string;
  time_window_valid?: boolean;
  rate_limit?: number;
  estimated_duration_seconds?: number;
  results: { target: string; decision: string; reason: string }[];
}

export interface ScopeRule {
  id: string;
  project_id: string;
  action: "include" | "exclude";
  type: string;
  value: string;
  reason?: string;
}

export interface ScanTask {
  id: string;
  project_id: string;
  tool: string;
  status: string;
  created_at: string;
}

export interface Asset {
  id: string;
  project_id: string;
  type: string;
  value: string;
  normalized_value: string;
  source_tools?: string[];
  first_seen: string;
  last_seen: string;
  tags?: Record<string, string>;
}

export interface WebEndpoint {
  id: string;
  project_id: string;
  asset_id: string;
  url: string;
  scheme?: string;
  host?: string;
  port?: number;
  path?: string;
  status_code?: number;
  title?: string;
  technologies?: string[];
  source_tool?: string;
  created_at: string;
}

export interface Port {
  id: string;
  asset_id: string;
  port: number;
  protocol: string;
  state: string;
  source_tool?: string;
  created_at: string;
}

export interface Service {
  id: string;
  asset_id: string;
  port_id?: string;
  name?: string;
  product?: string;
  version?: string;
  banner?: string;
  confidence: number;
  source_tool?: string;
  created_at: string;
}

export interface ToolHealth {
  tool: string;
  binary_path?: string;
  version?: string;
  dns_available: boolean;
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

export interface Evidence {
  id: string;
  finding_id: string;
  type: string;
  artifact_id?: string;
  excerpt?: string;
  created_by?: string;
  created_at: string;
}

export const api = {
  createProject: (data: { name: string; organization?: string; purpose?: string; start_time?: string; end_time?: string; rate_limit?: number }, signal?: AbortSignal) =>
    fetchAPI<Project>("/projects", { method: "POST", body: JSON.stringify(data), signal }),

  listProjects: (pagination?: PaginationParams, signal?: AbortSignal) =>
    fetchAPI<PaginatedResponse<Project>>(buildQueryString("/projects", { page: pagination?.page, page_size: pagination?.page_size }), { signal }),

  getProject: (id: string, signal?: AbortSignal) => fetchAPI<Project>(`/projects/${id}`, { signal }),

  updateProject: (id: string, data: Partial<Omit<Project, "id" | "created_at">>, signal?: AbortSignal) =>
    fetchAPI<Project>(`/projects/${id}`, { method: "PATCH", body: JSON.stringify(data), signal }),

  deleteProject: (id: string, signal?: AbortSignal) =>
    fetchAPI<{ status: string }>(`/projects/${id}`, { method: "DELETE", signal }),

  createTarget: (projectId: string, data: { type: string; value: string }, signal?: AbortSignal) =>
    fetchAPI<Target | ScopeConfirmationResponse>(`/projects/${projectId}/targets`, { method: "POST", body: JSON.stringify(data), signal }),

  listTargets: (projectId: string, pagination?: PaginationParams, signal?: AbortSignal) =>
    fetchAPI<PaginatedResponse<Target>>(buildQueryString(`/projects/${projectId}/targets`, { page: pagination?.page, page_size: pagination?.page_size }), { signal }),

  createScopeRule: (data: { project_id: string; action: string; type: string; value: string; reason?: string }, signal?: AbortSignal) =>
    fetchAPI<ScopeRule>("/scope-rules", { method: "POST", body: JSON.stringify(data), signal }),

  listScopeRules: (projectId: string, pagination?: PaginationParams, signal?: AbortSignal) =>
    fetchAPI<PaginatedResponse<ScopeRule>>(buildQueryString("/scope-rules", { project_id: projectId, page: pagination?.page, page_size: pagination?.page_size }), { signal }),

  dryRun: (projectId: string, signal?: AbortSignal) =>
    fetchAPI<DryRunResult>(`/scan-plans/dry-run?project_id=${projectId}`, { method: "POST", signal }),

  importTargets: (projectId: string, file: File, signal?: AbortSignal) => {
    const formData = new FormData();
    formData.append("file", file);
    return fetchAPI<ImportResult>(`/projects/${projectId}/targets/import`, {
      method: "POST",
      body: formData,
      signal,
    });
  },

  runTask: (data: { project_id: string; plan_id?: string; tool: string; target_id: string; command: string }, signal?: AbortSignal) =>
    fetchAPI<ScanTask>("/tasks/run", { method: "POST", body: JSON.stringify(data), signal }),

  getTask: (id: string, signal?: AbortSignal) => fetchAPI<ScanTask>(`/scan-tasks/${id}`, { signal }),

  cancelTask: (id: string, signal?: AbortSignal) =>
    fetchAPI<any>(`/scan-tasks/${id}/cancel`, { method: "POST", signal }),

  listArtifacts: (taskId: string, signal?: AbortSignal) =>
    fetchAPI<any[]>(`/tasks/${taskId}/artifacts`, { signal }),

  listToolHealth: (signal?: AbortSignal) => fetchAPI<ToolHealth[]>("/health/tools", { signal }),

  runHealthCheck: (signal?: AbortSignal) => fetchAPI<ToolHealth[]>("/health/check", { method: "POST", signal }),

  healthCheck: async () => fetchAPI<{ status: string }>("/health", { timeout: 5000 }),

  startAssetDiscovery: (projectId: string, signal?: AbortSignal) =>
    fetchAPI<{ status: string }>(`/projects/${projectId}/workflows/asset-discovery`, { method: "POST", signal }),

  listAssets: (projectId: string, pagination?: PaginationParams, signal?: AbortSignal) =>
    fetchAPI<PaginatedResponse<Asset>>(buildQueryString(`/projects/${projectId}/assets`, { page: pagination?.page, page_size: pagination?.page_size }), { signal }),

  listWebEndpoints: (projectId: string, pagination?: PaginationParams, signal?: AbortSignal) =>
    fetchAPI<PaginatedResponse<WebEndpoint>>(buildQueryString(`/projects/${projectId}/web-endpoints`, { page: pagination?.page, page_size: pagination?.page_size }), { signal }),

  listPorts: (assetId: string, signal?: AbortSignal) => fetchAPI<Port[]>(`/assets/${assetId}/ports`, { signal }),

  listServices: (assetId: string, signal?: AbortSignal) => fetchAPI<Service[]>(`/assets/${assetId}/services`, { signal }),

  startWebScreening: (projectId: string, signal?: AbortSignal) =>
    fetchAPI<{ status: string }>(`/projects/${projectId}/workflows/web-screening`, { method: "POST", signal }),

  listFindings: (projectId: string, status?: string, pagination?: PaginationParams, signal?: AbortSignal) =>
    fetchAPI<PaginatedResponse<Finding>>(buildQueryString(`/projects/${projectId}/findings`, { status, page: pagination?.page, page_size: pagination?.page_size }), { signal }),

  getFinding: (id: string, signal?: AbortSignal) =>
    fetchAPI<{ finding: Finding; evidence: Evidence[] }>(`/findings/${id}`, { signal }),

  updateFindingStatus: (id: string, status: string, signal?: AbortSignal) =>
    fetchAPI<{ status: string }>(`/findings/${id}/status`, { method: "PATCH", body: JSON.stringify({ status }), signal }),

  addEvidence: (findingId: string, data: { type: string; excerpt: string; created_by?: string }, signal?: AbortSignal) =>
    fetchAPI<Evidence>(`/findings/${findingId}/evidence`, { method: "POST", body: JSON.stringify(data), signal }),

  exportReportMD: (projectId: string, signal?: AbortSignal) =>
    fetchBlob(`/projects/${projectId}/reports/export.md`, { signal }),

  exportReportJSON: (projectId: string, signal?: AbortSignal) =>
    fetchBlob(`/projects/${projectId}/reports/export.json`, { signal }),

  // --- Runs ---
  getRun: (id: string, signal?: AbortSignal) =>
    fetchAPI<Run>(`/runs/${id}`, { signal }),
  getRunTasks: (id: string, signal?: AbortSignal) =>
    fetchAPI<ScanTask[]>(`/runs/${id}/tasks`, { signal }),
  cancelRun: (id: string, signal?: AbortSignal) =>
    fetchAPI<{ status: string }>(`/runs/${id}/cancel`, { method: "POST", signal }),

  // --- Pipeline Config (project-scoped) ---
  getPipelineConfig: (projectId: string, signal?: AbortSignal) =>
    fetchAPI<PipelineConfig>(`/projects/${projectId}/pipeline/config`, { signal }),
  updatePipelineConfig: (projectId: string, data: PipelineConfig, signal?: AbortSignal) =>
    fetchAPI<{ status: string }>(`/projects/${projectId}/pipeline/config`, {
      method: "POST",
      body: JSON.stringify(data),
      signal,
    }),

  // --- Tool Templates ---
  listToolTemplates: (signal?: AbortSignal) =>
    fetchAPI<ToolTemplate[]>("/tool-templates", { signal }),

  // --- Pipeline Run Stages ---
  listPipelineRunStages: (projectId: string, runId: string, signal?: AbortSignal) =>
    fetchAPI<{ stages: PipelineRunStage[] }>(`/projects/${projectId}/pipeline/runs/${runId}/stages`, { signal }),

  // --- Unified Scan ---
  createScan: (projectId: string, data: { mode: string; config: PipelineConfig }, signal?: AbortSignal) =>
    fetchAPI<{ run_id: string; status: string; mode: string }>(`/projects/${projectId}/scan`, { method: "POST", body: JSON.stringify(data), signal }),
  listScanRuns: (projectId: string, pagination?: PaginationParams, signal?: AbortSignal) =>
    fetchAPI<PaginatedResponse<PipelineRun>>(buildQueryString(`/projects/${projectId}/scan/runs`, { page: pagination?.page, page_size: pagination?.page_size }), { signal }),
  getScanRun: (projectId: string, runId: string, signal?: AbortSignal) =>
    fetchAPI<PipelineRun>(`/projects/${projectId}/pipeline/runs/${runId}`, { signal }),

  // --- Retest ---
  retestFinding: (id: string, signal?: AbortSignal) =>
    fetchAPI<any>(`/findings/${id}/retest`, { method: "POST", signal }),
  listRetests: (id: string, signal?: AbortSignal) =>
    fetchAPI<any[]>(`/findings/${id}/retests`, { signal }),
  batchUpdateFindingStatus: (ids: string[], status: string, signal?: AbortSignal) =>
    fetchAPI<any>(`/findings/batch-status`, { method: "PATCH", body: JSON.stringify({ ids, status }), signal }),

  // --- Dashboard ---
  getDashboardStats: (signal?: AbortSignal) =>
    fetchAPI<DashboardStats>("/dashboard/stats", { signal }),

  // --- Engine Search ---
  listEngineCredentials: (signal?: AbortSignal) =>
    fetchAPI<EngineCredential[]>("/engines/credentials", { signal }),
  saveEngineCredential: (data: { engine: string; api_key: string; extra?: string }, signal?: AbortSignal) =>
    fetchAPI<EngineCredential>("/engines/credentials", { method: "POST", body: JSON.stringify(data), signal }),
  deleteEngineCredential: (engine: string, signal?: AbortSignal) =>
    fetchAPI<void>(`/engines/credentials/${engine}`, { method: "DELETE", signal }),
  searchEngine: (params: { engine: string; query: string; page?: number; size?: number }, signal?: AbortSignal) =>
    fetchAPI<SearchEngineResponse>(buildQueryString("/engines/search", params), { signal }),
};

export interface Run {
  id: string;
  project_id: string;
  tool_template_id?: string;
  name: string;
  status: string;
  stage?: string;
  started_at?: string;
  finished_at?: string;
  created_at: string;
}

export interface ToolTemplate {
  id: string;
  name: string;
  description: string;
  tools_json: string;
  rate_limit: number;
  is_preset: boolean;
  created_at: string;
}

export interface PipelineRunStage {
  id: string;
  run_id: string;
  stage: string;
  status: string;
  error?: string;
  started_at?: string;
  completed_at?: string;
  created_at: string;
}

export interface PipelineRun {
  id: string;
  project_id: string;
  mode: string;
  status: string;
  stage?: string;
  error?: string;
  started_at: string;
  completed_at?: string;
  created_at: string;
}

export interface PipelineConfig {
  enable_fofa: boolean;
  fofa_result_limit: number;
  fofa_concurrency: number;
  enable_subfinder: boolean;
  subfinder_rate_limit: number;
  subfinder_threads: number;
  subfinder_timeout: number;
  enable_dnsx: boolean;
  dnsx_rate_limit: number;
  dnsx_threads: number;
  dnsx_timeout: number;
  enable_cdn_filter: boolean;
  port_range: string;
  naabu_rate: number;
  naabu_threads: number;
  naabu_timeout: number;
  enable_nerva: boolean;
  nerva_rate_limit: number;
  nerva_workers: number;
  nerva_timeout: number;
  enable_httpx: boolean;
  httpx_rate_limit: number;
  httpx_threads: number;
  enable_nuclei: boolean;
  nuclei_rate_limit: number;
  nuclei_rate_limit_per_min: number; // -rlm: requests per minute (sensitive targets)
  nuclei_concurrency: number; // -c: parallel templates/hosts
  nuclei_scan_depth: string; // "workflow" | "tags" | "both"
}

export interface DashboardRunItem {
  id: string;
  project_id: string;
  project_name: string;
  name: string;
  status: string;
  started_at?: string;
  created_at: string;
}

export interface DashboardFindingItem {
  id: string;
  project_id: string;
  project_name: string;
  title: string;
  severity: string;
  created_at: string;
}

export interface DashboardStats {
  total_projects: number;
  active_runs: number;
  pending_findings: number;
  online_workers: number;
  recent_runs: DashboardRunItem[];
  recent_findings: DashboardFindingItem[];
}

export const DEFAULT_PIPELINE_CONFIG: PipelineConfig = {
  enable_fofa: true,
  fofa_result_limit: 500,
  fofa_concurrency: 5,
  enable_subfinder: true,
  subfinder_rate_limit: 50,
  subfinder_threads: 10,
  subfinder_timeout: 300,
  enable_dnsx: true,
  dnsx_rate_limit: 100,
  dnsx_threads: 50,
  dnsx_timeout: 5,
  enable_cdn_filter: true,
  port_range: "top1000",
  naabu_rate: 1000,
  naabu_threads: 100,
  naabu_timeout: 600,
  enable_nerva: true,
  nerva_rate_limit: 100,
  nerva_workers: 50,
  nerva_timeout: 10,
  enable_httpx: true,
  httpx_rate_limit: 150,
  httpx_threads: 50,
  enable_nuclei: true,
  nuclei_rate_limit: 100,
  nuclei_rate_limit_per_min: 0,
  nuclei_concurrency: 25,
  nuclei_scan_depth: "tags",
};

export interface PortRangePreset {
  value: string;
  label: string;
  description: string;
}

export const PORT_RANGE_PRESETS: PortRangePreset[] = [
  {
    value: "top100",
    label: "Top 100 常用端口",
    description: "Naabu 默认快扫，覆盖 Web/SSH/SMB 等高频服务",
  },
  {
    value: "top1000",
    label: "Top 1000 常用端口",
    description: "更广覆盖，但漏掉 Redis 6379、ES 9200、Kubelet 10250 等",
  },
  {
    value: "high-risk",
    label: "高危端口（推荐）",
    description: "115 个攻击面端口，覆盖 Redis/ES/MongoDB/Docker/K8s/Ollama 等高价值目标",
  },
  {
    value: "full",
    label: "全端口（1-65535）",
    description: "最完整但最慢，建议仅在内网或小范围使用",
  },
  {
    value: "custom",
    label: "自定义",
    description: "手动指定端口，例如 80,443,8080 或 1-1000",
  },
];

// --- Engine Search ---

export interface EngineCredential {
  id: string;
  engine: string;
  api_key: string;
  email?: string;
  extra?: string;
  created_at: string;
  updated_at: string;
}

export interface SearchResult {
  engine: string;
  ip: string;
  port?: number;
  domain?: string;
  title?: string;
  service?: string;
  protocol?: string;
  location?: string;
  os?: string;
  raw?: unknown;
  [key: string]: unknown;
}

export interface SearchEngineResponse {
  engine: string;
  query: string;
  total: number;
  data: SearchResult[];
}

// Append to api object below existing methods
