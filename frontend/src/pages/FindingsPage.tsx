import { useEffect, useState } from "react";
import { useParams } from "react-router-dom";
import { api } from "../lib/api";
import { useStore } from "../lib/store";
import type { Finding, Evidence } from "../lib/api";

const severityColors: Record<string, string> = {
  critical: "bg-red-600 text-white",
  high: "bg-orange-500 text-white",
  medium: "bg-yellow-400 text-black",
  low: "bg-blue-300 text-black",
  info: "bg-gray-200 text-gray-700",
};

const statusLabels: Record<string, string> = {
  pending_review: "待审核",
  confirmed: "已确认",
  false_positive: "误报",
  accepted_risk: "已接受风险",
  ignored: "已忽略",
};

const statusColors: Record<string, string> = {
  pending_review: "bg-yellow-100 text-yellow-800",
  confirmed: "bg-green-100 text-green-800",
  false_positive: "bg-gray-100 text-gray-600",
  accepted_risk: "bg-blue-100 text-blue-800",
  ignored: "bg-gray-100 text-gray-500",
};

export default function FindingsPage() {
  const { id: projectId } = useParams<{ id: string }>();
  const { findings, setFindings, currentFinding, setCurrentFinding } = useStore();
  const [filter, setFilter] = useState<string>("");
  const [loading, setLoading] = useState(false);
  const [detailOpen, setDetailOpen] = useState(false);

  useEffect(() => {
    if (!projectId) return;
    setLoading(true);
    api
      .listFindings(projectId, filter)
      .then(setFindings)
      .finally(() => setLoading(false));
  }, [projectId, filter, setFindings]);

  const openDetail = async (findingId: string) => {
    const data = await api.getFinding(findingId);
    setCurrentFinding(data);
    setDetailOpen(true);
  };

  const closeDetail = () => {
    setDetailOpen(false);
    setCurrentFinding(null);
  };

  const changeStatus = async (findingId: string, status: string) => {
    await api.patchFindingStatus(findingId, status);
    if (projectId) {
      const updated = await api.listFindings(projectId, filter);
      setFindings(updated);
    }
    if (currentFinding) {
      const data = await api.getFinding(findingId);
      setCurrentFinding(data);
    }
  };

  return (
    <div className="space-y-4">
      <div className="flex items-center justify-between">
        <h1 className="text-xl font-bold">Findings</h1>
        <div className="flex gap-2">
          {["", "pending_review", "confirmed", "false_positive", "accepted_risk", "ignored"].map((s) => (
            <button
              key={s || "all"}
              onClick={() => setFilter(s)}
              className={`px-3 py-1 rounded text-sm ${
                filter === s ? "bg-slate-800 text-white" : "bg-gray-100 text-gray-700 hover:bg-gray-200"
              }`}
            >
              {s ? statusLabels[s] || s : "全部"}
            </button>
          ))}
        </div>
      </div>

      {loading && <p className="text-gray-500">加载中...</p>}

      <div className="bg-white rounded shadow overflow-x-auto">
        <table className="min-w-full text-sm">
          <thead className="bg-gray-50 text-gray-600">
            <tr>
              <th className="px-4 py-2 text-left">标题</th>
              <th className="px-4 py-2 text-left">严重级别</th>
              <th className="px-4 py-2 text-left">可信度</th>
              <th className="px-4 py-2 text-left">优先级</th>
              <th className="px-4 py-2 text-left">状态</th>
              <th className="px-4 py-2 text-left">操作</th>
            </tr>
          </thead>
          <tbody>
            {findings.map((f) => (
              <tr key={f.id} className="border-t hover:bg-gray-50">
                <td className="px-4 py-2 font-medium">{f.title}</td>
                <td className="px-4 py-2">
                  <span className={`px-2 py-0.5 rounded text-xs font-medium ${severityColors[f.severity] || "bg-gray-100"}`}>
                    {f.severity}
                  </span>
                </td>
                <td className="px-4 py-2">{f.confidence}</td>
                <td className="px-4 py-2">{f.priority}</td>
                <td className="px-4 py-2">
                  <span className={`px-2 py-0.5 rounded text-xs font-medium ${statusColors[f.status] || "bg-gray-100"}`}>
                    {statusLabels[f.status] || f.status}
                  </span>
                </td>
                <td className="px-4 py-2">
                  <button
                    onClick={() => openDetail(f.id)}
                    className="text-blue-600 hover:underline text-xs"
                  >
                    详情
                  </button>
                </td>
              </tr>
            ))}
            {findings.length === 0 && !loading && (
              <tr>
                <td colSpan={6} className="px-4 py-8 text-center text-gray-400">
                  暂无 Finding
                </td>
              </tr>
            )}
          </tbody>
        </table>
      </div>

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

  const addNote = async () => {
    if (!note.trim()) return;
    setAdding(true);
    await api.addEvidence(finding.id, { type: "note", excerpt: note.trim() });
    setNote("");
    setAdding(false);
    const data = await api.getFinding(finding.id);
    useStore.getState().setCurrentFinding(data);
  };

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/40">
      <div className="bg-white rounded shadow-lg w-full max-w-3xl max-h-[90vh] overflow-y-auto p-6">
        <div className="flex items-center justify-between mb-4">
          <h2 className="text-lg font-bold">Finding 详情</h2>
          <button onClick={onClose} className="text-gray-500 hover:text-gray-800">✕</button>
        </div>

        <div className="space-y-4">
          <div className="grid grid-cols-2 gap-4 text-sm">
            <div>
              <span className="text-gray-500">标题</span>
              <p className="font-medium">{finding.title}</p>
            </div>
            <div>
              <span className="text-gray-500">严重级别</span>
              <p>
                <span className={`px-2 py-0.5 rounded text-xs font-medium ${severityColors[finding.severity] || ""}`}>
                  {finding.severity}
                </span>
              </p>
            </div>
            <div>
              <span className="text-gray-500">可信度</span>
              <p className="font-medium">{finding.confidence}</p>
            </div>
            <div>
              <span className="text-gray-500">优先级</span>
              <p className="font-medium">{finding.priority}</p>
            </div>
            <div>
              <span className="text-gray-500">来源工具</span>
              <p className="font-medium">{finding.source_tool}</p>
            </div>
            <div>
              <span className="text-gray-500">规则 ID</span>
              <p className="font-medium">{finding.source_rule_id || "—"}</p>
            </div>
          </div>

          <div>
            <h3 className="font-semibold text-sm mb-2">状态变更</h3>
            <div className="flex gap-2">
              {["confirmed", "false_positive", "accepted_risk", "ignored", "pending_review"].map((s) => (
                <button
                  key={s}
                  onClick={() => onChangeStatus(finding.id, s)}
                  disabled={finding.status === s}
                  className={`px-3 py-1 rounded text-xs ${
                    finding.status === s
                      ? "bg-slate-800 text-white cursor-default"
                      : "bg-gray-100 text-gray-700 hover:bg-gray-200"
                  }`}
                >
                  {statusLabels[s]}
                </button>
              ))}
            </div>
          </div>

          <div>
            <h3 className="font-semibold text-sm mb-2">Evidence</h3>
            {evidence.length === 0 && <p className="text-gray-400 text-sm">暂无 Evidence</p>}
            <div className="space-y-2">
              {evidence.map((e) => (
                <div key={e.id} className="border rounded p-3 text-sm">
                  <div className="flex items-center gap-2 mb-1">
                    <span className="px-2 py-0.5 rounded bg-gray-100 text-xs">{e.type}</span>
                    <span className="text-gray-400 text-xs">{e.created_at}</span>
                  </div>
                  {e.excerpt && <pre className="whitespace-pre-wrap text-xs bg-gray-50 p-2 rounded">{e.excerpt}</pre>}
                </div>
              ))}
            </div>
          </div>

          <div>
            <h3 className="font-semibold text-sm mb-2">添加备注</h3>
            <textarea
              className="w-full border rounded p-2 text-sm"
              rows={3}
              value={note}
              onChange={(e) => setNote(e.target.value)}
              placeholder="输入备注..."
            />
            <button
              onClick={addNote}
              disabled={adding || !note.trim()}
              className="mt-2 px-4 py-1 bg-slate-800 text-white rounded text-sm disabled:opacity-50"
            >
              {adding ? "保存中..." : "保存备注"}
            </button>
          </div>
        </div>
      </div>
    </div>
  );
}
