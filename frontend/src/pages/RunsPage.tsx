import { useState, useCallback, useRef, useEffect } from "react";
import { api, PAGE_ALL } from "../lib/api";
import { useStore } from "../lib/store";
import { useResource, useSSE, usePolling, useTaskLiveOutput } from "../hooks";
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
  SeverityBadge,
  SkeletonList,
  Modal,
  Table,
  TableHeader,
  TableBody,
  TableRow,
  TableHead,
  TableCell,
} from "../components";
import { useToast } from "../components/Toast";
import { getApiBase } from "../lib/config";
import type { ScanTask, PipelineRun, PipelineRunStage, PipelineConfig } from "../lib/api";
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
  Copy,
  ChevronDown,
  ChevronRight,
  Eye,
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

  const [inspectingTask, setInspectingTask] = useState<ScanTask | null>(null);
  const [inspectingLogs, setInspectingLogs] = useState<string>("");
  const [logsLoading, setLogsLoading] = useState(false);
  const [logViewMode, setLogViewMode] = useState<"raw" | "parsed">("parsed");

  // Inline task output expansion state
  const [expandedTaskId, setExpandedTaskId] = useState<string | null>(null);
  const [expandedTaskLogs, setExpandedTaskLogs] = useState<Record<string, string>>({});
  const [expandedTaskLoading, setExpandedTaskLoading] = useState<Record<string, boolean>>({});

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
        setStages((prev) => mergeStageEvent(prev, msg));
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

  // Keep polling for a few cycles after the run finishes so the final batch
  // of task status updates (completed/failed) reaches the UI.  Without this
  // cooldown the polling stops the moment the run flips to "completed" and
  // the last few task cards stay stuck on "running" until the user reloads.
  const [cooldown, setCooldown] = useState(false);
  useEffect(() => {
    if (isCurrentRunActive) {
      setCooldown(false);
    } else if (selectedRun && currentRun && !cooldown) {
      setCooldown(true);
      const t = setTimeout(() => setCooldown(false), 15000);
      return () => clearTimeout(t);
    }
  }, [isCurrentRunActive, selectedRun, currentRun?.status]);

  usePolling(
    async () => {
      if (!selectedRun) return;
      await refreshRunDetails(selectedRun);
    },
    {
      interval: 3000,
      enabled: !!selectedRun && (isCurrentRunActive || cooldown),
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
    if (!cancelTargetRun || cancelling || !projectId) return;
    setCancelling(true);
    try {
      await api.cancelPipelineRun(projectId, cancelTargetRun.id);
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

  const handleViewReport = (_runId: string) => {
    window.location.href = `/reports?projectId=${projectId}`;
  };



  const handleInspectTask = async (task: ScanTask) => {
    setInspectingTask(task);
    setInspectingLogs("");
    setLogViewMode("raw");
    if (task.status === "running" || task.status === "queued") {
      setLogsLoading(false);
      return;
    }
    setLogsLoading(true);
    try {
      const artifacts = await api.listArtifacts(task.id);
      // Sort to prioritize stdout, then stderr, then others
      const stdout = artifacts.find(a => a.type === 'stdout');
      const stderr = artifacts.find(a => a.type === 'stderr');
      
      const targetArtifact = stdout || stderr;
      if (targetArtifact) {
        const content = await api.getArtifactContent(targetArtifact.id);
        setInspectingLogs(content);
      } else {
        setInspectingLogs("(无日志输出)");
      }
    } catch (err) {
      console.error("Failed to load logs:", err);
      setInspectingLogs("加载日志失败: " + (err instanceof Error ? err.message : String(err)));
    } finally {
      setLogsLoading(false);
    }
  };

  // Toggle inline task output expansion (fetch artifact lazily).
  const toggleTaskExpand = async (task: ScanTask) => {
    if (expandedTaskId === task.id) {
      setExpandedTaskId(null);
      return;
    }
    setExpandedTaskId(task.id);
    if (task.status === "running" || task.status === "queued") {
      return;
    }
    if (expandedTaskLogs[task.id] !== undefined) return; // already cached

    setExpandedTaskLoading((prev) => ({ ...prev, [task.id]: true }));
    try {
      const artifacts = await api.listArtifacts(task.id);
      const stdout = artifacts.find(a => a.type === 'stdout');
      const stderr = artifacts.find(a => a.type === 'stderr');
      const targetArtifact = stdout || stderr;
      if (targetArtifact) {
        const content = await api.getArtifactContent(targetArtifact.id);
        // Truncate to first 50 lines for inline preview
        const lines = content.split('\n');
        const preview = lines.slice(0, 50).join('\n');
        setExpandedTaskLogs((prev) => ({ ...prev, [task.id]: preview + (lines.length > 50 ? '\n...' : '') }));
      } else {
        setExpandedTaskLogs((prev) => ({ ...prev, [task.id]: '(无日志输出)' }));
      }
    } catch (err) {
      setExpandedTaskLogs((prev) => ({ ...prev, [task.id]: '加载失败: ' + (err instanceof Error ? err.message : String(err)) }));
    } finally {
      setExpandedTaskLoading((prev) => ({ ...prev, [task.id]: false }));
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
                                    <Button
                                        variant="ghost"
                                        size="sm"
                                        className="h-8 gap-1.5 text-xs"
                                        onClick={(e) => {
                                            e.stopPropagation();
                                            handleViewReport(run.id);
                                        }}
                                    >
                                        <FileText className="h-3.5 w-3.5" />
                                        查看报告
                                    </Button>
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
                                                        "text-[10px] mt-0.5 font-medium",
                                                        !s.error && "truncate",
                                                        s.status === 'completed' ? 'text-brand-success' :
                                                        s.status === 'running' ? 'text-primary' :
                                                        s.status === 'failed' ? 'text-destructive' :
                                                        'text-muted-foreground'
                                                    )}>
                                                        {s.status.toUpperCase()}
                                                    </div>
                                                    {s.error && (
                                                        <p className="text-[10px] mt-1 font-mono text-destructive/90 whitespace-pre-wrap break-all">
                                                            {s.error}
                                                        </p>
                                                    )}
                                                    {stageTasks.length === 0 && s.status !== 'pending' && !s.error && (
                                                        <p className="text-[10px] mt-1 text-muted-foreground">
                                                            此阶段无 cdncheck 子任务（可能尚未执行或 run 无 stage 记录）
                                                        </p>
                                                    )}
                                                </div>
                                            </div>
                                            {stageTasks.length > 0 && (
                                                <div className="pl-14 pr-4 pb-3 -mt-1 space-y-1.5">
                                                    {stageTasks.map((task) => {
                                                        const isExpanded = expandedTaskId === task.id;
                                                        const taskLive =
                                                          task.status === "running" ||
                                                          task.status === "queued";
                                                        const isLoading = taskLive
                                                          ? expandedLiveLoading
                                                          : expandedTaskLoading[task.id];
                                                        const logs = taskLive
                                                          ? expandedLiveLogs
                                                          : expandedTaskLogs[task.id];
                                                        return (
                                                        <div key={task.id}>
                                                            <div
                                                                className="flex items-center justify-between gap-3 text-[11px] py-1 border-l border-border/40 pl-3 hover:bg-primary/[0.03] cursor-pointer transition-colors group/task"
                                                                onClick={(e) => {
                                                                    e.stopPropagation();
                                                                    toggleTaskExpand(task);
                                                                }}
                                                            >
                                                                <div className="flex items-center gap-2 min-w-0">
                                                                    {isExpanded
                                                                        ? <ChevronDown className="h-3 w-3 text-primary shrink-0" />
                                                                        : <ChevronRight className="h-3 w-3 text-muted-foreground/40 shrink-0 group-hover/task:text-muted-foreground transition-colors" />
                                                                    }
                                                                    <span className="px-1.5 py-0.5 rounded bg-muted font-mono font-bold text-muted-foreground text-[10px] group-hover/task:bg-primary/10 group-hover/task:text-primary transition-colors">
                                                                        {task.tool}
                                                                    </span>
                                                                    <span className="font-mono text-muted-foreground opacity-50 truncate">#{task.id.slice(-6)}</span>
                                                                    <span className="font-mono text-muted-foreground/70 text-[10px]">
                                                                        {formatTaskDuration(task)}
                                                                    </span>
                                                                </div>
                                                                <div className="flex items-center gap-2">
                                                                    <Badge variant={
                                                                        task.status === 'completed' ? 'success' :
                                                                        task.status === 'failed' ? 'destructive' :
                                                                        task.status === 'running' ? 'info' :
                                                                        'secondary'
                                                                    } className="h-4 px-1 text-[9px] shrink-0">
                                                                        {task.status}
                                                                    </Badge>
                                                                    <Terminal className="h-3 w-3 text-muted-foreground/30 group-hover/task:text-primary transition-colors" />
                                                                </div>
                                                            </div>
                                                            {/* Inline output preview */}
                                                            {isExpanded && (
                                                                <div className="ml-6 mt-1 mb-2 border-l border-primary/20 pl-3 animate-in fade-in slide-in-from-top-1 duration-150">
                                                                    {isLoading && (!logs || logs === "") ? (
                                                                        <div className="flex items-center gap-2 py-2">
                                                                            <Loader2 className="h-3 w-3 animate-spin text-primary" />
                                                                            <span className="text-[10px] text-muted-foreground">
                                                                              {taskLive ? "等待工具输出…" : "加载中..."}
                                                                            </span>
                                                                        </div>
                                                                    ) : logs !== undefined ? (
                                                                        <div className="space-y-1.5">
                                                                            {taskLive && (
                                                                              <span className="text-[9px] font-bold uppercase tracking-wider text-brand-success">
                                                                                实时输出
                                                                              </span>
                                                                            )}
                                                                            <pre className="text-[10px] font-mono text-muted-foreground bg-black/20 rounded-md p-2 overflow-x-auto max-h-[200px] overflow-y-auto leading-relaxed whitespace-pre-wrap break-all">
                                                                                {logs || (taskLive ? "(尚无输出，工具可能仍在启动)" : "")}
                                                                            </pre>
                                                                            <button
                                                                                className="flex items-center gap-1 text-[10px] text-primary hover:text-primary/80 font-medium transition-colors"
                                                                                onClick={(e) => {
                                                                                    e.stopPropagation();
                                                                                    handleInspectTask(task);
                                                                                }}
                                                                            >
                                                                                <Eye className="h-3 w-3" />
                                                                                查看完整日志
                                                                            </button>
                                                                        </div>
                                                                    ) : null}
                                                                </div>
                                                            )}
                                                        </div>
                                                    )})}
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
        projectId={projectId ?? undefined}
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

      <TaskDetailsModal
        task={inspectingTask}
        logs={inspectingLive ? inspectingLiveLogs : inspectingLogs}
        loading={inspectingLive ? inspectingLiveLoading : logsLoading}
        live={inspectingLive}
        viewMode={logViewMode}
        onViewModeChange={setLogViewMode}
        onClose={() => {
          setInspectingTask(null);
          setInspectingLogs("");
        }}
      />
    </div>
  );
}

// Mapping of stages to the tools they are allowed to contain.
// This prevents tasks from appearing in multiple stages if they start/end 
// at the exact same time as a stage boundary.
const STAGE_TOOL_ALLOWLIST: Record<string, string[]> = {
  alive: ["nmap"],
  portscan: ["naabu"],
  fingerprint: ["nmap"],
  httpx: ["httpx"],
  vuln: ["nuclei"],
  subdomain: ["subfinder"],
  resolve: ["dnsx"],
  cdn_filter: ["cdncheck"],
  ffuf: ["ffuf"],
  urlfinder: ["urlfinder"],
  httpx_2: ["httpx"],
  vuln_2: ["nuclei"],
  search: ["fofa", "hunter", "quake"],
  passive_cert: ["crt"],
  passive_url: ["gau"],
};

function tasksInStage(stage: PipelineRunStage, allTasks: ScanTask[]): ScanTask[] {
  if (!stage.started_at) return [];
  const stageStart = new Date(stage.started_at).getTime();
  const stageEnd = stage.completed_at ? new Date(stage.completed_at).getTime() : Infinity;
  
  const allowedTools = STAGE_TOOL_ALLOWLIST[stage.stage];

  return allTasks.filter((t) => {
    if (!t.started_at) return false;
    const ts = new Date(t.started_at).getTime();
    
    // Time boundary check
    if (ts < stageStart || ts > stageEnd) return false;
    
    // Strict tool matching if allowlist exists for this stage
    if (allowedTools && !allowedTools.includes(t.tool)) return false;
    
    return true;
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
  alive: "主机存活探测",
  portscan: "端口扫描",
  fingerprint: "服务指纹",
  httpx: "Web 探活",
  vuln: "漏洞探测",
  ffuf: "目录爆破",
  urlfinder: "URL 发现",
  httpx_2: "Web 探活 (2)",
  vuln_2: "漏洞扫描 (2)",
  passive_cert: "证书子域",
  passive_url: "历史 URL",
  crawl: "站点爬虫",
};

// mergeStageEvent reduces an SSE pipeline_stage_change event into the local
// stages array. Existing stage rows get their status flipped; previously
// unseen stages (post-phase urlfinder/ffuf events arrive after the initial
// loadRunDetails snapshot is taken) are appended with a tmp- id that gets
// replaced when pipeline_complete triggers a reload from the server.
//
// Exported so the SSE merge logic is unit-testable without rendering RunsPage.
export interface StageEventMessage {
  run_id?: string;
  stage?: string;
  status?: string;
  error?: string;
}

export function mergeStageEvent(
  prev: PipelineRunStage[],
  msg: StageEventMessage,
): PipelineRunStage[] {
  if (!msg.stage) {
    return prev;
  }
  const idx = prev.findIndex((s) => s.stage === msg.stage);
  if (idx >= 0) {
    const existing = prev[idx];
    const nextStatus = msg.status ?? "running";
    if (existing.status === nextStatus && existing.error === msg.error) {
      return prev;
    }
    const next = [...prev];
    next[idx] = { ...next[idx], status: nextStatus, error: msg.error };
    return next;
  }
  const now = new Date().toISOString();
  return [
    ...prev,
    {
      id: `tmp-${msg.stage}-${Date.now()}`,
      run_id: msg.run_id ?? "",
      stage: msg.stage,
      status: msg.status ?? "running",
      error: msg.error,
      started_at: now,
      created_at: now,
    },
  ];
}

// --- Task Parsed View Component ---

function TaskParsedView({ task, logs }: { task: ScanTask; logs: string }) {
  if (!logs) return (
    <div className="flex flex-col items-center justify-center h-[300px] text-muted-foreground gap-2">
      <Activity className="h-8 w-8 opacity-20" />
      <span className="italic text-sm">NO DATA RECEIVED YET</span>
    </div>
  );

  const lines = logs.trim().split("\n");

  const tryParseJSONL = <T,>(lines: string[]): T[] => {
    const results: T[] = [];
    for (const line of lines) {
      try {
        if (line.trim() && (line.trim().startsWith("{") || line.trim().startsWith("["))) {
          results.push(JSON.parse(line));
        }
      } catch (e) {
        // ignore invalid lines
      }
    }
    return results;
  };

  const TableContainer = ({ children }: { children: React.ReactNode }) => (
    <div className="overflow-hidden rounded-xl border border-border/50 bg-card">
        {children}
    </div>
  );

  switch (task.tool) {
    case "nuclei": {
      const data = tryParseJSONL<any>(lines);
      return (
        <TableContainer>
            <Table>
            <TableHeader className="bg-muted/50">
                <TableRow>
                <TableHead className="w-[200px] text-[10px] uppercase font-bold">TEMPLATE ID</TableHead>
                <TableHead className="text-[10px] uppercase font-bold">TARGET</TableHead>
                <TableHead className="w-[100px] text-[10px] uppercase font-bold text-center">SEVERITY</TableHead>
                <TableHead className="text-[10px] uppercase font-bold text-right pr-6">MATCHER</TableHead>
                </TableRow>
            </TableHeader>
            <TableBody>
                {data.map((item, idx) => (
                <TableRow key={idx} className="hover:bg-muted/30 transition-colors border-border/40">
                    <TableCell className="font-mono text-[11px] text-primary font-bold py-3">{item["template-id"]}</TableCell>
                    <TableCell className="font-mono text-[11px] py-3 break-all">{item["matched-at"] || item["host"]}</TableCell>
                    <TableCell className="text-center py-3">
                    <SeverityBadge severity={item["info"]?.["severity"] || "info"} className="h-5 px-1.5" />
                    </TableCell>
                    <TableCell className="text-right pr-6 py-3">
                        <span className="text-[10px] px-1.5 py-0.5 rounded bg-muted font-mono text-muted-foreground">
                            {item["matcher-name"] || "match"}
                        </span>
                    </TableCell>
                </TableRow>
                ))}
                {data.length === 0 && (
                    <TableRow>
                        <TableCell colSpan={4} className="text-center py-20">
                            <div className="flex flex-col items-center gap-2">
                                <CheckCircle2 className="h-8 w-8 text-brand-success opacity-20" />
                                <span className="text-sm font-medium text-muted-foreground">CLEAN: NO VULNERABILITIES DETECTED</span>
                            </div>
                        </TableCell>
                    </TableRow>
                )}
            </TableBody>
            </Table>
        </TableContainer>
      );
    }

    case "httpx": {
      const data = tryParseJSONL<any>(lines);
      return (
        <TableContainer>
            <Table>
            <TableHeader className="bg-muted/50">
                <TableRow>
                <TableHead className="text-[10px] uppercase font-bold">URL</TableHead>
                <TableHead className="w-[80px] text-[10px] uppercase font-bold text-center">STATUS</TableHead>
                <TableHead className="text-[10px] uppercase font-bold">PAGE TITLE</TableHead>
                <TableHead className="text-[10px] uppercase font-bold pr-6">TECHNOLOGIES</TableHead>
                </TableRow>
            </TableHeader>
            <TableBody>
                {data.map((item, idx) => (
                <TableRow key={idx} className="hover:bg-muted/30 transition-colors border-border/40">
                    <TableCell className="font-mono text-[11px] text-primary py-3 max-w-[300px] truncate">{item["url"]}</TableCell>
                    <TableCell className="text-center py-3">
                    <Badge variant={item["status_code"] >= 200 && item["status_code"] < 300 ? "success" : "secondary"} className="h-5 px-1.5 font-mono text-[10px]">
                        {item["status_code"]}
                    </Badge>
                    </TableCell>
                    <TableCell className="max-w-[200px] truncate text-[11px] py-3 text-muted-foreground font-medium">{item["title"] || "-"}</TableCell>
                    <TableCell className="pr-6 py-3">
                    <div className="flex flex-wrap gap-1">
                        {(item["technologies"] || []).map((t: string) => (
                        <Badge key={t} variant="outline" className="text-[9px] px-1.5 h-4 font-bold border-primary/20 bg-primary/[0.03] text-primary/80">
                            {t}
                        </Badge>
                        ))}
                        {!(item["technologies"]?.length) && <span className="text-muted-foreground/30">—</span>}
                    </div>
                    </TableCell>
                </TableRow>
                ))}
            </TableBody>
            </Table>
        </TableContainer>
      );
    }

    case "naabu": {
      const data = tryParseJSONL<any>(lines);
      return (
        <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-3 p-4">
            {data.map((item, idx) => (
                <div key={idx} className="flex items-center justify-between p-3 rounded-xl border border-border/50 bg-muted/20 hover:bg-primary/[0.03] hover:border-primary/30 transition-all group">
                    <div className="flex flex-col gap-0.5">
                        <span className="text-[10px] font-bold text-muted-foreground/50 uppercase tracking-tighter">Host IP</span>
                        <span className="font-mono text-[11px] text-foreground font-medium">{item["ip"]}</span>
                    </div>
                    <div className="flex flex-col items-end gap-0.5">
                        <span className="text-[10px] font-bold text-muted-foreground/50 uppercase tracking-tighter text-right">Open Port</span>
                        <div className="flex items-center gap-1.5">
                            <span className="w-1.5 h-1.5 rounded-full bg-brand-success animate-pulse" />
                            <span className="font-mono text-sm text-primary font-bold">{item["port"]}</span>
                        </div>
                    </div>
                </div>
            ))}
            {data.length === 0 && (
                <div className="col-span-full py-20 text-center flex flex-col items-center gap-2 opacity-30">
                    <Zap className="h-8 w-8" />
                    <span className="text-sm italic uppercase font-bold">No Open Ports Found</span>
                </div>
            )}
        </div>
      );
    }

    case "subfinder": {
      const data = tryParseJSONL<any>(lines);
      return (
        <div className="p-5 space-y-4">
          <div className="flex items-center justify-between border-b border-border/50 pb-2">
            <h5 className="text-[10px] font-bold text-muted-foreground uppercase tracking-widest flex items-center gap-2">
                <Activity className="h-3 w-3" />
                Domains Discovered
            </h5>
            <Badge variant="secondary" className="font-mono text-[10px]">{data.length}</Badge>
          </div>
          <div className="grid grid-cols-1 sm:grid-cols-2 md:grid-cols-3 xl:grid-cols-4 gap-2">
            {data.map((item, idx) => (
              <div key={idx} className="px-3 py-2 bg-muted/30 rounded-lg border border-border/40 font-mono text-[11px] text-primary truncate hover:border-primary/30 transition-all cursor-default">
                {item["host"]}
              </div>
            ))}
          </div>
        </div>
      );
    }

    case "ffuf": {
      const data = tryParseJSONL<any>(lines);
      return (
        <TableContainer>
            <Table>
            <TableHeader className="bg-muted/50">
                <TableRow>
                <TableHead className="text-[10px] uppercase font-bold">DISCOVERED PATH</TableHead>
                <TableHead className="w-[80px] text-[10px] uppercase font-bold text-center">STATUS</TableHead>
                <TableHead className="w-[100px] text-[10px] uppercase font-bold text-right">SIZE</TableHead>
                <TableHead className="w-[100px] text-[10px] uppercase font-bold text-right pr-6">WORDS</TableHead>
                </TableRow>
            </TableHeader>
            <TableBody>
                {data.map((item, idx) => (
                <TableRow key={idx} className="hover:bg-muted/30 transition-colors border-border/40">
                    <TableCell className="font-mono text-[11px] text-primary py-3 break-all">{item["url"]}</TableCell>
                    <TableCell className="text-center py-3">
                    <Badge variant={item["status"] >= 200 && item["status"] < 300 ? "success" : "secondary"} className="h-5 px-1.5 font-mono text-[10px]">
                        {item["status"]}
                    </Badge>
                    </TableCell>
                    <TableCell className="text-right font-mono text-[11px] text-muted-foreground py-3">{item["length"]}</TableCell>
                    <TableCell className="text-right font-mono text-[11px] text-muted-foreground pr-6 py-3">{item["words"]}</TableCell>
                </TableRow>
                ))}
            </TableBody>
            </Table>
        </TableContainer>
      );
    }

    case "urlfinder": {
        try {
            const data = JSON.parse(logs);
            const results = Array.isArray(data) ? data : data.results || [];
            return (
                <TableContainer>
                    <Table>
                    <TableHeader className="bg-muted/50">
                        <TableRow>
                            <TableHead className="text-[10px] uppercase font-bold">SOURCE LINK / URL</TableHead>
                            <TableHead className="w-[150px] text-[10px] uppercase font-bold text-right pr-6">TYPE</TableHead>
                        </TableRow>
                    </TableHeader>
                    <TableBody>
                        {results.map((item: any, idx: number) => (
                            <TableRow key={idx} className="hover:bg-muted/30 transition-colors border-border/40">
                                <TableCell className="font-mono text-[11px] text-primary break-all py-3">{item.url || item.Url || item}</TableCell>
                                <TableCell className="text-right pr-6 py-3">
                                    <span className="text-[10px] px-1.5 py-0.5 rounded-full bg-muted font-bold text-muted-foreground uppercase tracking-tighter">
                                        {item.source || item.Source || "Extracted"}
                                    </span>
                                </TableCell>
                            </TableRow>
                        ))}
                    </TableBody>
                    </Table>
                </TableContainer>
            );
        } catch (e) {
            return <pre className="p-5 text-[11px] font-mono leading-relaxed bg-muted/20 rounded-xl">{logs}</pre>;
        }
    }

    case "nmap": {
        // Handle -oG (greppable) output for alive check
        if (logs.includes("Host:") && logs.includes("Status: Up")) {
            const aliveIps = lines
                .filter(l => l.includes("Host:") && l.includes("Status: Up"))
                .map(l => l.match(/Host: ([0-9.]+)/)?.[1])
                .filter(Boolean);
            return (
                <div className="p-5 space-y-4">
                    <div className="flex items-center justify-between border-b border-border/50 pb-2">
                        <h5 className="text-[10px] font-bold text-muted-foreground uppercase tracking-widest flex items-center gap-2">
                            <Activity className="h-3 w-3" />
                            Hosts Alive
                        </h5>
                        <Badge variant="success" className="font-mono text-[10px]">{aliveIps.length}</Badge>
                    </div>
                    <div className="grid grid-cols-2 sm:grid-cols-4 md:grid-cols-6 gap-2">
                        {aliveIps.map((ip, idx) => (
                            <div key={idx} className="px-2 py-2 bg-brand-success/10 border border-brand-success/20 rounded-lg font-mono text-[11px] text-brand-success text-center font-bold">
                                {ip}
                            </div>
                        ))}
                    </div>
                </div>
            );
        }
        
        if (logs.startsWith("<?xml")) {
            return (
                <div className="p-5 space-y-4">
                    <div className="flex items-center gap-2 border-b border-border/50 pb-2">
                        <span className="text-[10px] font-bold text-muted-foreground uppercase tracking-widest">Port & Service Inventory</span>
                    </div>
                    <div className="grid grid-cols-1 gap-4">
                        {logs.match(/<host[\s\S]*?<\/host>/g)?.map((hostXml, hIdx) => {
                            const ip = hostXml.match(/addr="([0-9.]+)"/)?.[1];
                            const ports = hostXml.match(/<port[\s\S]*?<\/port>/g)?.map(portXml => {
                                const port = portXml.match(/portid="([0-9]+)"/)?.[1];
                                const service = portXml.match(/name="([^"]+)"/)?.[1];
                                const product = portXml.match(/product="([^"]+)"/)?.[1];
                                const version = portXml.match(/version="([^"]+)"/)?.[1];
                                return { port, service, product, version };
                            });
                            if (!ports?.length) return null;
                            return (
                                <div key={hIdx} className="rounded-xl border border-border/50 bg-card overflow-hidden shadow-sm">
                                    <div className="bg-muted/40 px-4 py-2 font-mono text-[11px] font-bold border-b border-border/40 flex items-center justify-between">
                                        <div className="flex items-center gap-2">
                                            <div className="w-2 h-2 rounded-full bg-primary animate-pulse" />
                                            <span>HOST: {ip}</span>
                                        </div>
                                        <span className="text-[10px] text-muted-foreground font-normal">{ports.length} ports open</span>
                                    </div>
                                    <Table>
                                        <TableBody>
                                            {ports?.map((p, pIdx) => (
                                                <TableRow key={pIdx} className="hover:bg-muted/20 border-border/30 last:border-0">
                                                    <TableCell className="w-24 font-mono text-primary font-bold py-2 pl-6">{p.port}</TableCell>
                                                    <TableCell className="w-32 font-mono text-[11px] py-2">{p.service}</TableCell>
                                                    <TableCell className="text-muted-foreground text-[10px] py-2 font-medium">
                                                        {p.product} <span className="opacity-50">{p.version}</span>
                                                    </TableCell>
                                                </TableRow>
                                            ))}
                                        </TableBody>
                                    </Table>
                                </div>
                            );
                        })}
                    </div>
                </div>
            );
        }
        return <pre className="p-5 text-[11px] font-mono leading-relaxed bg-muted/20 rounded-xl">{logs}</pre>;
    }

    case "dnsx": {
      const data = tryParseJSONL<any>(lines);
      return (
        <TableContainer>
            <Table>
            <TableHeader className="bg-muted/50">
                <TableRow>
                <TableHead className="text-[10px] uppercase font-bold">DOMAIN / HOST</TableHead>
                <TableHead className="text-[10px] uppercase font-bold pr-6">RESOLUTION RECORDS</TableHead>
                </TableRow>
            </TableHeader>
            <TableBody>
                {data.map((item, idx) => (
                <TableRow key={idx} className="hover:bg-muted/30 transition-colors border-border/40">
                    <TableCell className="font-mono text-[11px] text-primary font-bold py-3">{item["host"]}</TableCell>
                    <TableCell className="font-mono pr-6 py-3">
                        <div className="flex flex-wrap gap-1.5">
                            {(item["a"] || []).map((a: string) => (
                                <Badge key={a} variant="outline" className="text-[10px] font-bold border-brand-success/30 bg-brand-success/[0.03] text-brand-success">
                                    A: {a}
                                </Badge>
                            ))}
                            {(item["cname"] || []).map((c: string) => (
                                <Badge key={c} variant="secondary" className="text-[10px] font-bold border-border bg-muted/50 text-muted-foreground">
                                    CN: {c}
                                </Badge>
                            ))}
                        </div>
                    </TableCell>
                </TableRow>
                ))}
            </TableBody>
            </Table>
        </TableContainer>
      );
    }

    case "fofa":
    case "hunter":
    case "quake": {
      const data = tryParseJSONL<any>(lines);
      // Single-element JSON array? Parse as a whole.
      const items = (data.length === 0 && lines.length > 0)
        ? (() => { try { const parsed = JSON.parse(logs); return Array.isArray(parsed) ? parsed : [parsed]; } catch { return []; } })()
        : data;
      return (
        <TableContainer>
          <Table>
            <TableHeader className="bg-muted/50">
              <TableRow>
                <TableHead className="text-[10px] uppercase font-bold">HOST</TableHead>
                <TableHead className="w-[140px] text-[10px] uppercase font-bold">IP</TableHead>
                <TableHead className="w-[70px] text-[10px] uppercase font-bold text-center">PORT</TableHead>
                <TableHead className="text-[10px] uppercase font-bold pr-6">TITLE / SERVER</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {items.map((item: any, idx: number) => (
                <TableRow key={idx} className="hover:bg-muted/30 transition-colors border-border/40">
                  <TableCell className="font-mono text-[11px] text-primary py-3 max-w-[250px] truncate">{item.host || item.domain || "-"}</TableCell>
                  <TableCell className="font-mono text-[11px] py-3">{item.ip || "-"}</TableCell>
                  <TableCell className="text-center py-3">
                    <Badge variant="secondary" className="h-5 px-1.5 font-mono text-[10px]">{item.port || "-"}</Badge>
                  </TableCell>
                  <TableCell className="text-[11px] text-muted-foreground pr-6 py-3 max-w-[200px] truncate">
                    {item.title || item.server || "-"}
                  </TableCell>
                </TableRow>
              ))}
              {items.length === 0 && (
                <TableRow><TableCell colSpan={4} className="text-center py-12 text-muted-foreground text-xs">无结果 (API 返回空或查询无匹配)</TableCell></TableRow>
              )}
            </TableBody>
          </Table>
        </TableContainer>
      );
    }

    case "crt": {
      const subdomains: string[] = (() => {
        try { const parsed = JSON.parse(logs); return Array.isArray(parsed) ? parsed : []; } catch { return []; }
      })();
      return (
        <div className="p-5 space-y-3">
          <div className="flex items-center justify-between">
            <h5 className="text-[10px] font-bold text-muted-foreground uppercase tracking-widest">证书透明度子域名</h5>
            <Badge variant="secondary" className="font-mono text-[10px]">{subdomains.length}</Badge>
          </div>
          <div className="grid grid-cols-1 sm:grid-cols-2 md:grid-cols-3 gap-2">
            {subdomains.map((d, i) => (
              <div key={i} className="px-3 py-2 bg-muted/30 rounded-lg border border-border/40 font-mono text-[11px] text-primary truncate hover:border-primary/30 transition-all cursor-default">{d}</div>
            ))}
          </div>
          {subdomains.length === 0 && (
            <p className="text-xs text-muted-foreground text-center py-8">无子域名发现</p>
          )}
        </div>
      );
    }

    case "gau": {
      const urls: string[] = (() => {
        try { const parsed = JSON.parse(logs); return Array.isArray(parsed) ? parsed : []; } catch { return []; }
      })();
      return (
        <div className="p-5 space-y-3">
          <div className="flex items-center justify-between">
            <h5 className="text-[10px] font-bold text-muted-foreground uppercase tracking-widest">历史 URL</h5>
            <Badge variant="secondary" className="font-mono text-[10px]">{urls.length}</Badge>
          </div>
          <div className="max-h-[400px] overflow-y-auto space-y-1">
            {urls.map((u, i) => (
              <div key={i} className="px-3 py-1.5 rounded font-mono text-[10px] text-muted-foreground hover:bg-muted/30 truncate">{u}</div>
            ))}
          </div>
          {urls.length === 0 && (
            <p className="text-xs text-muted-foreground text-center py-8">无历史 URL</p>
          )}
        </div>
      );
    }

    default:
      return (
        <div className="p-20 text-center flex flex-col items-center gap-4">
          <div className="p-4 rounded-full bg-muted/50 border border-border/50">
            <Terminal className="h-10 w-10 text-muted-foreground opacity-20" />
          </div>
          <div className="space-y-1">
            <h5 className="font-bold text-foreground uppercase tracking-widest text-sm">NO FORMATTER AVAILABLE</h5>
            <p className="text-xs text-muted-foreground">The results for tool <span className="font-mono text-primary">{task.tool}</span> are shown in raw format.</p>
          </div>
          <Button variant="outline" size="sm" onClick={() => {}} className="mt-2 border-border/50">
            Switch to Raw Logs
          </Button>
        </div>
      );
  }
}

// --- Task Details Modal Component ---

function TaskDetailsModal({
  task,
  logs,
  loading,
  live,
  viewMode,
  onViewModeChange,
  onClose,
}: {
  task: ScanTask | null;
  logs: string;
  loading: boolean;
  live?: boolean;
  viewMode: "raw" | "parsed";
  onViewModeChange: (mode: "raw" | "parsed") => void;
  onClose: () => void;
}) {
  const toast = useToast();

  const handleCopyCommand = (cmd: string) => {
    navigator.clipboard.writeText(cmd);
    toast("命令已复制到剪贴板", "success");
  };

  const handleDownloadLog = () => {
    if (!task) return;
    const blob = new Blob([logs], { type: "text/plain" });
    const url = URL.createObjectURL(blob);
    const a = document.createElement("a");
    a.href = url;
    a.download = `task_${task.tool}_${task.id.slice(-8)}.log`;
    document.body.appendChild(a);
    a.click();
    document.body.removeChild(a);
    URL.revokeObjectURL(url);
  };

  if (!task) return null;

  return (
    <Modal
      open={!!task}
      onClose={onClose}
      title={
        <div className="flex flex-col sm:flex-row sm:items-center justify-between w-full pr-12 gap-4">
          <div className="flex items-center gap-2 min-w-0">
            <Terminal className="h-5 w-5 text-primary shrink-0" />
            <span className="truncate font-bold text-lg">任务详情: {task.tool}</span>
            <Badge variant="secondary" className="font-mono text-[10px] px-2 py-0.5 rounded-full bg-muted/50 border-border/50 text-muted-foreground shrink-0 ml-1">
              #{task.id.slice(-8).toUpperCase()}
            </Badge>
          </div>
          <div className="flex bg-muted/50 p-1 rounded-xl border border-border/50 shrink-0">
            <button
              onClick={() => onViewModeChange("parsed")}
              className={cn(
                "px-4 py-1.5 text-[10px] font-bold rounded-lg transition-all duration-200",
                viewMode === "parsed" ? "bg-card text-foreground shadow-md ring-1 ring-black/5" : "text-muted-foreground hover:text-foreground hover:bg-white/5"
              )}
            >
              格式化输出
            </button>
            <button
              onClick={() => onViewModeChange("raw")}
              className={cn(
                "px-4 py-1.5 text-[10px] font-bold rounded-lg transition-all duration-200",
                viewMode === "raw" ? "bg-card text-foreground shadow-md ring-1 ring-black/5" : "text-muted-foreground hover:text-foreground hover:bg-white/5"
              )}
            >
              原始日志
            </button>
          </div>
        </div>
      }
      size="xl"
    >
      <div className="space-y-6">
        {/* Execution Summary Cards */}
        <div className="grid grid-cols-2 lg:grid-cols-4 gap-3">
          <div className="bg-muted/30 border border-border/50 rounded-xl p-3 flex flex-col gap-2">
            <div className="flex items-center gap-2 text-muted-foreground">
              <Activity className="h-3.5 w-3.5" />
              <span className="text-[10px] uppercase font-bold tracking-wider">执行状态</span>
            </div>
            <div className="flex items-center">
              <Badge variant={
                task.status === 'completed' ? 'success' :
                task.status === 'failed' ? 'destructive' :
                task.status === 'running' ? 'info' :
                'secondary'
              } className="uppercase text-[10px] h-5">
                {task.status}
              </Badge>
            </div>
          </div>
          
          <div className="bg-muted/30 border border-border/50 rounded-xl p-3 flex flex-col gap-2">
            <div className="flex items-center gap-2 text-muted-foreground">
              <Clock className="h-3.5 w-3.5" />
              <span className="text-[10px] uppercase font-bold tracking-wider">运行时长</span>
            </div>
            <div className="text-sm font-mono font-bold text-foreground">
              {formatTaskDuration(task) || "0.0s"}
            </div>
          </div>

          <div className="bg-muted/30 border border-border/50 rounded-xl p-3 flex flex-col gap-2">
            <div className="flex items-center gap-2 text-muted-foreground">
              <Zap className="h-3.5 w-3.5" />
              <span className="text-[10px] uppercase font-bold tracking-wider">退出代码</span>
            </div>
            <div className={cn(
              "text-sm font-mono font-bold",
              task.exit_code === 0 ? "text-brand-success" : "text-destructive"
            )}>
              {task.exit_code !== undefined ? task.exit_code : "-"}
            </div>
          </div>

          <div className="bg-muted/30 border border-border/50 rounded-xl p-3 flex flex-col gap-2">
            <div className="flex items-center gap-2 text-muted-foreground">
              <History className="h-3.5 w-3.5" />
              <span className="text-[10px] uppercase font-bold tracking-wider">启动时间</span>
            </div>
            <div className="text-sm font-mono text-foreground">
              {task.started_at ? new Date(task.started_at).toLocaleTimeString() : "-"}
            </div>
          </div>
        </div>

        {/* Command Line with Terminal-like Header */}
        <div className="border border-border/50 rounded-xl overflow-hidden bg-card/50">
          <div className="bg-muted/50 px-4 py-2 border-b border-border/50 flex items-center justify-between">
            <div className="flex items-center gap-2">
              <div className="flex gap-1.5">
                <div className="w-2.5 h-2.5 rounded-full bg-destructive/50" />
                <div className="w-2.5 h-2.5 rounded-full bg-warning/50" />
                <div className="w-2.5 h-2.5 rounded-full bg-brand-success/50" />
              </div>
              <span className="text-[10px] font-bold text-muted-foreground uppercase tracking-widest ml-2">Shell Command</span>
            </div>
            <Button 
                variant="ghost" 
                size="sm" 
                className="h-6 px-2 text-[10px] gap-1 hover:bg-primary/10 hover:text-primary transition-colors"
                onClick={() => handleCopyCommand(task.command_template || "")}
            >
                <Copy className="h-3 w-3" />
                COPY
            </Button>
          </div>
          <div className="p-4 font-mono text-[11px] break-all leading-relaxed text-muted-foreground selection:bg-primary/20">
            <span className="text-brand-success mr-2">$</span>
            {task.command_template || "(无命令信息)"}
          </div>
        </div>

        {/* Console Logs / Parsed Output Area */}
        <div className="space-y-3">
          <div className="flex items-center justify-between">
            <div className="flex items-center gap-2">
              <FileText className="h-4 w-4 text-primary" />
              <h4 className="text-xs font-bold uppercase tracking-wider text-foreground">
                {viewMode === "raw" ? "Console Output" : "Structure Data"}
              </h4>
            </div>
          </div>
          
          <div className={cn(
              "relative min-h-[450px] rounded-xl border border-border/50 overflow-hidden shadow-sm transition-all",
              viewMode === "raw" ? "bg-[#0d1117] ring-1 ring-white/5" : "bg-card"
          )}>
            {loading && !(live && logs) ? (
              <div className="absolute inset-0 flex flex-col items-center justify-center gap-4 bg-background/60 backdrop-blur-md z-10">
                <div className="relative">
                  <Loader2 className="h-10 w-10 animate-spin text-primary opacity-20" />
                  <Activity className="h-5 w-5 text-primary absolute top-1/2 left-1/2 -translate-x-1/2 -translate-y-1/2 animate-pulse" />
                </div>
                <div className="flex flex-col items-center gap-1">
                  <span className="text-sm font-bold text-foreground">
                    {live ? "等待工具输出" : "正在拉取扫描结果"}
                  </span>
                  <span className="text-[10px] text-muted-foreground uppercase tracking-tighter">
                    {live ? "每 2 秒刷新 stdout/stderr" : "Syncing from worker node..."}
                  </span>
                </div>
              </div>
            ) : viewMode === "raw" ? (
              <div className="font-mono text-[11px] leading-relaxed text-slate-300 overflow-auto h-full max-h-[600px] p-5 scrollbar-thin scrollbar-thumb-white/10 scrollbar-track-transparent">
                {logs ? (
                  <pre className="whitespace-pre-wrap break-all selection:bg-primary/40 selection:text-white">
                    {logs}
                  </pre>
                ) : (
                  <div className="flex flex-col h-[400px] items-center justify-center text-muted-foreground gap-3">
                    <Terminal className="h-10 w-10 opacity-10" />
                    <span className="italic opacity-50 text-sm">
                      {live ? "尚无输出（工具可能仍在启动）" : "NO LOG OUTPUT RECORDED"}
                    </span>
                  </div>
                )}
              </div>
            ) : (
              <div className="h-full overflow-auto max-h-[600px] scrollbar-thin scrollbar-thumb-border scrollbar-track-transparent">
                <TaskParsedView task={task} logs={logs} />
              </div>
            )}
          </div>
        </div>

        <div className="flex items-center justify-between pt-4 border-t border-border/50">
          <div className="text-[10px] text-muted-foreground font-medium uppercase tracking-widest">
            {live
              ? `实时输出 · ${Math.round(logs.length / 1024)} KB`
              : loading
                ? "Waiting for response..."
                : logs
                  ? `Received ${Math.round(logs.length / 1024)} KB data`
                  : "No data available"}
          </div>
          <div className="flex gap-3">
            {logs && !loading && (
              <Button variant="outline" onClick={handleDownloadLog} size="sm" className="gap-2 h-9 border-border/50 hover:bg-primary/5 hover:border-primary/30 transition-all">
                <Download className="h-3.5 w-3.5" />
                下载完整日志
              </Button>
            )}
            <Button variant="secondary" onClick={onClose} size="sm" className="h-9 px-6 font-bold shadow-sm">
              关闭面板
            </Button>
          </div>
        </div>
      </div>
    </Modal>
  );
}

