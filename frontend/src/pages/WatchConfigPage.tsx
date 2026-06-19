import { useEffect, useState, useCallback } from "react";
import { useProjectId } from "../components/ProjectLayout";
import { api, WatchConfig } from "../lib/api";
import {
  useToast,
  Card,
  CardHeader,
  CardTitle,
  CardContent,
  Button,
  SkeletonCard,
  EmptyState,
} from "../components";
import {
  Clock,
  RefreshCw,
  Play,
  Pause,
  Eye,
  Zap,
} from "lucide-react";

const INTERVAL_OPTIONS = [
  { value: 4, label: "每 4 小时" },
  { value: 6, label: "每 6 小时" },
  { value: 8, label: "每 8 小时" },
  { value: 12, label: "每 12 小时" },
  { value: 24, label: "每 24 小时" },
  { value: 48, label: "每 48 小时" },
  { value: 72, label: "每 72 小时" },
  { value: 168, label: "每周" },
];

export default function WatchConfigPage() {
  const projectId = useProjectId();
  const toast = useToast();
  const [config, setConfig] = useState<WatchConfig | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [saving, setSaving] = useState(false);

  const fetchConfig = useCallback(async () => {
    if (!projectId) return;
    setLoading(true);
    setError(null);
    try {
      const cfg = await api.getWatchConfig(projectId);
      setConfig(cfg);
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err));
    } finally {
      setLoading(false);
    }
  }, [projectId]);

  useEffect(() => {
    fetchConfig();
  }, [fetchConfig]);

  const save = useCallback(
    async (data: Partial<WatchConfig>) => {
      if (!projectId || !config) return;
      const next = { ...config, ...data };
      setConfig(next);
      setSaving(true);
      try {
        const updated = await api.updateWatchConfig(projectId, data);
        setConfig(updated);
        toast("已保存", "success");
      } catch (err) {
        setConfig(config);
        toast("保存失败", "error");
      } finally {
        setSaving(false);
      }
    },
    [projectId, config, toast],
  );

  const formatTime = (t?: string) => {
    if (!t) return "从未";
    return new Date(t).toLocaleString();
  };

  const nextTick = (last?: string, interval?: number) => {
    if (!last || !interval) return null;
    const n = new Date(last).getTime() + interval * 3600 * 1000;
    if (n <= Date.now()) return "现在";
    return new Date(n).toLocaleString();
  };

  if (!projectId) {
    return <EmptyState title="未选择项目" description="请先选择一个项目" />;
  }

  return (
    <div className="space-y-6 animate-in fade-in duration-500">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold tracking-tight text-foreground flex items-center gap-2">
            <Clock className="h-6 w-6 text-emerald-400" />
            持续监控
          </h1>
          <p className="text-muted-foreground mt-1">
            配置项目的自动扫描计划，实现资产持续监控
          </p>
        </div>
        <Button
          variant="secondary"
          size="sm"
          onClick={fetchConfig}
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
        </>
      ) : error ? (
        <Card>
          <CardContent className="py-12 text-center">
            <Clock className="mx-auto h-10 w-10 text-muted-foreground/50" />
            <p className="mt-4 text-muted-foreground">{error}</p>
            <Button
              variant="secondary"
              size="sm"
              className="mt-4"
              onClick={fetchConfig}
            >
              重试
            </Button>
          </CardContent>
        </Card>
      ) : config ? (
        <>
          <Card>
            <CardHeader>
              <CardTitle className="flex items-center gap-2">
                {config.watch_enabled ? (
                  <Play className="h-5 w-5 text-emerald-400" />
                ) : (
                  <Pause className="h-5 w-5 text-muted-foreground" />
                )}
                监控开关
              </CardTitle>
            </CardHeader>
            <CardContent>
              <div className="flex items-center justify-between">
                <div>
                  <p className="text-sm font-medium text-foreground">
                    {config.watch_enabled ? "已启用" : "已停用"}
                  </p>
                  <p className="text-xs text-muted-foreground mt-1">
                    {config.watch_enabled
                      ? `上次扫描: ${formatTime(config.watch_last_tick_at)}`
                      : "关闭后不会自动执行扫描"}
                  </p>
                  {config.watch_enabled && (
                    <p className="text-xs text-muted-foreground">
                      下次扫描:{" "}
                      {nextTick(
                        config.watch_last_tick_at,
                        config.watch_interval_hours,
                      ) || "计算中..."}
                    </p>
                  )}
                </div>
                <label className="relative inline-flex items-center cursor-pointer">
                  <input
                    type="checkbox"
                    className="sr-only peer"
                    checked={config.watch_enabled}
                    onChange={(e) => save({ watch_enabled: e.target.checked })}
                    disabled={saving}
                  />
                  <div className="w-11 h-6 bg-white/10 peer-checked:bg-emerald-500 rounded-full peer peer-checked:after:translate-x-full after:content-[''] after:absolute after:top-0.5 after:left-[2px] after:bg-white after:rounded-full after:h-5 after:w-5 after:transition-all" />
                </label>
              </div>
            </CardContent>
          </Card>

          <Card>
            <CardHeader>
              <CardTitle className="flex items-center gap-2">
                <RefreshCw className="h-5 w-5 text-blue-400" />
                扫描间隔
              </CardTitle>
            </CardHeader>
            <CardContent>
              <div className="space-y-4">
                <div className="flex flex-wrap gap-2">
                  {INTERVAL_OPTIONS.map((opt) => (
                    <button
                      key={opt.value}
                      disabled={saving || !config.watch_enabled}
                      className={`px-3 py-1.5 rounded-lg text-sm font-medium transition-colors ${
                        config.watch_interval_hours === opt.value
                          ? "bg-blue-500/20 text-blue-400 border border-blue-500/30"
                          : "bg-white/5 text-muted-foreground hover:bg-white/10 border border-transparent"
                      } disabled:opacity-50 disabled:cursor-not-allowed`}
                      onClick={() => save({ watch_interval_hours: opt.value })}
                    >
                      {opt.label}
                    </button>
                  ))}
                </div>
                <p className="text-xs text-muted-foreground">
                  当前间隔: {config.watch_interval_hours} 小时
                  {config.watch_enabled &&
                    config.watch_last_tick_at &&
                    ` · 下次扫描: ${nextTick(config.watch_last_tick_at, config.watch_interval_hours) || "计算中..."}`}
                </p>
              </div>
            </CardContent>
          </Card>

          <Card>
            <CardHeader>
              <CardTitle className="flex items-center gap-2">
                <Eye className="h-5 w-5 text-violet-400" />
                扫描模式
              </CardTitle>
            </CardHeader>
            <CardContent>
              <div className="space-y-4">
                <label className="flex items-start gap-3 cursor-pointer">
                  <input
                    type="checkbox"
                    className="mt-0.5 rounded border-white/20 bg-white/5 text-violet-500 focus:ring-violet-500/30"
                    checked={config.watch_passive_only}
                    onChange={(e) =>
                      save({ watch_passive_only: e.target.checked })
                    }
                    disabled={saving || !config.watch_enabled}
                  />
                  <div>
                    <span className="text-sm font-medium text-foreground flex items-center gap-1.5">
                      <Zap className="h-3.5 w-3.5 text-violet-400" />
                      仅被动搜索
                    </span>
                    <p className="text-xs text-muted-foreground mt-1">
                      启用后只执行被动信息收集（FOFA / Hunter / CRT.sh），
                      不进行端口扫描、服务识别和漏洞检测。适合低频快速监控。
                    </p>
                  </div>
                </label>
              </div>
            </CardContent>
          </Card>

          <Card>
            <CardHeader>
              <CardTitle className="flex items-center gap-2">
                <Clock className="h-5 w-5 text-amber-400" />
                状态摘要
              </CardTitle>
            </CardHeader>
            <CardContent>
              <div className="grid grid-cols-3 gap-4">
                <div className="bg-white/5 rounded-lg p-3 text-center">
                  <p className="text-lg font-bold text-foreground">
                    {config.watch_enabled ? "运行中" : "已停止"}
                  </p>
                  <p className="text-xs text-muted-foreground mt-0.5">
                    监控状态
                  </p>
                </div>
                <div className="bg-white/5 rounded-lg p-3 text-center">
                  <p className="text-lg font-bold text-foreground">
                    {config.watch_interval_hours}h
                  </p>
                  <p className="text-xs text-muted-foreground mt-0.5">
                    扫描间隔
                  </p>
                </div>
                <div className="bg-white/5 rounded-lg p-3 text-center">
                  <p className="text-lg font-bold text-foreground">
                    {formatTime(config.watch_last_tick_at)}
                  </p>
                  <p className="text-xs text-muted-foreground mt-0.5 truncate">
                    上次扫描
                  </p>
                </div>
              </div>
            </CardContent>
          </Card>
        </>
      ) : null}
    </div>
  );
}
