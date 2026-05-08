import { useState, useEffect, useCallback, useMemo } from "react";
import { Link } from "react-router-dom";
import { api, Finding, API_BASE, PAGE_ALL } from "../lib/api";
import { getApiToken } from "../lib/config";
import { renderMarkdown } from "../lib/markdown";
import {
  Button,
  SeverityBadge,
  StatusBadge,
  EmptyState,
  SkeletonList,
  useProjectId,
  useToast,
  Card,
  CardHeader,
  CardTitle,
  CardDescription,
  CardContent,
  Badge
} from "../components";
import { useStore } from "../lib/store";
import {
  FileText,
  Download,
  Eye,
  FileJson,
  FileCode,
  ChevronRight,
  CheckCircle2,
  ShieldCheck,
  AlertTriangle,
  Info,
  ListOrdered,
  BookOpen,
  X
} from "lucide-react";
import { cn } from "../lib/utils";

interface FindingDetail {
  finding: Finding;
  evidence: { id: string; type: string; excerpt?: string }[];
}

function isTauri(): boolean {
  return !!(window as any).__TAURI_INTERNALS__ || !!(window as any).__TAURI__;
}

async function saveWithTauriDialog(blob: Blob, defaultName: string): Promise<boolean> {
  const { save } = await import("@tauri-apps/plugin-dialog");
  const { invoke } = await import("@tauri-apps/api/core");

  const ext = defaultName.split(".").pop() || "";
  const filters =
    ext === "md"
      ? [{ name: "Markdown", extensions: ["md"] }]
      : [{ name: "JSON", extensions: ["json"] }];

  const filePath = await save({
    defaultPath: defaultName,
    filters,
  });

  if (!filePath) {
    return false;
  }

  const arrayBuffer = await blob.arrayBuffer();
  const contents = Array.from(new Uint8Array(arrayBuffer));
  await invoke("save_file", { path: filePath, contents });
  return true;
}

function downloadWithAnchor(blob: Blob, filename: string): void {
  try {
    const url = URL.createObjectURL(blob);
    const a = document.createElement("a");
    a.href = url;
    a.download = filename;
    document.body.appendChild(a);
    a.click();
    document.body.removeChild(a);
    URL.revokeObjectURL(url);
  } catch (e) {
    throw new Error("浏览器下载失败: " + (e instanceof Error ? e.message : String(e)));
  }
}

const SEVERITY_META: Record<string, { label: string; icon: any; color: string; bg: string }> = {
  critical: { label: "严重", icon: AlertTriangle, color: "text-rose-500", bg: "bg-rose-500/10 border-rose-500/20" },
  high: { label: "高危", icon: ShieldCheck, color: "text-orange-500", bg: "bg-orange-500/10 border-orange-500/20" },
  medium: { label: "中危", icon: ShieldCheck, color: "text-amber-500", bg: "bg-amber-500/10 border-amber-500/20" },
  low: { label: "低危", icon: Info, color: "text-cyan-500", bg: "bg-cyan-500/10 border-cyan-500/20" },
  info: { label: "信息", icon: Info, color: "text-muted-foreground", bg: "bg-muted/30 border-border/50" },
};

