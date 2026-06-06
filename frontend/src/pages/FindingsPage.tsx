import { useEffect, useMemo, useState } from "react";
import { Link } from "react-router-dom";
import { api, PAGE_ALL } from "../lib/api";
import { useStore } from "../lib/store";
import { useResource } from "../hooks";
import {
  EmptyState,
  Input,
  Select,
  SkeletonList,
  useProjectId,
  useToast,
  Card,
  CardHeader,
  CardTitle,
  CardDescription,
  CardContent,
  Button,
  Badge,
  SeverityBadge,
  StatusBadge
} from "../components";
import type { Finding, Evidence, ToolCallTrace } from "../lib/api";
import {
  Search,
  AlertCircle,
  ArrowRight,
  FileText,
  Clock,
  History,
  ShieldAlert,
  MessageSquare,
  X,
  Plus,
  Activity,
} from "lucide-react";
import { cn } from "../lib/utils";
import { dedupeFindingsForDisplay } from "../lib/finding-dedup";

const statusLabels: Record<string, string> = {
  pending_review: "待审核",
  confirmed: "已确认",
  false_positive: "误报",
  accepted_risk: "已接受风险",
  ignored: "已忽略",
};

export default function FindingsPage() {
  const projectId = useProjectId();
  const findings = useStore((state) => state.findings) ?? [];
  const setFindings = useStore((state) => state.setFindings);
  const currentFinding = useStore((state) => state.currentFinding);
  const setCurrentFinding = useStore((state) => state.setCurrentFinding);

  const [statusFilter, setStatusFilter] = useState<string>("");
  const [severityFilter, setSeverityFilter] = useState<string>("");
  const [keyword, setKeyword] = useState<string>("");
  const [debouncedKeyword, setDebouncedKeyword] = useState<string>("");
  const [detailOpen, setDetailOpen] = useState(false);
  const [selectedIds, setSelectedIds] = useState<Set<string>>(new Set());
  const [batchStatus, setBatchStatus] = useState<string>("");
  const [batchUpdating, setBatchUpdating] = useState(false);

  const findingStatusHistory = useStore((state) => state.findingStatusHistory);
  const recordStatusChange = useStore((state) => state.recordStatusChange);
  const toast = useToast();

  useEffect(() => {
    const timer = setTimeout(() => setDebouncedKeyword(keyword), 300);
    return () => clearTimeout(timer);
  }, [keyword]);

  const { loading } = useResource(
    async (signal) => {
      if (!projectId) return;
      const res = await api.listFindings(projectId, undefined, PAGE_ALL, signal);
      setFindings(res.data ?? []);
    },
    [projectId],
    undefined
  );

  const openDetail = async (findingId: string) => {
    try {
      const data = await api.getFinding(findingId);
      setCurrentFinding(data);
      setDetailOpen(true);
    } catch (err) {
    }
  };

  const closeDetail = () => {
    setDetailOpen(false);
    setCurrentFinding(null);
  };

  const changeStatus = async (findingId: string, status: string) => {
    const previousStatus = findings.find((f) => f.id === findingId)?.status;
    setFindings((prev) => prev.map((f) => (f.id === findingId ? { ...f, status } : f)));
    recordStatusChange(findingId, status);
    try {
      await api.updateFindingStatus(findingId, status);
      toast("状态已更新", "success");
    } catch (err) {
      if (previousStatus !== undefined) {
        setFindings((prev) => prev.map((f) => (f.id === findingId ? { ...f, status: previousStatus } : f)));
      }
    }
  };

  const toggleSelect = (id: string) => {
    setSelectedIds((prev) => {
      const next = new Set(prev);
      next[next.has(id) ? "delete" : "add"](id);
      return next;
    });
  };

  const toggleSelectAll = () => {
    if (selectedIds.size === filteredFindings.length && filteredFindings.length > 0) {
      setSelectedIds(new Set());
    } else {
      setSelectedIds(new Set(filteredFindings.map((f) => f.id)));
    }
  };

  const batchChangeStatus = async () => {
    if (!batchStatus || selectedIds.size === 0) return;
    setBatchUpdating(true);
    try {
      await api.batchUpdateFindingStatus(Array.from(selectedIds), batchStatus);
      selectedIds.forEach((id) => recordStatusChange(id, batchStatus));
      toast(`已成功批量更新 ${selectedIds.size} 项状态`, "success");
      setSelectedIds(new Set());
      setBatchStatus("");
      if (projectId) {
        const updated = await api.listFindings(projectId, undefined, PAGE_ALL);
        setFindings(updated.data ?? []);
      }
    } catch (err) {
    } finally {
      setBatchUpdating(false);
    }
  };

  const displayFindings = useMemo(() => dedupeFindingsForDisplay(findings), [findings]);

  const filteredFindings = useMemo(() => {
    let result = displayFindings;
    if (statusFilter) result = result.filter((f) => f.status === statusFilter);
    if (severityFilter) result = result.filter((f) => f.severity === severityFilter);
    if (debouncedKeyword.trim()) {
      const kw = debouncedKeyword.trim().toLowerCase();
      result = result.filter((f) => f.title.toLowerCase().includes(kw) || (f.summary && f.summary.toLowerCase().includes(kw)));
    }
    return result;
  }, [displayFindings, statusFilter, severityFilter, debouncedKeyword]);

  if (!projectId) {
    return (
      <div className="flex flex-col items-center justify-center py-20 text-center">
        <div className="h-16 w-16 rounded-full bg-muted flex items-center justify-center mb-4">
            <ShieldAlert className="h-8 w-8 text-muted-foreground opacity-50" />
        </div>
        <h2 className="text-xl font-bold">漏洞发现</h2>
        <p className="text-muted-foreground mt-1 mb-6">请先从左侧菜单或总览选择一个项目</p>
        <Link to="/">
            <Button variant="primary">前往总览</Button>
        </Link>
      </div>
    );
  }

  return (
    <div className="space-y-8 animate-in fade-in duration-500 pb-20">
      <div className="flex items-start justify-between">
        <div>
          <div className="flex items-center gap-2 text-rose-500 font-bold text-xs uppercase tracking-widest mb-1.5">
            <AlertCircle className="h-3.5 w-3.5" />
            Step 4: Finding Audit
          </div>
          <h1 className="text-3xl font-bold tracking-tight text-foreground">发现审核</h1>
          <p className="text-muted-foreground mt-1">对扫描发现的漏洞进行分类、验证和风险评估。</p>
        </div>
      </div>

      <div className="space-y-6">
        <div className="flex flex-col gap-4">
            <div className="flex flex-wrap items-center gap-4">
               <div className="relative flex-1 max-w-md">
                  <Search className="absolute left-3 top-2.5 h-4 w-4 text-muted-foreground" />
                  <Input 
                    placeholder="按漏洞名称或摘要搜索..." 
                    className="pl-10 h-10" 
                    value={keyword}
                    onChange={(e) => setKeyword(e.target.value)}
                  />
               </div>
               <div className="flex items-center gap-2">
                  <Badge variant="outline" className="h-10 px-4 rounded-md border-border bg-card shadow-sm text-xs text-muted-foreground">
                    <History className="h-3.5 w-3.5 mr-2" />
                    {displayFindings.length === findings.length
                      ? `共 ${findings.length} 条记录`
                      : `共 ${displayFindings.length} 条（已去重，原始 ${findings.length} 条）`}
                  </Badge>
               </div>
            </div>

            <div className="flex flex-wrap items-center gap-6 py-2 border-b border-border/50">
               <div className="flex items-center gap-2">
                  <span className="text-xs font-bold text-muted-foreground uppercase tracking-wider">Severity:</span>
                  <div className="flex items-center gap-1">
                     {["", "critical", "high", "medium", "low", "info"].map((s) => (
                        <button
                          key={s || 'all'}
                          onClick={() => setSeverityFilter(s)}
                          className={cn(
                            "px-2.5 py-1 rounded-md text-[11px] font-bold transition-all",
                            severityFilter === s 
                                ? "bg-primary text-primary-foreground shadow-sm scale-105" 
                                : "text-muted-foreground hover:bg-muted"
                          )}
                        >
                          {s ? s.toUpperCase() : "ALL"}
                        </button>
                     ))}
                  </div>
               </div>

               <div className="flex items-center gap-2">
                  <span className="text-xs font-bold text-muted-foreground uppercase tracking-wider">Status:</span>
                  <div className="flex items-center gap-1">
                     {["", "pending_review", "confirmed", "false_positive"].map((s) => (
                        <button
                          key={s || 'all'}
                          onClick={() => setStatusFilter(s)}
                          className={cn(
                            "px-2.5 py-1 rounded-md text-[11px] font-bold transition-all",
                            statusFilter === s 
                                ? "bg-primary text-primary-foreground shadow-sm scale-105" 
                                : "text-muted-foreground hover:bg-muted"
                          )}
                        >
                          {s ? statusLabels[s] : "ALL"}
                        </button>
                     ))}
                  </div>
               </div>
            </div>
        </div>

        {loading ? (
            <SkeletonList count={6} />
        ) : filteredFindings.length === 0 ? (
            <Card className="border-dashed py-20 text-center">
                <EmptyState title="未找到漏洞发现" description="试试调整筛选条件，或者运行一次新的扫描。" />
            </Card>
        ) : (
            <div className="space-y-4">
                <div className="flex items-center gap-2 mb-2">
                    <input
                      type="checkbox"
                      checked={filteredFindings.length > 0 && selectedIds.size === filteredFindings.length}
                      onChange={toggleSelectAll}
                      className="h-4 w-4 rounded border-border bg-card text-primary focus:ring-primary"
                    />
                    <span className="text-xs text-muted-foreground font-medium">全选当前页</span>
                </div>
                
                <div className="grid gap-3">
                    {filteredFindings.map((f) => (
                        <Card 
                            key={f.id} 
                            className={cn(
                                "group border-l-4 transition-all hover:bg-muted/30 cursor-pointer",
                                f.severity === 'critical' ? 'border-l-rose-600' : 
                                f.severity === 'high' ? 'border-l-orange-500' :
                                f.severity === 'medium' ? 'border-l-amber-500' :
                                f.severity === 'low' ? 'border-l-cyan-500' : 'border-l-muted-foreground/30',
                                selectedIds.has(f.id) && "bg-primary/[0.03] ring-1 ring-primary/20"
                            )}
                            onClick={() => openDetail(f.id)}
                        >
                            <CardContent className="p-4 flex items-center gap-4">
                                <div onClick={(e) => e.stopPropagation()} className="flex items-center">
                                    <input
                                      type="checkbox"
                                      checked={selectedIds.has(f.id)}
                                      onChange={() => toggleSelect(f.id)}
                                      className="h-4 w-4 rounded border-border bg-card text-primary focus:ring-primary"
                                    />
                                </div>
                                
                                <div className="flex-1 min-w-0">
                                    <div className="flex items-center gap-2 mb-1">
                                        <SeverityBadge severity={f.severity} />
                                        <h3 className="font-bold text-foreground truncate group-hover:text-primary transition-colors">
                                            {f.title}
                                        </h3>
                                    </div>
                                    <div className="flex flex-wrap items-center gap-x-4 gap-y-1">
                                        <div className="flex items-center gap-1.5 text-xs text-muted-foreground">
                                            <Badge variant="secondary" className="h-4 px-1 text-[9px] font-mono">
                                                {f.source_tool}
                                            </Badge>
                                        </div>
                                        <div className="flex items-center gap-1.5 text-[11px] text-muted-foreground">
                                            <Clock className="h-3 w-3" />
                                            {findingStatusHistory[f.id] ? formatTimeAgo(findingStatusHistory[f.id].updatedAt) : "发现于最近一次扫描"}
                                        </div>
                                        <div className="flex items-center gap-1.5 text-[11px] text-muted-foreground">
                                            <span className="font-semibold text-muted-foreground/60">CONFIDENCE:</span>
                                            <span className="font-mono">{f.confidence}</span>
                                        </div>
                                        {f.asset_id && (
                                            <div className="flex items-center gap-1.5 text-[11px] text-muted-foreground">
                                                <span className="font-semibold text-muted-foreground/60">ASSET:</span>
                                                <Link 
                                                    to={`/projects/${projectId}/assets?highlight=${f.asset_id}`}
                                                    className="font-mono text-primary hover:underline truncate max-w-[120px]"
                                                    title={f.asset_id}
                                                >
                                                    {f.asset_id.slice(-8)}
                                                </Link>
                                            </div>
                                        )}
                                    </div>
                                </div>

                                <div className="flex items-center gap-6" onClick={e => e.stopPropagation()}>
                                    <Select 
                                        value={f.status}
                                        onChange={(e) => changeStatus(f.id, e.target.value)}
                                        className="h-8 w-32 text-xs font-medium"
                                    >
                                        {statusOptions.map(opt => (
                                            <option key={opt.value} value={opt.value}>{opt.label}</option>
                                        ))}
                                    </Select>
                                    <Button variant="ghost" size="sm" className="h-8 w-8 p-0 text-muted-foreground opacity-0 group-hover:opacity-100 transition-all">
                                        <ArrowRight className="h-4 w-4" />
                                    </Button>
                                </div>
                            </CardContent>
                        </Card>
                    ))}
                </div>
            </div>
        )}
      </div>

      {/* Floating Batch Actions Bar */}
      {selectedIds.size > 0 && (
        <div className="fixed bottom-8 left-1/2 -translate-x-1/2 z-50 animate-in slide-in-from-bottom-8 duration-300">
           <div className="flex items-center gap-4 px-6 py-3 rounded-full bg-zinc-900 text-zinc-100 shadow-2xl ring-1 ring-white/10 border border-white/10 backdrop-blur-md">
              <span className="text-sm font-bold border-r border-white/20 pr-4">
                已选中 {selectedIds.size} 项
              </span>
              <div className="flex items-center gap-2">
                 <Select
                    value={batchStatus}
                    onChange={e => setBatchStatus(e.target.value)}
                    className="h-8 w-32 text-xs bg-white/10 text-white border-none focus:ring-0"
                 >
                    <option value="" className="bg-zinc-900 text-white">批量修改状态...</option>
                    {Object.entries(statusLabels).map(([v, l]) => (
                       <option key={v} value={v} className="bg-zinc-900 text-white">{l}</option>
                    ))}
                 </Select>
                 <Button
                    variant="primary"
                    size="sm"
                    className="h-8 rounded-full px-4"
                    onClick={batchChangeStatus}
                    loading={batchUpdating}
                 >
                    立即执行
                 </Button>
                 <Button
                    variant="ghost"
                    size="sm"
                    className="h-8 w-8 p-0 text-white/70 hover:bg-white/10 hover:text-white rounded-full"
                    onClick={() => setSelectedIds(new Set())}
                 >
                    <X className="h-4 w-4" />
                 </Button>
              </div>
           </div>
        </div>
      )}
      {detailOpen && currentFinding && (
        <FindingDetail
          finding={currentFinding.finding}
          evidence={currentFinding.evidence}
          onClose={closeDetail}
          onChangeStatus={changeStatus}
        />
      )}
    </div>
  );
}

