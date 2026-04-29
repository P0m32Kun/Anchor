import { useEffect, useState, createContext, useContext } from "react";
import { useParams, Outlet, useNavigate } from "react-router-dom";
import { useStore } from "../lib/store";
import { api } from "../lib/api";
import { EmptyState, SkeletonCard } from "./";
import { useToast } from "./Toast";

const ProjectContext = createContext<string | null>(null);
export const useProjectId = () => useContext(ProjectContext);

export default function ProjectLayout() {
  const { projectId } = useParams<{ projectId: string }>();
  const navigate = useNavigate();
  const toast = useToast();
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [expired, setExpired] = useState(false);

  const currentProjectId = useStore((s) => s.currentProjectId);
  const setCurrentProjectId = useStore((s) => s.setCurrentProjectId);
  const setCurrentProject = useStore((s) => s.setCurrentProject);

  useEffect(() => {
    if (!projectId) return;

    // 切换项目时重置 store
    if (projectId !== currentProjectId) {
      setCurrentProjectId(projectId);
    }

    setLoading(true);
    setError(null);

    api
      .getProject(projectId)
      .then((project) => {
        setCurrentProject(project);
        setExpired(
          project.end_time ? new Date(project.end_time) < new Date() : false
        );
        setLoading(false);
      })
      .catch((err) => {
        setError(err.message || "项目不存在");
        setLoading(false);
        setCurrentProject(null);
        toast("项目不存在或已被删除", "error");
        setTimeout(() => navigate("/projects"), 2000);
      });
  }, [projectId, currentProjectId, setCurrentProjectId, setCurrentProject, navigate, toast]);

  if (loading) {
    return (
      <div className="p-8 space-y-4">
        <SkeletonCard />
        <SkeletonCard />
        <SkeletonCard />
      </div>
    );
  }

  if (error) {
    return (
      <div className="p-8">
        <EmptyState
          title="项目不存在"
          description={error}
          actionLabel="返回项目列表"
          onAction={() => navigate("/projects")}
        />
      </div>
    );
  }

  return (
    <ProjectContext.Provider value={projectId || null}>
      <div className="space-y-4">
        {expired && (
          <div className="bg-accent-yellow/10 border border-accent-yellow/20 text-accent-yellow px-4 py-2 rounded-apple text-sm">
            ⚠️ 该项目测试窗口已过期
          </div>
        )}
        <Outlet />
      </div>
    </ProjectContext.Provider>
  );
}
