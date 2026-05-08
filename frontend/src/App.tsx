/**
 * Route Registry — ALL application routes must be defined here.
 * When adding a new page, update this table and ensure the component file exists.
 *
 * | Path                     | Component      | Purpose                          | Navbar Link |
 * |--------------------------|----------------|----------------------------------|-------------|
 * | /                        | DashboardPage  | Main dashboard with stats        | Yes         |
 * | /projects                | ProjectPage    | Project list & creation          | Yes         |
 * | /targets                 | LegacyRouteGuard| Redirect to /projects/:id/targets | Yes         |
 * | /assets                  | LegacyRouteGuard| Redirect to /projects/:id/assets  | Yes         |
 * | /runs                    | LegacyRouteGuard| Redirect to /projects/:id/runs    | Yes         |
 * | /findings                | LegacyRouteGuard| Redirect to /projects/:id/findings| Yes         |
 * | /reports                 | LegacyRouteGuard| Redirect to /projects/:id/reports | Yes         |
 * | /workers                 | WorkersPage    | Worker node management           | Yes         |
 * | /settings                | SettingsPage   | App configuration                | Yes         |
 * | /projects/:projectId     | ProjectLayout  | Project wrapper + redirect       | No          |
 * | /projects/:projectId     | Navigate       | Index → redirects to targets     | No          |
 * | /projects/:projectId/targets | TargetPage   | Nested: targets                  | No          |
 * | /projects/:projectId/assets  | AssetPage    | Nested: assets                   | No          |
 * | /projects/:projectId/runs    | RunsPage     | Nested: runs                     | No          |
 * | /projects/:projectId/findings| FindingsPage | Nested: findings                 | No          |
 * | /projects/:projectId/reports | ReportsPage  | Nested: reports                  | No          |
 * | /projects/:id            | ProjectPage    | Legacy project detail            | No          |
 * | /projects/:id/assets     | AssetPage      | Legacy (same as nested)          | No          |
 * | /projects/:id/targets    | TargetPage     | Legacy (same as nested)          | No          |
 * | /projects/:id/runs       | RunsPage       | Legacy (same as nested)          | No          |
 * | /projects/:id/findings   | FindingsPage   | Legacy (same as nested)          | No          |
 * | /projects/:id/reports    | ReportsPage    | Legacy (same as nested)          | No          |
 *
 * TODO: Legacy routes (/projects/:id/*) do not use useParams() in their
 * components. They render the same content as the base page. Consider
 * removing or properly implementing them.
 */

import { Routes, Route, useLocation, Navigate, useNavigate } from "react-router-dom";
import { useEffect, useState } from "react";
import { ToastProvider, Navbar, useToast, ErrorBoundary, Button } from "./components";
import { setGlobalErrorHandler, setConsecutiveErrorCallback, api, API_BASE } from "./lib/api";
import { resetApiBase, needsApiBaseConfig, getApiToken, resetApiToken, needsApiToken } from "./lib/config";
import ApiBaseSetup from "./components/ApiBaseSetup";
import { useStore } from "./lib/store";
import ProjectLayout from "./components/ProjectLayout";
import DashboardPage from "./pages/DashboardPage";
import ProjectPage from "./pages/ProjectPage";
import TargetPage from "./pages/TargetPage";
import AssetPage from "./pages/AssetPage";
import RunsPage from "./pages/RunsPage";
import FindingsPage from "./pages/FindingsPage";
import ReportsPage from "./pages/ReportsPage";
import WorkersPage from "./pages/WorkersPage";
import SettingsPage from "./pages/SettingsPage";
import EnginesPage from "./pages/EnginesPage";
import EngineKeysPage from "./pages/EngineKeysPage";

function LegacyRouteGuard() {
  const location = useLocation();
  const lastProjectId = useStore((s) => s.currentProjectId);

  useEffect(() => {
    console.warn(`[Deprecation] Accessed legacy route: ${location.pathname}`);
  }, [location]);

  if (!lastProjectId) return <Navigate to="/projects" replace />;

  const legacyMap: Record<string, string> = {
    "/targets": `/projects/${lastProjectId}/targets`,
    "/assets": `/projects/${lastProjectId}/assets`,
    "/runs": `/projects/${lastProjectId}/runs`,
    "/findings": `/projects/${lastProjectId}/findings`,
    "/reports": `/projects/${lastProjectId}/reports`,
  };

  const redirectTo = legacyMap[location.pathname];
  return redirectTo ? <Navigate to={redirectTo} replace /> : <Navigate to="/projects" replace />;
}

