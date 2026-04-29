import { useState, useEffect, useCallback } from "react";
import { Link, useNavigate } from "react-router-dom";
import { api, Finding, API_BASE } from "../lib/api";
import { Button } from "../components/Button";
import { SeverityBadge, StatusBadge } from "../components/Badge";
import { EmptyState } from "../components/EmptyState";
import { useProjectId } from "../components";

// Extended type to include finding details.
interface FindingDetail {
  finding: Finding;
  evidence: { id: string; type: string; excerpt?: string }[];
}

export default function ReportsPage() {
  const projectId = useProjectId();
  const navigate = useNavigate();

  const [findings, setFindings] = useState<FindingDetail[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [previewText, setPreviewText] = useState<string | null>(null);
  const [previewRawText, setPreviewRawText] = useState<string | null>(null);
  const [showPreview, setShowPreview] = useState(false);
  const [exporting, setExporting] = useState<string | null>(null);

  useEffect(() => {
    if (!projectId) return;
    loadData();
  }, [projectId]);

  const loadData = async () => {
    if (!projectId) return;
    try {
      setLoading(true);
      setError(null);

      // Fetch report-eligible findings (confirmed + accepted_risk).
      const allFindings = await api.listFindings(projectId);
      const reportFindings = (allFindings ?? []).filter(
        (f) => f.status === "confirmed" || f.status === "accepted_risk"
      );

      // Enrich with evidence.
      const enriched: FindingDetail[] = [];
      for (const f of reportFindings) {
        try {
          const detail = await api.getFinding(f.id);
          enriched.push({
            finding: f,
            evidence: detail.evidence || [],
          });
        } catch {
          enriched.push({ finding: f, evidence: [] });
        }
      }

      setFindings(enriched);
    } catch (e: any) {
      setError(e.message || "Failed to load findings");
    } finally {
      setLoading(false);
    }
  };

  const handlePreviewMarkdown = useCallback(async () => {
    if (!projectId) return;
    try {
      setPreviewRawText(null);
      const res = await fetch(`${API_BASE}/projects/${projectId}/reports/export.md`);
      if (!res.ok) {
        const data = await res.json().catch(() => null);
        throw new Error(data?.error?.message || `Failed: ${res.status}`);
      }
      const text = await res.text();
      setPreviewRawText(text);
      setPreviewText(text);
      setShowPreview(true);
    } catch (e: any) {
      setError(e.message || "Failed to generate preview");
    }
  }, [projectId]);

  const handleExport = async (format: "md" | "json") => {
    if (!projectId) return;
    try {
      setExporting(format);
      setError(null);
      const blob =
        format === "md"
          ? await api.exportReportMD(projectId)
          : await api.exportReportJSON(projectId);

      const url = URL.createObjectURL(blob);
      const a = document.createElement("a");
      a.href = url;
      a.download = `report_${projectId}.${format}`;
      document.body.appendChild(a);
      a.click();
      document.body.removeChild(a);
      URL.revokeObjectURL(url);
    } catch (e: any) {
      setError(e.message || `Export ${format.toUpperCase()} failed`);
    } finally {
      setExporting(null);
    }
  };

  const confirmedCount = findings.filter((f) => f.finding.status === "confirmed").length;
  const acceptedCount = findings.filter((f) => f.finding.status === "accepted_risk").length;



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
    <div className="max-w-5xl mx-auto">
      {/* Header */}
      <div className="flex items-center justify-between mb-6">
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
        <div className="flex gap-2">
          <Button
            variant="secondary"
            size="sm"
            onClick={handlePreviewMarkdown}
            disabled={loading || showPreview}
          >
            {showPreview ? "已生成预览" : "Markdown 预览"}
          </Button>
          <Button
            variant="primary"
            size="sm"
            onClick={() => handleExport("md")}
            disabled={exporting !== null}
          >
            {exporting === "md" ? "导出中..." : "导出 Markdown"}
          </Button>
          <Button
            variant="secondary"
            size="sm"
            onClick={() => handleExport("json")}
            disabled={exporting !== null}
          >
            {exporting === "json" ? "导出中..." : "导出 JSON"}
          </Button>
        </div>
      </div>

      {/* Error */}
      {error && (
        <div className="bg-brand-danger/10 border border-brand-danger/20 text-brand-danger px-4 py-3 rounded-lg mb-4">
          {error}
        </div>
      )}

      {/* Loading */}
      {loading && (
        <div className="text-center text-zinc-500 py-12">加载中...</div>
      )}

      {/* Report Outline */}
      {!loading && (
        <div className="mb-6">
          <div className="flex gap-4 mb-4 text-sm text-zinc-500">
            <span>📋 概览</span>
            <span>📐 范围</span>
            <span>🔬 方法</span>
            <span>📊 风险统计</span>
            <span>🐛 漏洞详情</span>
            <span>✅ 接受风险</span>
            <span>📎 附录</span>
          </div>

          {/* Finding List */}
          <div className="space-y-2">
            {findings.length === 0 && (
              <div className="text-center text-zinc-500 py-8">
                暂无可报告的发现 (confirmed/accepted_risk)
              </div>
            )}
            {findings.map((fd) => (
              <div
                key={fd.finding.id}
                className="bg-zinc-900/50 backdrop-blur-md border border-zinc-800/80 rounded-xl p-4 hover:border-zinc-700/80 transition-all"
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

      {/* Markdown Preview Modal */}
      {showPreview && previewText && (
        <div className="bg-zinc-900/50 backdrop-blur-md border border-zinc-800/80 rounded-xl overflow-hidden">
          <div className="flex items-center justify-between px-4 py-3 bg-zinc-900/80 border-b border-zinc-800/60">
            <span className="font-medium text-sm text-zinc-200">Markdown 报告预览</span>
            <button
              onClick={() => { setShowPreview(false); setPreviewText(null); setPreviewRawText(null); }}
              className="text-zinc-500 hover:text-zinc-300 transition-colors"
            >
              ✕
            </button>
          </div>
          <div className="p-4 overflow-auto max-h-[70vh]">
            <pre className="text-xs font-mono whitespace-pre-wrap text-zinc-300">
              {previewRawText}
            </pre>
          </div>
        </div>
      )}
    </div>
  );
}
