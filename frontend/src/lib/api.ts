import { getApiBase } from "./config";
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

async function request(
  path: string,
  opts?: RequestInit & { timeout?: number }
): Promise<Response> {
  const controller = new AbortController();
  const timeoutId = setTimeout(() => controller.abort(), opts?.timeout ?? 30000);

  // Merge external signal if provided
  if (opts?.signal) {
    opts.signal.addEventListener("abort", () => controller.abort(), { once: true });
  }

  try {
    const res = await fetch(`${API_BASE}${path}`, {
      ...opts,
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
      const code = res.status >= 500 ? "HTTP_5xx" : "HTTP_4xx";
      throw new APIError(message, code);
    }

    resetConsecutiveErrors();
    return res;
  } catch (err: any) {
    let apiErr: APIError;

    if (err instanceof APIError) {
      apiErr = err;
    } else if (err instanceof DOMException && err.name === "AbortError") {
      apiErr = new APIError("请求超时，请检查网络", "TIMEOUT");
    } else if (err instanceof TypeError) {
      apiErr = new APIError("网络连接失败，请检查后端服务是否运行", "NETWORK_ERROR");
    } else {
      apiErr = new APIError(err?.message || "请求失败", "UNKNOWN");
    }

    // Notify global handler (e.g., Toast) without swallowing the error
    if (globalErrorHandler) {
      globalErrorHandler(apiErr);
    }

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
  const headers: Record<string, string> = isFormData ? {} : { "Content-Type": "application/json" };
  if (opts?.headers) {
    Object.assign(headers, opts.headers as Record<string, string>);
  }

  const res = await request(path, { ...opts, headers, timeout: opts?.timeout ?? 30000 });

  if (res.status === 204) {
    return null as T;
  }

  const contentType = res.headers.get("content-type") || "";
  if (!contentType.includes("application/json")) {
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

  listProjects: (signal?: AbortSignal) => fetchAPI<Project[]>("/projects", { signal }),

  getProject: (id: string, signal?: AbortSignal) => fetchAPI<Project>(`/projects/${id}`, { signal }),

  updateProject: (id: string, data: Partial<Omit<Project, "id" | "created_at">>, signal?: AbortSignal) =>
    fetchAPI<Project>(`/projects/${id}`, { method: "PATCH", body: JSON.stringify(data), signal }),

  deleteProject: (id: string, signal?: AbortSignal) =>
    fetchAPI<{ status: string }>(`/projects/${id}`, { method: "DELETE", signal }),

  createTarget: (projectId: string, data: { type: string; value: string }, signal?: AbortSignal) =>
    fetchAPI<Target | ScopeConfirmationResponse>(`/projects/${projectId}/targets`, { method: "POST", body: JSON.stringify(data), signal }),

  listTargets: (projectId: string, signal?: AbortSignal) => fetchAPI<Target[]>(`/projects/${projectId}/targets`, { signal }),

  createScopeRule: (data: { project_id: string; action: string; type: string; value: string; reason?: string }, signal?: AbortSignal) =>
    fetchAPI<ScopeRule>("/scope-rules", { method: "POST", body: JSON.stringify(data), signal }),

  listScopeRules: (projectId: string, signal?: AbortSignal) =>
    fetchAPI<ScopeRule[]>(`/scope-rules?project_id=${projectId}`, { signal }),

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

  listAssets: (projectId: string, signal?: AbortSignal) => fetchAPI<Asset[]>(`/projects/${projectId}/assets`, { signal }),

  listWebEndpoints: (projectId: string, signal?: AbortSignal) => fetchAPI<WebEndpoint[]>(`/projects/${projectId}/web-endpoints`, { signal }),

  listPorts: (assetId: string, signal?: AbortSignal) => fetchAPI<Port[]>(`/assets/${assetId}/ports`, { signal }),

  listServices: (assetId: string, signal?: AbortSignal) => fetchAPI<Service[]>(`/assets/${assetId}/services`, { signal }),

  startWebScreening: (projectId: string, signal?: AbortSignal) =>
    fetchAPI<{ status: string }>(`/projects/${projectId}/workflows/web-screening`, { method: "POST", signal }),

  listFindings: (projectId: string, status?: string, signal?: AbortSignal) =>
    fetchAPI<Finding[]>(`/projects/${projectId}/findings${status ? `?status=${status}` : ""}`, { signal }),

  getFinding: (id: string, signal?: AbortSignal) =>
    fetchAPI<{ finding: Finding; evidence: Evidence[] }>(`/findings/${id}`, { signal }),

  patchFindingStatus: (id: string, status: string, signal?: AbortSignal) =>
    fetchAPI<{ status: string }>(`/findings/${id}/status`, { method: "PATCH", body: JSON.stringify({ status }), signal }),

  updateFindingStatus: (id: string, status: string, signal?: AbortSignal) =>
    fetchAPI<{ status: string }>(`/findings/${id}/status`, { method: "PATCH", body: JSON.stringify({ status }), signal }),

  addEvidence: (findingId: string, data: { type: string; excerpt: string; created_by?: string }, signal?: AbortSignal) =>
    fetchAPI<Evidence>(`/findings/${findingId}/evidence`, { method: "POST", body: JSON.stringify(data), signal }),

  exportReportMD: (projectId: string, signal?: AbortSignal) =>
    fetchBlob(`/projects/${projectId}/reports/export.md`, { signal }),

  exportReportJSON: (projectId: string, signal?: AbortSignal) =>
    fetchBlob(`/projects/${projectId}/reports/export.json`, { signal }),

  // --- Runs ---
  listRuns: (projectId: string, signal?: AbortSignal) =>
    fetchAPI<Run[]>(`/projects/${projectId}/runs`, { signal }),
  createRun: (projectId: string, data: { tool_template_id: string; name: string }, signal?: AbortSignal) =>
    fetchAPI<Run>(`/projects/${projectId}/runs`, { method: "POST", body: JSON.stringify(data), signal }),
  getRun: (id: string, signal?: AbortSignal) =>
    fetchAPI<Run>(`/runs/${id}`, { signal }),
  getRunTasks: (id: string, signal?: AbortSignal) =>
    fetchAPI<ScanTask[]>(`/runs/${id}/tasks`, { signal }),
  cancelRun: (id: string, signal?: AbortSignal) =>
    fetchAPI<{ status: string }>(`/runs/${id}/cancel`, { method: "POST", signal }),

  // --- Tool Templates ---
  listToolTemplates: (signal?: AbortSignal) =>
    fetchAPI<ToolTemplate[]>("/tool-templates", { signal }),

  // --- Retest ---
  retestFinding: (id: string, signal?: AbortSignal) =>
    fetchAPI<any>(`/findings/${id}/retest`, { method: "POST", signal }),
  listRetests: (id: string, signal?: AbortSignal) =>
    fetchAPI<any[]>(`/findings/${id}/retests`, { signal }),
  batchUpdateFindingStatus: (ids: string[], status: string, signal?: AbortSignal) =>
    fetchAPI<any>(`/findings/batch-status`, { method: "PATCH", body: JSON.stringify({ ids, status }), signal }),
};

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

export interface ToolTemplate {
  id: string;
  name: string;
  description: string;
  tools_json: string;
  rate_limit: number;
  is_preset: boolean;
  created_at: string;
}