function AppHealthCheck({ children }: { children: React.ReactNode }) {
  const [healthy, setHealthy] = useState<boolean | null>(null);
  const [errorInfo, setErrorInfo] = useState<{ message: string; url: string } | null>(null);
  const [diagResult, setDiagResult] = useState<string | null>(null);
  const navigate = useNavigate();

  const healthUrl = `${API_BASE}/health`;

  useEffect(() => {
    api.healthCheck()
      .then(() => setHealthy(true))
      .catch((err: any) => {
        const is401 = err?.code === "HTTP_4xx" && err?.message?.includes("认证失败");
        const message = is401 ? "Token 无效，请检查输入的 API Token" : (err?.message || String(err));
        setErrorInfo({ message, url: healthUrl });
        setHealthy(false);
      });
  }, []);

  const runDiag = async () => {
    setDiagResult("测试中...");
    try {
      const token = getApiToken();
      const headers: Record<string, string> = token ? { Authorization: `Bearer ${token}` } : {};
      const res = await fetch(healthUrl, { method: "GET", mode: "cors", headers });
      const text = await res.text();
      setDiagResult(`HTTP ${res.status}: ${text}`);
    } catch (e: any) {
      setDiagResult(`Fetch 失败: ${e?.message || String(e)}`);
    }
  };

  if (healthy === null) {
    return (
      <div className="flex items-center justify-center h-screen bg-surface text-text-primary">
        <div className="text-center">
          <div className="animate-pulse text-2xl mb-4">🔄</div>
          <p className="text-text-secondary">检查服务状态中...</p>
        </div>
      </div>
    );
  }

  if (!healthy) {
    return (
      <div className="flex flex-col items-center justify-center h-screen bg-surface text-text-primary px-4">
        <div className="text-4xl mb-4">🔌</div>
        <h1 className="text-xl font-semibold mb-2">后端服务未启动</h1>
        <p className="text-text-secondary mb-2">请确认 Anchor 服务正在运行</p>
        {errorInfo && (
          <div className="bg-surface-elevated border border-white/10 rounded-lg p-4 mb-4 max-w-md text-left text-sm">
            <p className="text-text-secondary mb-1"><strong>请求 URL:</strong> {errorInfo.url}</p>
            <p className="text-text-secondary"><strong>错误信息:</strong> {errorInfo.message}</p>
          </div>
        )}
        <div className="flex gap-3 mb-4">
          <Button onClick={() => navigate(0)}>重试</Button>
          <Button variant="secondary" onClick={runDiag}>网络诊断</Button>
        </div>
        {diagResult && (
          <div className="bg-surface-elevated border border-white/10 rounded-lg p-3 max-w-md text-left text-sm text-text-secondary">
            <strong>诊断结果:</strong> {diagResult}
          </div>
        )}
        <button
          className="text-xs text-text-secondary underline mt-4"
          onClick={() => { resetApiBase(); resetApiToken(); navigate(0); }}
        >
          重置 API 设置并刷新
        </button>
      </div>
    );
  }

  return children;
}

function AppContent() {
  const toast = useToast();

  useEffect(() => {
    setGlobalErrorHandler((err) => {
      let title = "错误";
      if (err.code === "TIMEOUT") title = "请求超时";
      else if (err.code === "NETWORK_ERROR") title = "网络错误";
      else if (err.code === "HTTP_5xx") title = "服务器错误";
      else if (err.code === "HTTP_4xx") title = "请求错误";
      else if (err.code === "NON_JSON_RESPONSE") title = "响应格式错误";
      if (err.message?.includes("认证失败")) {
        title = "认证失败";
      }
      toast(`${title}：${err.message}`, "error");
    });
    return () => {
      setGlobalErrorHandler(() => {});
    };
  }, [toast]);

  useEffect(() => {
    setConsecutiveErrorCallback(() => {
      toast("后端服务异常，请检查服务状态", "error");
    });
    return () => setConsecutiveErrorCallback(() => {});
  }, [toast]);

  return (
    <div className="min-h-screen bg-background text-foreground">
      <Navbar />
      <main className="pl-64 min-h-screen">
        <div className="container py-8 max-w-6xl">
          <ErrorBoundary>
            <Routes>
            <Route path="/" element={<DashboardPage />} />
            <Route path="/targets" element={<LegacyRouteGuard />} />
            <Route path="/assets" element={<LegacyRouteGuard />} />
            <Route path="/runs" element={<LegacyRouteGuard />} />
            <Route path="/findings" element={<LegacyRouteGuard />} />
            <Route path="/reports" element={<LegacyRouteGuard />} />
            <Route path="/engines" element={<EnginesPage />} />
            <Route path="/engines/keys" element={<EngineKeysPage />} />
            <Route path="/workers" element={<WorkersPage />} />
            <Route path="/settings" element={<SettingsPage />} />
            <Route path="/projects" element={<ProjectPage />} />
            <Route path="/projects/:projectId" element={<ProjectLayout />}>
              <Route index element={<Navigate to="targets" replace />} />
              <Route path="targets" element={<TargetPage />} />
              <Route path="assets" element={<AssetPage />} />
              <Route path="runs" element={<RunsPage />} />
              <Route path="findings" element={<FindingsPage />} />
              <Route path="reports" element={<ReportsPage />} />
            </Route>
            {/* Legacy routes for backward compat — Sprint 1.11 will remove */}
            <Route path="/projects/:id" element={<ProjectPage />} />
            <Route path="/projects/:id/assets" element={<AssetPage />} />
            <Route path="/projects/:id/targets" element={<TargetPage />} />
            <Route path="/projects/:id/runs" element={<RunsPage />} />
            <Route path="/projects/:id/findings" element={<FindingsPage />} />
            <Route path="/projects/:id/reports" element={<ReportsPage />} />
          </Routes>
        </ErrorBoundary>
      </main>
    </div>
  );
}

function App() {
  const needsConfig = needsApiBaseConfig() || needsApiToken();

  if (needsConfig) {
    return (
      <ToastProvider>
        <ApiBaseSetup />
      </ToastProvider>
    );
  }

  return (
    <ToastProvider>
      <AppHealthCheck>
        <AppContent />
      </AppHealthCheck>
    </ToastProvider>
  );
}

export default App;