function FindingDetail({
  finding,
  evidence,
  onClose,
  onChangeStatus,
}: {
  finding: Finding;
  evidence: Evidence[];
  onClose: () => void;
  onChangeStatus: (id: string, status: string) => void;
}) {
  const [note, setNote] = useState("");
  const [adding, setAdding] = useState(false);
  const [trace, setTrace] = useState<ToolCallTrace | null>(null);
  const [traceLoading, setTraceLoading] = useState(true);
  const toast = useToast();

  useEffect(() => {
    let cancelled = false;
    setTraceLoading(true);
    api.getFindingTrace(finding.id)
      .then((data) => {
        if (!cancelled) setTrace(data);
      })
      .catch(() => {
        if (!cancelled) setTrace(null);
      })
      .finally(() => {
        if (!cancelled) setTraceLoading(false);
      });
    return () => {
      cancelled = true;
    };
  }, [finding.id]);

  const addNote = async () => {
    if (!note.trim()) return;
    setAdding(true);
    try {
      await api.addEvidence(finding.id, { type: "note", excerpt: note.trim() });
      setNote("");
      const data = await api.getFinding(finding.id);
      useStore.getState().setCurrentFinding(data);
    } catch (err) {
    } finally {
      setAdding(false);
    }
  };

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center p-4 bg-background/80 backdrop-blur-sm animate-in fade-in duration-200">
      <Card className="w-full max-w-4xl max-h-[90vh] flex flex-col shadow-2xl animate-in zoom-in-95 duration-200">
        <CardHeader className="flex flex-row items-center justify-between border-b pb-4">
          <div className="flex items-center gap-3">
             <div className="h-10 w-10 rounded-lg bg-primary/10 flex items-center justify-center">
                <FileText className="h-6 w-6 text-primary" />
             </div>
             <div>
                <CardTitle className="text-xl">漏洞详情报告</CardTitle>
                <CardDescription className="text-xs font-mono uppercase tracking-tighter">#{finding.id.slice(-12)}</CardDescription>
             </div>
          </div>
          <Button variant="ghost" size="sm" className="h-8 w-8 p-0" onClick={onClose}>
             <X className="h-5 w-5" />
          </Button>
        </CardHeader>
        
        <div className="flex-1 overflow-y-auto p-6 custom-scrollbar">
          <div className="grid grid-cols-1 md:grid-cols-3 gap-6 mb-8">
            <Card className="bg-muted/30 border-none">
                <CardContent className="p-4">
                    <div className="text-[10px] font-bold text-muted-foreground uppercase mb-2">Severity</div>
                    <SeverityBadge severity={finding.severity} className="h-7 px-3 text-sm" />
                </CardContent>
            </Card>
            <Card className="bg-muted/30 border-none">
                <CardContent className="p-4">
                    <div className="text-[10px] font-bold text-muted-foreground uppercase mb-2">Source Tool</div>
                    <div className="text-lg font-bold text-foreground">{finding.source_tool}</div>
                    <div className="text-[10px] text-muted-foreground font-mono truncate">{finding.source_rule_id || 'N/A'}</div>
                </CardContent>
            </Card>
            <Card className="bg-muted/30 border-none">
                <CardContent className="p-4">
                    <div className="text-[10px] font-bold text-muted-foreground uppercase mb-2">Current Status</div>
                    <StatusBadge status={finding.status} />
                </CardContent>
            </Card>
          </div>

          <div className="space-y-8">
             <section>
                <h3 className="text-sm font-bold flex items-center gap-2 mb-3">
                    <Activity className="h-4 w-4 text-primary" />
                    调用溯源
                </h3>
                <div className="rounded-xl border p-4 bg-muted/10 space-y-2 text-sm">
                    {traceLoading ? (
                        <p className="text-xs text-muted-foreground italic">加载溯源链...</p>
                    ) : trace?.run || trace?.tool_call_log ? (
                        <>
                            {trace.run && (
                                <div className="flex items-center gap-2 text-xs">
                                    <span className="text-muted-foreground">Run</span>
                                    <span className="font-mono font-semibold">{trace.run.id.slice(-12)}</span>
                                    <Badge variant="outline" className="h-4 px-1 text-[9px] uppercase">{trace.run.status}</Badge>
                                </div>
                            )}
                            {trace.tool_call_log && (
                                <div className="flex flex-wrap items-center gap-2 text-xs">
                                    <Badge variant="secondary" className="font-mono text-[10px]">{trace.tool_call_log.tool}</Badge>
                                    <span className="text-muted-foreground">{trace.tool_call_log.action}</span>
                                    <Badge variant="outline" className="h-4 px-1 text-[9px]">{trace.tool_call_log.status}</Badge>
                                </div>
                            )}
                            {!trace.tool_call_log && (
                                <p className="text-xs text-muted-foreground">工具调用日志尚未关联到此 Finding</p>
                            )}
                        </>
                    ) : (
                        <p className="text-xs text-muted-foreground italic">暂无溯源数据（可能来自历史扫描或未记录 source_task_id）</p>
                    )}
                </div>
             </section>

             <section>
                <h3 className="text-sm font-bold flex items-center gap-2 mb-3">
                    <MessageSquare className="h-4 w-4 text-primary" />
                    漏洞描述
                </h3>
                <div className="rounded-xl border p-4 bg-card shadow-sm">
                    <h4 className="text-lg font-bold mb-2">{finding.title}</h4>
                    <p className="text-muted-foreground text-sm leading-relaxed whitespace-pre-wrap">
                        {finding.summary || "未提供详细摘要。"}
                    </p>
                </div>
             </section>

             <section>
                <h3 className="text-sm font-bold flex items-center gap-2 mb-3">
                    <ShieldAlert className="h-4 w-4 text-primary" />
                    证据链 (Evidence)
                </h3>
                <div className="space-y-3">
                    {evidence.length === 0 ? (
                        <div className="p-8 text-center border rounded-xl border-dashed text-muted-foreground text-sm italic">
                            暂无原始证据记录
                        </div>
                    ) : (
                        evidence.map((e) => (
                            <div key={e.id} className="rounded-xl border bg-muted/20 overflow-hidden">
                                <div className="px-4 py-2 border-b bg-muted/30 flex items-center justify-between">
                                    <Badge variant="outline" className="text-[9px] uppercase font-bold tracking-widest">{e.type}</Badge>
                                    <span className="text-[10px] text-muted-foreground font-mono">{new Date(e.created_at).toLocaleString()}</span>
                                </div>
                                <div className="p-4 overflow-x-auto">
                                    {e.excerpt ? (
                                        <pre className="text-xs font-mono text-foreground/90 whitespace-pre">
                                            {e.excerpt}
                                        </pre>
                                    ) : (
                                        <span className="text-xs italic text-muted-foreground">Binary or empty content</span>
                                    )}
                                </div>
                            </div>
                        ))
                    )}
                </div>
             </section>

             <section>
                <h3 className="text-sm font-bold flex items-center gap-2 mb-3">
                    <History className="h-4 w-4 text-primary" />
                    工作流更新
                </h3>
                <div className="rounded-xl border p-4 bg-muted/10 space-y-4">
                    <div>
                        <div className="text-[10px] font-bold text-muted-foreground uppercase mb-2 tracking-widest">Update Finding Status</div>
                        <div className="flex flex-wrap gap-2">
                        {["confirmed", "false_positive", "accepted_risk", "ignored", "pending_review"].map((s) => (
                            <Button
                            key={s}
                            variant={finding.status === s ? "primary" : "outline"}
                            size="sm"
                            onClick={() => onChangeStatus(finding.id, s)}
                            className="h-8 text-xs px-3"
                            >
                            {statusLabels[s]}
                            </Button>
                        ))}
                        </div>
                    </div>
                    
                    <div className="pt-4 border-t border-border/50">
                        <div className="text-[10px] font-bold text-muted-foreground uppercase mb-2 tracking-widest">Add Internal Note</div>
                        <div className="flex gap-3">
                            <textarea
                                className="flex min-h-[80px] w-full rounded-md border border-input bg-transparent px-3 py-2 text-sm shadow-sm placeholder:text-muted-foreground focus-visible:outline-none focus-visible:ring-1 focus-visible:ring-ring disabled:cursor-not-allowed disabled:opacity-50"
                                value={note}
                                onChange={(e) => setNote(e.target.value)}
                                placeholder="输入关于此漏洞的内部审计备注..."
                            />
                        </div>
                        <div className="flex justify-end mt-2">
                             <Button
                                onClick={addNote}
                                disabled={adding || !note.trim()}
                                variant="secondary"
                                size="sm"
                             >
                                <Plus className="mr-2 h-3.5 w-3.5" />
                                提交备注
                             </Button>
                        </div>
                    </div>
                </div>
             </section>
          </div>
        </div>

        <div className="p-4 border-t bg-muted/30 flex justify-between items-center">
            <Button variant="secondary" size="sm" onClick={async () => {
                try {
                    await api.retestFinding(finding.id);
                    toast("漏洞复测任务已下发至 Worker", "success");
                } catch (e) {
                    toast("复测启动失败", "error");
                }
            }}>
                <History className="mr-2 h-4 w-4" />
                漏洞复测 (Retest)
            </Button>
            <Button variant="secondary" size="sm" onClick={onClose}>
                完成审计并关闭
            </Button>
        </div>
      </Card>
    </div>
  );
}

const statusOptions = [
  { value: "pending_review", label: "待审核" },
  { value: "confirmed", label: "已确认" },
  { value: "false_positive", label: "误报" },
  { value: "accepted_risk", label: "已接受风险" },
  { value: "ignored", label: "已忽略" },
];

function formatTimeAgo(timestamp: number): string {
  const now = Date.now();
  const diff = now - timestamp;
  const minutes = Math.floor(diff / 60000);
  if (minutes < 1) return "刚刚";
  if (minutes < 60) return `${minutes} 分钟前`;
  const hours = Math.floor(minutes / 60);
  if (hours < 24) return `${hours} 小时前`;
  const days = Math.floor(hours / 24);
  return `${days} 天前`;
}
