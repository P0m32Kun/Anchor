/**
 * Route Registry — ALL application routes must be defined here.
 * When adding a new page, update this table and ensure the component file exists.
 *
 * | Path                     | Component      | Purpose                          | Navbar Link |
 * |--------------------------|----------------|----------------------------------|-------------|
 * | /                        | DashboardPage  | Main dashboard with stats        | Yes         |
 * | /projects                | ProjectPage    | Project list & creation          | Yes         |
 * | /targets                 | TargetPage     | Target management & import       | Yes         |
 * | /assets                  | AssetPage      | Asset inventory                  | Yes         |
 * | /runs                    | RunsPage       | Scan run history & controls      | Yes         |
 * | /findings                | FindingsPage   | Security findings list           | Yes         |
 * | /reports                 | ReportsPage    | Report generation & export       | Yes         |
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

import { Routes, Route } from "react-router-dom";
import { useEffect } from "react";
import { ToastProvider, Navbar, useToast } from "./components";
import { setGlobalErrorHandler } from "./lib/api";
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

function AppContent() {
  const toast = useToast();

  useEffect(() => {
    setGlobalErrorHandler((err) => {
      toast(err.message, "error");
    });
    return () => {
      setGlobalErrorHandler(() => {});
    };
  }, [toast]);

  return (
    <div className="min-h-screen flex flex-col bg-surface text-text-primary">
      <Navbar />
      <main className="flex-1 px-6 py-6 max-w-7xl mx-auto w-full">
        <Routes>
          <Route path="/" element={<DashboardPage />} />
          <Route path="/targets" element={<TargetPage />} />
          <Route path="/assets" element={<AssetPage />} />
          <Route path="/runs" element={<RunsPage />} />
          <Route path="/findings" element={<FindingsPage />} />
          <Route path="/reports" element={<ReportsPage />} />
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
      </main>
    </div>
  );
}

function App() {
  return (
    <ToastProvider>
      <AppContent />
    </ToastProvider>
  );
}

export default App;
