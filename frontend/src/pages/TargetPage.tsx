import { useState, useRef, useCallback, useMemo } from "react";
import { Link, useNavigate } from "react-router-dom";
import { api, type ImportResult, type DryRunResult, type Project, type Target, type ScopeConfirmationResponse, type ScopeRule, PAGE_ALL } from "../lib/api";
import { useStore } from "../lib/store";
import { useResource } from "../hooks";
import {
  useProjectId,
  ConfirmDialog,
  useToast,
  EmptyState,
  Badge,
  SkeletonList,
  Card,
  CardHeader,
  CardTitle,
  CardDescription,
  CardContent,
  Button,
  Input,
  Select,
  Table,
  TableHeader,
  TableBody,
  TableRow,
  TableHead,
  TableCell
} from "../components";
import {
  Target as TargetIcon,
  ShieldCheck,
  Plus,
  Play,
  CheckCircle2,
  AlertCircle,
  Clock,
  Zap,
  FileUp,
  X,
  Users,
  Ban,
  Check
} from "lucide-react";
import { cn } from "../lib/utils";

function ProjectInfo({ project }: { project: Project }) {
  const now = new Date();
  const start = project.start_time ? new Date(project.start_time) : null;
  const end = project.end_time ? new Date(project.end_time) : null;
  const isExpired = end && end < now;
  const isActive = (!start || start <= now) && (!end || end >= now);

  return (
    <Card className="bg-muted/30 border-none shadow-none">
      <CardContent className="p-4 flex flex-wrap items-center gap-x-8 gap-y-3">
        <div className="flex items-center gap-2">
            <div className="h-8 w-8 rounded-full bg-primary/10 flex items-center justify-center">
                <Users className="h-4 w-4 text-primary" />
            </div>
            <div>
                <div className="text-[10px] uppercase font-bold text-muted-foreground tracking-tight">Organization</div>
                <div className="text-sm font-medium">{project.organization || "—"}</div>
            </div>
        </div>

        <div className="flex items-center gap-2">
            <div className="h-8 w-8 rounded-full bg-primary/10 flex items-center justify-center">
                <Clock className="h-4 w-4 text-primary" />
            </div>
            <div>
                <div className="text-[10px] uppercase font-bold text-muted-foreground tracking-tight">Window</div>
                <div className="flex items-center gap-2">
                   <span className="text-sm font-medium">{start && end ? `${start.toLocaleDateString()} - ${end.toLocaleDateString()}` : "Unlimited"}</span>
                   {isExpired && <Badge variant="destructive" className="h-4 px-1 text-[9px]">EXPIRED</Badge>}
                   {isActive && <Badge variant="success" className="h-4 px-1 text-[9px]">ACTIVE</Badge>}
                </div>
            </div>
        </div>

        <div className="flex items-center gap-2">
            <div className="h-8 w-8 rounded-full bg-primary/10 flex items-center justify-center">
                <Zap className="h-4 w-4 text-primary" />
            </div>
            <div>
                <div className="text-[10px] uppercase font-bold text-muted-foreground tracking-tight">Rate Limit</div>
                <div className="text-sm font-medium">{project.rate_limit && project.rate_limit > 0 ? `${project.rate_limit} pkts/s` : "No Limit"}</div>
            </div>
        </div>
      </CardContent>
    </Card>
  );
}

