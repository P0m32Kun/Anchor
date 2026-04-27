import { Routes, Route } from "react-router-dom";
import { ToastProvider, Navbar } from "./components";
import ProjectPage from "./pages/ProjectPage";
import TargetPage from "./pages/TargetPage";
import AssetPage from "./pages/AssetPage";
import RunsPage from "./pages/RunsPage";
import FindingsPage from "./pages/FindingsPage";
import ReportsPage from "./pages/ReportsPage";

function App() {
  return (
    <ToastProvider>
      <div className="min-h-screen flex flex-col bg-surface text-text-primary">
        <Navbar />
        <main className="flex-1 px-6 py-6 max-w-6xl mx-auto w-full">
          <Routes>
            <Route path="/" element={<ProjectPage />} />
            <Route path="/projects/:id" element={<TargetPage />} />
            <Route path="/projects/:id/assets" element={<AssetPage />} />
            <Route path="/projects/:id/findings" element={<FindingsPage />} />
            <Route path="/projects/:id/reports" element={<ReportsPage />} />
            <Route path="/runs" element={<RunsPage />} />
          </Routes>
        </main>
      </div>
    </ToastProvider>
  );
}

export default App;
