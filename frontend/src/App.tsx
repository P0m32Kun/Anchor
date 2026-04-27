import { Routes, Route, Link } from "react-router-dom";
import ProjectPage from "./pages/ProjectPage";
import TargetPage from "./pages/TargetPage";
import AssetPage from "./pages/AssetPage";
import RunsPage from "./pages/RunsPage";
import FindingsPage from "./pages/FindingsPage";
import ReportsPage from "./pages/ReportsPage";

function App() {
  return (
    <div className="min-h-screen flex flex-col">
      <nav className="bg-slate-800 text-white px-4 py-3 flex gap-6">
        <Link to="/" className="font-bold text-lg">SecBench</Link>
        <Link to="/" className="hover:underline">项目</Link>
        <Link to="/runs" className="hover:underline">运行</Link>
      </nav>
      <main className="flex-1 p-6">
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
  );
}

export default App;
