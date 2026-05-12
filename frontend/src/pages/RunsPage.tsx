import { useState, useCallback, useRef, useEffect } from "react";
import { api, PAGE_ALL } from "../lib/api";
import { useStore } from "../lib/store";
import { useResource, useSSE, usePolling } from "../hooks";
import {
  EmptyState,
  useProjectId,
  ConfirmDialog,
  Button,
  ScanModal,
  Card,
  CardHeader,
  CardTitle,
  CardDescription,
  CardContent,
  Badge,
  SkeletonList
} from "../components";
import { useToast } from "../components/Toast";
import { getApiBase } from "../lib/config";
import type { ScanTask, PipelineRun, PipelineRunStage, PipelineConfig, Report } from "../lib/api";
import type { ScanMode } from "../components/ScanModal";
import {
  Play,
  Activity,
  Clock,
  XCircle,
  CheckCircle2,
  AlertCircle,
  Terminal,
  History,
  Zap,
  Loader2,
  ArrowRight,
  FileText,
  Download,
  ChevronDown,
  Trash2,
} from "lucide-react";
import { cn } from "../lib/utils";

const modeVariants: Record<string, any> = {
  quick: "warning",
  standard: "info",
  deep: "purple",
  custom: "secondary",
};

const canCancel = (status: string) =>
  status === "pending" || status === "running";

