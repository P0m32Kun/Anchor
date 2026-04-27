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
            {/* Legacy routes for backward compat */}
            <Route path="/projects/:id" element={<ProjectPage />} />
            <Route path="/projects/:id/assets" element={<AssetPage />} />
            <Route path="/projects/:id/findings" element={<FindingsPage />} />
            <Route path="/projects/:id/reports" element={<ReportsPage />} />
          </Routes>
        </main>
      </div>
    </ToastProvider>
  );
}

export default App;
