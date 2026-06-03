import { getApiBase, getApiToken } from "./config";
export const API_BASE = getApiBase();

export class APIError extends Error {
  code: "TIMEOUT" | "NETWORK_ERROR" | "HTTP_5xx" | "HTTP_4xx" | "NON_JSON_RESPONSE" | "UNKNOWN";
  httpStatus?: number;
  retryable: boolean;
  constructor(message: string, code: APIError["code"] = "UNKNOWN", httpStatus?: number) {
    super(message);
    this.code = code;
    this.httpStatus = httpStatus;
    this.retryable = code === "TIMEOUT" || code === "NETWORK_ERROR" || code === "HTTP_5xx";
  }
}

const HTTP_MESSAGES: Record<number, string> = {
  400: "请求参数有误",
  401: "登录已过期，请重新输入 API Token",
  403: "没有权限执行此操作",
  404: "请求的资源不存在",
  409: "操作冲突，请刷新后重试",
  422: "输入数据有误，请检查",
  429: "请求过于频繁，请稍后重试",
  500: "服务器内部错误，请稍后重试",
  502: "服务暂时不可用，请稍后重试",
  503: "服务正在维护中，请稍后重试",
  504: "服务响应超时，请稍后重试",
};

export function friendlyMessage(err: APIError): string {
  if (err.code === "TIMEOUT") return "请求超时，请检查网络后重试";
  if (err.code === "NETWORK_ERROR") return "网络连接失败，请检查后端服务是否运行";
  if (err.code === "NON_JSON_RESPONSE") return "服务器响应异常，请检查后端服务";
  if (err.code === "HTTP_4xx" && err.httpStatus) {
    return HTTP_MESSAGES[err.httpStatus] || `请求失败（${err.httpStatus}）`;
  }
  if (err.code === "HTTP_5xx" && err.httpStatus) {
    return HTTP_MESSAGES[err.httpStatus] || "服务器错误，请稍后重试";
  }
  return err.message || "操作失败，请稍后重试";
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
  opts?: RequestInit & { timeout?: number; skipGlobalError?: boolean }
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
      throw new APIError(message, res.status >= 500 ? "HTTP_5xx" : "HTTP_4xx", res.status);
    }

    resetConsecutiveErrors();
    return res;
  } catch (err) {
    if (err instanceof DOMException && err.name === "AbortError") {
      // Caller's own signal abort — rethrow as-is (not an error)
      if (opts?.signal?.aborted) throw err;
      // Our timeout fired
      throw new APIError("请求超时，请检查网络后重试", "TIMEOUT");
    }
    const apiErr = classifyError(err);

    if (globalErrorHandler && !opts?.skipGlobalError) globalErrorHandler(apiErr);

    if (!opts?.skipGlobalError) {
      consecutiveErrors++;
      if (consecutiveErrors >= 3 && consecutiveErrorCallback) {
        consecutiveErrorCallback();
      }
    }

    throw apiErr;
  } finally {
    clearTimeout(timeoutId);
  }
}