export default function RunsPage() {
  const projectId = useProjectId();
  const toast = useToast();
  const runs = useStore((state) => state.runs) ?? [];
  const setRuns = useStore((state) => state.setRuns);
  const setRunsError = useStore((state) => state.setRunsError);
  const targets = useStore((state) => state.targets) ?? [];
  const setTargets = useStore((state) => state.setTargets);
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
  const [reports, setReports] = useState<Map<string, Report>>(new Map());
  const [generatingReports, setGeneratingReports] = useState<Set<string>>(new Set());

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

  const {
    loading: runsLoading,
    reload: loadRuns,
  } = useResource(
    async (signal) => {
      if (!projectId) return;
      const data = await api.listScanRuns(projectId, PAGE_ALL, signal);
      setRuns(data.data ?? []);
      setRunsError(null);
      clearToastError();
    },
    [projectId],
    undefined
  );

  const {
    loading: targetsLoading,
  } = useResource(
    async (signal) => {
      if (!projectId) return;
      const data = await api.listTargets(projectId, PAGE_ALL, signal);
      setTargets(data.data ?? []);
    },
    [projectId],
    undefined
  );

  const hasTargets = targets.length > 0;
  const canStartScan = hasTargets && !targetsLoading;

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
    } finally {
      setTasksLoading(false);
      setStagesLoading(false);
    }
  };

  const refreshRunDetails = useCallback(
    async (runId: string, signal?: AbortSignal) => {
      try {
        const [taskData, stageData] = await Promise.all([
          api.getRunTasks(runId, signal).catch(() => null),
          projectId
            ? api.listPipelineRunStages(projectId, runId, signal).catch(() => null)
            : Promise.resolve(null),
        ]);
        if (taskData) setTasks(taskData);
        if (stageData) setStages(stageData.stages ?? []);
      } catch (err) {
        if (err instanceof DOMException && err.name === "AbortError") return;
      }
    },
    [projectId]
  );

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
      if (msg.event === "report_progress") {
        const reportMsg = msg as { report_id: string; run_id: string; status: string; title?: string };
        if (reportMsg.status === "complete" || reportMsg.status === "failed") {
          setGeneratingReports((prev) => { const next = new Set(prev); next.delete(reportMsg.run_id); return next; });
          api.getReport(reportMsg.report_id).then((r) => {
            setReports((prev) => new Map(prev).set(reportMsg.run_id, r));
          }).catch(() => {});
          if (reportMsg.status === "complete") {
            toast("报告生成完成", "success");
          } else {
            toast("报告生成失败", "error");
          }
        }
      }
    },
  });

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

  const currentRun = selectedRun ? runs.find((r) => r.id === selectedRun) : null;
  const isCurrentRunActive =
    currentRun?.status === "running" || currentRun?.status === "pending";

  usePolling(
    async () => {
      if (!selectedRun) return;
      await refreshRunDetails(selectedRun);
    },
    {
      interval: 3000,
      enabled: !!selectedRun && isCurrentRunActive,
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
    } finally {
      setCreating(false);
    }
  };

  const openCancelDialog = (run: PipelineRun) => {
    setCancelTargetRun(run);
    setCancelDialogOpen(true);
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
    } finally {
      setCancelling(false);
      setCancelDialogOpen(false);
      setCancelTargetRun(null);
    }
  };

  const handleGenerateReport = async (runId: string) => {
    if (generatingReports.has(runId)) return;
    setGeneratingReports((prev) => new Set(prev).add(runId));
    try {
      const rpt = await api.createReport(runId);
      if (rpt.status === "complete") {
        setReports((prev) => new Map(prev).set(runId, rpt));
        setGeneratingReports((prev) => { const next = new Set(prev); next.delete(runId); return next; });
        toast("报告已生成", "success");
      } else if (rpt.status === "failed") {
        setGeneratingReports((prev) => { const next = new Set(prev); next.delete(runId); return next; });
        toast(rpt.error_message || "报告生成失败", "error");
      }
      // If "generating", wait for SSE callback.
    } catch (err) {
      setGeneratingReports((prev) => { const next = new Set(prev); next.delete(runId); return next; });
      const msg = err instanceof Error ? err.message : "请求生成报告失败";
      toast(msg, "error");
    }
  };

  const handleDeleteReport = async (runId: string) => {
    const rpt = reports.get(runId);
    if (!rpt) return;
    try {
      await api.deleteReport(rpt.id);
      setReports((prev) => { const next = new Map(prev); next.delete(runId); return next; });
      toast("报告已删除", "success");
    } catch (err) {
      const msg = err instanceof Error ? err.message : "删除报告失败";
      toast(msg, "error");
    }
  };

  return (
    <div className="space-y-8 animate-in fade-in duration-500">
      <div className="flex items-start justify-between">
        <div>
          <div className="flex items-center gap-2 text-primary font-bold text-xs uppercase tracking-widest mb-1.5">
            <Zap className="h-3.5 w-3.5 fill-current" />
            Step 3: Scan Engine
          </div>
          <h1 className="text-3xl font-bold tracking-tight text-foreground">扫描执行</h1>
          <p className="text-muted-foreground mt-1">启动漏扫流水线，实时观察任务进度与各阶段状态。</p>
        </div>
        <div className="flex items-center gap-3">
            {projectId && (
                <div className="flex items-center gap-2 px-3 py-1.5 rounded-full bg-muted/50 border border-border text-xs font-medium">
                    {isLive ? (
                        <>
                            <span className="relative flex h-2 w-2">
                                <span className="animate-ping absolute inline-flex h-full w-full rounded-full bg-brand-success opacity-75" />
                                <span className="relative inline-flex rounded-full h-2 w-2 bg-brand-success" />
                            </span>
                            <span className="text-brand-success">SSE 实时连接</span>
                        </>
                    ) : (
                        <>
                            <span className="h-2 w-2 rounded-full bg-muted-foreground/30" />
                            <span className="text-muted-foreground">轮询模式</span>
                        </>
                    )}
                </div>
            )}
            <div className="flex items-center gap-3">
              {!hasTargets && !targetsLoading && (
                <div className="flex items-center gap-1.5 text-[11px] text-destructive/80 bg-destructive/5 border border-destructive/10 px-2.5 py-1.5 rounded-lg">
                  <AlertCircle className="h-3 w-3 shrink-0" />
                  <span>未添加扫描目标</span>
                </div>
              )}
              <Button
                variant="primary"
                onClick={() => setShowScanModal(true)}
                disabled={!canStartScan}
                title={!hasTargets ? "请先前往「目标管理」添加扫描目标" : ""}
              >
                <Play className="mr-2 h-4 w-4 fill-current" />
                启动扫描
              </Button>
            </div>
        </div>
      </div>

      <div className="grid gap-8 lg:grid-cols-[1fr_400px]">
        <section className="space-y-4">
          <div className="flex items-center justify-between">
            <h2 className="text-xl font-semibold tracking-tight flex items-center gap-2">
                <History className="h-5 w-5 text-muted-foreground" />
                执行历史
            </h2>
          </div>

          <div className="space-y-3">
            {runsLoading && runs.length === 0 ? (
                <SkeletonList count={5} />
            ) : runs.length === 0 ? (
                <Card className="border-dashed p-12 text-center">
                    <EmptyState
                        title="暂无扫描记录"
                        description="请在右上方点击「启动扫描」开始你的第一次任务。"
                    />
                </Card>
            ) : (
                runs.map((run) => (
                    <Card 
                        key={run.id} 
                        className={cn(
                            "group cursor-pointer transition-all hover:border-primary/50",
                            selectedRun === run.id && "ring-1 ring-primary border-primary shadow-sm bg-primary/[0.02]"
                        )}
                        onClick={() => loadRunDetails(run.id)}
                    >
                        <CardContent className="p-4 flex items-center justify-between gap-4">
                            <div className="flex items-center gap-4 flex-1">
                                <div className={cn(
                                    "h-10 w-10 rounded-full flex items-center justify-center shrink-0 border",
                                    run.status === 'running' ? 'bg-primary/10 border-primary/20 text-primary animate-pulse' :
                                    run.status === 'completed' ? 'bg-brand-success/10 border-brand-success/20 text-brand-success' :
                                    run.status === 'failed' ? 'bg-destructive/10 border-destructive/20 text-destructive' :
                                    'bg-muted border-border text-muted-foreground'
                                )}>
                                    {run.status === 'running' ? <Activity className="h-5 w-5" /> : 
                                     run.status === 'completed' ? <CheckCircle2 className="h-5 w-5" /> :
                                     run.status === 'failed' ? <XCircle className="h-5 w-5" /> :
                                     <Clock className="h-5 w-5" />
                                    }
                                </div>
                                <div className="min-w-0">
                                    <div className="flex items-center gap-2">
                                        <span className="font-bold text-foreground truncate max-w-[200px]">
                                            {run.id.slice(-8).toUpperCase()}
                                        </span>
                                        <Badge variant={modeVariants[run.mode] || 'secondary'} className="h-5 px-1.5 text-[10px] uppercase">
                                            {run.mode}
                                        </Badge>
                                    </div>
                                    <div className="flex items-center gap-2 mt-1 text-xs text-muted-foreground">
                                        <span>{new Date(run.created_at).toLocaleString()}</span>
                                        {run.stage && (
                                            <>
                                                <span className="text-border">|</span>
                                                <span className="font-medium text-primary uppercase text-[10px] tracking-tight">{run.stage}</span>
                                            </>
                                        )}
                                    </div>
                                </div>
                            </div>

                            <div className="flex items-center gap-3">
                                <div className="hidden sm:block text-right">
                                    <div className={cn(
                                        "text-xs font-bold uppercase",
                                        run.status === 'running' ? 'text-primary' :
                                        run.status === 'completed' ? 'text-brand-success' :
                                        run.status === 'failed' ? 'text-destructive' :
                                        'text-muted-foreground'
                                    )}>
                                        {run.status}
                                    </div>
                                </div>
                                {(run.status === 'completed' || run.status === 'failed') && (
                                    <ReportButton
                                        runId={run.id}
                                        report={reports.get(run.id)}
                                        generating={generatingReports.has(run.id)}
                                        onGenerate={() => handleGenerateReport(run.id)}
                                        onDownload={(reportId) => api.downloadReport(reportId)}
                                        onDelete={() => handleDeleteReport(run.id)}
                                    />
                                )}
                                {canCancel(run.status) && (
                                    <Button
                                        variant="ghost"
                                        size="sm"
                                        className="h-8 w-8 p-0 text-muted-foreground hover:text-destructive hover:bg-destructive/10"
                                        onClick={(e) => {
                                            e.stopPropagation();
                                            openCancelDialog(run);
                                        }}
                                    >
                                        <XCircle className="h-4 w-4" />
                                    </Button>
                                )}
                                <ArrowRight className="h-4 w-4 text-muted-foreground opacity-50 group-hover:opacity-100 transition-all" />
                            </div>
                        </CardContent>
                    </Card>
                ))
            )}
          </div>
        </section>

        <aside className="space-y-6">
            <h2 className="text-xl font-semibold tracking-tight flex items-center gap-2">
                <Terminal className="h-5 w-5 text-muted-foreground" />
                流水线详情
            </h2>

            {!selectedRun ? (
                <div className="rounded-xl border border-dashed p-10 text-center bg-muted/30">
                    <p className="text-sm text-muted-foreground italic">选择左侧扫描记录查看详细进度</p>
                </div>
            ) : (
                <div className="space-y-6">
                    <Card className="overflow-hidden">
                        <CardHeader className="bg-muted/30 pb-3">
                            <CardTitle className="text-sm">阶段进度报告</CardTitle>
                            <CardDescription className="text-xs">Pipeline Execution Stages</CardDescription>
                        </CardHeader>
                        <CardContent className="p-0">
                            {stagesLoading || tasksLoading ? (
                                <div className="p-8 text-center flex flex-col items-center gap-2">
                                    <Loader2 className="h-6 w-6 animate-spin text-primary" />
                                    <span className="text-xs text-muted-foreground font-mono">Loading pipeline stages...</span>
                                </div>
                            ) : stages.length === 0 ? (
                                <div className="p-8 text-center text-xs text-muted-foreground italic">
                                    暂无阶段数据 (可能为手动触发的单个任务)
                                </div>
                            ) : (
                                <div className="divide-y border-t">
                                    {stages.map((s, idx) => {
                                        const stageTasks = tasksInStage(s, tasks);
                                        return (
                                        <div key={s.id} className="group">
                                            <div className="relative flex items-center gap-4 p-4 hover:bg-muted/30 transition-colors">
                                                <div className="flex flex-col items-center relative h-full">
                                                    <div className={cn(
                                                        "z-10 h-6 w-6 rounded-full border-2 bg-background flex items-center justify-center shrink-0",
                                                        s.status === 'completed' ? 'border-brand-success text-brand-success' :
                                                        s.status === 'running' ? 'border-primary text-primary animate-pulse' :
                                                        s.status === 'failed' ? 'border-destructive text-destructive' :
                                                        'border-border text-muted-foreground'
                                                    )}>
                                                        {s.status === 'completed' ? <CheckCircle2 className="h-3 w-3" /> :
                                                         s.status === 'failed' ? <AlertCircle className="h-3 w-3" /> :
                                                         <span className="text-[9px] font-bold">{idx + 1}</span>
                                                        }
                                                    </div>
                                                    {idx < stages.length - 1 && (
                                                        <div className="absolute top-6 bottom-[-16px] w-[1px] bg-border group-hover:bg-primary/20 transition-colors" />
                                                    )}
                                                </div>
                                                <div className="flex-1 min-w-0">
                                                    <div className="flex items-center justify-between">
                                                        <span className="text-sm font-semibold uppercase tracking-tight truncate pr-2">
                                                            {STAGE_LABELS[s.stage] || s.stage}
                                                        </span>
                                                        <span className="text-[10px] font-mono text-muted-foreground">
                                                            {formatStageDuration(s)}
                                                        </span>
                                                    </div>
                                                    <div className={cn(
                                                        "text-[10px] mt-0.5 font-medium truncate",
                                                        s.status === 'completed' ? 'text-brand-success' :
                                                        s.status === 'running' ? 'text-primary' :
                                                        s.status === 'failed' ? 'text-destructive' :
                                                        'text-muted-foreground'
                                                    )}>
                                                        {s.status.toUpperCase()}
                                                        {s.error && <span className="ml-2 font-mono text-destructive opacity-80">— {s.error}</span>}
                                                    </div>
                                                </div>
                                            </div>
                                            {stageTasks.length > 0 && (
                                                <div className="pl-14 pr-4 pb-3 -mt-1 space-y-1.5">
                                                    {stageTasks.map((task) => (
                                                        <div key={task.id} className="flex items-center justify-between gap-3 text-[11px] py-1 border-l border-border/40 pl-3">
                                                            <div className="flex items-center gap-2 min-w-0">
                                                                <span className="px-1.5 py-0.5 rounded bg-muted font-mono font-bold text-muted-foreground text-[10px]">
                                                                    {task.tool}
                                                                </span>
                                                                <span className="font-mono text-muted-foreground opacity-50 truncate">#{task.id.slice(-6)}</span>
                                                                <span className="font-mono text-muted-foreground/70 text-[10px]">
                                                                    {formatTaskDuration(task)}
                                                                </span>
                                                            </div>
                                                            <Badge variant={
                                                                task.status === 'completed' ? 'success' :
                                                                task.status === 'failed' ? 'destructive' :
                                                                task.status === 'running' ? 'info' :
                                                                'secondary'
                                                            } className="h-4 px-1 text-[9px] shrink-0">
                                                                {task.status}
                                                            </Badge>
                                                        </div>
                                                    ))}
                                                </div>
                                            )}
                                        </div>
                                        );
                                    })}
                                </div>
                            )}
                        </CardContent>
                    </Card>
                </div>
            )}
        </aside>
      </div>

      <ScanModal
        open={showScanModal}
        onClose={() => setShowScanModal(false)}
        onStart={handleStartScan}
        loading={creating}
      />

      <ConfirmDialog
        open={cancelDialogOpen}
        onClose={() => setCancelDialogOpen(false)}
        onConfirm={handleCancelRun}
        title="确认终止扫描？"
        description={
            cancelTargetRun
              ? `即将向扫描任务「${cancelTargetRun.id.slice(-8).toUpperCase()}」发送 SIGINT 信号。已产生的发现结果将被保留，但后续阶段将不再执行。`
              : ""
          }
        confirmText="强制终止"
        cancelText="稍后处理"
        variant="danger"
        loading={cancelling}
      />
    </div>
  );
}

