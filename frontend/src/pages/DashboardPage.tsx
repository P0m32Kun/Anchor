import { useNavigate } from "react-router-dom";
import { useEffect, useState, useRef } from "react";
import { api } from "../lib/api";
import { useStore } from "../lib/store";
import { EmptyState, StatusBadge, SeverityBadge } from "../components";
import type { DashboardStats } from "../lib/api";

function formatRelativeTime(dateStr: string): string {
  const date = new Date(dateStr);
  const now = new Date();
  const diffMs = now.getTime() - date.getTime();
  const diffSec = Math.floor(diffMs / 1000);
  if (diffSec < 60) return "刚刚";
  const diffMin = Math.floor(diffSec / 60);
  if (diffMin < 60) return `${diffMin} 分钟前`;
  const diffHour = Math.floor(diffMin / 60);
  if (diffHour < 24) return `${diffHour} 小时前`;
  const diffDay = Math.floor(diffHour / 24);
  if (diffDay < 30) return `${diffDay} 天前`;
  return date.toLocaleDateString();
}

export default function DashboardPage() {
  const navigate = useNavigate();
  const currentProject = useStore((state) => state.currentProject);

  const [stats, setStats] = useState<DashboardStats | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const abortRef = useRef<AbortController | null>(null);

  const fetchDashboard = async (signal?: AbortSignal) => {
    setLoading(true);
    setError(null);
    try {
      const data = await api.getDashboardStats(signal);
      setStats(data);
    } catch (err) {
      if (err instanceof DOMException && err.name === "AbortError") return;
      setError(err instanceof Error ? err.message : "加载数据失败");
      console.error(err);
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    abortRef.current = new AbortController();
    fetchDashboard(abortRef.current.signal);

    const interval = setInterval(() => {
      abortRef.current?.abort();
      abortRef.current = new AbortController();
      fetchDashboard(abortRef.current.signal);
    }, 10000);

    return () => {
      abortRef.current?.abort();
      clearInterval(interval);
    };
  }, []);

  const isEmpty = !loading && stats && stats.total_projects === 0;

  return (
    <div className="space-y-6">
      {/* 统计卡片 */}
      <div className="grid grid-cols-2 lg:grid-cols-4 gap-4">
        <StatCard
          title="总项目数"
          value={stats?.total_projects ?? 0}
          loading={loading}
        />
        <StatCard
          title="活跃扫描"
          value={stats?.active_runs ?? 0}
          loading={loading}
          active={(stats?.active_runs ?? 0) > 0}
          glow="blue"
        />
        <StatCard
          title="待处理 Findings"
          value={stats?.pending_findings ?? 0}
          loading={loading}
          active={(stats?.pending_findings ?? 0) > 0}
          glow="blue"
        />
        <StatCard
          title="在线 Worker"
          value={stats?.online_workers ?? 0}
          loading={loading}
          active={(stats?.online_workers ?? 0) > 0}
        />
      </div>

      {/* 错误提示 */}
      {error && (
        <div className="rounded-lg bg-brand-danger/10 border border-brand-danger/20 text-brand-danger text-sm p-3 flex items-center justify-between">
          <span>加载数据失败：{error}</span>
          <button
            onClick={() => fetchDashboard()}
            className="underline hover:no-underline font-medium"
          >
            重试
          </button>
        </div>
      )}

      {/* 快速操作 */}
      <div className="flex gap-3">
        <button
          onClick={() => navigate("/projects")}
          className="btn-cyber-primary"
        >
          + 创建项目
        </button>
        <button
          onClick={() => {
            if (currentProject) {
              navigate(`/projects/${currentProject.id}/targets`);
            } else {
              navigate("/targets");
            }
          }}
          className="btn-cyber-secondary"
        >
          导入目标
        </button>
      </div>

      {/* 主体内容 */}
      {isEmpty ? (
        <div className="cyber-glass p-8">
          <EmptyState
            title="欢迎使用 Dashboard"
            description="创建项目并开始扫描，以查看跨项目统计和最近活动"
            actionLabel="创建项目"
            onAction={() => navigate("/projects")}
          />
        </div>
      ) : (
        <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
          {/* 最近活动 */}
          <div className="cyber-glass p-4">
            <div className="flex items-center justify-between mb-3">
              <h3 className="text-sm font-semibold text-text-secondary">
                最近活动
              </h3>
              <button
                onClick={() => navigate("/runs")}
                className="text-xs text-brand-primary hover:text-brand-secondary transition-colors"
              >
                查看全部 →
              </button>
            </div>

            {!stats || !stats.recent_runs || stats.recent_runs.length === 0 ? (
              <div className="py-8">
                <EmptyState
                  title="暂无扫描活动"
                  description="开始扫描后，这里将显示最近的活动"
                />
              </div>
            ) : (
              <div className="space-y-2">
                {stats.recent_runs.map((run) => (
                  <div
                    key={run.id}
                    onClick={() => navigate("/runs")}
                    className="flex items-center justify-between p-3 rounded-lg bg-white/[0.03] hover:bg-white/[0.06] cursor-pointer transition-colors"
                  >
                    <div className="min-w-0">
                      <div className="text-sm font-medium text-text-secondary truncate">
                        {run.name}
                      </div>
                      <div className="text-xs text-text-tertiary mt-0.5">
                        {run.project_name} ·{" "}
                        {run.started_at
                          ? formatRelativeTime(run.started_at)
                          : "未开始"}
                      </div>
                    </div>
                    <StatusBadge status={run.status} />
                  </div>
                ))}
              </div>
            )}
          </div>

          {/* 待处理 Findings */}
          <div className="cyber-glass p-4">
            <div className="flex items-center justify-between mb-3">
              <h3 className="text-sm font-semibold text-text-secondary">
                待处理 Findings
              </h3>
              <button
                onClick={() => navigate("/findings")}
                className="text-xs text-brand-primary hover:text-brand-secondary transition-colors"
              >
                查看全部 →
              </button>
            </div>

            {!stats || !stats.recent_findings || stats.recent_findings.length === 0 ? (
              <div className="py-8">
                <EmptyState
                  title="暂无待处理 Findings"
                  description="扫描发现的安全问题将在这里显示"
                />
              </div>
            ) : (
              <div className="space-y-2">
                {stats.recent_findings.map((finding) => (
                  <div
                    key={finding.id}
                    onClick={() => navigate("/findings")}
                    className="flex items-center justify-between p-3 rounded-lg bg-white/[0.03] hover:bg-white/[0.06] cursor-pointer transition-colors"
                  >
                    <div className="min-w-0 flex-1">
                      <div className="text-sm font-medium text-text-secondary truncate">
                        {finding.title}
                      </div>
                      <div className="text-xs text-text-tertiary mt-0.5">
                        {finding.project_name}
                      </div>
                    </div>
                    <SeverityBadge
                      severity={finding.severity}
                      className="ml-2 shrink-0"
                    />
                  </div>
                ))}
              </div>
            )}
          </div>
        </div>
      )}
    </div>
  );
}

function StatCard({
  title,
  value,
  loading,
  active,
  glow,
}: {
  title: string;
  value: number;
  loading: boolean;
  active?: boolean;
  glow?: "blue" | "red";
}) {
  const glowClass = glow === "blue" && active ? " glow-blue" : "";
  return (
    <div className={`cyber-glass p-4${glowClass}`}>
      <div className="text-xs text-text-tertiary mb-1">{title}</div>
      {loading && value === 0 ? (
        <div className="h-8 bg-white/[0.06] rounded-apple-sm animate-pulse w-16" />
      ) : (
        <div
          className={`text-2xl font-bold ${
            active ? "text-brand-primary" : "text-text-primary"
          }`}
        >
          {value}
        </div>
      )}
    </div>
  );
}
