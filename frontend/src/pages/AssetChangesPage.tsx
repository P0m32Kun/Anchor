import { useEffect, useState, useCallback } from "react";
import { useProjectId } from "../components/ProjectLayout";
import { api, AssetChange, CHANGE_TYPE_LABELS, SIGNAL_SEVERITY_COLORS } from "../lib/api";
import {
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
  History,
  RefreshCw,
  Plus,
  Minus,
  Server,
  Globe,
  Wifi,
  Shield,
  ChevronDown,
  Filter,
} from "lucide-react";

const CHANGE_TYPE_ICONS: Record<string, React.ElementType> = {
  port_new: Wifi,
  port_gone: Wifi,
  service_new: Server,
  service_gone: Server,
  service_version: Server,
  endpoint_new: Globe,
  endpoint_gone: Globe,
  endpoint_changed: Globe,
  cert_expiring: Shield,
  cert_changed: Shield,
  asset_new: Plus,
  asset_gone: Minus,
};

const CHANGE_TYPES = [
  { value: "all", label: "全部" },
  { value: "asset_new", label: "新资产" },
  { value: "asset_gone", label: "资产消失" },
  { value: "port_new", label: "新端口" },
  { value: "port_gone", label: "端口关闭" },
  { value: "service_new", label: "新服务" },
  { value: "service_gone", label: "服务移除" },
  { value: "service_version_change", label: "版本变更" },
  { value: "endpoint_new", label: "新端点" },
  { value: "endpoint_gone", label: "端点消失" },
  { value: "endpoint_changed", label: "端点变更" },
  { value: "cert_expiring", label: "证书过期" },
  { value: "cert_changed", label: "证书变更" },
];

export default function AssetChangesPage() {
  const projectId = useProjectId();
  const [changes, setChanges] = useState<AssetChange[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [filterType, setFilterType] = useState("all");
  const [expandedId, setExpandedId] = useState<string | null>(null);

  const fetchChanges = useCallback(async () => {
    if (!projectId) return;
    setLoading(true);
    setError(null);
    try {
      const result = await api.listAssetChanges(projectId, { limit: 200 });
      setChanges(result);
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err));
    } finally {
      setLoading(false);
    }
  }, [projectId]);

  useEffect(() => {
    fetchChanges();
  }, [fetchChanges]);

  const filteredChanges = filterType === "all"
    ? changes
    : changes.filter((c) => c.change_type === filterType);

  const formatTime = (t: string) => new Date(t).toLocaleString();

  if (!projectId) {
    return <EmptyState title="未选择项目" description="请先选择一个项目" />;
  }

  return (
    <div className="space-y-6 animate-in fade-in duration-500">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold tracking-tight text-foreground flex items-center gap-2">
            <History className="h-6 w-6 text-blue-400" />
            资产变更历史
          </h1>
          <p className="text-muted-foreground mt-1">
            追踪项目中所有资产的状态变化
          </p>
        </div>
        <Button
          variant="secondary"
          size="sm"
          onClick={fetchChanges}
          loading={loading}
          className="gap-1"
        >
          <RefreshCw className="h-4 w-4" />
          刷新
        </Button>
      </div>

      {loading ? (
        <>
          <SkeletonCard />
          <SkeletonCard />
          <SkeletonCard />
        </>
      ) : error ? (
        <Card>
          <CardContent className="py-12 text-center">
            <History className="mx-auto h-10 w-10 text-muted-foreground/50" />
            <p className="mt-4 text-muted-foreground">{error}</p>
            <Button
              variant="secondary"
              size="sm"
              className="mt-4"
              onClick={fetchChanges}
            >
              重试
            </Button>
          </CardContent>
        </Card>
      ) : (
        <Card>
          <CardHeader>
            <CardTitle className="flex items-center gap-2">
              <Filter className="h-4 w-4 text-muted-foreground" />
              变更记录
              {filteredChanges.length > 0 && (
                <Badge variant="outline" className="ml-1 text-[10px]">
                  {filteredChanges.length}
                </Badge>
              )}
            </CardTitle>
          </CardHeader>
          <CardContent className="p-0">
            <div className="flex items-center gap-2 px-4 py-3 border-b border-border bg-muted/30 flex-wrap">
              {CHANGE_TYPES.map((t) => (
                <Badge
                  key={t.value}
                  variant={filterType === t.value ? "default" : "outline"}
                  className={`cursor-pointer text-[11px] transition-colors ${
                    filterType === t.value
                      ? "bg-blue-500/20 text-blue-400 border-blue-500/30"
                      : "hover:bg-white/10"
                  }`}
                  onClick={() => setFilterType(t.value)}
                >
                  {t.label}
                </Badge>
              ))}
            </div>

            {filteredChanges.length === 0 ? (
              <div className="py-12 text-center">
                <History className="mx-auto h-10 w-10 text-muted-foreground/30" />
                <p className="mt-4 text-muted-foreground">
                  {filterType !== "all"
                    ? "没有匹配的变更记录"
                    : "暂无资产变更记录"}
                </p>
              </div>
            ) : (
              <div className="divide-y divide-border">
                {filteredChanges.map((change) => {
                  const Icon =
                    CHANGE_TYPE_ICONS[change.change_type] || History;
                  const severityColor =
                    SIGNAL_SEVERITY_COLORS[change.severity] ||
                    SIGNAL_SEVERITY_COLORS.info;
                  const isExpanded = expandedId === change.id;

                  return (
                    <div
                      key={change.id}
                      className="group hover:bg-muted/30 transition-colors"
                    >
                      <div
                        className="flex items-start gap-3 p-4 cursor-pointer"
                        onClick={() =>
                          setExpandedId(isExpanded ? null : change.id)
                        }
                      >
                        <div
                          className={`mt-0.5 p-1.5 rounded-lg ${severityColor.split(" ")[1] || "bg-white/10"}`}
                        >
                          <Icon className="h-3.5 w-3.5" />
                        </div>
                        <div className="flex-1 min-w-0">
                          <p className="text-sm font-medium text-foreground">
                            {change.change_summary}
                          </p>
                          <div className="flex items-center gap-2 mt-1 flex-wrap">
                            <Badge
                              variant="outline"
                              className={`text-[10px] px-1.5 py-0 ${severityColor}`}
                            >
                              {CHANGE_TYPE_LABELS[change.change_type] ||
                                change.change_type}
                            </Badge>
                            {change.asset_value && (
                              <span className="text-xs text-muted-foreground font-mono truncate max-w-[200px]">
                                {change.asset_value}
                              </span>
                            )}
                            <span className="text-xs text-muted-foreground/60">
                              {formatTime(change.created_at)}
                            </span>
                          </div>
                        </div>
                        <ChevronDown
                          className={`h-4 w-4 text-muted-foreground mt-1 shrink-0 transition-transform ${
                            isExpanded ? "rotate-180" : ""
                          }`}
                        />
                      </div>
                      {isExpanded && change.detail_json && (
                        <div className="px-4 pb-4 pl-14">
                          <pre className="text-xs text-muted-foreground bg-white/5 rounded-lg p-3 overflow-x-auto whitespace-pre-wrap">
                            {(() => {
                              try {
                                return JSON.stringify(
                                  JSON.parse(change.detail_json),
                                  null,
                                  2,
                                );
                              } catch {
                                return change.detail_json;
                              }
                            })()}
                          </pre>
                        </div>
                      )}
                    </div>
                  );
                })}
              </div>
            )}
          </CardContent>
        </Card>
      )}
    </div>
  );
}
