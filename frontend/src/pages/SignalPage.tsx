import { useEffect, useState, useCallback } from "react";
import { useProjectId } from "../components/ProjectLayout";
import { useNavigate } from "react-router-dom";
import {
  api,
  Signal,
  SignalStats,
  SIGNAL_SOURCE_LABELS,
  SIGNAL_SEVERITY_COLORS,
  SIGNAL_STATUS_LABELS,
} from "../lib/api";
import {
  useToast,
  Card,
  CardHeader,
  CardTitle,
  CardContent,
  Badge,
  Button,
  SkeletonCard,
  EmptyState,
} from "../components";
import {
  Bell,
  BellOff,
  CheckCheck,
  Eye,
  EyeOff,
  RefreshCw,
  ShieldAlert,
  Box,
  Wifi,
  Globe,
  Shield,
  Server,
} from "lucide-react";

const SIGNAL_SOURCE_ICONS: Record<string, React.ElementType> = {
  finding: ShieldAlert,
  new_asset: Box,
  disappeared_asset: EyeOff,
  asset_change: RefreshCw,
  new_endpoint: Globe,
  new_port: Wifi,
  new_service: Server,
  cert_expiring: Shield,
};

const SEVERITY_ORDER: Record<string, number> = {
  critical: 0,
  high: 1,
  medium: 2,
  low: 3,
  info: 4,
};

