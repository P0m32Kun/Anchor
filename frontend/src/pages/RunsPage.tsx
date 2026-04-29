import { useEffect, useState, useCallback, useRef } from "react";
import { useNavigate } from "react-router-dom";
import { api } from "../lib/api";
import { useStore } from "../lib/store";
import { EmptyState, useProjectId, ConfirmDialog, Button } from "../components";
import { useToast } from "../components/Toast";
import { useSSE, usePolling } from "../hooks";
import { getApiBase } from "../lib/config";
import type { ScanTask, ToolTemplate, Run } from "../lib/api";

const statusColors: Record<string, string> = {
  pending: "bg-accent-yellow/15 text-accent-yellow",
  running: "bg-brand-primary/15 text-brand-primary",
  completed: "bg-brand-success/15 text-brand-success",
  failed: "bg-brand-danger/15 text-brand-danger",
  cancelled: "bg-white/[0.04] text-text-tertiary",
};

const statusLabels: Record<string, string> = {
  pending: "待启动",
  running: "运行中",
  completed: "已完成",
  failed: "失败",
  cancelled: "已取消",
};

const taskStatusColors: Record<string, string> = {
  created: "bg-white/[0.04] text-text-tertiary",
  queued: "bg-accent-yellow/15 text-accent-yellow",
  running: "bg-brand-primary/15 text-brand-primary",
  completed: "bg-brand-success/15 text-brand-success",
  failed: "bg-brand-danger/15 text-brand-danger",
  cancelled: "bg-white/[0.04] text-text-tertiary",
};

const canCancel = (status: string) =>
  status === "pending" || status === "running";