async function fetchAPI<T>(path: string, opts?: RequestInit & { timeout?: number; skipGlobalError?: boolean }): Promise<T> {
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
  targets?: Target[];
  denied_targets?: { value: string; reason: string }[];
  needs_scope_confirmation?: boolean;
  message?: string;
  suggested_rules?: {
    action: "include" | "exclude";
    type: string;
    value: string;
  }[];
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
  command_template?: string;
  status: string;
  error_message?: string;
  started_at?: string;
  finished_at?: string;
  exit_code?: number;
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

export interface ServicePort {
  id: string;
  project_id: string;
  asset_id: string;
  ip: string;
  port: number;
  protocol: string;
  state: string;
  service_name: string;
  title: string;
  technologies: string[];
  url: string;
  source_tools: string[];
  is_web: boolean;
  created_at: string;
}

export interface ToolHealth {
  tool: string;
  binary_path?: string;
  version?: string;
  dns_available: boolean;
}

export interface ExcludedDomain {
  id: string;
  domain: string;
  reason: string;
  builtin: boolean;
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
  matched_template?: string;
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

export interface WorkerNode {
  id: string;
  name: string;
  endpoint: string;
  mode: string;
  status: string;
  busy: boolean;
  last_seen?: string;
  created_at: string;
}

export const api = {
  // ... rest of api object
  listWorkers: (signal?: AbortSignal) =>
    fetchAPI<WorkerNode[]>("/workers", { signal }),
  deleteWorker: (id: string, signal?: AbortSignal) =>
    fetchAPI<void>(`/workers/${id}`, { method: "DELETE", signal }),
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

  deleteTarget: (projectId: string, targetId: string, signal?: AbortSignal) =>
    fetchAPI<{ status: string }>(`/projects/${projectId}/targets/${targetId}`, { method: "DELETE", signal }),

  createScopeRule: (data: { project_id: string; action: string; type: string; value: string; reason?: string }, signal?: AbortSignal) =>
    fetchAPI<ScopeRule>("/scope-rules", { method: "POST", body: JSON.stringify(data), signal }),

  deleteScopeRule: (ruleId: string, signal?: AbortSignal) =>
    fetchAPI<{ status: string }>(`/scope-rules/${ruleId}`, { method: "DELETE", signal }),

  parseScopeValue: (value: string, signal?: AbortSignal) =>
    fetchAPI<{ rules: Array<{ type: string; value: string }> }>("/scope-rules/parse", { method: "POST", body: JSON.stringify({ value }), signal }),

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

  getArtifactContent: (artifactId: string, signal?: AbortSignal) =>
    request(`/artifacts/content?id=${artifactId}`, { signal }).then((res) => res.text()),

  getTaskOutput: (
    taskId: string,
    params: { stream?: "stdout" | "stderr"; offset?: number },
    signal?: AbortSignal
  ) =>
    fetchAPI<{ stream: string; offset: number; content: string; done: boolean }>(
      buildQueryString(`/tasks/${taskId}/output`, {
        stream: params.stream ?? "stdout",
        offset: params.offset ?? 0,
      }),
      { signal }
    ),

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

  listServicePorts: (projectId: string, signal?: AbortSignal) =>
    fetchAPI<ServicePort[]>(`/projects/${projectId}/service-ports`, { signal }),

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

  // --- Runs ---
  getRun: (id: string, signal?: AbortSignal) =>
    fetchAPI<Run>(`/runs/${id}`, { signal }),
  getRunTasks: (id: string, signal?: AbortSignal) =>
    fetchAPI<ScanTask[]>(`/runs/${id}/tasks`, { signal }),
  cancelRun: (id: string, signal?: AbortSignal) =>
    fetchAPI<{ status: string }>(`/runs/${id}/cancel`, { method: "POST", signal }),
  cancelPipelineRun: (projectId: string, runId: string, signal?: AbortSignal) =>
    fetchAPI<{ status: string }>(`/projects/${projectId}/pipeline/runs/${runId}/cancel`, { method: "POST", signal }),

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

  // --- Scan Run Metrics ---
  getScanRunMetrics: (projectId: string, runId: string, signal?: AbortSignal) =>
    fetchAPI<ScanRunMetrics>(`/projects/${projectId}/pipeline/runs/${runId}/metrics`, { signal }),

  // --- Scan Work Items ---
  listScanRunWorks: (projectId: string, runId: string, signal?: AbortSignal) =>
    fetchAPI<{ items: ScanWorkItem[]; total: number }>(`/projects/${projectId}/pipeline/runs/${runId}/works`, { signal }),
  listAssetWorks: (assetId: string, runId: string, signal?: AbortSignal) =>
    fetchAPI<{ asset_id: string; run_id: string; items: ScanWorkItem[]; total: number }>(`/assets/${assetId}/works?run_id=${runId}`, { signal }),

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
  getEngineQuota: (engine: string, signal?: AbortSignal) =>
    fetchAPI<{ engine: string; quota: { points: { name: string; value: number; unit: string }[] } }>(`/engines/quota?engine=${engine}`, { signal, skipGlobalError: true }),

  // --- Nuclei Custom Templates ---
  listNucleiCustomSources: (signal?: AbortSignal) =>
    fetchAPI<NucleiCustomSource[]>("/nuclei/custom/sources", { signal }),

  createNucleiCustomGitSource: (data: { name: string; install_path: string; uri: string; branch?: string }, signal?: AbortSignal) =>
    fetchAPI<NucleiCustomSource>("/nuclei/custom/sources/git", { method: "POST", body: JSON.stringify(data), signal }),

  createNucleiCustomUploadSource: (name: string, installPath: string, file: File, signal?: AbortSignal) => {
    const formData = new FormData();
    formData.append("name", name);
    formData.append("install_path", installPath);
    formData.append("routing_policy", "manual");
    formData.append("file", file);
    return fetchAPI<NucleiCustomSource>("/nuclei/custom/sources/upload", { method: "POST", body: formData, signal });
  },

  refreshNucleiCustomSource: (id: string, signal?: AbortSignal) =>
    fetchAPI<NucleiCustomSource>(`/nuclei/custom/sources/${id}/refresh`, { method: "POST", signal }),

  patchNucleiCustomSource: (id: string, data: { name?: string; enabled?: boolean; routing_policy?: string }, signal?: AbortSignal) =>
    fetchAPI<NucleiCustomSource>(`/nuclei/custom/sources/${id}`, { method: "PATCH", body: JSON.stringify(data), signal }),

  patchNucleiCustomSourceEnabled: (id: string, enabled: boolean, signal?: AbortSignal) =>
    fetchAPI<NucleiCustomSource>(`/nuclei/custom/sources/${id}/enabled`, { method: "PATCH", body: JSON.stringify({ enabled }), signal }),

  deleteNucleiCustomSource: (id: string, signal?: AbortSignal) =>
    fetchAPI<void>(`/nuclei/custom/sources/${id}`, { method: "DELETE", signal }),

  listNucleiCustomFiles: (id: string, signal?: AbortSignal) =>
    fetchAPI<NucleiCustomFileEntry[]>(`/nuclei/custom/sources/${id}/files`, { signal }),

  readNucleiCustomFile: (id: string, path: string, signal?: AbortSignal) =>
    fetchBlob(`/nuclei/custom/sources/${id}/files/${path}`, { signal }),

  writeNucleiCustomFile: (id: string, path: string, content: string, signal?: AbortSignal) =>
    fetchAPI<void>(`/nuclei/custom/sources/${id}/files/${path}`, { method: "PUT", body: content, signal }),

  deleteNucleiCustomFile: (id: string, path: string, signal?: AbortSignal) =>
    fetchAPI<void>(`/nuclei/custom/sources/${id}/files/${path}`, { method: "DELETE", signal }),

  validateNucleiCustomSource: (id: string, signal?: AbortSignal) =>
    fetchAPI<NucleiCustomValidationResult>(`/nuclei/custom/sources/${id}/validate`, { method: "POST", signal }),

  validateAllNucleiCustom: (signal?: AbortSignal) =>
    fetchAPI<NucleiCustomValidationResult[]>("/nuclei/custom/validate", { method: "POST", signal }),

  publishNucleiCustom: (signal?: AbortSignal) =>
    fetchAPI<{ version: string }>("/nuclei/custom/publish", { method: "POST", signal }),

  getNucleiCustomManifest: (signal?: AbortSignal) =>
    fetchAPI<NucleiCustomManifest>("/nuclei/custom/manifest", { signal }),

  // --- Dictionaries ---
  listDictionaries: (category?: string, signal?: AbortSignal) =>
    fetchAPI<Dictionary[]>(buildQueryString("/dictionaries", { category }), { signal }),

  createDictionary: (data: { name: string; description?: string; category: string; file: File }, signal?: AbortSignal) => {
    const formData = new FormData();
    formData.append("name", data.name);
    formData.append("category", data.category);
    if (data.description) formData.append("description", data.description);
    formData.append("file", data.file);
    return fetchAPI<Dictionary>("/dictionaries", { method: "POST", body: formData, signal });
  },

  getDictionary: (id: string, signal?: AbortSignal) =>
    fetchAPI<Dictionary>(`/dictionaries/${id}`, { signal }),

  patchDictionary: (id: string, data: Partial<Omit<Dictionary, "id" | "created_at" | "updated_at" | "file_path" | "line_count" | "size_bytes">>, signal?: AbortSignal) =>
    fetchAPI<Dictionary>(`/dictionaries/${id}`, { method: "PATCH", body: JSON.stringify(data), signal }),

  patchDictionaryEnabled: (id: string, enabled: boolean, signal?: AbortSignal) =>
    fetchAPI<Dictionary>(`/dictionaries/${id}/enabled`, { method: "PATCH", body: JSON.stringify({ enabled }), signal }),

  deleteDictionary: (id: string, signal?: AbortSignal) =>
    fetchAPI<void>(`/dictionaries/${id}`, { method: "DELETE", signal }),

  readDictionaryContent: (id: string, signal?: AbortSignal) =>
    request(`/dictionaries/${id}/content`, { signal }).then((res) => res.text()),

  writeDictionaryContent: (id: string, content: string, signal?: AbortSignal) =>
    fetchAPI<Dictionary>(`/dictionaries/${id}/content`, { method: "PUT", body: content, signal }),

  // --- HTTPX Fingerprints ---
  listHttpxFingerprints: (type?: string, signal?: AbortSignal) =>
    fetchAPI<HttpxFingerprint[]>(buildQueryString("/httpx/fingerprints", { type }), { signal }),

  createHttpxFingerprint: (data: { name: string; description?: string; type: string; file: File }, signal?: AbortSignal) => {
    const formData = new FormData();
    formData.append("name", data.name);
    formData.append("type", data.type);
    if (data.description) formData.append("description", data.description);
    formData.append("file", data.file);
    return fetchAPI<HttpxFingerprint>("/httpx/fingerprints", { method: "POST", body: formData, signal });
  },

  getHttpxFingerprint: (id: string, signal?: AbortSignal) =>
    fetchAPI<HttpxFingerprint>(`/httpx/fingerprints/${id}`, { signal }),

  patchHttpxFingerprint: (id: string, data: Partial<Omit<HttpxFingerprint, "id" | "created_at" | "updated_at" | "file_path">>, signal?: AbortSignal) =>
    fetchAPI<HttpxFingerprint>(`/httpx/fingerprints/${id}`, { method: "PATCH", body: JSON.stringify(data), signal }),

  patchHttpxFingerprintEnabled: (id: string, enabled: boolean, signal?: AbortSignal) =>
    fetchAPI<HttpxFingerprint>(`/httpx/fingerprints/${id}/enabled`, { method: "PATCH", body: JSON.stringify({ enabled }), signal }),

  deleteHttpxFingerprint: (id: string, signal?: AbortSignal) =>
    fetchAPI<void>(`/httpx/fingerprints/${id}`, { method: "DELETE", signal }),

  readHttpxFingerprintContent: (id: string, signal?: AbortSignal) =>
    request(`/httpx/fingerprints/${id}/content`, { signal }).then((res) => res.text()),

  writeHttpxFingerprintContent: (id: string, content: string, signal?: AbortSignal) =>
    fetchAPI<HttpxFingerprint>(`/httpx/fingerprints/${id}/content`, { method: "PUT", body: content, signal }),

  listFindingTemplates: (sourceTool?: string, signal?: AbortSignal) =>
    fetchAPI<FindingTemplate[]>(buildQueryString("/finding-templates", { source_tool: sourceTool }), { signal }),

  getFindingTemplate: (id: string, signal?: AbortSignal) =>
    fetchAPI<FindingTemplate>(`/finding-templates/${id}`, { signal }),

  createFindingTemplate: (data: Partial<Omit<FindingTemplate, "id" | "created_at" | "updated_at">>, signal?: AbortSignal) =>
    fetchAPI<FindingTemplate>("/finding-templates", { method: "POST", body: JSON.stringify(data), signal }),

  patchFindingTemplate: (id: string, data: Partial<Omit<FindingTemplate, "id" | "created_at" | "updated_at">>, signal?: AbortSignal) =>
    fetchAPI<FindingTemplate>(`/finding-templates/${id}`, { method: "PATCH", body: JSON.stringify(data), signal }),

  deleteFindingTemplate: (id: string, signal?: AbortSignal) =>
    fetchAPI<void>(`/finding-templates/${id}`, { method: "DELETE", signal }),

  acceptFindingTemplateUpstream: (id: string, signal?: AbortSignal) =>
    fetchAPI<FindingTemplate>(`/finding-templates/${id}/accept-upstream`, { method: "POST", signal }),

  // --- Excluded Domains ---

  listExcludedDomains: (signal?: AbortSignal) =>
    fetchAPI<{ builtin: ExcludedDomain[]; custom: ExcludedDomain[]; total: number }>("/excluded-domains", { signal }),

  listDefaultDomains: (signal?: AbortSignal) =>
    fetchAPI<{ domains: string[]; total: number }>("/excluded-domains/defaults", { signal }),

  addExcludedDomain: (domain: string, reason?: string, signal?: AbortSignal) =>
    fetchAPI<ExcludedDomain>("/excluded-domains", {
      method: "POST",
      body: JSON.stringify({ domain, reason: reason || "" }),
      signal,
    }),

  batchAddExcludedDomains: (domains: Array<{ domain: string; reason?: string }>, signal?: AbortSignal) =>
    fetchAPI<{ created: number; domains: ExcludedDomain[] }>("/excluded-domains/batch", {
      method: "POST",
      body: JSON.stringify({ domains }),
      signal,
    }),

  deleteExcludedDomain: (domain: string, signal?: AbortSignal) =>
    fetchAPI<{ status: string }>(`/excluded-domains/${domain}`, { method: "DELETE", signal }),

  resetExcludedDomains: (signal?: AbortSignal) =>
    fetchAPI<{ status: string }>("/excluded-domains/reset", { method: "POST", signal }),

  checkExcludedDomain: (domain: string, signal?: AbortSignal) =>
    fetchAPI<{ domain: string; excluded: boolean; reason: string }>(
      `/excluded-domains/check?domain=${encodeURIComponent(domain)}`,
      { signal }
    ),
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
  work_total?: number;
  work_done?: number;
  work_running?: number;
  round?: number;
  started_at?: string;
  completed_at?: string;
  created_at: string;
}

export interface ScanRunMetrics {
  engine_state: "running" | "wind_down" | "stopped";
  assets_discovered: number;
  works_pending: number;
  works_done: number;
  works_skipped: number;
  works_running: number;
  works_failed: number;
  queue_depth: { high: number; medium: number; low: number };
  last_new_asset_at?: string;
}

export interface ScanWorkItem {
  id: string;
  run_id: string;
  project_id: string;
  asset_id: string;
  action: string;
  status: string;
  skip_reason?: string;
  stage?: string;
  error?: string;
  started_at?: string;
  completed_at?: string;
  task_id?: string;
  created_at: string;
}

export interface PipelineRun {
  id: string;
  project_id: string;
  mode: string;
  status: string;
  stage?: string;
  error?: string;
  engine_state: string;
  last_new_asset_at?: string;
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
  enable_nmap_service: boolean;
  nmap_service_timeout: number;
  enable_httpx: boolean;
  httpx_rate_limit: number;
  httpx_threads: number;
  enable_nuclei: boolean;
  nuclei_rate_limit: number;
  nuclei_rate_limit_per_min: number; // -rlm: requests per minute (sensitive targets)
  nuclei_concurrency: number; // -c: parallel templates/hosts
  nuclei_scan_depth: string; // "workflow" | "tags" | "both"
  enable_ffuf: boolean;
  ffuf_rate_limit: number;
  ffuf_timeout: number;
  ffuf_dictionary_id: string;
  katana_timeout: number;
  // External-scan-only fields
  enable_passive_search: boolean;
  enable_passive_cert: boolean;
  enable_passive_url: boolean;
  subfinder_mode: string; // passive | active | off
  enable_katana: boolean;
  katana_max_depth: number;
  katana_rate_limit: number;
  ffuf_tier: string; // small | medium | off
  skip_portscan_on_cdn_host: boolean;
  nuclei_require_fingerprint: boolean;
  passive_search_result_limit: number;
  passive_search_concurrency: number;
  subfinder_provider_config: string;
}

export interface Dictionary {
  id: string;
  name: string;
  description?: string;
  category: "dirscan" | "subdomain" | "vhost" | "custom";
  file_path: string;
  line_count: number;
  size_bytes: number;
  builtin: boolean;
  enabled: boolean;
  created_at: string;
  updated_at: string;
}

export interface HttpxFingerprint {
  id: string;
  name: string;
  description?: string;
  type: "favicon" | "tech_detect";
  file_path: string;
  enabled: boolean;
  builtin: boolean;
  created_at: string;
  updated_at: string;
}

export interface FindingTemplate {
  id: string;
  source_tool: string;
  match_keys: string[];
  title: string;
  severity: "" | "info" | "low" | "medium" | "high" | "critical";
  summary: string;
  remediation: string;
  enabled: boolean;
  is_builtin: boolean;
  user_modified: boolean;
  builtin_payload?: string;
  created_at: string;
  updated_at: string;
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
  subfinder_timeout: 30,
  enable_dnsx: true,
  dnsx_rate_limit: 100,
  dnsx_threads: 50,
  dnsx_timeout: 5,
  enable_cdn_filter: true,
  port_range: "top1000",
  naabu_rate: 1000,
  naabu_threads: 100,
  naabu_timeout: 5000,
  enable_nmap_service: true,
  nmap_service_timeout: 180,
  enable_httpx: true,
  httpx_rate_limit: 150,
  httpx_threads: 50,
  enable_nuclei: true,
  nuclei_rate_limit: 100,
  nuclei_rate_limit_per_min: 0,
  nuclei_concurrency: 25,
  nuclei_scan_depth: "tags",
  enable_ffuf: true,
  ffuf_rate_limit: 6,
  ffuf_timeout: 30,
  ffuf_dictionary_id: "",
  enable_katana: true,
  katana_max_depth: 2,
  katana_rate_limit: 10,
  katana_timeout: 10,
  enable_passive_search: false,
  enable_passive_cert: false,
  enable_passive_url: false,
  subfinder_mode: "active",
  ffuf_tier: "off",
  skip_portscan_on_cdn_host: false,
  nuclei_require_fingerprint: false,
  passive_search_result_limit: 500,
  passive_search_concurrency: 3,
  subfinder_provider_config: "",
};

export const DEFAULT_EXTERNAL_PIPELINE_CONFIG: PipelineConfig = {
  ...DEFAULT_PIPELINE_CONFIG,
  port_range: "top100",
  naabu_rate: 300,
  naabu_threads: 50,
  nuclei_scan_depth: "workflow",
  nuclei_rate_limit: 20,
  nuclei_concurrency: 5,
  nuclei_rate_limit_per_min: 30,
  ffuf_rate_limit: 4,
  enable_passive_search: true,
  enable_passive_cert: true,
  enable_passive_url: true,
  subfinder_mode: "passive",
  katana_timeout: 10,
  ffuf_tier: "small",
  skip_portscan_on_cdn_host: true,
  nuclei_require_fingerprint: true,
  passive_search_result_limit: 500,
  passive_search_concurrency: 3,
};

// Mirrors internal/worker/commands.go:HighRiskPorts — curated list of high-value
// attack-surface ports (Redis/ES/MongoDB/Docker/K8s/Ollama, …) that Naabu's
// built-in top-N presets miss. Used as the default pre-fill for the -p custom
// port input.
export const DEFAULT_HIGH_RISK_PORTS =
  "21,22,23,25,53,80,81,88,110,135,139,143,389,443,445,465,587,636,873,993,995," +
  "1080,1099,1433,1521,1723,2049,2082,2375,2376,2480,3000,3128,3306,3389," +
  "4040,4194,4369,4444,4848,5000,5432,5601,5672,5900,5901,5984,6379,6443," +
  "7000,7001,7002,7077,7474,8000,8001,8008,8009,8020,8060,8080,8081,8086,8088,8090,8161,8200,8443,8500,8531,8888,8983," +
  "9000,9001,9042,9043,9060,9080,9090,9091,9092,9100,9200,9300,9418,9443,9981," +
  "10000,10250,10255,11211,11434,15672,15692,16379,18091," +
  "27017,27018,27019,28017,50000,50070,50075,61613,61616";

// Maps to naabu's -tp flag values supported by the backend (see
// internal/worker/commands.go:BuildNaabuCommand).
export const TP_PRESET_VALUES = ["top100", "top1000", "full"] as const;
export type TpPresetValue = (typeof TP_PRESET_VALUES)[number];

export const TP_PRESET_LABELS: Record<TpPresetValue, string> = {
  top100: "Top 100 常用端口",
  top1000: "Top 1000 常用端口",
  full: "全端口 (1-65535)",
};

// --- Engine Search ---

export interface EngineCredential {
  id: string;
  engine: string;
  api_key: string;
  extra?: string;
  created_at: string;
  updated_at: string;
}

export interface SearchResult {
  engine: string;
  ip: string;
  url?: string;
  host?: string;
  port?: number;
  domain?: string;
  title?: string;
  service?: string;
  protocol?: string;
  location?: string;
  os?: string;
  status_code?: number;
  country?: string;
  city?: string;
  organization?: string;
  icp?: string;
  raw?: unknown;
  [key: string]: unknown;
}

export interface SearchEngineResponse {
  engine: string;
  query: string;
  total: number;
  data: SearchResult[];
}

// --- Nuclei Custom Templates ---

export interface NucleiCustomSource {
  id: string;
  name: string;
  type: "git" | "upload" | "file";
  uri?: string;
  branch?: string;
  enabled: boolean;
  builtin: boolean;
  routing_policy: string;
  status: "draft" | "ready" | "error";
  last_sync_at?: string;
  last_validate_at?: string;
  last_error?: string;
  created_at: string;
  updated_at: string;
}

export interface NucleiCustomFileEntry {
  path: string;
  is_dir: boolean;
  size: number;
  mod_time: string;
}

export interface NucleiCustomValidationResult {
  source_id: string;
  ok: boolean;
  errors?: string[];
}

export interface NucleiCustomManifest {
  version: string;
  sources: { id: string; name: string; files: string[]; checksum: string }[];
  created_at: string;
}

