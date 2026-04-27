import { useState, useEffect, useCallback } from "react";
import { useParams, Link } from "react-router-dom";
import { api, Finding } from "../lib/api";

// Extended type to include finding details.
interface FindingDetail {
  finding: Finding;
  evidence: { id: string; type: string; excerpt?: string }[];
}

export default function ReportsPage() {
  const { id } = useParams<{ id: string }>();
  const projectId = id!;

  const [findings, setFindings] = useState<FindingDetail[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [previewText, setPreviewText] = useState<string | null>(null);
  const [previewRawText, setPreviewRawText] = useState<string | null>(null);
  const [showPreview, setShowPreview] = useState(false);
  const [exporting, setExporting] = useState<string | null>(null);

  useEffect(() => {
    loadData();
  }, [projectId]);

  const loadData = async () => {
    try {
      setLoading(true);
      setError(null);

      // Fetch report-eligible findings (confirmed + accepted_risk).
      const allFindings = await api.listFindings(projectId);
      const reportFindings = allFindings.filter(
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
    try {
      setPreviewRawText(null);
      const res = await fetch(`http://localhost:8080/projects/${projectId}/reports/export.md`);
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

  const severityBadge = (sev: string) => {
    const colors: Record<string, string> = {
      critical: "bg-red-700 text-white",
      high: "bg-orange-600 text-white",
      medium: "bg-yellow-500 text-black",
      low: "bg-blue-500 text-white",
      info: "bg-gray-400 text-white",
    };
    return `px-2 py-0.5 rounded text-xs font-medium ${colors[sev] || "bg-gray-300"}`;
  };

  const statusBadge = (status: string) => {
    const colors: Record<string, string> = {
      confirmed: "bg-green-200 text-green-900",
      accepted_risk: "bg-yellow-200 text-yellow-900",
    };
    return `px-2 py-0.5 rounded text-xs font-medium ${colors[status] || "bg-gray-200"}`;
  };

  const confirmedCount = findings.filter((f) => f.finding.status === "confirmed").length;
  const acceptedCount = findings.filter((f) => f.finding.status === "accepted_risk").length;



  return (
    <div className="max-w-5xl mx-auto">
      {/* Header */}
      <div className="flex items-center justify-between mb-6">
        <div>
          <Link to={`/projects/${projectId}`} className="text-sm text-blue-600 hover:underline mb-1 block">
            ← 返回项目
          </Link>
          <h1 className="text-2xl font-bold">安全评估报告</h1>
          <p className="text-gray-500 text-sm mt-1">
            {confirmedCount} 个确认漏洞，{acceptedCount} 个接受风险
          </p>
        </div>
        <div className="flex gap-2">
          <button
            onClick={handlePreviewMarkdown}
            disabled={loading || showPreview}
            className="px-4 py-2 bg-gray-600 text-white rounded-lg hover:bg-gray-700 disabled:opacity-50 text-sm"
          >
            {showPreview ? "已生成预览" : "Markdown 预览"}
          </button>
          <button
            onClick={() => handleExport("md")}
            disabled={exporting !== null}
            className="px-4 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700 disabled:opacity-50 text-sm"
          >
            {exporting === "md" ? "导出中..." : "导出 Markdown"}
          </button>
          <button
            onClick={() => handleExport("json")}
            disabled={exporting !== null}
            className="px-4 py-2 bg-green-600 text-white rounded-lg hover:bg-green-700 disabled:opacity-50 text-sm"
          >
            {exporting === "json" ? "导出中..." : "导出 JSON"}
          </button>
        </div>
      </div>

      {/* Error */}
      {error && (
        <div className="bg-red-50 border border-red-200 text-red-700 px-4 py-3 rounded-lg mb-4">
          {error}
        </div>
      )}

      {/* Loading */}
      {loading && (
        <div className="text-center text-gray-500 py-12">加载中...</div>
      )}

      {/* Report Outline */}
      {!loading && (
        <div className="mb-6">
          <div className="flex gap-4 mb-4 text-sm text-gray-600">
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
              <div className="text-center text-gray-400 py-8">
                暂无可报告的发现 (confirmed/accepted_risk)
              </div>
            )}
            {findings.map((fd) => (
              <div
                key={fd.finding.id}
                className="bg-white border rounded-lg p-4 hover:shadow-sm transition-shadow"
              >
                <div className="flex items-center justify-between">
                  <div className="flex items-center gap-3">
                    <span className={severityBadge(fd.finding.severity)}>
                      {fd.finding.severity}
                    </span>
                    <span className="font-medium">{fd.finding.title}</span>
                  </div>
                  <div className="flex items-center gap-2">
                    <span className={statusBadge(fd.finding.status)}>
                      {fd.finding.status === "confirmed" ? "已确认" : "接受风险"}
                    </span>
                    <span className="text-xs text-gray-400">
                      可信度 {fd.finding.confidence}
                    </span>
                  </div>
                </div>
                {fd.finding.summary && (
                  <p className="text-sm text-gray-600 mt-2 line-clamp-2">{fd.finding.summary}</p>
                )}
                <div className="flex items-center gap-2 mt-2 text-xs text-gray-400">
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
        <div className="border rounded-lg bg-white">
          <div className="flex items-center justify-between px-4 py-2 bg-gray-50 border-b rounded-t-lg">
            <span className="font-medium text-sm">Markdown 报告预览</span>
            <button
              onClick={() => { setShowPreview(false); setPreviewText(null); setPreviewRawText(null); }}
              className="text-gray-400 hover:text-gray-600"
            >
              ✕
            </button>
          </div>
          <div className="p-4 overflow-auto max-h-[70vh]">
            <pre className="text-xs font-mono whitespace-pre-wrap text-gray-700">
              {previewRawText}
            </pre>
          </div>
        </div>
      )}
    </div>
  );
}