function FileImport({ projectId, onImported }: { projectId: string; onImported: () => void }) {
  const [dragOver, setDragOver] = useState(false);
  const [importing, setImporting] = useState(false);
  const [result, setResult] = useState<ImportResult | null>(null);
  const fileRef = useRef<HTMLInputElement>(null);
  const toast = useToast();

  const handleFile = useCallback(
    async (file: File) => {
      if (!file.name.endsWith(".txt") && !file.name.endsWith(".csv")) {
        toast("仅支持 .txt 或 .csv 文件", "warning");
        return;
      }
      setImporting(true);
      setResult(null);
      try {
        const res = await api.importTargets(projectId, file);
        setResult(res);
        onImported();
        toast(`导入完成: 成功 ${res.imported} 个`, "success");
      } catch (err) {
      } finally {
        setImporting(false);
        setDragOver(false);
      }
    },
    [projectId, onImported, toast],
  );

  const handleDrop = useCallback(
    (e: React.DragEvent) => {
      e.preventDefault();
      const file = e.dataTransfer.files[0];
      if (file) handleFile(file);
    },
    [handleFile],
  );

  return (
    <div className="space-y-4">
      <div
        onDragOver={(e) => {
          e.preventDefault();
          setDragOver(true);
        }}
        onDragLeave={() => setDragOver(false)}
        onDrop={handleDrop}
        onClick={() => fileRef.current?.click()}
        className={cn(
            "relative group border-2 border-dashed rounded-xl p-10 text-center cursor-pointer transition-all",
            importing ? "opacity-50 cursor-not-allowed" : "hover:border-primary/50 hover:bg-primary/[0.02]",
            dragOver ? "border-primary bg-primary/10" : "border-border"
        )}
      >
        <input
          ref={fileRef}
          type="file"
          accept=".txt,.csv"
          className="hidden"
          onChange={(e) => {
            const file = e.target.files?.[0];
            if (file) handleFile(file);
            e.target.value = "";
          }}
        />
        <div className="flex flex-col items-center">
            <div className={cn(
                "mb-4 h-12 w-12 rounded-full bg-muted flex items-center justify-center transition-transform group-hover:scale-110",
                dragOver && "bg-primary text-primary-foreground"
            )}>
                {importing ? <div className="animate-spin text-lg">⏳</div> : <FileUp className="h-6 w-6" />}
            </div>
            <div className="font-semibold text-foreground">
                {importing ? "正在上传并处理..." : "点击或拖拽文件到此处"}
            </div>
            <p className="mt-1 text-sm text-muted-foreground max-w-[280px] mx-auto">
                支持每行一个目标的 .txt 或 .csv 文本文件。
            </p>
        </div>
      </div>

      {result && (
        <Card className="bg-muted/30 border-dashed animate-in slide-in-from-top-2 duration-300">
          <CardHeader className="py-3 px-4">
            <div className="flex items-center justify-between">
                <CardTitle className="text-sm flex items-center gap-2">
                    <CheckCircle2 className="h-4 w-4 text-brand-success" />
                    导入报告
                </CardTitle>
                <Button variant="ghost" size="sm" className="h-6 w-6 p-0" onClick={() => setResult(null)}>
                    <X className="h-3 w-3" />
                </Button>
            </div>
          </CardHeader>
          <CardContent className="px-4 pb-4">
            <div className="grid grid-cols-2 sm:grid-cols-5 gap-3">
              <StatItem label="成功" value={result.imported} color="text-brand-success" />
              <StatItem label="总目标" value={result.expanded ?? 0} color="text-brand-primary" />
              <StatItem label="重复" value={result.duplicates} />
              <StatItem label="拒绝" value={result.denied} color="text-brand-warning" />
              <StatItem label="错误" value={result.errors} color="text-brand-danger" />
            </div>
            {result.denied_targets.length > 0 && (
              <div className="mt-4 pt-4 border-t border-border/50">
                <div className="text-xs font-semibold text-brand-warning mb-2 uppercase tracking-tight">Scope Denied:</div>
                <div className="space-y-1.5 max-h-32 overflow-auto pr-2 custom-scrollbar">
                    {result.denied_targets.map((d, i) => (
                    <div key={i} className="text-xs flex items-center justify-between p-2 rounded bg-background/50 border border-border/50">
                        <code className="text-brand-warning font-mono">{d.value}</code>
                        <span className="text-muted-foreground italic text-[10px]">{d.reason}</span>
                    </div>
                    ))}
                </div>
              </div>
            )}
          </CardContent>
        </Card>
      )}
    </div>
  );
}

function StatItem({ label, value, color = "text-muted-foreground" }: { label: string; value: number; color?: string }) {
    return (
        <div className="rounded-lg bg-background/50 p-2 border border-border/50">
            <div className="text-[10px] font-bold text-muted-foreground uppercase leading-none mb-1.5">{label}</div>
            <div className={cn("text-lg font-bold tabular-nums leading-none", color)}>{value}</div>
        </div>
    )
}

