import { useNavigate } from "react-router-dom";
import { useEffect, useState, useRef } from "react";
import type React from "react";
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
  const projectPath = (suffix: string) => currentProject ? `/projects/${currentProject.id}/${suffix}` : "/projects";

  return (
    <div className="page-shell space-y-6">
      <div className="page-header">
        <div>
          <div className="page-eyebrow">Command Center</div>
          <h1 className="page-title">安全测试工作台</h1>
          <p className="page-subtitle">
            统一查看跨项目风险、扫描状态和待审核发现，并按目标到报告的路径推进交付。
          </p>
        </div>
        <div className="flex gap-2">
          <button onClick={() => navigate("/projects")} className="btn-cyber-primary">
            新建项目
          </button>
          <button onClick={() => navigate(projectPath("targets"))} className="btn-cyber-secondary">
            导入目标
          </button>
        </div>
      </div>

      <div className="grid grid-cols-2 lg:grid-cols-4 gap-4">
        <StatCard title="项目" value={stats?.total_projects ?? 0} loading={loading} />
        <StatCard title="运行中扫描" value={stats?.active_runs ?? 0} loading={loading} active={(stats?.active_runs ?? 0) > 0} />
        <StatCard title="待审核发现" value={stats?.pending_findings ?? 0} loading={loading} active={(stats?.pending_findings ?? 0) > 0} />
        <StatCard title="在线 Worker" value={stats?.online_workers ?? 0} loading={loading} active={(stats?.online_workers ?? 0) > 0} />
      </div>

      {error && (
        <div className="rounded-lg bg-brand-danger/10 border border-brand-danger/20 text-brand-danger text-sm p-3 flex items-center justify-between">
          <span>加载数据失败：{error}</span>
          <button onClick={() => fetchDashboard()} className="underline hover:no-underline font-medium">
            重试
          </button>
        </div>
      )}

      {isEmpty ? (
        <div className="panel p-8">
          <EmptyState
            title="还没有项目"
            description="先创建项目，再导入目标、运行扫描、审核发现并导出报告。"
            actionLabel="创建项目"
            onAction={() => navigate("/projects")}
          />
        </div>
      ) : (
        <div className="grid grid-cols-1 xl:grid-cols-[1fr_360px] gap-6">
          <div className="space-y-6">
            <ActivityPanel
              title="最近扫描活动"
              description="快速定位运行中、失败和刚完成的扫描"
              actionLabel="查看扫描"
              onAction={() => navigate(projectPath("runs"))}
            >
              {!stats || !stats.recent_runs || stats.recent_runs.length === 0 ? (
                <div className="py-8">
                  <EmptyState title="暂无扫描活动" description="启动扫描后，这里会显示阶段状态和最近运行记录" />
                </div>
              ) : (
                <div className="divide-y divide-white/[0.06]">
                  {stats.recent_runs.map((run) => (
                    <button
                      key={run.id}
                      onClick={() => navigate(projectPath("runs"))}
                      className="flex w-full items-center justify-between gap-4 py-3 text-left hover:bg-white/[0.03]"
                    >
                      <div className="min-w-0">
                        <div className="text-sm font-medium text-text-primary truncate">{run.name}</div>
                        <div className="text-xs text-text-tertiary mt-1">
                          {run.project_name} · {run.started_at ? formatRelativeTime(run.started_at) : "未开始"}
                        </div>
                      </div>
                      <StatusBadge status={run.status} />
                    </button>
                  ))}
                </div>
              )}
            </ActivityPanel>

            <ActivityPanel
              title="待审核发现"
              description="先确认真实风险，再进入报告交付"
              actionLabel="进入队列"
              onAction={() => navigate(projectPath("findings"))}
            >
              {!stats || !stats.recent_findings || stats.recent_findings.length === 0 ? (
                <div className="py-8">
                  <EmptyState title="暂无待处理发现" description="扫描产生的 Finding 会进入审核队列" />
                </div>
              ) : (
                <div className="divide-y divide-white/[0.06]">
                  {stats.recent_findings.map((finding) => (
                    <button
                      key={finding.id}
                      onClick={() => navigate(projectPath("findings"))}
                      className="flex w-full items-center justify-between gap-4 py-3 text-left hover:bg-white/[0.03]"
                    >
                      <div className="min-w-0">
                        <div className="text-sm font-medium text-text-primary truncate">{finding.title}</div>
                        <div className="text-xs text-text-tertiary mt-1">{finding.project_name}</div>
                      </div>
                      <SeverityBadge severity={finding.severity} className="shrink-0" />
                    </button>
                  ))}
                </div>
              )}
            </ActivityPanel>
          </div>

          <aside className="panel h-fit">
            <div className="panel-header">
              <div>
                <h2 className="panel-title">推荐操作路径</h2>
                <p className="text-xs text-text-tertiary mt-1">按安全测试交付流程组织，而不是按数据表组织</p>
              </div>
            </div>
            <div className="panel-body space-y-3">
              {[
                ["1", "确认项目授权", "配置 Scope、时间窗口和速率限制", projectPath("targets")],
                ["2", "导入并归一化目标", "批量导入域名、URL、IP 或 CIDR", projectPath("targets")],
                ["3", "发现资产与服务", "沉淀 Web 端点、端口与技术栈", projectPath("assets")],
                ["4", "运行扫描并观察阶段", "关注失败阶段和 Worker 状态", projectPath("runs")],
                ["5", "审核 Finding 并交付报告", "确认、标误报、接受风险并导出", projectPath("findings")],
              ].map(([step, title, desc, path]) => (
                <button
                  key={step}
                  onClick={() => navigate(path)}
                  className="surface-item flex w-full gap-3 p-3 text-left"
                >
                  <span className="flex h-7 w-7 shrink-0 items-center justify-center rounded-md bg-brand-primary/12 text-xs font-semibold text-brand-primary ring-1 ring-brand-primary/25">
                    {step}
                  </span>
                  <span>
                    <span className="block text-sm font-medium text-text-primary">{title}</span>
                    <span className="mt-1 block text-xs text-text-tertiary">{desc}</span>
                  </span>
                </button>
              ))}
            </div>
          </aside>
        </div>
      )}
    </div>
  );
}

