import { useEffect, useState } from "react";
import { useNavigate } from "react-router-dom";
import { api } from "../lib/api";
import { useStore } from "../lib/store";
import { Button } from "../components/Button";
import { StatusBadge } from "../components/Badge";
import { useToast, ConfirmDialog } from "../components";

export default function ProjectPage() {
  const navigate = useNavigate();
  const projects = useStore((state) => state.projects) ?? [];
  const setProjects = useStore((state) => state.setProjects);
  const setCurrentProject = useStore((state) => state.setCurrentProject);
  const [name, setName] = useState("");
  const [org, setOrg] = useState("");
  const [purpose, setPurpose] = useState("");
  const [startTime, setStartTime] = useState("");
  const [endTime, setEndTime] = useState("");
  const [rateLimit, setRateLimit] = useState(0);
  const [loading, setLoading] = useState(false);
  const [deletingId, setDeletingId] = useState<string | null>(null);
  const [deleteDialogOpen, setDeleteDialogOpen] = useState(false);
  const [deleteTarget, setDeleteTarget] = useState<typeof projects[0] | null>(null);
  const toast = useToast();

  useEffect(() => {
    const ctrl = new AbortController();
    api.listProjects(ctrl.signal)
      .then((data) => setProjects(data ?? []))
      .catch((err) => {
        if (err instanceof DOMException && err.name === "AbortError") return;
        toast("加载项目列表失败: " + (err instanceof Error ? err.message : String(err)), "error");
        console.error(err);
      });
    return () => ctrl.abort();
  }, [setProjects, toast]);

  const handleCreate = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!name.trim()) {
      toast("项目名称不能为空", "warning");
      return;
    }
    if (loading) return;
    setLoading(true);
    try {
      const p = await api.createProject({
        name,
        organization: org || undefined,
        purpose: purpose || undefined,
        start_time: startTime ? new Date(startTime).toISOString() : undefined,
        end_time: endTime ? new Date(endTime).toISOString() : undefined,
        rate_limit: rateLimit,
      });
      setProjects([p, ...projects]);
      setCurrentProject(p);
      setName("");
      setOrg("");
      setPurpose("");
      setStartTime("");
      setEndTime("");
      setRateLimit(0);
    } catch (err) {
      toast("创建失败: " + String(err), "error");
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="max-w-3xl">
      <h1 className="text-2xl font-bold mb-6 text-text-primary tracking-tight">项目管理</h1>

      <form onSubmit={handleCreate} className="card-dark p-6 mb-8 space-y-5">
        <h2 className="text-base font-semibold text-brand-primary flex items-center gap-2">
          <span className="w-1.5 h-1.5 rounded-full bg-brand-primary shadow-[0_0_8px_rgba(47,129,247,0.6)]" />
          新建项目
        </h2>
        
        <div className="space-y-4">
          <input
            className="input-dark"
            placeholder="项目名称 *"
            value={name}
            onChange={(e) => setName(e.target.value)}
            required
          />
          <div className="flex gap-4">
            <input
              className="flex-1 input-dark"
              placeholder="组织/客户"
              value={org}
              onChange={(e) => setOrg(e.target.value)}
            />
            <input
              className="flex-1 input-dark"
              placeholder="目的/描述"
              value={purpose}
              onChange={(e) => setPurpose(e.target.value)}
            />
          </div>
          <div className="flex gap-4">
            <div className="flex-1">
              <label className="block text-[11px] font-medium text-text-tertiary mb-1.5 uppercase tracking-wider">开始时间</label>
              <input
                type="datetime-local"
                className="input-dark"
                value={startTime}
                onChange={(e) => setStartTime(e.target.value)}
              />
            </div>
            <div className="flex-1">
              <label className="block text-[11px] font-medium text-text-tertiary mb-1.5 uppercase tracking-wider">结束时间</label>
              <input
                type="datetime-local"
                className="input-dark"
                value={endTime}
                onChange={(e) => setEndTime(e.target.value)}
              />
            </div>
            <div>
              <label className="block text-[11px] font-medium text-text-tertiary mb-1.5 uppercase tracking-wider">速率限制</label>
              <input
                type="number"
                min={0}
                className="w-28 input-dark"
                value={rateLimit}
                onChange={(e) => setRateLimit(Math.max(0, Number(e.target.value)))}
              />
            </div>
          </div>
        </div>
        
        <div className="pt-2">
          <Button type="submit" variant="primary" loading={loading} className="w-full sm:w-auto px-8">
            创建项目
          </Button>
        </div>
      </form>

      <div className="space-y-4">
        {projects.map((p) => {
          const now = new Date();
          const start = p.start_time ? new Date(p.start_time) : null;
          const end = p.end_time ? new Date(p.end_time) : null;
          const isActive = (!start || start <= now) && (!end || end >= now);
          const isExpired = end && end < now;
          const status = isExpired ? "expired" : isActive ? "active" : "pending";

          return (
            <div
              key={p.id}
              className="card-dark p-5 group relative"
            >
              <div
                onClick={() => {
                  setCurrentProject(p);
                  navigate("/");
                }}
                className="cursor-pointer"
              >
                <div className="flex items-center justify-between">
                  <div className="flex items-center gap-3">
                    <div className="font-semibold text-text-primary group-hover:text-brand-primary transition-colors text-base">{p.name}</div>
                    <StatusBadge status={status} />
                  </div>
                  <div className="text-text-quaternary opacity-0 group-hover:opacity-100 transition-opacity">
                    <svg className="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth="2">
                      <path strokeLinecap="round" strokeLinejoin="round" d="M9 5l7 7-7 7" />
                    </svg>
                  </div>
                </div>
                <div className="text-[13px] text-text-tertiary mt-3 flex flex-wrap gap-x-4 gap-y-1">
                  <div className="flex items-center gap-1.5">
                    <span className="text-text-quaternary">组织:</span>
                    <span className="text-text-secondary">{p.organization || "—"}</span>
                  </div>
                  {p.purpose && (
                    <div className="flex items-center gap-1.5">
                      <span className="text-text-quaternary">目的:</span>
                      <span className="text-text-secondary">{p.purpose}</span>
                    </div>
                  )}
                  <div className="flex items-center gap-1.5">
                    <span className="text-text-quaternary">创建:</span>
                    <span className="text-text-secondary">{new Date(p.created_at).toLocaleDateString()}</span>
                  </div>
                </div>
              </div>
              {/* 删除按钮 */}
              <div className="absolute top-3 right-3">
                <button
                  onClick={(e) => {
                    e.stopPropagation();
                    setDeleteTarget(p);
                    setDeleteDialogOpen(true);
                  }}
                  className="opacity-0 group-hover:opacity-100 transition-opacity p-1 rounded hover:bg-red-500/10 text-text-quaternary hover:text-red-400"
                  title="删除项目"
                >
                  <svg className="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth="2">
                    <path strokeLinecap="round" strokeLinejoin="round" d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16" />
                  </svg>
                </button>
              </div>
            </div>
          );
        })}
        {projects.length === 0 && (
          <div className="text-text-tertiary text-center py-12 liquid-glass rounded-apple border-dashed border-2 border-white/5">
            <div className="mb-2 text-lg font-medium text-text-secondary">暂无项目</div>
            <p className="text-sm">点击上方“新建项目”开始您的第一次扫描</p>
          </div>
        )}
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
