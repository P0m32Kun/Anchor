import { useState, useEffect, useCallback, useMemo } from "react";
import { Link, useNavigate } from "react-router-dom";
import { api, Finding, API_BASE } from "../lib/api";
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
  { key: "critical", label: "严重", colorClass: "text-red-400 bg-red-400/10 border-red-400/20" },
  { key: "high", label: "高危", colorClass: "text-orange-400 bg-orange-400/10 border-orange-400/20" },
  { key: "medium", label: "中危", colorClass: "text-yellow-400 bg-yellow-400/10 border-yellow-400/20" },
  { key: "low", label: "低危", colorClass: "text-teal-400 bg-teal-400/10 border-teal-400/20" },
  { key: "info", label: "信息", colorClass: "text-zinc-400 bg-zinc-400/10 border-zinc-400/20" },
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

      const allFindings = await api.listFindings(projectId, undefined, signal);
      const reportFindings = (allFindings ?? []).filter(
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
      <div className="max-w-5xl mx-auto">
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
    <div className="max-w-5xl mx-auto space-y-6">
      {/* Title area */}
      <div>
        <Link to={`/projects/${projectId}`} className="text-sm text-brand-primary hover:text-brand-primary/80 mb-1 block transition-colors">
          ← 返回项目
        </Link>
        <h1 className="text-2xl font-bold text-zinc-100 tracking-tight">安全评估报告</h1>
        <p className="text-zinc-500 text-sm mt-1">
          <span className="font-mono text-zinc-300">{confirmedCount}</span> 个确认漏洞，
          <span className="font-mono text-zinc-300">{acceptedCount}</span> 个接受风险
        </p>
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
          <h2 className="text-sm font-semibold text-zinc-300 mb-3">报告大纲</h2>

          {/* Section overview */}
          <div className="flex flex-wrap gap-3 mb-4 text-sm text-zinc-500">
            <span className="flex items-center gap-1">📋 概览</span>
            <span className="flex items-center gap-1">📐 范围</span>
            <span className="flex items-center gap-1">🔬 方法</span>
            <span className="flex items-center gap-1">📊 风险统计</span>
            <span className="flex items-center gap-1">🐛 漏洞详情</span>
            <span className="flex items-center gap-1">✅ 接受风险</span>
            <span className="flex items-center gap-1">📎 附录</span>
          </div>

          {/* Findings outline — clickable */}
          {findings.length > 0 && (
            <div className="bg-zinc-900/30 border border-zinc-800/60 rounded-lg p-3">
              <h3 className="text-xs font-medium text-zinc-400 mb-2 uppercase tracking-wider">
                Findings 列表
              </h3>
              <div className="space-y-1 max-h-48 overflow-y-auto">
                {findings.map((fd, idx) => (
                  <button
                    key={fd.finding.id}
                    onClick={() => scrollToFinding(fd.finding.id)}
                    className="w-full flex items-center gap-2 px-2 py-1.5 rounded hover:bg-zinc-800/60 transition-colors text-left"
                    title={`查看 ${fd.finding.title}`}
                  >
                    <span className="text-xs text-zinc-500 font-mono w-5 shrink-0">
                      {idx + 1}.
                    </span>
                    <SeverityBadge severity={fd.finding.severity} />
                    <span className="text-sm text-zinc-300 truncate">
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
                className="bg-zinc-900/50 backdrop-blur-md border border-zinc-800/80 rounded-xl p-4 hover:border-zinc-700/80 transition-all scroll-mt-6"
              >
                <div className="flex items-center justify-between">
                  <div className="flex items-center gap-3">
                    <SeverityBadge severity={fd.finding.severity} />
                    <span className="font-medium text-zinc-200">{fd.finding.title}</span>
                  </div>
                  <div className="flex items-center gap-2">
                    <StatusBadge status={fd.finding.status} />
                    <span className="text-xs text-zinc-500 font-mono">
                      可信度 {fd.finding.confidence}
                    </span>
                  </div>
                </div>
                {fd.finding.summary && (
                  <p className="text-sm text-zinc-400 mt-2 line-clamp-2">{fd.finding.summary}</p>
                )}
                <div className="flex items-center gap-2 mt-2 text-xs text-zinc-500">
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
        <div className="bg-zinc-900/50 backdrop-blur-md border border-zinc-800/80 rounded-xl overflow-hidden">
          <div className="flex items-center justify-between px-4 py-3 bg-zinc-900/80 border-b border-zinc-800/60">
            <span className="font-medium text-sm text-zinc-200">Markdown 报告预览</span>
            <div className="flex items-center gap-2">
              {/* Mode toggle */}
              <div className="flex bg-zinc-800/80 rounded-lg p-0.5">
                <button
                  onClick={() => setPreviewMode("rendered")}
                  className={`px-3 py-1 text-xs rounded-md transition-colors ${
                    previewMode === "rendered"
                      ? "bg-zinc-700 text-zinc-100"
                      : "text-zinc-400 hover:text-zinc-200"
                  }`}
                  aria-pressed={previewMode === "rendered"}
                >
                  预览
                </button>
                <button
                  onClick={() => setPreviewMode("raw")}
                  className={`px-3 py-1 text-xs rounded-md transition-colors ${
                    previewMode === "raw"
                      ? "bg-zinc-700 text-zinc-100"
                      : "text-zinc-400 hover:text-zinc-200"
                  }`}
                  aria-pressed={previewMode === "raw"}
                >
                  原始
                </button>
              </div>
              <button
                onClick={() => { setShowPreview(false); setPreviewText(null); setPreviewRawText(null); }}
                className="text-zinc-500 hover:text-zinc-300 transition-colors ml-1"
                aria-label="关闭预览"
              >
                ✕
              </button>
            </div>
          </div>
          <div className="p-4 overflow-auto max-h-[70vh]">
            {previewMode === "rendered" ? (
              <div
                className="prose prose-invert prose-sm max-w-none prose-headings:text-zinc-100 prose-p:text-zinc-300 prose-a:text-brand-primary hover:prose-a:text-brand-primary/80 prose-strong:text-zinc-200 prose-code:text-zinc-200 prose-code:bg-zinc-800 prose-pre:bg-zinc-800/80 prose-pre:text-zinc-300 prose-li:text-zinc-300"
                dangerouslySetInnerHTML={{ __html: renderMarkdown(previewText) }}
              />
            ) : (
              <pre className="text-xs font-mono whitespace-pre-wrap text-zinc-300">
                {previewRawText}
              </pre>
            )}
          </div>
        </div>
      )}
    </div>
  );
}
