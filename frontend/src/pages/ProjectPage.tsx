import { useEffect, useState } from "react";
import type React from "react";
import { useNavigate } from "react-router-dom";
import { api, PAGE_ALL } from "../lib/api";
import { useStore } from "../lib/store";
import { Button } from "../components/Button";
import { useToast, ConfirmDialog, EmptyState, SkeletonList } from "../components";

export default function ProjectPage() {
  const navigate = useNavigate();
  const projects = useStore((state) => state.projects) ?? [];
  const setProjects = useStore((state) => state.setProjects);
  const setCurrentProject = useStore((state) => state.setCurrentProject);
  const [name, setName] = useState("");
  const [org, setOrg] = useState("");
  const [purpose, setPurpose] = useState("");
  const [creating, setCreating] = useState(false);
  const [deletingId, setDeletingId] = useState<string | null>(null);
  const projectsLoading = useStore((state) => state.projectsLoading);
  const setProjectsLoading = useStore((state) => state.setProjectsLoading);
  const setProjectsError = useStore((state) => state.setProjectsError);
  const [deleteDialogOpen, setDeleteDialogOpen] = useState(false);
  const [deleteTarget, setDeleteTarget] = useState<typeof projects[0] | null>(null);
  const toast = useToast();

  useEffect(() => {
    const ctrl = new AbortController();
    setProjectsLoading(true);
    setProjectsError(null);
    api.listProjects(PAGE_ALL, ctrl.signal)
      .then((res) => {
        setProjects(res.data ?? []);
        setProjectsError(null);
      })
      .catch((err) => {
        if (err instanceof DOMException && err.name === "AbortError") return;
        const msg = err instanceof Error ? err.message : String(err);
        setProjectsError(msg);
        toast("加载项目列表失败: " + msg, "error");
        console.error(err);
      })
      .finally(() => setProjectsLoading(false));
    return () => ctrl.abort();
  }, [setProjects, setProjectsLoading, setProjectsError, toast]);

  const handleCreate = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!name.trim()) {
      toast("项目名称不能为空", "warning");
      return;
    }
    if (creating) return;
    setCreating(true);
    try {
      const p = await api.createProject({
        name,
        organization: org || undefined,
        purpose: purpose || undefined,
      });
      setProjects([p, ...projects]);
      setCurrentProject(p);
      setName("");
      setOrg("");
      setPurpose("");
    } catch (err) {
      toast("创建失败: " + (err instanceof Error ? err.message : String(err)), "error");
    } finally {
      setCreating(false);
    }
  };

  return (
    <div className="page-shell space-y-6">
      <div className="page-header">
        <div>
          <div className="page-eyebrow">Projects</div>
          <h1 className="page-title">项目与授权边界</h1>
          <p className="page-subtitle">每个项目承载一次授权测试交付，后续目标、资产、扫描、发现和报告都围绕项目展开。</p>
        </div>
      </div>

      <div className="grid gap-6 xl:grid-cols-[380px_1fr]">
        <form onSubmit={handleCreate} className="panel h-fit">
          <div className="panel-header">
            <div>
              <h2 className="panel-title">新建项目</h2>
              <p className="text-xs text-text-tertiary mt-1">先建立授权容器，再导入目标范围</p>
            </div>
          </div>
          <div className="panel-body space-y-4">
            <input
              className="input-dark"
              placeholder="项目名称 *"
              value={name}
              onChange={(e) => setName(e.target.value)}
              required
            />
            <input
              className="input-dark"
              placeholder="组织/客户"
              value={org}
              onChange={(e) => setOrg(e.target.value)}
            />
            <input
              className="input-dark"
              placeholder="目的/描述"
              value={purpose}
              onChange={(e) => setPurpose(e.target.value)}
            />
            <Button type="submit" variant="primary" loading={creating} className="w-full">
              创建项目
            </Button>
          </div>
        </form>

        <section className="panel">
          <div className="panel-header">
            <div>
              <h2 className="panel-title">项目列表</h2>
              <p className="text-xs text-text-tertiary mt-1">选择项目后从目标与 Scope 开始推进</p>
            </div>
            <span className="chip">{projects.length} 个项目</span>
          </div>
          <div className="panel-body">
            {projectsLoading ? (
              <SkeletonList count={3} />
            ) : projects.length === 0 ? (
              <EmptyState
                title="暂无项目"
                description="创建第一个项目后即可导入目标并启动扫描"
              />
            ) : (
              <div className="space-y-3">
                {projects.map((p) => {
                  return (
                    <div
                      key={p.id}
                      className="group rounded-lg border border-white/[0.08] bg-white/[0.025] p-4 transition-colors hover:border-sky-400/30 hover:bg-sky-500/5"
                    >
                      <div className="flex items-start justify-between gap-3">
                        <button
                          onClick={() => {
                            setCurrentProject(p);
                            navigate(`/projects/${p.id}/targets`);
                          }}
                          className="min-w-0 text-left"
                        >
                          <div className="font-semibold text-text-primary group-hover:text-brand-primary transition-colors text-base">{p.name}</div>
                          <div className="text-[13px] text-text-tertiary mt-2 flex flex-wrap gap-x-4 gap-y-1">
                            <span>组织: <span className="text-text-secondary">{p.organization || "-"}</span></span>
                            {p.purpose && <span>目的: <span className="text-text-secondary">{p.purpose}</span></span>}
                            <span>创建: <span className="text-text-secondary">{new Date(p.created_at).toLocaleDateString()}</span></span>
                          </div>
                        </button>
                        <button
                          onClick={(e) => {
                            e.stopPropagation();
                            setDeleteTarget(p);
                            setDeleteDialogOpen(true);
                          }}
                          className="rounded p-1 text-text-quaternary transition-colors hover:bg-red-500/10 hover:text-red-400"
                          title="删除项目"
                        >
                          <svg className="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth="2">
                            <path strokeLinecap="round" strokeLinejoin="round" d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16" />
                          </svg>
                        </button>
                      </div>
                      <div className="mt-4 flex flex-wrap gap-2">
                        {[
                          ["目标", "targets"],
                          ["资产", "assets"],
                          ["扫描", "runs"],
                          ["发现", "findings"],
                          ["报告", "reports"],
                        ].map(([label, route]) => (
                          <button
                            key={route}
                            onClick={() => {
                              setCurrentProject(p);
                              navigate(`/projects/${p.id}/${route}`);
                            }}
                            className="chip hover:border-sky-400/40 hover:text-sky-100"
                          >
                            {label}
                          </button>
                        ))}
                      </div>
                    </div>
                  );
                })}
              </div>
            )}
          </div>
        </section>
      </div>

      <ConfirmDialog
        open={deleteDialogOpen}
        onClose={() => {
          setDeleteDialogOpen(false);
          setDeleteTarget(null);
        }}
        onConfirm={async () => {
          if (!deleteTarget) return;
          setDeletingId(deleteTarget.id);
          try {
            await api.deleteProject(deleteTarget.id);
            setProjects(projects.filter((proj) => proj.id !== deleteTarget.id));
            setDeleteDialogOpen(false);
            setDeleteTarget(null);
            toast("项目已删除", "success");
          } catch (err) {
            toast("删除失败: " + (err instanceof Error ? err.message : String(err)), "error");
          } finally {
            setDeletingId(null);
          }
        }}
        title="删除项目"
        description={deleteTarget ? `确认删除项目 "${deleteTarget.name}"？此操作不可恢复。` : ""}
        confirmText="删除"
        cancelText="取消"
        variant="danger"
        loading={deletingId !== null}
      />
    </div>
  );
}