// --- Report Button Component ---

function ReportButton({
  runId: _runId,
  report,
  generating,
  onGenerate,
  onDownload,
  onDelete,
}: {
  runId: string;
  report?: Report;
  generating: boolean;
  onGenerate: () => void;
  onDownload: (reportId: string) => void;
  onDelete: () => void;
}) {
  const [menuOpen, setMenuOpen] = useState(false);
  const menuRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    const handler = (e: MouseEvent) => {
      if (menuRef.current && !menuRef.current.contains(e.target as Node)) {
        setMenuOpen(false);
      }
    };
    document.addEventListener("mousedown", handler);
    return () => document.removeEventListener("mousedown", handler);
  }, []);

  if (generating) {
    return (
      <Button variant="ghost" size="sm" disabled className="h-8 gap-1.5 text-xs">
        <Loader2 className="h-3.5 w-3.5 animate-spin" />
        生成中
      </Button>
    );
  }

  if (report && report.status === "complete") {
    return (
      <div className="relative" ref={menuRef}>
        <div className="flex">
          <Button
            variant="ghost"
            size="sm"
            className="h-8 gap-1.5 text-xs rounded-r-none border-r border-border"
            onClick={(e) => {
              e.stopPropagation();
              onDownload(report.id);
            }}
          >
            <Download className="h-3.5 w-3.5" />
            下载
          </Button>
          <Button
            variant="ghost"
            size="sm"
            className="h-8 w-7 rounded-l-none px-0"
            onClick={(e) => {
              e.stopPropagation();
              setMenuOpen(!menuOpen);
            }}
          >
            <ChevronDown className="h-3.5 w-3.5" />
          </Button>
        </div>
        {menuOpen && (
          <div className="absolute right-0 top-full mt-1 z-50 w-40 rounded-md border border-border bg-popover shadow-md">
            <button
              className="flex w-full items-center gap-2 px-3 py-2 text-xs hover:bg-accent transition-colors"
              onClick={(e) => {
                e.stopPropagation();
                onDownload(report.id);
                setMenuOpen(false);
              }}
            >
              <FileText className="h-3.5 w-3.5" />
              HTML 报告
            </button>
            <button
              className="flex w-full items-center gap-2 px-3 py-2 text-xs text-destructive hover:bg-destructive/10 transition-colors"
              onClick={(e) => {
                e.stopPropagation();
                onDelete();
                setMenuOpen(false);
              }}
            >
              <Trash2 className="h-3.5 w-3.5" />
              删除报告
            </button>
          </div>
        )}
      </div>
    );
  }

  if (report && report.status === "failed") {
    return (
      <Button
        variant="ghost"
        size="sm"
        className="h-8 gap-1.5 text-xs text-destructive hover:bg-destructive/10"
        onClick={(e) => {
          e.stopPropagation();
          onGenerate();
        }}
      >
        <AlertCircle className="h-3.5 w-3.5" />
        重试
      </Button>
    );
  }

  return (
    <Button
      variant="ghost"
      size="sm"
      className="h-8 gap-1.5 text-xs"
      onClick={(e) => {
        e.stopPropagation();
        onGenerate();
      }}
    >
      <FileText className="h-3.5 w-3.5" />
      生成报告
    </Button>
  );
}

