import { useEffect, useMemo, useState } from "react";
import { useNavigate } from "react-router-dom";
import { api, PAGE_ALL } from "../lib/api";
import { useStore } from "../lib/store";
import { EmptyState, Input, Select, SkeletonList, useProjectId, useToast } from "../components";
import type { Finding, Evidence } from "../lib/api";

const severityColors: Record<string, string> = {
  critical: "bg-brand-danger text-white",
  high: "bg-brand-warning text-white",
  medium: "bg-accent-yellow text-black",
  low: "bg-accent-teal text-black",
  info: "bg-white/[0.04] text-text-tertiary",
};

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
  const loading = useStore((state) => state.findingsLoading);
  const setFindingsLoading = useStore((state) => state.setFindingsLoading);
  const setFindingsError = useStore((state) => state.setFindingsError);
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

  // Debounce keyword input by 300ms
  useEffect(() => {
    const timer = setTimeout(() => setDebouncedKeyword(keyword), 300);
    return () => clearTimeout(timer);
  }, [keyword]);

  useEffect(() => {
    if (!projectId) return;
    const ctrl = new AbortController();
    setFindingsLoading(true);
    setFindingsError(null);
    api
      .listFindings(projectId, undefined, PAGE_ALL, ctrl.signal)
      .then((res) => setFindings(res.data ?? []))
      .catch((err) => {
        if (err instanceof DOMException && err.name === "AbortError") return;
        setFindingsError(err instanceof Error ? err.message : String(err));
        console.error(err);
      })
      .finally(() => setFindingsLoading(false));
    return () => ctrl.abort();
  }, [projectId, setFindings, setFindingsLoading, setFindingsError]);

  const openDetail = async (findingId: string) => {
    try {
      const data = await api.getFinding(findingId);
      setCurrentFinding(data);
      setDetailOpen(true);
    } catch (err) {
      toast("加载详情失败: " + (err instanceof Error ? err.message : String(err)), "error");
    }
  };

  const closeDetail = () => {
    setDetailOpen(false);
    setCurrentFinding(null);
  };

  const changeStatus = async (findingId: string, status: string) => {
    const previousStatus = findings.find((f) => f.id === findingId)?.status;

    // Optimistic update
    setFindings((prev) =>
      prev.map((f) => (f.id === findingId ? { ...f, status } : f))
    );
    recordStatusChange(findingId, status);

    try {
      await api.updateFindingStatus(findingId, status);
      toast("状态已更新", "success");
    } catch (err) {
      // Rollback on failure
      if (previousStatus !== undefined) {
        setFindings((prev) =>
          prev.map((f) => (f.id === findingId ? { ...f, status: previousStatus } : f))
        );
      }
      toast("更新失败: " + String(err), "error");
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
      toast(`已批量更新 ${selectedIds.size} 条 finding 状态`, "success");
      setSelectedIds(new Set());
      setBatchStatus("");
      if (projectId) {
        const updated = await api.listFindings(projectId, undefined, PAGE_ALL);
        setFindings(updated.data ?? []);
      }
    } catch (err) {
      toast("批量更新失败: " + (err instanceof Error ? err.message : String(err)), "error");
    } finally {
      setBatchUpdating(false);
    }
  };

  // Client-side filtering: severity + status + keyword
  const filteredFindings = useMemo(() => {
    let result = findings;
    if (statusFilter) {
      result = result.filter((f) => f.status === statusFilter);
    }
    if (severityFilter) {
      result = result.filter((f) => f.severity === severityFilter);
    }
    if (debouncedKeyword.trim()) {
      const kw = debouncedKeyword.trim().toLowerCase();
      result = result.filter(
        (f) =>
          f.title.toLowerCase().includes(kw) ||
          (f.summary && f.summary.toLowerCase().includes(kw))
      );
    }
    return result;
  }, [findings, statusFilter, severityFilter, debouncedKeyword]);

  const navigate = useNavigate();

  if (!projectId) {
    return (
      <div className="p-8">
        <EmptyState
          title="请先选择一个项目"
          description="选择一个项目后查看安全发现"
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
          <div className="page-eyebrow">Step 4</div>
          <h1 className="page-title">发现审核</h1>
          <p className="page-subtitle">按严重级别、状态和关键词筛选，确认真实风险、标记误报或接受风险。</p>
        </div>
      </div>

      {/* Severity filter */}
      <div className="flex items-center gap-2 flex-wrap">
        <span className="text-sm text-zinc-400">严重级别：</span>
        {["", "critical", "high", "medium", "low", "info"].map((s) => (
          <button
            key={s || "all"}
            onClick={() => setSeverityFilter(s)}
            className={`px-3 py-1.5 rounded-lg text-sm transition-colors ${
              severityFilter === s
                ? "bg-slate-800 text-white"
                : "bg-zinc-800/60 text-zinc-300 hover:bg-zinc-700/60"
            }`}
          >
            {s ? s.charAt(0).toUpperCase() + s.slice(1) : "全部"}
          </button>
        ))}
      </div>

      {/* Status filter */}
      <div className="flex items-center gap-2 flex-wrap">
        <span className="text-sm text-zinc-400">状态：</span>
        {["", "pending_review", "confirmed", "false_positive", "accepted_risk", "ignored"].map((s) => (
          <button
            key={s || "all"}
            onClick={() => setStatusFilter(s)}
            className={`px-3 py-1.5 rounded-lg text-sm transition-colors ${
              statusFilter === s
                ? "bg-slate-800 text-white"
                : "bg-zinc-800/60 text-zinc-300 hover:bg-zinc-700/60"
            }`}
          >
            {s ? statusLabels[s] || s : "全部"}
          </button>
        ))}
      </div>

      {/* Keyword search */}
      <div className="max-w-md">
        <Input
          placeholder="搜索标题或描述..."
          value={keyword}
          onChange={(e) => setKeyword(e.target.value)}
        />
      </div>

      {/* Result count */}
      <div className="text-sm text-zinc-400">
        共 {filteredFindings.length} 个 findings
        {(statusFilter || severityFilter || debouncedKeyword) && "（已筛选）"}
      </div>

      {/* Batch operations */}
      {selectedIds.size > 0 && (
        <div className="flex items-center gap-3 bg-zinc-800/40 border border-zinc-700/50 rounded-lg px-4 py-2">
          <span className="text-sm text-zinc-300">已选择 {selectedIds.size} 项</span>
          <select
            value={batchStatus}
            onChange={(e) => setBatchStatus(e.target.value)}
            className="bg-zinc-800 border border-zinc-700 rounded px-2 py-1 text-xs text-zinc-200"
          >
            <option value="">选择状态...</option>
            {Object.entries(statusLabels).map(([value, label]) => (
              <option key={value} value={value}>{label}</option>
            ))}
          </select>
          <button
            onClick={batchChangeStatus}
            disabled={!batchStatus || batchUpdating}
            className="px-3 py-1 bg-slate-800 text-white rounded text-xs disabled:opacity-50"
          >
            {batchUpdating ? "更新中..." : "批量修改"}
          </button>
          <button
            onClick={() => setSelectedIds(new Set())}
            className="text-zinc-400 hover:text-zinc-200 text-xs"
          >
            取消选择
          </button>
        </div>
      )}

      {loading && <SkeletonList count={5} />}

      {!loading && filteredFindings.length === 0 && (
        <div className="panel p-8">
          <EmptyState
            title="暂无 Finding"
            description="当前项目还没有任何安全发现，请先运行扫描任务"
          />
        </div>
      )}

      {!loading && filteredFindings.length > 0 && (
        <div className="panel overflow-auto max-h-[560px]">
        <table className="min-w-full text-sm">
          <thead className="bg-zinc-800/40 text-zinc-400">
            <tr>
              <th className="px-3 py-2 text-left w-10">
                <input
                  type="checkbox"
                  checked={filteredFindings.length > 0 && selectedIds.size === filteredFindings.length}
                  onChange={toggleSelectAll}
                  className="rounded border-zinc-600 bg-zinc-800 text-blue-600 focus:ring-0"
                />
              </th>
              <th className="px-4 py-2 text-left">标题</th>
              <th className="px-4 py-2 text-left">严重级别</th>
              <th className="px-4 py-2 text-left">可信度</th>
              <th className="px-4 py-2 text-left">优先级</th>
              <th className="px-4 py-2 text-left">状态</th>
              <th className="px-4 py-2 text-left">上次修改</th>
              <th className="px-4 py-2 text-left">操作</th>
            </tr>
          </thead>
          <tbody>
            {filteredFindings.map((f) => (
              <tr key={f.id} className="border-t hover:bg-zinc-800/40">
                <td className="px-3 py-2">
                  <input
                    type="checkbox"
                    checked={selectedIds.has(f.id)}
                    onChange={() => toggleSelect(f.id)}
                    className="rounded border-zinc-600 bg-zinc-800 text-blue-600 focus:ring-0"
                  />
                </td>
                <td className="px-4 py-2 font-medium">{f.title}</td>
                <td className="px-4 py-2">
                  <span className={`px-2 py-0.5 rounded text-xs font-medium ${severityColors[f.severity] || "bg-zinc-800/60"}`}>
                    {f.severity}
                  </span>
                </td>
                <td className="px-4 py-2">{f.confidence}</td>
                <td className="px-4 py-2">{f.priority}</td>
                <td className="px-4 py-2">
                  <Select
                    value={f.status}
                    options={statusOptions}
                    onChange={(value) => changeStatus(f.id, value)}
                    className="w-28"
                  />
                </td>
                <td className="px-4 py-2 text-xs text-zinc-500">
                  {findingStatusHistory[f.id]
                    ? formatTimeAgo(findingStatusHistory[f.id].updatedAt)
                    : "—"}
                </td>
                <td className="px-4 py-2 flex gap-2">
                  <button
                    onClick={() => openDetail(f.id)}
                    className="text-blue-600 hover:underline text-xs"
                  >
                    详情
                  </button>
                  <button
                    onClick={async () => {
                      try {
                        await api.retestFinding(f.id);
                        toast("复测已发起", "success");
                      } catch (e) {
                        toast("复测失败: " + String(e), "error");
                      }
                    }}
                    className="text-green-600 hover:underline text-xs"
                  >
                    复测
                  </button>
                </td>
              </tr>
            ))}
          </tbody>
        </table>
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
  const toast = useToast();

  const addNote = async () => {
    if (!note.trim()) return;
    setAdding(true);
    try {
      await api.addEvidence(finding.id, { type: "note", excerpt: note.trim() });
      setNote("");
      const data = await api.getFinding(finding.id);
      useStore.getState().setCurrentFinding(data);
    } catch (err) {
      toast("保存备注失败: " + String(err), "error");
    } finally {
      setAdding(false);
    }
  };

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/40">
      <div className="panel w-full max-w-3xl max-h-[90vh] overflow-y-auto p-6">
        <div className="flex items-center justify-between mb-4">
          <h2 className="text-lg font-bold">Finding 详情</h2>
          <button onClick={onClose} className="text-zinc-400 hover:text-zinc-200">✕</button>
        </div>

        <div className="space-y-4">
          <div className="grid grid-cols-2 gap-4 text-sm">
            <div>
              <span className="text-zinc-400">标题</span>
              <p className="font-medium">{finding.title}</p>
            </div>
            <div>
              <span className="text-zinc-400">严重级别</span>
              <p>
                <span className={`px-2 py-0.5 rounded text-xs font-medium ${severityColors[finding.severity] || ""}`}>
                  {finding.severity}
                </span>
              </p>
            </div>
            <div>
              <span className="text-zinc-400">可信度</span>
              <p className="font-medium">{finding.confidence}</p>
            </div>
            <div>
              <span className="text-zinc-400">优先级</span>
              <p className="font-medium">{finding.priority}</p>
            </div>
            <div>
              <span className="text-zinc-400">来源工具</span>
              <p className="font-medium">{finding.source_tool}</p>
            </div>
            <div>
              <span className="text-zinc-400">规则 ID</span>
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
                      : "bg-zinc-800/60 text-zinc-300 hover:bg-gray-200"
                  }`}
                >
                  {statusLabels[s]}
                </button>
              ))}
            </div>
          </div>

          <div>
            <h3 className="font-semibold text-sm mb-2">Evidence</h3>
            {evidence.length === 0 && <p className="text-zinc-500 text-sm">暂无 Evidence</p>}
            <div className="space-y-2">
              {evidence.map((e) => (
                <div key={e.id} className="border border-white/[0.10] rounded-lg p-3 text-sm">
                  <div className="flex items-center gap-2 mb-1">
                    <span className="px-2 py-0.5 rounded bg-zinc-800/60 text-xs">{e.type}</span>
                    <span className="text-zinc-500 text-xs">{e.created_at}</span>
                  </div>
                  {e.excerpt && <pre className="whitespace-pre-wrap text-xs bg-zinc-800/40 p-2 rounded">{e.excerpt}</pre>}
                </div>
              ))}
            </div>
          </div>

          <div>
            <h3 className="font-semibold text-sm mb-2">添加备注</h3>
            <textarea
              className="w-full rounded-lg border border-white/[0.10] bg-slate-950/40 p-2 text-sm text-text-primary placeholder:text-text-quaternary"
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