export default function SignalPage() {
  const projectId = useProjectId();
  const navigate = useNavigate();
  const toast = useToast();

  const [signals, setSignals] = useState<Signal[]>([]);
  const [stats, setStats] = useState<SignalStats | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [statusFilter, setStatusFilter] = useState<string>("");
  const [severityFilter, setSeverityFilter] = useState<string>("");
  const [sourceKindFilter, setSourceKindFilter] = useState<string>("");
  const [selectedIds, setSelectedIds] = useState<Set<string>>(new Set());
  const [actionLoading, setActionLoading] = useState(false);

  const fetchData = useCallback(async () => {
    if (!projectId) return;
    setLoading(true);
    setError(null);
    try {
      const [signalList, signalStats] = await Promise.all([
        api.listSignals(projectId, {
          status: statusFilter || undefined,
          severity: severityFilter || undefined,
          source_kind: sourceKindFilter || undefined,
        }),
        api.getSignalStats(projectId),
      ]);
      signalList.sort((a, b) => {
        const sa = SEVERITY_ORDER[a.severity] ?? 99;
        const sb = SEVERITY_ORDER[b.severity] ?? 99;
        if (sa !== sb) return sa - sb;
        return new Date(b.last_seen).getTime() - new Date(a.last_seen).getTime();
      });
      setSignals(signalList);
      setStats(signalStats);
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err));
    } finally {
      setLoading(false);
    }
  }, [projectId, statusFilter, severityFilter, sourceKindFilter]);

  useEffect(() => {
    fetchData();
  }, [fetchData]);

  const handleStatusUpdate = async (id: string, status: string) => {
    try {
      await api.updateSignalStatus(id, status);
      setSignals((prev) =>
        prev.map((s) => (s.id === id ? { ...s, status } : s))
      );
      setStats((prev) => {
        if (!prev) return null;
        const updated = { ...prev };
        const oldSignal = signals.find((s) => s.id === id);
        if (oldSignal) {
          if (oldSignal.status === "new") updated.new--;
          if (oldSignal.status === "acknowledged") updated.acknowledged--;
          if (oldSignal.status === "resolved") updated.resolved--;
        }
        if (status === "new") updated.new++;
        if (status === "acknowledged") updated.acknowledged++;
        if (status === "resolved") updated.resolved++;
        return updated;
      });
    } catch {
      toast("状态更新失败", "error");
    }
  };

  const handleBatchAction = async (status: string) => {
    if (selectedIds.size === 0) {
      toast("请先选择信号", "warning");
      return;
    }
    setActionLoading(true);
    try {
      await api.batchUpdateSignalStatus(Array.from(selectedIds), status);
      setSignals((prev) =>
        prev.map((s) =>
          selectedIds.has(s.id) ? { ...s, status } : s
        )
      );
      setSelectedIds(new Set());
      toast(`已更新 ${selectedIds.size} 个信号`, "success");
      fetchData();
    } catch {
      toast("批量操作失败", "error");
    } finally {
      setActionLoading(false);
    }
  };

  const toggleSelect = (id: string) => {
    setSelectedIds((prev) => {
      const next = new Set(prev);
      if (next.has(id)) next.delete(id);
      else next.add(id);
      return next;
    });
  };

  const toggleSelectAll = () => {
    if (selectedIds.size === signals.length) {
      setSelectedIds(new Set());
    } else {
      setSelectedIds(new Set(signals.map((s) => s.id)));
    }
  };

  if (!projectId) {
    return (
      <EmptyState
        title="未选择项目"
        description="请先在左侧导航栏选择一个项目"
        actionLabel="返回项目列表"
        onAction={() => navigate("/projects")}
      />
    );
  }

  const parseMetadataJson = (sig: Signal): Record<string, string> => {
    try {
      if (sig.metadata) return JSON.parse(sig.metadata);
    } catch {}
    return {};
  };

  return (
    <div className="space-y-6 animate-in fade-in duration-500">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold tracking-tight text-foreground flex items-center gap-2">
            <Bell className="h-6 w-6 text-primary" />
            信号收件箱
          </h1>
          <p className="text-muted-foreground mt-1">
            持续资产监控告警 — 资产变化、端口变更、证书过期一目了然
          </p>
        </div>
        <Button variant="secondary" size="sm" onClick={fetchData} loading={loading}>
          <RefreshCw className="h-4 w-4 mr-2" />
          刷新
        </Button>
      </div>

      {stats && (
        <div className="grid grid-cols-4 gap-4">
          <Card className="border-destructive/20">
            <CardContent className="p-4 text-center">
              <div className="text-2xl font-bold text-destructive">{stats.new}</div>
              <div className="text-xs text-muted-foreground mt-1">未处理</div>
            </CardContent>
          </Card>
          <Card className="border-yellow-500/20">
            <CardContent className="p-4 text-center">
              <div className="text-2xl font-bold text-yellow-500">{stats.acknowledged}</div>
              <div className="text-xs text-muted-foreground mt-1">已确认</div>
            </CardContent>
          </Card>
          <Card className="border-emerald-500/20">
            <CardContent className="p-4 text-center">
              <div className="text-2xl font-bold text-emerald-500">{stats.resolved}</div>
              <div className="text-xs text-muted-foreground mt-1">已解决</div>
            </CardContent>
          </Card>
          <Card>
            <CardContent className="p-4 text-center">
              <div className="text-2xl font-bold">{stats.total}</div>
              <div className="text-xs text-muted-foreground mt-1">总计</div>
            </CardContent>
          </Card>
        </div>
      )}

      <Card>
        <CardHeader className="pb-3">
          <div className="flex items-center justify-between flex-wrap gap-3">
            <CardTitle className="text-lg">信号列表</CardTitle>
            <div className="flex items-center gap-2">
              {selectedIds.size > 0 && (
                <>
                  <span className="text-xs text-muted-foreground">{selectedIds.size} 已选</span>
                  <Button
                    variant="secondary"
                    size="sm"
                    onClick={() => handleBatchAction("acknowledged")}
                    loading={actionLoading}
                    className="text-xs"
                  >
                    <Eye className="h-3 w-3 mr-1" />
                    已确认
                  </Button>
                  <Button
                    variant="secondary"
                    size="sm"
                    onClick={() => handleBatchAction("resolved")}
                    loading={actionLoading}
                    className="text-xs"
                  >
                    <CheckCheck className="h-3 w-3 mr-1" />
                    已解决
                  </Button>
                </>
              )}
            </div>
          </div>
        </CardHeader>
        <CardContent className="p-0">
          <div className="flex items-center gap-3 px-4 py-3 border-b border-border bg-muted/30 flex-wrap">
            <Button
              variant={statusFilter === "" ? "primary" : "secondary"}
              size="sm"
              onClick={() => setStatusFilter("")}
              className="h-7 text-xs"
            >
              全部
            </Button>
            {[
              { key: "new", label: "未处理" },
              { key: "acknowledged", label: "已确认" },
              { key: "resolved", label: "已解决" },
            ].map(({ key, label }) => (
              <Button
                key={key}
                variant={statusFilter === key ? "primary" : "secondary"}
                size="sm"
                onClick={() => setStatusFilter(key)}
                className="h-7 text-xs"
              >
                {label}
              </Button>
            ))}
            <div className="h-4 w-px bg-border mx-1" />
            <select
              className="h-7 rounded-lg border border-border bg-muted/50 px-2 text-xs font-medium text-muted-foreground focus:outline-none focus:ring-1 focus:ring-primary/50"
              value={severityFilter}
              onChange={(e) => setSeverityFilter(e.target.value)}
            >
              <option value="">全部严重级别</option>
              <option value="critical">严重</option>
              <option value="high">高</option>
              <option value="medium">中</option>
              <option value="low">低</option>
              <option value="info">信息</option>
            </select>
            <select
              className="h-7 rounded-lg border border-border bg-muted/50 px-2 text-xs font-medium text-muted-foreground focus:outline-none focus:ring-1 focus:ring-primary/50"
              value={sourceKindFilter}
              onChange={(e) => setSourceKindFilter(e.target.value)}
            >
              <option value="">全部类型</option>
              {Object.entries(SIGNAL_SOURCE_LABELS).map(([key, label]) => (
                <option key={key} value={key}>
                  {label}
                </option>
              ))}
            </select>
            {signals.length > 0 && (
              <button
                className="text-xs text-muted-foreground hover:text-foreground transition-colors ml-auto"
                onClick={toggleSelectAll}
              >
                {selectedIds.size === signals.length ? "取消全选" : "全选"}
              </button>
            )}
          </div>

          {loading ? (
            <div className="divide-y divide-border">
              {[...Array(5)].map((_, i) => (
                <div key={i} className="p-4">
                  <SkeletonCard />
                </div>
              ))}
            </div>
          ) : error ? (
            <div className="p-8 text-center">
              <BellOff className="h-12 w-12 text-muted-foreground mx-auto mb-3 opacity-30" />
              <p className="text-muted-foreground">{error}</p>
              <Button variant="secondary" className="mt-3" onClick={fetchData}>
                重试
              </Button>
            </div>
          ) : signals.length === 0 ? (
            <div className="p-12 text-center">
              <EmptyState
                title="暂无信号"
                description={
                  statusFilter || severityFilter || sourceKindFilter
                    ? "当前筛选条件下没有匹配的信号"
                    : "启用持续监控后，系统会自动检测资产变化并生成告警信号"
                }
                actionLabel="清除筛选"
                onAction={() => {
                  setStatusFilter("");
                  setSeverityFilter("");
                  setSourceKindFilter("");
                }}
              />
            </div>
          ) : (
            <div className="divide-y divide-border">
              {signals.map((sig) => {
                const Icon = SIGNAL_SOURCE_ICONS[sig.source_kind] || Bell;
                const meta = parseMetadataJson(sig);
                const isSelected = selectedIds.has(sig.id);
                return (
                  <div
                    key={sig.id}
                    className="group flex items-start gap-4 p-4 hover:bg-muted/30 transition-colors cursor-pointer"
                    onClick={() => toggleSelect(sig.id)}
                  >
                    <div className="mt-0.5">
                      <div
                        className={`w-8 h-8 rounded-lg flex items-center justify-center ${
                          isSelected
                            ? "bg-primary/20 text-primary"
                            : "bg-muted text-muted-foreground"
                        } transition-colors`}
                      >
                        {isSelected ? (
                          <CheckCheck className="h-4 w-4" />
                        ) : (
                          <Icon className="h-4 w-4" />
                        )}
                      </div>
                    </div>
                    <div className="flex-1 min-w-0">
                      <div className="flex items-start gap-2 flex-wrap">
                        <span className="font-medium text-sm text-foreground">
                          {sig.title}
                        </span>
                        <Badge
                          variant="outline"
                          className="text-[10px] px-1.5 py-0"
                        >
                          {SIGNAL_SOURCE_LABELS[sig.source_kind] || sig.source_kind}
                        </Badge>
                      </div>
                      {Object.keys(meta).length > 0 && (
                        <div className="flex flex-wrap gap-2 mt-1.5">
                          {Object.entries(meta).map(([k, v]) => (
                            <span
                              key={k}
                              className="text-[10px] text-muted-foreground bg-muted/50 px-1.5 py-0.5 rounded"
                            >
                              {k}: {v}
                            </span>
                          ))}
                        </div>
                      )}
                      <div className="flex items-center gap-2 mt-2 text-[10px] text-muted-foreground">
                        <span>
                          {new Date(sig.first_seen).toLocaleString()}
                        </span>
                        {sig.last_seen !== sig.first_seen && (
                          <>
                            <span>→</span>
                            <span>
                              {new Date(sig.last_seen).toLocaleString()}
                            </span>
                          </>
                        )}
                      </div>
                    </div>
                    <div className="flex items-center gap-2 shrink-0">
                      <Badge
                        variant="outline"
                        className={`text-[10px] px-1.5 py-0 ${
                          SIGNAL_STATUS_LABELS[sig.status] || ""
                        }`}
                      >
                        {SIGNAL_STATUS_LABELS[sig.status] || sig.status}
                      </Badge>
                      <span
                        className={`text-[10px] font-semibold px-2 py-0.5 rounded-full border ${SIGNAL_SEVERITY_COLORS[sig.severity] || ""}`}
                      >
                        {sig.severity.toUpperCase()}
                      </span>
                    </div>
                    <div className="flex items-center gap-1 shrink-0 opacity-0 group-hover:opacity-100 transition-opacity">
                      {sig.status !== "acknowledged" && (
                        <Button
                          variant="ghost"
                          size="sm"
                          className="h-7 w-7 p-0"
                          title="确认"
                          onClick={(e) => {
                            e.stopPropagation();
                            handleStatusUpdate(sig.id, "acknowledged");
                          }}
                        >
                          <Eye className="h-3 w-3" />
                        </Button>
                      )}
                      {sig.status !== "resolved" && (
                        <Button
                          variant="ghost"
                          size="sm"
                          className="h-7 w-7 p-0"
                          title="解决"
                          onClick={(e) => {
                            e.stopPropagation();
                            handleStatusUpdate(sig.id, "resolved");
                          }}
                        >
                          <CheckCheck className="h-3 w-3" />
                        </Button>
                      )}
                    </div>
                  </div>
                );
              })}
            </div>
          )}
        </CardContent>
      </Card>
    </div>
  );
}