function tasksInStage(stage: PipelineRunStage, allTasks: ScanTask[]): ScanTask[] {
  if (!stage.started_at) return [];
  const stageStart = new Date(stage.started_at).getTime();
  const stageEnd = stage.completed_at ? new Date(stage.completed_at).getTime() : Infinity;
  return allTasks.filter((t) => {
    if (!t.started_at) return false;
    const ts = new Date(t.started_at).getTime();
    return ts >= stageStart && ts <= stageEnd;
  });
}

function formatDurationMs(ms: number): string {
  if (ms < 0) return "";
  const s = ms / 1000;
  if (s < 60) return `${s.toFixed(1)}s`;
  const m = Math.floor(s / 60);
  const rs = Math.round(s % 60);
  return `${m}m${rs}s`;
}

function formatStageDuration(s: PipelineRunStage): string {
  if (!s.started_at) return "";
  const start = new Date(s.started_at).getTime();
  const end = s.completed_at ? new Date(s.completed_at).getTime() : Date.now();
  return formatDurationMs(end - start);
}

function formatTaskDuration(t: ScanTask): string {
  if (!t.started_at) return "";
  const start = new Date(t.started_at).getTime();
  const end = t.finished_at ? new Date(t.finished_at).getTime() : Date.now();
  return formatDurationMs(end - start);
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