export default function RunsPage() {
  const projectId = useProjectId();
  const navigate = useNavigate();
  const toast = useToast();
  const runs = useStore((state) => state.runs) ?? [];
  const runsLoading = useStore((state) => state.runsLoading);
  const setRuns = useStore((state) => state.setRuns);
  const setRunsLoading = useStore((state) => state.setRunsLoading);
  const setRunsError = useStore((state) => state.setRunsError);
  const [templates, setTemplates] = useState<ToolTemplate[]>([]);
  const [selectedRun, setSelectedRun] = useState<string | null>(null);
  const [tasks, setTasks] = useState<ScanTask[]>([]);
  const [tasksLoading, setTasksLoading] = useState(false);
  const [showCreate, setShowCreate] = useState(false);
  const [creating, setCreating] = useState(false);
  const [cancelling, setCancelling] = useState(false);
  const [cancelDialogOpen, setCancelDialogOpen] = useState(false);
  const [cancelTargetRun, setCancelTargetRun] = useState<Run | null>(null);

  // Prevent duplicate toasts from SSE/polling rapid failures
  const lastToastErrorRef = useRef<string | null>(null);

  const maybeToastError = useCallback(
    (msg: string) => {
      if (lastToastErrorRef.current !== msg) {
        toast(msg, "error");
        lastToastErrorRef.current = msg;
      }
    },
    [toast]
  );

  const clearToastError = useCallback(() => {
    lastToastErrorRef.current = null;
  }, []);

  const loadRuns = useCallback(
    async (signal?: AbortSignal) => {
      if (!projectId) return;
      setRunsLoading(true);
      setRunsError(null);
      try {
        const data = await api.listRuns(projectId, signal);
        setRuns(data ?? []);
        clearToastError();
      } catch (err) {
        if (err instanceof DOMException && err.name === "AbortError") return;
        const msg = err instanceof Error ? err.message : String(err);
        setRunsError(msg);
        maybeToastError(msg);
        console.error(err);
      } finally {
        setRunsLoading(false);
      }
    },
    [projectId, setRuns, setRunsLoading, setRunsError, maybeToastError, clearToastError]
  );

  const loadTemplates = async (signal?: AbortSignal) => {
    try {
      const data = await api.listToolTemplates(signal);
      setTemplates(data ?? []);
    } catch (err) {
      if (err instanceof DOMException && err.name === "AbortError") return;
      const msg = err instanceof Error ? err.message : "加载工具模板失败";
      toast(msg, "error");
      console.error(err);
    }
  };

  const loadTasks = async (runId: string, signal?: AbortSignal) => {
    setSelectedRun(runId);
    setTasksLoading(true);
    try {
      const data = await api.getRunTasks(runId, signal);
      setTasks(data ?? []);
    } catch (err) {
      if (err instanceof DOMException && err.name === "AbortError") return;
      const msg = err instanceof Error ? err.message : "加载任务详情失败";
      toast(msg, "error");
      console.error(err);
    } finally {
      setTasksLoading(false);
    }
  };

  useEffect(() => {
    if (!projectId) return;
    const ctrl = new AbortController();
    loadRuns(ctrl.signal);
    loadTemplates(ctrl.signal);
    return () => ctrl.abort();
  }, [projectId, loadRuns]);

  // SSE for real-time updates
  const { status: sseStatus } = useSSE(`${getApiBase()}/events`, {
    projectId: projectId ?? undefined,
    onMessage: (raw) => {
      const msg = raw as {
        event?: string;
        run_id?: string;
        project_id?: string;
      };
      if (
        msg.event === "task_update" ||
        msg.event === "asset_discovery_complete" ||
        msg.event === "web_screening_complete"
      ) {
        loadRuns();
        if (selectedRun && msg.run_id === selectedRun) {
          loadTasks(selectedRun);
        }
      }
    },
  });

  // Polling fallback when SSE is unavailable
  const isLive = sseStatus === "open";
  const shouldPoll = !isLive && !!projectId;

  usePolling(
    async () => {
      try {
        const data = await api.listRuns(projectId!);
        setRuns(data ?? []);
        setRunsError(null);
        clearToastError();
        return data ?? [];
      } catch (err) {
        if (err instanceof DOMException && err.name === "AbortError") return [];
        const msg = err instanceof Error ? err.message : "轮询运行状态失败";
        setRunsError(msg);
        maybeToastError(msg);
        return [];
      }
    },
    {
      interval: 5000,
      enabled: shouldPoll,
      pauseOnHidden: true,
    }
  );

  const handleCreate = async (templateId: string, name: string) => {
    if (!projectId || creating) return;
    setCreating(true);
    try {
      await api.createRun(projectId, {
        tool_template_id: templateId,
        name: name || "未命名扫描",
      });
      toast("扫描任务已启动", "success");
      setShowCreate(false);
      await loadRuns();
    } catch (err) {
      const msg = err instanceof Error ? err.message : "启动扫描失败";
      toast(msg, "error");
      console.error(err);
    } finally {
      setCreating(false);
    }
  };

  const openCancelDialog = (run: Run) => {
    setCancelTargetRun(run);
    setCancelDialogOpen(true);
  };

  const closeCancelDialog = () => {
    setCancelDialogOpen(false);
    setCancelTargetRun(null);
  };

  const handleCancelRun = async () => {
    if (!cancelTargetRun || cancelling) return;
    setCancelling(true);
    try {
      await api.cancelRun(cancelTargetRun.id);
      toast("扫描任务已取消", "success");
      await loadRuns();
      if (selectedRun === cancelTargetRun.id) {
        await loadTasks(cancelTargetRun.id);
      }
    } catch (err) {
      const msg = err instanceof Error ? err.message : "取消扫描失败";
      toast(msg, "error");
      console.error(err);
    } finally {
      setCancelling(false);
      closeCancelDialog();
    }
  };

  const runsError = useStore((state) => state.runsError);

  return (
    <div className="max-w-5xl space-y-6">
      {/* Title area */}
      <div className="flex items-center justify-between">
        <h1 className="text-2xl font-bold text-zinc-100 tracking-tight">
          扫描执行
        </h1>
        {projectId && (
          <Button variant="primary" size="sm" onClick={() => setShowCreate(true)}>
            新建扫描
          </Button>
        )}
      </div>

      {/* Content area */}
      <section className="bg-zinc-900/50 backdrop-blur-md border border-zinc-800/80 rounded-xl p-6">
        <div className="flex justify-between items-center mb-4">
          <h2 className="text-base font-medium text-zinc-200">执行历史</h2>
          {projectId && (
            <div className="flex items-center gap-2">
              {sseStatus === "open" ? (
                <span className="flex items-center gap-1.5 text-xs text-brand-success">
                  <span className="relative flex h-2 w-2">
                    <span className="animate-ping absolute inline-flex h-full w-full rounded-full bg-brand-success opacity-75" />
                    <span className="relative inline-flex rounded-full h-2 w-2 bg-brand-success" />
                  </span>
                  实时连接
                </span>
              ) : shouldPoll ? (
                <span className="flex items-center gap-1.5 text-xs text-accent-yellow">
                  <span className="inline-flex rounded-full h-2 w-2 bg-accent-yellow" />
                  轮询中
                </span>
              ) : (
                <span className="flex items-center gap-1.5 text-xs text-zinc-500">
                  <span className="inline-flex rounded-full h-2 w-2 bg-zinc-500" />
                  未连接
                </span>
              )}
            </div>
          )}
        </div>

        {runsLoading && runs.length === 0 && (
          <div className="space-y-2 py-4">
            {Array.from({ length: 3 }).map((_, i) => (
              <div
                key={i}
                className="h-10 bg-zinc-800/40 animate-pulse rounded-lg"
              />
            ))}
          </div>
        )}

        <div className="divide-y divide-zinc-800/60">
          {runs.map((run) => (
            <div
              key={run.id}
              className={`py-3 px-2 flex items-center justify-between text-sm rounded-lg transition-all ${
                selectedRun === run.id
                  ? "bg-zinc-800/60"
                  : "hover:bg-zinc-800/40"
              }`}
            >
              <button
                onClick={() => loadTasks(run.id)}
                className="flex items-center gap-4 flex-1 text-left cursor-pointer"
              >
                <span className="font-medium text-zinc-200">{run.name}</span>
                <span className="text-zinc-500 text-xs font-mono">
                  {run.id.slice(-8)}
                </span>
              </button>
              <div className="flex items-center gap-3">
                <span
                  className={`px-2 py-0.5 rounded text-xs font-medium ${
                    statusColors[run.status] ||
                    "bg-zinc-800/60 text-zinc-400"
                  }`}
                >
                  {statusLabels[run.status] || run.status}
                </span>
                <span className="text-zinc-500 text-xs">
                  {run.created_at
                    ? new Date(run.created_at).toLocaleString("zh-CN")
                    : "—"}
                </span>
                {canCancel(run.status) && (
                  <Button
                    variant="ghost"
                    size="sm"
                    onClick={() => openCancelDialog(run)}
                    title="取消扫描"
                  >
                    <svg
                      className="w-3.5 h-3.5 text-zinc-400 hover:text-brand-danger transition-colors"
                      viewBox="0 0 24 24"
                      fill="none"
                      stroke="currentColor"
                      strokeWidth="2"
                      strokeLinecap="round"
                      strokeLinejoin="round"
                    >
                      <line x1="18" y1="6" x2="6" y2="18" />
                      <line x1="6" y1="6" x2="18" y2="18" />
                    </svg>
                  </Button>
                )}
              </div>
            </div>
          ))}

          {runs.length === 0 && !runsLoading && !projectId && (
            <EmptyState
              title="请先选择一个项目"
              description="选择一个项目后查看扫描任务"
              actionLabel="前往项目列表"
              onAction={() => navigate("/projects")}
            />
          )}

          {runs.length === 0 && !runsLoading && projectId && (
            <>
              <EmptyState
                title="暂无扫描任务"
                description="按照以下步骤开始你的第一次扫描"
              />
              <div className="flex items-center justify-center gap-2 text-sm text-text-tertiary -mt-4 pb-4">
                <button
                  onClick={() => navigate("/projects")}
                  className="flex items-center gap-1.5 px-3 py-1.5 rounded-lg bg-white/[0.04] border border-white/[0.06] hover:bg-white/[0.08] transition-colors"
                >
                  <span className="w-5 h-5 rounded-full bg-brand-primary/20 text-brand-primary text-xs flex items-center justify-center font-medium">
                    1
                  </span>
                  创建项目
                </button>
                <span className="text-zinc-600">→</span>
                <button
                  onClick={() => navigate("/targets")}
                  className="flex items-center gap-1.5 px-3 py-1.5 rounded-lg bg-white/[0.04] border border-white/[0.06] hover:bg-white/[0.08] transition-colors"
                >
                  <span className="w-5 h-5 rounded-full bg-brand-primary/20 text-brand-primary text-xs flex items-center justify-center font-medium">
                    2
                  </span>
                  导入目标
                </button>
                <span className="text-zinc-600">→</span>
                <button
                  onClick={() => setShowCreate(true)}
                  className="flex items-center gap-1.5 px-3 py-1.5 rounded-lg bg-brand-primary/10 border border-brand-primary/20 text-brand-primary hover:bg-brand-primary/20 transition-colors"
                >
                  <span className="w-5 h-5 rounded-full bg-brand-primary/20 text-brand-primary text-xs flex items-center justify-center font-medium">
                    3
                  </span>
                  启动扫描
                </button>
              </div>
            </>
          )}
        </div>

        {runsError && (
          <div className="mt-4 p-3 rounded-lg bg-brand-danger/10 border border-brand-danger/20 text-brand-danger text-sm">
            加载失败：{runsError}
          </div>
        )}
      </section>

      {selectedRun && (
        <section className="bg-zinc-900/50 backdrop-blur-md border border-zinc-800/80 rounded-xl p-6">
          <div className="flex justify-between items-center mb-4">
            <h2 className="text-base font-medium text-zinc-200">任务详情</h2>
            {tasksLoading && (
              <span className="text-zinc-500 text-sm">加载中...</span>
            )}
          </div>
          <div className="divide-y divide-zinc-800/60">
            {tasks.map((task) => (
              <div
                key={task.id}
                className="py-3 px-2 flex items-center justify-between text-sm"
              >
                <div className="flex items-center gap-4">
                  <span className="font-medium text-zinc-200 w-20">
                    {task.tool}
                  </span>
                  <span className="font-mono text-zinc-500 text-xs">
                    {task.id.slice(-8)}
                  </span>
                </div>
                <span
                  className={`px-2 py-0.5 rounded text-xs font-medium ${
                    taskStatusColors[task.status] ||
                    "bg-zinc-800/60 text-zinc-400"
                  }`}
                >
                  {task.status}
                </span>
              </div>
            ))}
            {tasks.length === 0 && !tasksLoading && (
              <div className="py-8 text-center text-zinc-500">暂无任务</div>
            )}
          </div>
        </section>
      )}

      {/* Create Run Modal */}
      {showCreate && (
        <CreateRunModal
          templates={templates}
          onCreate={handleCreate}
          onClose={() => setShowCreate(false)}
          creating={creating}
        />
      )}

      {/* Cancel Confirm Dialog */}
      <ConfirmDialog
        open={cancelDialogOpen}
        onClose={closeCancelDialog}
        onConfirm={handleCancelRun}
        title="确认取消扫描？"
        description={
          cancelTargetRun
            ? `即将取消扫描「${cancelTargetRun.name}」，此操作不可撤销。`
            : ""
        }
        confirmText="确认取消"
        cancelText="返回"
        variant="danger"
        loading={cancelling}
      />
    </div>
  );
}