function ActivityPanel({
  title,
  description,
  actionLabel,
  onAction,
  children,
}: {
  title: string;
  description: string;
  actionLabel: string;
  onAction: () => void;
  children: React.ReactNode;
}) {
  return (
    <section className="panel">
      <div className="panel-header">
        <div>
          <h2 className="panel-title">{title}</h2>
          <p className="text-xs text-text-tertiary mt-1">{description}</p>
        </div>
        <button onClick={onAction} className="text-xs link-cyber">
          {actionLabel}
        </button>
      </div>
      <div className="panel-body">{children}</div>
    </section>
  );
}

const STAT_COLOR_MAP: Record<string, { active: string; text: string }> = {
  "项目":     { active: "border-brand-purple/35 bg-brand-purple/5",        text: "metric-value-purple" },
  "运行中扫描": { active: "border-brand-primary/35 bg-brand-primary/5",      text: "metric-value" },
  "待审核发现": { active: "border-brand-warning/35 bg-brand-warning/5",      text: "metric-value-warning" },
  "在线 Worker": { active: "border-brand-success/35 bg-brand-success/5",    text: "metric-value-success" },
};

function StatCard({
  title,
  value,
  loading,
  active,
}: {
  title: string;
  value: number;
  loading: boolean;
  active?: boolean;
}) {
  const colors = STAT_COLOR_MAP[title] ?? { active: "border-brand-primary/35 bg-brand-primary/5", text: "metric-value" };
  return (
    <div className={`metric-card ${active ? colors.active : ""}`}>
      <div className="metric-label">{title}</div>
      {loading && value === 0 ? (
        <div className="h-8 bg-white/[0.06] rounded-lg animate-pulse w-16 mt-3" />
      ) : (
        <div className={`metric-value ${active ? colors.text : "!text-text-primary"}`}>
          {value}
        </div>
      )}
    </div>
  );
}
