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
import { setGlobalErrorHandler, setConsecutiveErrorCallback, api } from "./lib/api";
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
  const navigate = useNavigate();

  useEffect(() => {
    api.healthCheck().then(() => setHealthy(true)).catch(() => setHealthy(false));
  }, []);

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
      <div className="flex flex-col items-center justify-center h-screen bg-surface text-text-primary">
        <div className="text-4xl mb-4">🔌</div>
        <h1 className="text-xl font-semibold mb-2">后端服务未启动</h1>
        <p className="text-text-secondary mb-6">请确认 Anchor 服务正在运行</p>
        <Button onClick={() => navigate(0)}>重试</Button>
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
    <div className="min-h-screen flex flex-col bg-surface text-text-primary">
      <Navbar />
      <main className="flex-1 px-6 py-6 max-w-7xl mx-auto w-full">
        <ErrorBoundary>
          <Routes>
            <Route path="/" element={<DashboardPage />} />
            <Route path="/targets" element={<LegacyRouteGuard />} />
            <Route path="/assets" element={<LegacyRouteGuard />} />
            <Route path="/runs" element={<LegacyRouteGuard />} />
            <Route path="/findings" element={<LegacyRouteGuard />} />
            <Route path="/reports" element={<LegacyRouteGuard />} />
            <Route path="/workers" element={<WorkersPage />} />
            <Route path="/settings" element={<SettingsPage />} />
            <Route path="/projects" element={<ProjectPage />} />
            <Route path="/projects/:projectId" element={<ProjectLayout />}>
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
  return (
    <ToastProvider>
      <AppHealthCheck>
        <AppContent />
      </AppHealthCheck>
    </ToastProvider>
  );
}

export default App;
