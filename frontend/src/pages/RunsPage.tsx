import { useEffect, useState, useCallback, useRef } from "react";
import { useNavigate } from "react-router-dom";
import { api, PAGE_ALL } from "../lib/api";
import { useStore } from "../lib/store";
import { EmptyState, useProjectId, ConfirmDialog, Button, ScanModal } from "../components";
import { useToast } from "../components/Toast";
import { useSSE, usePolling } from "../hooks";
import { getApiBase } from "../lib/config";
import type { ScanTask, PipelineRun, PipelineRunStage, PipelineConfig } from "../lib/api";
import type { ScanMode } from "../components/ScanModal";

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

const modeLabels: Record<string, string> = {
  quick: "快速",
  standard: "标准",
  deep: "深度",
  custom: "自定义",
};

const modeColors: Record<string, string> = {
  quick: "bg-accent-yellow/10 text-accent-yellow",
  standard: "bg-brand-primary/10 text-brand-primary",
  deep: "bg-accent-purple/10 text-accent-purple",
  custom: "bg-white/[0.06] text-text-secondary",
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
  const [selectedRun, setSelectedRun] = useState<string | null>(null);
  const [tasks, setTasks] = useState<ScanTask[]>([]);
  const [tasksLoading, setTasksLoading] = useState(false);
  const [stages, setStages] = useState<PipelineRunStage[]>([]);
  const [stagesLoading, setStagesLoading] = useState(false);
  const [showScanModal, setShowScanModal] = useState(false);
  const [creating, setCreating] = useState(false);
  const [cancelling, setCancelling] = useState(false);
  const [cancelDialogOpen, setCancelDialogOpen] = useState(false);
  const [cancelTargetRun, setCancelTargetRun] = useState<PipelineRun | null>(null);

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
        const data = await api.listScanRuns(projectId, PAGE_ALL, signal);
        setRuns(data.data ?? []);
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

  const loadRunDetails = async (runId: string, signal?: AbortSignal) => {
    setSelectedRun(runId);
    setTasksLoading(true);
    setStagesLoading(true);
    try {
      const [taskData, stageData] = await Promise.all([
        api.getRunTasks(runId, signal).catch(() => [] as ScanTask[]),
        projectId
          ? api.listPipelineRunStages(projectId, runId, signal).catch(() => ({ stages: [] as PipelineRunStage[] }))
          : Promise.resolve({ stages: [] as PipelineRunStage[] }),
      ]);
      setTasks(taskData ?? []);
      setStages(stageData.stages ?? []);
    } catch (err) {
      if (err instanceof DOMException && err.name === "AbortError") return;
      const msg = err instanceof Error ? err.message : "加载任务详情失败";
      toast(msg, "error");
      console.error(err);
    } finally {
      setTasksLoading(false);
      setStagesLoading(false);
    }
  };

  useEffect(() => {
    if (!projectId) return;
    const ctrl = new AbortController();
    loadRuns(ctrl.signal);
    return () => ctrl.abort();
  }, [projectId, loadRuns]);

  // SSE for real-time updates
  const sseUrl = projectId ? `${getApiBase()}/projects/${projectId}/events` : `${getApiBase()}/events`;
  const { status: sseStatus } = useSSE(sseUrl, {
    onMessage: (raw) => {
      const msg = raw as {
        event?: string;
        run_id?: string;
        stage?: string;
        status?: string;
        error?: string;
      };
      if (msg.event === "pipeline_stage_change" && msg.run_id === selectedRun) {
        setStages((prev) => {
          const idx = prev.findIndex((s) => s.stage === msg.stage);
          if (idx >= 0) {
            const next = [...prev];
            next[idx] = { ...next[idx], status: msg.status ?? "running", error: msg.error };
            return next;
          }
          return prev;
        });
        if (msg.status === "failed" && msg.error) {
          toast(`阶段 ${msg.stage} 失败: ${msg.error}`, "error");
        }
      }
      if (msg.event === "pipeline_complete") {
        loadRuns();
        if (selectedRun && msg.run_id === selectedRun) {
          loadRunDetails(selectedRun);
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
        const data = await api.listScanRuns(projectId!, PAGE_ALL);
        setRuns(data.data ?? []);
        setRunsError(null);
        clearToastError();
        return data.data ?? [];
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

  const handleStartScan = async (mode: ScanMode, config: PipelineConfig) => {
    if (!projectId || creating) return;
    setCreating(true);
    try {
      await api.createScan(projectId, { mode, config });
      toast("扫描任务已启动", "success");
      setShowScanModal(false);
      await loadRuns();
    } catch (err) {
      const msg = err instanceof Error ? err.message : "启动扫描失败";
      toast(msg, "error");
      console.error(err);
    } finally {
      setCreating(false);
    }
  };

  const openCancelDialog = (run: PipelineRun) => {
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
        await loadRunDetails(cancelTargetRun.id);
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
    <div className="page-shell space-y-6">
      <div className="page-header">
        <div>
          <div className="page-eyebrow">Step 3</div>
          <h1 className="page-title">扫描执行</h1>
          <p className="page-subtitle">查看运行历史、实时连接状态、阶段进度和任务明细。</p>
        </div>
        {projectId && (
          <Button variant="primary" size="sm" onClick={() => setShowScanModal(true)}>
            新建扫描
          </Button>
        )}
      </div>

      <section className="panel p-6">
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
              <div key={i} className="h-10 bg-white/[0.02] animate-pulse rounded-lg" />
            ))}
          </div>
        )}

        <div className="divide-y divide-glass-border-light">
          {runs.map((run) => (
            <div
              key={run.id}
              className={`py-3 px-2 flex items-center justify-between text-sm rounded-lg transition-all ${
                selectedRun === run.id
                  ? "bg-white/[0.04]"
                  : "hover:bg-white/[0.02]"
              }`}
            >
              <button
                onClick={() => loadRunDetails(run.id)}
                className="flex items-center gap-3 flex-1 text-left cursor-pointer"
              >
                <span
                  className={`px-2 py-0.5 rounded text-[11px] font-medium ${
                    modeColors[run.mode] || modeColors.standard
                  }`}
                >
                  {modeLabels[run.mode] || run.mode}
                </span>
                <span className="text-zinc-500 text-xs font-mono">
                  {run.id.slice(-8)}
                </span>
                {run.stage && (
                  <span className="text-xs text-zinc-500 bg-white/[0.04] px-2 py-0.5 rounded">
                    {run.stage}
                  </span>
                )}
              </button>
              <div className="flex items-center gap-3">
                <span
                  className={`px-2 py-0.5 rounded text-xs font-medium ${
                    statusColors[run.status] || "bg-white/[0.04] text-zinc-400"
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
                  onClick={() => setShowScanModal(true)}
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
        <section className="panel p-6">
          <div className="flex justify-between items-center mb-4">
            <h2 className="text-base font-medium text-zinc-200">任务详情</h2>
            {(tasksLoading || stagesLoading) && (
              <span className="text-zinc-500 text-sm">加载中...</span>
            )}
          </div>

          {stages.length > 0 && (
            <div className="mb-4">
              <h3 className="text-xs font-medium text-zinc-400 mb-2">阶段进度</h3>
              <StageProgress stages={stages} />
            </div>
          )}

          <div className="divide-y divide-glass-border-light">
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
                    taskStatusColors[task.status] || "bg-white/[0.04] text-zinc-400"
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

      <ScanModal
        open={showScanModal}
        onClose={() => setShowScanModal(false)}
        onStart={handleStartScan}
        loading={creating}
      />

      <ConfirmDialog
        open={cancelDialogOpen}
        onClose={closeCancelDialog}
        onConfirm={handleCancelRun}
        title="确认取消扫描？"
        description={
          cancelTargetRun
            ? `即将取消扫描「${cancelTargetRun.id.slice(-8)}」，此操作不可撤销。`
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

const STAGE_LABELS: Record<string, string> = {
  classify: "目标分类",
  search: "FOFA 搜索",
  subdomain: "子域名发现",
  resolve: "DNS 解析",
  cdn_filter: "CDN 过滤",
  portscan: "端口扫描",
  fingerprint: "服务指纹",
  httpx: "Web 探活",
  vuln: "漏洞探测",
};

const STAGE_STATUS_COLORS: Record<string, string> = {
  pending: "bg-zinc-700",
  running: "bg-brand-primary animate-pulse",
  completed: "bg-brand-success",
  failed: "bg-brand-danger",
  skipped: "bg-zinc-600",
};

function StageProgress({ stages }: { stages: PipelineRunStage[] }) {
  return (
    <div className="space-y-1.5">
      {stages.map((s) => (
        <div key={s.id} className="flex items-center gap-3 text-sm">
          <div
            className={`w-2 h-2 rounded-full shrink-0 ${
              STAGE_STATUS_COLORS[s.status] || "bg-zinc-600"
            }`}
          />
          <span className="text-zinc-300 w-24 shrink-0">
            {STAGE_LABELS[s.stage] || s.stage}
          </span>
          <span
            className={`text-xs ${
              s.status === "failed"
                ? "text-brand-danger"
                : s.status === "running"
                ? "text-brand-primary"
                : s.status === "completed"
                ? "text-brand-success"
                : "text-zinc-500"
            }`}
          >
            {s.status === "pending" && "待执行"}
            {s.status === "running" && "执行中"}
            {s.status === "completed" && "已完成"}
            {s.status === "failed" && `失败${s.error ? ": " + s.error : ""}`}
            {s.status === "skipped" && "已跳过"}
          </span>
          {s.started_at && (
            <span className="text-zinc-600 text-xs ml-auto">
              {new Date(s.started_at).toLocaleTimeString("zh-CN")}
            </span>
          )}
        </div>
      ))}
    </div>
  );
}