export default function ReportsPage() {
  const projectId = useProjectId();
  const navigate = useNavigate();
  const toast = useToast();

  const [findings, setFindings] = useState<FindingDetail[]>([]);
  const loading = useStore((state) => state.reportsLoading);
  const error = useStore((state) => state.reportsError);
  const setReportsLoading = useStore((state) => state.setReportsLoading);
  const setReportsError = useStore((state) => state.setReportsError);
  const [previewText, setPreviewText] = useState<string | null>(null);
  const [previewRawText, setPreviewRawText] = useState<string | null>(null);
  const [showPreview, setShowPreview] = useState(false);
  const [previewMode, setPreviewMode] = useState<"rendered" | "raw">("rendered");
  const [exporting, setExporting] = useState<string | null>(null);

  useEffect(() => {
    if (!projectId) return;
    const ctrl = new AbortController();
    loadData(ctrl.signal);
    return () => ctrl.abort();
  }, [projectId]);

  const loadData = async (signal?: AbortSignal) => {
    if (!projectId) return;
    try {
      setReportsLoading(true);
      setReportsError(null);

      const allFindings = await api.listFindings(projectId, undefined, PAGE_ALL, signal);
      const reportFindings = (allFindings.data ?? []).filter(
        (f) => f.status === "confirmed" || f.status === "accepted_risk"
      );

      const enriched: FindingDetail[] = [];
      for (const f of reportFindings) {
        try {
          const detail = await api.getFinding(f.id, signal);
          enriched.push({
            finding: f,
            evidence: detail.evidence || [],
          });
        } catch {
          enriched.push({ finding: f, evidence: [] });
        }
      }

      setFindings(enriched);
    } catch (e: unknown) {
      if (e instanceof DOMException && e.name === "AbortError") return;
      const msg = e instanceof Error ? e.message : "Failed to load findings";
      setReportsError(msg);
    } finally {
      setReportsLoading(false);
    }
  };

  const handlePreviewMarkdown = useCallback(async () => {
    if (!projectId) return;
    try {
      setPreviewRawText(null);
      const token = getApiToken();
      const headers: Record<string, string> = token ? { Authorization: `Bearer ${token}` } : {};
      const res = await fetch(`${API_BASE}/projects/${projectId}/reports/export.md`, { headers });
      if (!res.ok) {
        const data = await res.json().catch(() => null);
        throw new Error(data?.error?.message || `Failed: ${res.status}`);
      }
      const text = await res.text();
      setPreviewRawText(text);
      setPreviewText(text);
      setShowPreview(true);
      setPreviewMode("rendered");
    } catch (e: unknown) {
      const msg = e instanceof Error ? e.message : "Failed to generate preview";
      setReportsError(msg);
      toast("预览生成失败：" + msg, "error");
    }
  }, [projectId, toast]);

  const handleExport = async (format: "md" | "json") => {
    if (!projectId) return;
    if (findings.length === 0) {
      toast("无 findings 可导出", "warning");
      return;
    }
    try {
      setExporting(format);
      setReportsError(null);
      const blob = format === "md" ? await api.exportReportMD(projectId) : await api.exportReportJSON(projectId);
      const filename = `report_${projectId}.${format}`;
      if (isTauri()) {
        const saved = await saveWithTauriDialog(blob, filename);
        if (saved) toast("导出成功", "success");
      } else {
        downloadWithAnchor(blob, filename);
        toast("下载已启动", "success");
      }
    } catch (e: unknown) {
      const msg = e instanceof Error ? e.message : `Export ${format.toUpperCase()} failed`;
      setReportsError(msg);
      toast("导出失败：" + msg, "error");
    } finally {
      setExporting(null);
    }
  };

  const scrollToFinding = useCallback((id: string) => {
    const el = document.getElementById(`finding-${id}`);
    if (el) el.scrollIntoView({ behavior: "smooth", block: "start" });
  }, []);

  const confirmedCount = findings.filter((f) => f.finding.status === "confirmed").length;
  const acceptedCount = findings.filter((f) => f.finding.status === "accepted_risk").length;

  const severityCounts = useMemo(() => {
    const counts: Record<string, number> = {};
    for (const fd of findings) {
      const s = fd.finding.severity;
      counts[s] = (counts[s] || 0) + 1;
    }
    return counts;
  }, [findings]);

  if (!projectId) {
    return (
        <div className="flex flex-col items-center justify-center py-20 text-center">
          <div className="h-16 w-16 rounded-full bg-muted flex items-center justify-center mb-4">
              <FileText className="h-8 w-8 text-muted-foreground opacity-50" />
          </div>
          <h2 className="text-xl font-bold">评估报告</h2>
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
          <div className="flex items-center gap-2 text-primary font-bold text-xs uppercase tracking-widest mb-1.5">
            <CheckCircle2 className="h-3.5 w-3.5" />
            Step 5: Report Delivery
          </div>
          <h1 className="text-3xl font-bold tracking-tight text-foreground">安全评估报告</h1>
          <p className="text-muted-foreground mt-1">
             汇整所有确认风险，生成标准化的安全测试交付文档。
          </p>
        </div>
        <div className="flex gap-2">
            <Button variant="secondary" size="sm" onClick={handlePreviewMarkdown} disabled={loading || showPreview}>
                <Eye className="mr-2 h-4 w-4" />
                预览 Markdown
            </Button>
            <Button variant="primary" size="sm" onClick={() => handleExport("md")} loading={exporting === 'md'}>
                <Download className="mr-2 h-4 w-4" />
                导出 MD
            </Button>
        </div>
      </div>

      <div className="grid gap-6 md:grid-cols-5">
         {Object.entries(SEVERITY_META).map(([key, meta]) => (
            <Card key={key} className={cn("relative overflow-hidden border-none shadow-sm", meta.bg)}>
                <CardContent className="p-4 flex flex-col items-center justify-center text-center">
                    <meta.icon className={cn("h-5 w-5 mb-2", meta.color)} />
                    <div className="text-[10px] font-bold uppercase opacity-60 mb-1">{meta.label}</div>
                    <div className={cn("text-2xl font-bold tabular-nums", meta.color)}>
                        {severityCounts[key] || 0}
                    </div>
                </CardContent>
            </Card>
         ))}
      </div>

      <div className="grid gap-8 lg:grid-cols-[1fr_360px]">
        <section className="space-y-6">
            <div className="flex items-center justify-between">
                <h2 className="text-xl font-semibold tracking-tight flex items-center gap-2">
                    <BookOpen className="h-5 w-5 text-muted-foreground" />
                    报告内容预览
                </h2>
                <Badge variant="secondary" className="font-mono">{findings.length} Findings Included</Badge>
            </div>

            {loading ? (
                <SkeletonList count={5} />
            ) : findings.length === 0 ? (
                <Card className="border-dashed py-20 text-center">
                    <EmptyState 
                        title="暂无可报告的发现" 
                        description="只有「已确认」或「已接受风险」状态的漏洞会出现在报告中。" 
                    />
                </Card>
            ) : (
                <div className="space-y-4">
                    {findings.map((fd, idx) => (
                        <Card 
                            key={fd.finding.id} 
                            id={`finding-${fd.finding.id}`}
                            className="group transition-all hover:border-primary/50 scroll-mt-24"
                        >
                            <CardHeader className="p-4 pb-2">
                                <div className="flex items-start justify-between">
                                    <div className="flex items-center gap-3">
                                        <span className="text-xs font-mono text-muted-foreground">{idx + 1}.</span>
                                        <SeverityBadge severity={fd.finding.severity} />
                                        <h3 className="font-bold text-foreground">{fd.finding.title}</h3>
                                    </div>
                                    <StatusBadge status={fd.finding.status} />
                                </div>
                            </CardHeader>
                            <CardContent className="p-4 pt-0">
                                {fd.finding.summary && (
                                    <p className="text-sm text-muted-foreground leading-relaxed mt-2 line-clamp-3">
                                        {fd.finding.summary}
                                    </p>
                                )}
                                <div className="flex items-center gap-4 mt-4 text-[11px] text-muted-foreground border-t pt-3">
                                    <div className="flex items-center gap-1">
                                        <FileCode className="h-3 w-3" />
                                        <span>来源: {fd.finding.source_tool}</span>
                                    </div>
                                    <div className="flex items-center gap-1">
                                        <ShieldCheck className="h-3 w-3" />
                                        <span>置信度: {fd.finding.confidence}</span>
                                    </div>
                                    {fd.evidence.length > 0 && (
                                        <div className="flex items-center gap-1 text-primary">
                                            <CheckCircle2 className="h-3 w-3" />
                                            <span>{fd.evidence.length} 条证据链</span>
                                        </div>
                                    )}
                                </div>
                            </CardContent>
                        </Card>
                    ))}
                </div>
            )}
        </section>

        <aside className="space-y-6">
            <Card className="bg-muted/30">
                <CardHeader>
                    <CardTitle className="text-base flex items-center gap-2">
                        <ListOrdered className="h-4 w-4 text-primary" />
                        漏洞导航
                    </CardTitle>
                    <CardDescription className="text-xs">
                        快速跳转至特定漏洞详情
                    </CardDescription>
                </CardHeader>
                <CardContent className="p-2 pt-0">
                    <div className="space-y-1 max-h-[400px] overflow-auto custom-scrollbar px-2">
                        {findings.map((fd, idx) => (
                            <button
                                key={fd.finding.id}
                                onClick={() => scrollToFinding(fd.finding.id)}
                                className="w-full flex items-center gap-3 p-2 rounded-md hover:bg-background text-left transition-all group"
                            >
                                <span className="text-[10px] font-mono text-muted-foreground w-4">{idx+1}.</span>
                                <div className={cn(
                                    "h-1.5 w-1.5 rounded-full shrink-0",
                                    fd.finding.severity === 'critical' ? 'bg-rose-500' :
                                    fd.finding.severity === 'high' ? 'bg-orange-500' :
                                    fd.finding.severity === 'medium' ? 'bg-amber-500' : 'bg-muted-foreground'
                                )} />
                                <span className="text-xs text-muted-foreground group-hover:text-foreground truncate flex-1">{fd.finding.title}</span>
                                <ChevronRight className="h-3 w-3 opacity-0 group-hover:opacity-100 transition-all -translate-x-1 group-hover:translate-x-0" />
                            </button>
                        ))}
                        {findings.length === 0 && (
                            <div className="p-4 text-center text-xs text-muted-foreground italic">列表为空</div>
                        )}
                    </div>
                </CardContent>
            </Card>

            <Card>
                <CardHeader>
                    <CardTitle className="text-base flex items-center gap-2">
                        <Download className="h-4 w-4 text-primary" />
                        导出交付
                    </CardTitle>
                </CardHeader>
                <CardContent className="space-y-3">
                    <Button variant="secondary" className="w-full justify-between group" onClick={() => handleExport("md")} loading={exporting === 'md'}>
                        <div className="flex items-center">
                            <FileCode className="mr-2 h-4 w-4 text-muted-foreground group-hover:text-primary" />
                            <span>Markdown (.md)</span>
                        </div>
                        <ArrowRight className="h-3 w-3 opacity-50 group-hover:translate-x-1 transition-transform" />
                    </Button>
                    <Button variant="secondary" className="w-full justify-between group" onClick={() => handleExport("json")} loading={exporting === 'json'}>
                        <div className="flex items-center">
                            <FileJson className="mr-2 h-4 w-4 text-muted-foreground group-hover:text-primary" />
                            <span>JSON Data (.json)</span>
                        </div>
                        <ArrowRight className="h-3 w-3 opacity-50 group-hover:translate-x-1 transition-transform" />
                    </Button>
                    <p className="text-[10px] text-muted-foreground px-1 leading-relaxed">
                        Markdown 格式适用于人工润色排版。JSON 格式适用于与其他自动化风险管理平台集成。
                    </p>
                </CardContent>
            </Card>
        </aside>
      </div>

      {showPreview && previewText && (
        <div className="fixed inset-0 z-50 flex items-center justify-center p-4 bg-background/80 backdrop-blur-sm animate-in fade-in duration-200">
          <Card className="w-full max-w-5xl h-[90vh] flex flex-col shadow-2xl animate-in zoom-in-95 duration-200 overflow-hidden">
            <CardHeader className="flex flex-row items-center justify-between border-b bg-muted/30 py-4">
              <div className="flex items-center gap-3">
                 <div className="h-10 w-10 rounded-lg bg-primary/10 flex items-center justify-center">
                    <Eye className="h-6 w-6 text-primary" />
                 </div>
                 <div>
                    <CardTitle className="text-xl">报告预览模式</CardTitle>
                    <CardDescription className="text-xs">Generated Markdown Report Preview</CardDescription>
                 </div>
              </div>
              <div className="flex items-center gap-2">
                <div className="flex items-center bg-muted p-1 rounded-md border mr-4">
                    <Button 
                        variant={previewMode === 'rendered' ? 'secondary' : 'ghost'} 
                        size="sm" 
                        className="h-7 text-xs px-3"
                        onClick={() => setPreviewMode('rendered')}
                    >
                        Rendered
                    </Button>
                    <Button 
                        variant={previewMode === 'raw' ? 'secondary' : 'ghost'} 
                        size="sm" 
                        className="h-7 text-xs px-3"
                        onClick={() => setPreviewMode('raw')}
                    >
                        Raw Source
                    </Button>
                </div>
                <Button variant="ghost" size="sm" className="h-8 w-8 p-0" onClick={() => setShowPreview(false)}>
                    <X className="h-5 w-5" />
                </Button>
              </div>
            </CardHeader>
            <div className="flex-1 overflow-auto p-8 custom-scrollbar bg-card">
              {previewMode === "rendered" ? (
                <div
                  className="prose prose-invert prose-slate max-w-none 
                    prose-headings:text-foreground prose-headings:font-bold 
                    prose-p:text-muted-foreground prose-strong:text-foreground
                    prose-code:text-primary prose-code:bg-primary/5 prose-code:px-1 prose-code:rounded
                    prose-pre:bg-muted/50 prose-pre:border prose-pre:text-muted-foreground
                    prose-li:text-muted-foreground"
                  dangerouslySetInnerHTML={{ __html: renderMarkdown(previewText) }}
                />
              ) : (
                <pre className="text-xs font-mono whitespace-pre-wrap text-muted-foreground leading-relaxed p-4 bg-muted/20 rounded-lg border">
                  {previewRawText}
                </pre>
              )}
            </div>
            <div className="p-4 border-t bg-muted/30 flex justify-end gap-3">
                <Button variant="ghost" size="sm" onClick={() => setShowPreview(false)}>取消</Button>
                <Button variant="primary" size="sm" onClick={() => handleExport("md")}>
                    <Download className="mr-2 h-4 w-4" />
                    下载此版本 (.md)
                </Button>
            </div>
          </Card>
        </div>
      )}
    </div>
  );
}