export default function TargetPage() {
  const navigate = useNavigate();
  const projectId = useProjectId();
  const currentProject = useStore((state) => state.currentProject);
  const setTargets = useStore((state) => state.setTargets);
  const targets = useStore((state) => state.targets) ?? [];
  const toast = useToast();

  const [scopeRules, setScopeRules] = useState<ScopeRule[]>([]);

  const { excludedRules, includedRules } = useMemo(() => {
    const excluded: ScopeRule[] = [];
    const included: ScopeRule[] = [];
    for (const r of scopeRules) {
      if (r.action === "exclude") excluded.push(r);
      else included.push(r);
    }
    return { excludedRules: excluded, includedRules: included };
  }, [scopeRules]);

  const {
    reload: loadScopeRules,
  } = useResource(
    async (signal) => {
      if (!projectId) return;
      const data = await api.listScopeRules(projectId, PAGE_ALL, signal);
      setScopeRules(data.data ?? []);
    },
    [projectId],
    undefined
  );

  const {
    loading,
    error,
    reload: loadTargets,
  } = useResource(
    async (signal) => {
      if (!projectId) return;
      const data = await api.listTargets(projectId, PAGE_ALL, signal);
      setTargets(data.data ?? []);
    },
    [projectId],
    undefined
  );

  const [targetValue, setTargetValue] = useState("");
  const [targetType, setTargetType] = useState("auto");
  const [scopeAction, setScopeAction] = useState<"include" | "exclude">("include");
  const [scopeValue, setScopeValue] = useState("");
  const [dryRunResult, setDryRunResult] = useState<DryRunResult | null>(null);
  const [scopeConfirmOpen, setScopeConfirmOpen] = useState(false);
  const [pendingScopeConfirm, setPendingScopeConfirm] = useState<{
    message: string;
    suggested: ScopeConfirmationResponse["suggested_rule"];
    pendingType: string;
    pendingValue: string;
  } | null>(null);
  const [scopeConfirmLoading, setScopeConfirmLoading] = useState(false);

  const [addingTarget, setAddingTarget] = useState(false);
  const [addingScope, setAddingScope] = useState(false);
  const [dryRunLoading, setDryRunLoading] = useState(false);
  const [dryRunConfirmOpen, setDryRunConfirmOpen] = useState(false);

  const addTarget = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!projectId || !targetValue.trim()) {
      toast("请输入目标值", "warning");
      return;
    }
    if (addingTarget) return;
    setAddingTarget(true);
    try {
      const res = await api.createTarget(projectId, { type: targetType, value: targetValue });
      if ("needs_scope_confirmation" in res && res.needs_scope_confirmation) {
        setPendingScopeConfirm({
          message: res.message,
          suggested: res.suggested_rule,
          pendingType: targetType,
          pendingValue: targetValue,
        });
        setScopeConfirmOpen(true);
        return;
      }
      setTargets([...targets, res as Target]);
      setTargetValue("");
      toast("目标已添加", "success");
    } catch (err) {
    } finally {
      setAddingTarget(false);
    }
  };

  const handleConfirmScope = async () => {
    if (!pendingScopeConfirm || !projectId) return;
    setScopeConfirmLoading(true);
    try {
      const { suggested, pendingType, pendingValue } = pendingScopeConfirm;
      await api.createScopeRule({
        project_id: projectId,
        action: suggested.action,
        type: suggested.type,
        value: suggested.value,
      });
      // 等待后端 SQLite WAL 刷新，确保新规则对后续查询可见
      await new Promise((r) => setTimeout(r, 500));
      await loadScopeRules();
      const t = await api.createTarget(projectId, { type: pendingType, value: pendingValue });
      if ("needs_scope_confirmation" in t && t.needs_scope_confirmation) {
        toast("Scope 规则已添加，但目标仍需要确认，请手动检查。", "warning");
      } else {
        setTargets([...targets, t as Target]);
        setTargetValue("");
        toast("目标已添加", "success");
      }
    } catch (err) {
    } finally {
      setScopeConfirmLoading(false);
      setScopeConfirmOpen(false);
      setPendingScopeConfirm(null);
    }
  };

  const inferScopeType = (value: string): string => {
    const v = value.trim();
    if (v.includes("/")) return "cidr";
    if (/^https?:\/\//i.test(v)) return "url";
    if (/^\d+\.\d+\.\d+\.\d+$/.test(v)) return "ip";
    return "domain";
  };

  const addScopeRule = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!projectId || !scopeValue.trim()) {
      toast("请输入 Scope 规则值", "warning");
      return;
    }
    if (addingScope) return;
    setAddingScope(true);
    try {
      await api.createScopeRule({
        project_id: projectId,
        action: scopeAction,
        type: inferScopeType(scopeValue),
        value: scopeValue,
      });
      setScopeValue("");
      await loadScopeRules();
      toast("规则已添加", "success");
    } catch (err) {
    } finally {
      setAddingScope(false);
    }
  };

  const runDryRun = async () => {
    if (!projectId || dryRunLoading) return;
    setDryRunLoading(true);
    try {
      const res = await api.dryRun(projectId);
      setDryRunResult(res);
      toast("授权检测完成", "success");
    } catch (err) {
    } finally {
      setDryRunLoading(false);
      setDryRunConfirmOpen(false);
    }
  };

  if (!currentProject) {
    return (
      <div className="flex flex-col items-center justify-center py-20 text-center">
        <div className="h-16 w-16 rounded-full bg-muted flex items-center justify-center mb-4">
            <TargetIcon className="h-8 w-8 text-muted-foreground opacity-50" />
        </div>
        <h2 className="text-xl font-bold">目标管理</h2>
        <p className="text-muted-foreground mt-1 mb-6">请先从左侧菜单或总览选择一个项目</p>
        <Link to="/">
            <Button variant="primary">前往总览</Button>
        </Link>
      </div>
    );
  }

  return (
    <div className="space-y-8 animate-in fade-in duration-500">
      <div className="flex items-start justify-between">
        <div>
          <div className="flex items-center gap-2 text-brand-success font-bold text-xs uppercase tracking-widest mb-1.5">
            <ShieldCheck className="h-3.5 w-3.5" />
            Step 1: Scoping
          </div>
          <h1 className="text-3xl font-bold tracking-tight text-foreground">目标与 Scope</h1>
          <p className="text-muted-foreground mt-1">确认授权边界，导入需要测试的域名、URL、IP 或 CIDR。</p>
        </div>
        <div className="flex gap-2">
            <Button variant="secondary" size="sm" onClick={() => setDryRunConfirmOpen(true)} disabled={dryRunLoading}>
                <Zap className={cn("mr-2 h-4 w-4 text-brand-warning", dryRunLoading && "animate-pulse")} />
                {dryRunLoading ? "正在评估..." : "授权 Dry Run"}
            </Button>
            <Button variant="primary" size="sm" onClick={() => navigate(`/projects/${currentProject.id}/runs`)}>
                <Play className="mr-2 h-4 w-4 fill-current" />
                下一步：扫描
            </Button>
        </div>
      </div>

      <ProjectInfo project={currentProject} />

      <div className="grid gap-8 lg:grid-cols-[1fr_360px]">
        <section className="space-y-6">
            <div className="flex items-center justify-between">
                <h2 className="text-xl font-semibold tracking-tight">目标列表</h2>
                <div className="flex items-center gap-2">
                    <Badge variant="secondary" className="h-6">{targets.length} Targets</Badge>
                </div>
            </div>

            <Card>
                {loading ? (
                <div className="p-6">
                    <SkeletonList count={5} />
                </div>
                ) : error ? (
                <div className="py-20 text-center">
                    <AlertCircle className="h-10 w-10 text-destructive mx-auto mb-4 opacity-50" />
                    <p className="text-destructive font-medium mb-4">加载失败: {error}</p>
                    <Button variant="secondary" size="sm" onClick={() => loadTargets()}>
                        重试
                    </Button>
                </div>
                ) : targets.length === 0 ? (
                <div className="py-20 text-center">
                    <EmptyState
                        title="暂无目标"
                        description="当前项目还没有添加任何目标，请在右侧添加或批量导入。"
                    />
                </div>
                ) : (
                <Table>
                    <TableHeader>
                        <TableRow>
                            <TableHead className="w-24">类型</TableHead>
                            <TableHead>目标值</TableHead>
                            <TableHead className="w-48 text-right text-muted-foreground">创建时间</TableHead>
                        </TableRow>
                    </TableHeader>
                    <TableBody>
                        {targets.map((t) => (
                        <TableRow key={t.id}>
                            <TableCell>
                                <Badge variant="secondary" className="font-mono text-[10px] uppercase">
                                    {t.type}
                                </Badge>
                            </TableCell>
                            <TableCell className="font-medium font-mono text-sm">{t.value}</TableCell>
                            <TableCell className="text-right text-xs text-muted-foreground">
                                {new Date(t.created_at).toLocaleString()}
                            </TableCell>
                        </TableRow>
                        ))}
                    </TableBody>
                </Table>
                )}
            </Card>
        </section>

        <aside className="space-y-6">
            <Card>
                <CardHeader>
                    <CardTitle className="text-lg">添加单个目标</CardTitle>
                    <CardDescription>支持 IP, CIDR, 域名, URL 等类型。</CardDescription>
                </CardHeader>
                <CardContent className="space-y-4">
                    <form onSubmit={addTarget} className="space-y-3">
                        <div className="grid grid-cols-[100px_1fr] gap-2">
                            <Select
                                value={targetType}
                                onChange={(e) => setTargetType(e.target.value)}
                            >
                                <option value="auto">自动</option>
                                <option value="domain">域名</option>
                                <option value="url">URL</option>
                                <option value="ip">IP</option>
                                <option value="cidr">CIDR</option>
                            </Select>
                            <Input
                                placeholder="example.com"
                                value={targetValue}
                                onChange={(e) => setTargetValue(e.target.value)}
                            />
                        </div>
                        <Button type="submit" variant="primary" loading={addingTarget} className="w-full">
                            <Plus className="mr-2 h-4 w-4" />
                            添加目标
                        </Button>
                    </form>
                </CardContent>
            </Card>

            <Card>
                <CardHeader>
                    <CardTitle className="text-lg">批量导入</CardTitle>
                </CardHeader>
                <CardContent>
                    <FileImport projectId={currentProject.id} onImported={() => { loadTargets(); loadScopeRules(); }} />
                </CardContent>
            </Card>

            <Card className="bg-muted/30">
                <CardHeader>
                    <CardTitle className="text-sm">Scope 规则</CardTitle>
                    <CardDescription className="text-xs text-muted-foreground">决定哪些目标在授权范围内。</CardDescription>
                </CardHeader>
                <CardContent className="space-y-4">
                    <form onSubmit={addScopeRule} className="space-y-3">
                        <div className="grid grid-cols-[80px_1fr] gap-2">
                            <Select
                                value={scopeAction}
                                onChange={(e) => setScopeAction(e.target.value as any)}
                            >
                                <option value="include">包含</option>
                                <option value="exclude">排除</option>
                            </Select>
                            <Input
                                placeholder="例如 *.example.com"
                                value={scopeValue}
                                onChange={(e) => setScopeValue(e.target.value)}
                            />
                        </div>
                        <Button type="submit" size="sm" disabled={addingScope} variant="secondary" className="w-full h-8">
                            添加规则
                        </Button>
                    </form>

                    {/* 已排除的 IP / 域名 */}
                    {excludedRules.length > 0 && (
                        <div className="pt-3 border-t border-border/50">
                            <div className="text-[10px] font-bold text-muted-foreground uppercase tracking-tight mb-2 flex items-center gap-1.5">
                                <Ban className="h-3 w-3 text-brand-danger" />
                                已排除 ({excludedRules.length})
                            </div>
                            <div className="space-y-1.5 max-h-40 overflow-auto pr-1 custom-scrollbar">
                                {excludedRules.map((rule) => (
                                    <div
                                        key={rule.id}
                                        className="flex items-center justify-between p-1.5 rounded bg-background/50 border border-border/50"
                                    >
                                        <div className="flex items-center gap-1.5 min-w-0">
                                            <Ban className="h-3 w-3 text-brand-danger shrink-0" />
                                            <code className="text-xs font-mono truncate">{rule.value}</code>
                                        </div>
                                        <Badge variant="outline" className="text-[9px] h-4 px-1 shrink-0 ml-2">
                                            {rule.type}
                                        </Badge>
                                    </div>
                                ))}
                            </div>
                        </div>
                    )}

                    {/* 已包含的规则（如果有） */}
                    {includedRules.length > 0 && (
                        <div className="pt-3 border-t border-border/50">
                            <div className="text-[10px] font-bold text-muted-foreground uppercase tracking-tight mb-2 flex items-center gap-1.5">
                                <Check className="h-3 w-3 text-brand-success" />
                                已包含 ({includedRules.length})
                            </div>
                            <div className="space-y-1.5 max-h-40 overflow-auto pr-1 custom-scrollbar">
                                {includedRules.map((rule) => (
                                    <div
                                        key={rule.id}
                                        className="flex items-center justify-between p-1.5 rounded bg-background/50 border border-border/50"
                                    >
                                        <div className="flex items-center gap-1.5 min-w-0">
                                            <Check className="h-3 w-3 text-brand-success shrink-0" />
                                            <code className="text-xs font-mono truncate">{rule.value}</code>
                                        </div>
                                        <Badge variant="outline" className="text-[9px] h-4 px-1 shrink-0 ml-2">
                                            {rule.type}
                                        </Badge>
                                    </div>
                                ))}
                            </div>
                        </div>
                    )}
                </CardContent>
            </Card>
        </aside>
      </div>

      {dryRunResult && (
        <Card className="border-brand-warning/30 bg-brand-warning/[0.02]">
           <CardHeader className="flex flex-row items-center justify-between py-4">
              <div className="flex items-center gap-2">
                  <div className="h-8 w-8 rounded-full bg-brand-warning/10 flex items-center justify-center">
                    <ShieldCheck className="h-4 w-4 text-brand-warning" />
                  </div>
                  <div>
                    <CardTitle className="text-base text-brand-warning">授权检测评估 (Dry Run Result)</CardTitle>
                    <CardDescription className="text-xs">
                        基于当前 {dryRunResult.results?.length ?? 0} 个目标的 Scope 评估
                    </CardDescription>
                  </div>
              </div>
              <Button variant="ghost" size="sm" onClick={() => setDryRunResult(null)}>关闭报告</Button>
           </CardHeader>
           <CardContent>
                <div className="grid grid-cols-1 md:grid-cols-3 gap-4 mb-6">
                    <div className="p-3 rounded-lg border bg-background/50">
                        <div className="text-[10px] font-bold text-muted-foreground uppercase mb-1">Time Window</div>
                        <div className="flex items-center gap-2">
                            {dryRunResult.time_window_valid ? 
                                <span className="text-sm font-semibold text-brand-success flex items-center gap-1"><CheckCircle2 className="h-3.5 w-3.5" /> Valid</span> : 
                                <span className="text-sm font-semibold text-brand-danger flex items-center gap-1"><AlertCircle className="h-3.5 w-3.5" /> Invalid</span>
                            }
                        </div>
                    </div>
                    <div className="p-3 rounded-lg border bg-background/50">
                        <div className="text-[10px] font-bold text-muted-foreground uppercase mb-1">Estimated Time</div>
                        <div className="text-sm font-semibold">
                            {dryRunResult.estimated_duration_seconds ? 
                                (dryRunResult.estimated_duration_seconds < 60 ? `${dryRunResult.estimated_duration_seconds}s` : `${Math.round(dryRunResult.estimated_duration_seconds/60)}m`) 
                                : "N/A"}
                        </div>
                    </div>
                    <div className="p-3 rounded-lg border bg-background/50">
                        <div className="text-[10px] font-bold text-muted-foreground uppercase mb-1">Decision Mode</div>
                        <div className="text-sm font-semibold uppercase">{dryRunResult.mode}</div>
                    </div>
                </div>

                {dryRunResult.results && (
                   <div className="border rounded-lg overflow-hidden">
                      <div className="max-h-60 overflow-auto custom-scrollbar">
                        <Table>
                            <TableBody>
                                {dryRunResult.results.map((r, i) => (
                                    <TableRow key={i}>
                                        <TableCell className="w-12 text-center p-2">
                                            {r.decision === "allow" ? 
                                                <div className="h-2 w-2 rounded-full bg-brand-success mx-auto" /> : 
                                                <div className="h-2 w-2 rounded-full bg-brand-danger mx-auto" />
                                            }
                                        </TableCell>
                                        <TableCell className="p-2 font-mono text-xs">{r.target}</TableCell>
                                        <TableCell className={cn("p-2 text-xs", r.decision === 'allow' ? 'text-brand-success' : 'text-brand-danger')}>
                                            {r.reason}
                                        </TableCell>
                                    </TableRow>
                                ))}
                            </TableBody>
                        </Table>
                      </div>
                   </div>
                )}
           </CardContent>
        </Card>
      )}

      <ConfirmDialog
        open={scopeConfirmOpen}
        onClose={() => {
          setScopeConfirmOpen(false);
          setPendingScopeConfirm(null);
        }}
        onConfirm={handleConfirmScope}
        title="自动修正 Scope"
        description={
          pendingScopeConfirm
            ? `添加此目标需要额外的 Scope 授权规则。是否自动添加规则: [${pendingScopeConfirm.suggested.action}] ${pendingScopeConfirm.suggested.type} = ${pendingScopeConfirm.suggested.value}？`
            : ""
        }
        confirmText="添加并继续"
        cancelText="取消"
        loading={scopeConfirmLoading}
      />

      <ConfirmDialog
        open={dryRunConfirmOpen}
        onClose={() => setDryRunConfirmOpen(false)}
        onConfirm={runDryRun}
        title="启动授权 Dry Run"
        description="系统将模拟扫描引擎对当前项目的所有目标进行授权校验，并评估预计耗时。此操作不会产生实际扫描流量。"
        confirmText="开始评估"
        cancelText="取消"
        loading={dryRunLoading}
      />
    </div>
  );
}
