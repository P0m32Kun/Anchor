import { getApiBase } from "./config";
export const API_BASE = getApiBase();

export class APIError extends Error {
  code?: string;
  constructor(message: string, code?: string) {
    super(message);
    this.code = code;
  }
}

async function fetchJSON<T>(path: string, opts?: RequestInit): Promise<T> {
  const res = await fetch(`${API_BASE}${path}`, {
    headers: { "Content-Type": "application/json" },
    ...opts,
  });
  if (!res.ok) {
    const data = await res.json().catch(() => null);
    const message = data?.error?.message || `${res.status}: ${await res.text()}`;
    throw new APIError(message, data?.error?.code);
  }
  return res.json();
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
  created_at: string;
}

export interface Target {
  id: string;
  project_id: string;
  type: string;
  value: string;
  created_at: string;
}

export interface ImportResult {
  imported: number;
  duplicates: number;
  denied: number;
  errors: number;
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
  createProject: (data: { name: string; organization?: string; purpose?: string; start_time?: string; end_time?: string; rate_limit?: number }) =>
    fetchJSON<Project>("/projects", { method: "POST", body: JSON.stringify(data) }),

  listProjects: () => fetchJSON<Project[]>("/projects"),

  getProject: (id: string) => fetchJSON<Project>(`/projects/${id}`),

  createTarget: (projectId: string, data: { type: string; value: string }) =>
    fetchJSON<Target>(`/projects/${projectId}/targets`, { method: "POST", body: JSON.stringify(data) }),

  listTargets: (projectId: string) => fetchJSON<Target[]>(`/projects/${projectId}/targets`),

  createScopeRule: (data: { project_id: string; action: string; type: string; value: string; reason?: string }) =>
    fetchJSON<ScopeRule>("/scope-rules", { method: "POST", body: JSON.stringify(data) }),

  listScopeRules: (projectId: string) =>
    fetchJSON<ScopeRule[]>(`/scope-rules?project_id=${projectId}`),

  dryRun: (projectId: string) =>
    fetchJSON<DryRunResult>(`/scan-plans/dry-run?project_id=${projectId}`, { method: "POST" }),

  importTargets: async (projectId: string, file: File): Promise<ImportResult> => {
    const formData = new FormData();
    formData.append("file", file);
    const res = await fetch(`${API_BASE}/projects/${projectId}/targets/import`, {
      method: "POST",
      body: formData,
    });
    if (!res.ok) {
      const data = await res.json().catch(() => null);
      throw new APIError(data?.error?.message || res.statusText);
    }
    return res.json();
  },

  runTask: (data: { project_id: string; plan_id?: string; tool: string; target_id: string; command: string }) =>
    fetchJSON<ScanTask>("/tasks/run", { method: "POST", body: JSON.stringify(data) }),

  getTask: (id: string) => fetchJSON<ScanTask>(`/scan-tasks/${id}`),

  cancelTask: (id: string) =>
    fetchJSON<any>(`/scan-tasks/${id}/cancel`, { method: "POST" }),

  listArtifacts: (taskId: string) =>
    fetchJSON<any[]>(`/tasks/${taskId}/artifacts`),

  listToolHealth: () => fetchJSON<ToolHealth[]>("/health/tools"),

  runHealthCheck: () => fetchJSON<ToolHealth[]>("/health/check", { method: "POST" }),

  startAssetDiscovery: (projectId: string) =>
    fetchJSON<{ status: string }>(`/projects/${projectId}/workflows/asset-discovery`, { method: "POST" }),

  listAssets: (projectId: string) => fetchJSON<Asset[]>(`/projects/${projectId}/assets`),

  listWebEndpoints: (projectId: string) => fetchJSON<WebEndpoint[]>(`/projects/${projectId}/web-endpoints`),

  listPorts: (assetId: string) => fetchJSON<Port[]>(`/assets/${assetId}/ports`),

  listServices: (assetId: string) => fetchJSON<Service[]>(`/assets/${assetId}/services`),

  startWebScreening: (projectId: string) =>
    fetchJSON<{ status: string }>(`/projects/${projectId}/workflows/web-screening`, { method: "POST" }),

  listFindings: (projectId: string, status?: string) =>
    fetchJSON<Finding[]>(`/projects/${projectId}/findings${status ? `?status=${status}` : ""}`),

  getFinding: (id: string) =>
    fetchJSON<{ finding: Finding; evidence: Evidence[] }>(`/findings/${id}`),

  patchFindingStatus: (id: string, status: string) =>
    fetchJSON<{ status: string }>(`/findings/${id}/status`, { method: "PATCH", body: JSON.stringify({ status }) }),

  addEvidence: (findingId: string, data: { type: string; excerpt: string; created_by?: string }) =>
    fetchJSON<Evidence>(`/findings/${findingId}/evidence`, { method: "POST", body: JSON.stringify(data) }),

  exportReportMD: async (projectId: string): Promise<Blob> => {
    const res = await fetch(`${API_BASE}/projects/${projectId}/reports/export.md`);
    if (!res.ok) {
      const data = await res.json().catch(() => null);
      throw new APIError(data?.error?.message || `Export failed: ${res.statusText}`);
    }
    return res.blob();
  },

  exportReportJSON: async (projectId: string): Promise<Blob> => {
    const res = await fetch(`${API_BASE}/projects/${projectId}/reports/export.json`);
    if (!res.ok) {
      const data = await res.json().catch(() => null);
      throw new APIError(data?.error?.message || `Export failed: ${res.statusText}`);
    }
    return res.blob();
  },

  // --- Runs ---
  listRuns: (projectId: string) =>
    fetchJSON<Run[]>(`/projects/${projectId}/runs`),
  createRun: (projectId: string, data: { tool_template_id: string; name: string }) =>
    fetchJSON<Run>(`/projects/${projectId}/runs`, { method: "POST", body: JSON.stringify(data) }),
  getRun: (id: string) =>
    fetchJSON<Run>(`/runs/${id}`),
  getRunTasks: (id: string) =>
    fetchJSON<ScanTask[]>(`/runs/${id}/tasks`),

  // --- Tool Templates ---
  listToolTemplates: () =>
    fetchJSON<ToolTemplate[]>("/tool-templates"),

  // --- Retest ---
  retestFinding: (id: string) =>
    fetchJSON<any>(`/findings/${id}/retest`, { method: "POST" }),
  listRetests: (id: string) =>
    fetchJSON<any[]>(`/findings/${id}/retests`),
  batchUpdateFindingStatus: (ids: string[], status: string) =>
    fetchJSON<any>(`/findings/batch-status`, { method: "PATCH", body: JSON.stringify({ ids, status }) }),
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