function CreateRunModal({
  templates,
  onCreate,
  onClose,
  creating,
}: {
  templates: ToolTemplate[];
  onCreate: (templateId: string, name: string) => void;
  onClose: () => void;
  creating: boolean;
}) {
  const [selectedTemplate, setSelectedTemplate] = useState("");
  const [name, setName] = useState("");

  return (
    <div
      className="fixed inset-0 bg-black/50 backdrop-blur-sm flex items-center justify-center z-50"
      onClick={onClose}
    >
      <div
        className="bg-zinc-900 border border-zinc-800 rounded-xl p-6 w-full max-w-md space-y-4"
        onClick={(e) => e.stopPropagation()}
      >
        <h2 className="text-lg font-semibold text-zinc-100">新建扫描</h2>
        <div className="space-y-3">
          <div>
            <label className="text-sm text-zinc-400 block mb-1">扫描名称</label>
            <input
              type="text"
              value={name}
              onChange={(e) => setName(e.target.value)}
              placeholder="例如：外网初筛"
              className="w-full bg-zinc-800 border border-zinc-700 rounded-lg px-3 py-2 text-sm text-zinc-100 focus:outline-none focus:border-brand-primary/50"
            />
          </div>
          <div>
            <label className="text-sm text-zinc-400 block mb-1">
              工具模板
            </label>
            <div className="space-y-2 max-h-64 overflow-y-auto">
              {templates.map((t) => (
                <div
                  key={t.id}
                  onClick={() => setSelectedTemplate(t.id)}
                  className={`p-3 rounded-lg border cursor-pointer transition-all ${
                    selectedTemplate === t.id
                      ? "border-brand-primary bg-brand-primary/10"
                      : "border-zinc-700 hover:border-zinc-600"
                  }`}
                >
                  <div className="text-sm font-medium text-zinc-200">
                    {t.name}
                  </div>
                  <div className="text-xs text-zinc-500 mt-0.5">
                    {t.description}
                  </div>
                </div>
              ))}
              {templates.length === 0 && (
                <div className="text-sm text-zinc-500 py-2">
                  暂无可用模板
                </div>
              )}
            </div>
          </div>
        </div>
        <div className="flex gap-3 pt-2">
          <button
            onClick={onClose}
            disabled={creating}
            className="flex-1 bg-zinc-800 text-zinc-300 text-sm py-2 rounded-lg hover:bg-zinc-700 transition-colors disabled:opacity-50"
          >
            取消
          </button>
          <button
            onClick={() => onCreate(selectedTemplate, name || "未命名扫描")}
            disabled={!selectedTemplate || creating}
            className="flex-1 bg-brand-primary text-white text-sm py-2 rounded-lg hover:opacity-90 transition-opacity disabled:opacity-50"
          >
            {creating ? "创建中..." : "开始扫描"}
          </button>
        </div>
      </div>
    </div>
  );
}
