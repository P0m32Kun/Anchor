const API_BASE = "http://localhost:8080";

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

export interface ToolHealth {
  tool: string;
  binary_path?: string;
  version?: string;
  dns_available: boolean;
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
};
