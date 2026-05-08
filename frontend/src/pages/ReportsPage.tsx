import { useState, useEffect, useCallback, useMemo } from "react";
import { useNavigate } from "react-router-dom";
import { api, Finding, API_BASE, PAGE_ALL } from "../lib/api";
import { getApiToken } from "../lib/config";
import { renderMarkdown } from "../lib/markdown";
import { Button } from "../components/Button";
import { SeverityBadge, StatusBadge } from "../components/Badge";
import { EmptyState, SkeletonCard } from "../components";
import { useProjectId, useToast } from "../components";
import { useStore } from "../lib/store";

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

/** Severity config for summary cards. */
const SEVERITY_META: { key: string; label: string; colorClass: string }[] = [
  { key: "critical", label: "严重", colorClass: "text-brand-danger bg-brand-danger/10 border-brand-danger/20" },
  { key: "high", label: "高危", colorClass: "text-brand-warning bg-brand-warning/10 border-brand-warning/20" },
  { key: "medium", label: "中危", colorClass: "text-accent-yellow bg-accent-yellow/10 border-accent-yellow/20" },
  { key: "low", label: "低危", colorClass: "text-accent-teal bg-accent-teal/10 border-accent-teal/20" },
  { key: "info", label: "信息", colorClass: "text-text-tertiary bg-white/[0.04] border-white/[0.08]" },
];

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
      const blob =
        format === "md"
          ? await api.exportReportMD(projectId)
          : await api.exportReportJSON(projectId);

      const filename = `report_${projectId}.${format}`;

      if (isTauri()) {
        const saved = await saveWithTauriDialog(blob, filename);
        if (saved) {
          toast("导出成功", "success");
        }
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
    if (el) {
      el.scrollIntoView({ behavior: "smooth", block: "start" });
    }
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
      <div className="page-shell">
        <EmptyState
          title="请先选择一个项目"
          description="选择一个项目后生成和导出报告"
          actionLabel="前往项目列表"
          onAction={() => navigate("/projects")}
        />
      </div>
    );
  }

  return (
    <div className="page-shell space-y-6">
      <div className="page-header">
        <div>
          <div className="page-eyebrow text-brand-purple">Step 5</div>
          <h1 className="page-title">安全评估报告</h1>
          <p className="page-subtitle">
            <span className="font-mono text-text-secondary">{confirmedCount}</span> 个确认漏洞，
            <span className="font-mono text-text-secondary">{acceptedCount}</span> 个接受风险可进入交付报告。
          </p>
        </div>
      </div>

      {/* Findings Summary */}
      {!loading && findings.length > 0 && (
        <div className="flex flex-wrap gap-2">
          {SEVERITY_META.map(({ key, label, colorClass }) => {
            const count = severityCounts[key] || 0;
            return (
              <div
                key={key}
                className={`flex items-center gap-2 px-3 py-1.5 rounded-lg border text-sm ${colorClass}`}
              >
                <span className="font-medium">{label}</span>
                <span className="font-mono font-bold">{count}</span>
              </div>
            );
          })}
        </div>
      )}

      {/* Operation area */}
      <div className="flex gap-2">
        <Button
          variant="secondary"
          size="sm"
          onClick={handlePreviewMarkdown}
          disabled={loading || showPreview}
          title={showPreview ? "预览已打开" : "生成 Markdown 预览"}
        >
          {showPreview ? "已生成预览" : "Markdown 预览"}
        </Button>
        <Button
          variant="primary"
          size="sm"
          onClick={() => handleExport("md")}
          disabled={exporting !== null}
          title={findings.length === 0 ? "无 findings 可导出" : "导出 Markdown 报告"}
        >
          {exporting === "md" ? "导出中..." : "导出 Markdown"}
        </Button>
        <Button
          variant="secondary"
          size="sm"
          onClick={() => handleExport("json")}
          disabled={exporting !== null}
          title={findings.length === 0 ? "无 findings 可导出" : "导出 JSON 报告"}
        >
          {exporting === "json" ? "导出中..." : "导出 JSON"}
        </Button>
      </div>

      {/* Status area */}
      {error && (
        <div className="bg-brand-danger/10 border border-brand-danger/20 text-brand-danger px-4 py-3 rounded-lg">
          {error}
        </div>
      )}
      {loading && (
        <div className="space-y-4">
          {Array.from({ length: 3 }).map((_, i) => (
            <SkeletonCard key={i} lines={4} />
          ))}
        </div>
      )}

      {!loading && findings.length === 0 && !error && (
        <EmptyState
          title="暂无可报告的发现"
          description="当前项目还没有 confirmed 或 accepted_risk 状态的 finding。请先运行扫描任务并审核 findings。"
        />
      )}

      {/* Report Outline */}
      {!loading && findings.length > 0 && (
        <div className="mb-6">
          <h2 className="text-sm font-semibold text-text-secondary mb-3">报告大纲</h2>

          {/* Section overview */}
          <div className="flex flex-wrap gap-3 mb-4 text-sm text-text-quaternary">
            <span className="chip">概览</span>
            <span className="chip">范围</span>
            <span className="chip">方法</span>
            <span className="chip">风险统计</span>
            <span className="chip">漏洞详情</span>
            <span className="chip">接受风险</span>
            <span className="chip">附录</span>
          </div>

          {/* Findings outline — clickable */}
          {findings.length > 0 && (
            <div className="panel p-3">
              <h3 className="text-xs font-medium text-text-tertiary mb-2 uppercase tracking-wider">
                Findings 列表
              </h3>
              <div className="space-y-1 max-h-48 overflow-y-auto">
                {findings.map((fd, idx) => (
                  <button
                    key={fd.finding.id}
                    onClick={() => scrollToFinding(fd.finding.id)}
                    className="w-full flex items-center gap-2 px-2 py-1.5 rounded hover:bg-brand-primary/[0.055] transition-colors text-left"
                    title={`查看 ${fd.finding.title}`}
                  >
                    <span className="text-xs text-text-quaternary font-mono w-5 shrink-0">
                      {idx + 1}.
                    </span>
                    <SeverityBadge severity={fd.finding.severity} />
                    <span className="text-sm text-text-secondary truncate">
                      {fd.finding.title}
                    </span>
                  </button>
                ))}
              </div>
            </div>
          )}

          {/* Finding List */}
          <div className="space-y-3 mt-4">
            {findings.map((fd) => (
              <div
                key={fd.finding.id}
                id={`finding-${fd.finding.id}`}
                className="panel p-4 hover:border-brand-primary/35 transition-all scroll-mt-6"
              >
                <div className="flex items-center justify-between">
                  <div className="flex items-center gap-3">
                    <SeverityBadge severity={fd.finding.severity} />
                    <span className="font-medium text-text-secondary">{fd.finding.title}</span>
                  </div>
                  <div className="flex items-center gap-2">
                    <StatusBadge status={fd.finding.status} />
                    <span className="text-xs text-text-quaternary font-mono">
                      可信度 {fd.finding.confidence}
                    </span>
                  </div>
                </div>
                {fd.finding.summary && (
                  <p className="text-sm text-text-tertiary mt-2 line-clamp-2">{fd.finding.summary}</p>
                )}
                <div className="flex items-center gap-2 mt-2 text-xs text-text-quaternary">
                  <span>来源: {fd.finding.source_tool}</span>
                  {fd.evidence.length > 0 && (
                    <span>• {fd.evidence.length} 条证据</span>
                  )}
                </div>
              </div>
            ))}
          </div>
        </div>
      )}

      {/* Markdown Preview Panel */}
      {showPreview && previewText && (
        <div className="panel overflow-hidden">
          <div className="flex items-center justify-between px-4 py-3 border-b border-brand-primary/10 bg-brand-primary/[0.045]">
            <span className="font-medium text-sm text-text-secondary">Markdown 报告预览</span>
            <div className="flex items-center gap-2">
              {/* Mode toggle */}
              <div className="flex bg-surface-elevated-2/70 rounded-lg p-0.5 ring-1 ring-brand-primary/10">
                <button
                  onClick={() => setPreviewMode("rendered")}
                  className={`px-3 py-1 text-xs rounded-md transition-colors ${
                    previewMode === "rendered"
                      ? "bg-brand-primary/15 text-text-primary"
                      : "text-text-tertiary hover:text-text-secondary"
                  }`}
                  aria-pressed={previewMode === "rendered"}
                >
                  预览
                </button>
                <button
                  onClick={() => setPreviewMode("raw")}
                  className={`px-3 py-1 text-xs rounded-md transition-colors ${
                    previewMode === "raw"
                      ? "bg-brand-primary/15 text-text-primary"
                      : "text-text-tertiary hover:text-text-secondary"
                  }`}
                  aria-pressed={previewMode === "raw"}
                >
                  原始
                </button>
              </div>
              <button
                onClick={() => { setShowPreview(false); setPreviewText(null); setPreviewRawText(null); }}
                className="text-text-quaternary hover:text-text-secondary transition-colors ml-1"
                aria-label="关闭预览"
              >
                ✕
              </button>
            </div>
          </div>
          <div className="p-4 overflow-auto max-h-[70vh]">
            {previewMode === "rendered" ? (
              <div
                className="prose prose-invert prose-sm max-w-none prose-headings:text-text-primary prose-p:text-text-secondary prose-a:text-brand-primary hover:prose-a:text-brand-primary/80 prose-strong:text-text-primary prose-code:text-text-secondary prose-code:bg-surface-elevated-2 prose-pre:bg-surface-elevated-2/80 prose-pre:text-text-secondary prose-li:text-text-secondary"
                dangerouslySetInnerHTML={{ __html: renderMarkdown(previewText) }}
              />
            ) : (
              <pre className="text-xs font-mono whitespace-pre-wrap text-text-secondary">
                {previewRawText}
              </pre>
            )}
          </div>
        </div>
      )}
    </div>
  );
}
