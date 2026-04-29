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
 * | /projects/:id            | ProjectPage    | Legacy project detail (params    | No          |
 * | /projects/:id/assets     | AssetPage      | not fully handled)               | No          |
 * | /projects/:id/targets    | TargetPage     |                                  | No          |
 * | /projects/:id/runs       | RunsPage       |                                  | No          |
 * | /projects/:id/findings   | FindingsPage   |                                  | No          |
 * | /projects/:id/reports    | ReportsPage    |                                  | No          |
 *
 * TODO: Legacy routes (/projects/:id/*) do not use useParams() in their
 * components. They render the same content as the base page. Consider
 * removing or properly implementing them.
 */

import { Routes, Route } from "react-router-dom";
import { ToastProvider, Navbar } from "./components";
import DashboardPage from "./pages/DashboardPage";
import ProjectPage from "./pages/ProjectPage";
import TargetPage from "./pages/TargetPage";
import AssetPage from "./pages/AssetPage";
import RunsPage from "./pages/RunsPage";
import FindingsPage from "./pages/FindingsPage";
import ReportsPage from "./pages/ReportsPage";
import WorkersPage from "./pages/WorkersPage";
import SettingsPage from "./pages/SettingsPage";

function App() {
  return (
    <ToastProvider>
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
            {/* Legacy routes for backward compat */}
            <Route path="/projects/:id" element={<ProjectPage />} />
            <Route path="/projects/:id/assets" element={<AssetPage />} />
            <Route path="/projects/:id/targets" element={<TargetPage />} />
            <Route path="/projects/:id/runs" element={<RunsPage />} />
            <Route path="/projects/:id/findings" element={<FindingsPage />} />
            <Route path="/projects/:id/reports" element={<ReportsPage />} />
          </Routes>
        </main>
      </div>
    </ToastProvider>
  );
}

export default App;
