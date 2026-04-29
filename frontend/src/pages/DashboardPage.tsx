import { useNavigate } from "react-router-dom";
import { useCallback, useEffect, useState, useMemo } from "react";
import { API_BASE, api } from "../lib/api";
import { useStore } from "../lib/store";
import { EmptyState, StatusBadge, SeverityBadge } from "../components";
import type { Project, Run, Finding } from "../lib/api";

interface Worker {
  id: string;
  name: string;
  mode: string;
  status: string;
}

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

  const [projects, setProjects] = useState<Project[]>([]);
  const [runs, setRuns] = useState<Run[]>([]);
  const [pendingFindings, setPendingFindings] = useState<Finding[]>([]);
  const [onlineWorkers, setOnlineWorkers] = useState(0);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const fetchDashboardData = useCallback(async (signal?: AbortSignal) => {
    setLoading(true);
    setError(null);

    try {
      const projectsList = await api.listProjects(signal);
      const validProjects = projectsList ?? [];
      setProjects(validProjects);

      // Fetch workers independently — don't block dashboard on worker failure
      fetch(`${API_BASE}/workers`, { signal })
        .then((res) => (res.ok ? res.json() : []))
        .then((workers: Worker[]) => {
          const count = workers.filter(
            (w) => w.status === "online" || w.status === "busy"
          ).length;
          setOnlineWorkers(count);
        })
        .catch(() => setOnlineWorkers(0));

      if (validProjects.length > 0) {
        const runsPromises = validProjects.map((p) =>
          api.listRuns(p.id, signal).catch(() => [] as Run[])
        );
        const findingsPromises = validProjects.map((p) =>
          api
            .listFindings(p.id, "pending_review", signal)
            .catch(() => [] as Finding[])
        );

        const [runsResults, findingsResults] = await Promise.all([
          Promise.all(runsPromises),
          Promise.all(findingsPromises),
        ]);

        const allRuns = runsResults.flat();
        allRuns.sort((a, b) => {
          const aTime = a.started_at || a.created_at;
          const bTime = b.started_at || b.created_at;
          return new Date(bTime).getTime() - new Date(aTime).getTime();
        });

        const allFindings = findingsResults.flat();
        allFindings.sort(
          (a, b) =>
            new Date(b.created_at).getTime() - new Date(a.created_at).getTime()
        );

        setRuns(allRuns);
        setPendingFindings(allFindings);
      } else {
        setRuns([]);
        setPendingFindings([]);
      }
    } catch (err) {
      if (err instanceof DOMException && err.name === "AbortError") return;
      const message = err instanceof Error ? err.message : "加载数据失败";
      setError(message);
      console.error(err);
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    const ctrl = new AbortController();
    fetchDashboardData(ctrl.signal);
    const interval = setInterval(() => {
      if (!ctrl.signal.aborted) {
        fetchDashboardData(ctrl.signal);
      }
    }, 10000);
    return () => {
      ctrl.abort();
      clearInterval(interval);
    };
  }, [fetchDashboardData]);

  const stats = useMemo(
    () => ({
      totalProjects: projects.length,
      activeRuns: runs.filter((r) => r.status === "running").length,
      pendingFindings: pendingFindings.length,
      onlineWorkers,
    }),
    [projects.length, runs, pendingFindings.length, onlineWorkers]
  );

  const recentRuns = useMemo(() => runs.slice(0, 5), [runs]);
  const recentFindings = useMemo(
    () => pendingFindings.slice(0, 5),
    [pendingFindings]
  );

  const getProjectName = useCallback(
    (projectId: string) => {
      return projects.find((p) => p.id === projectId)?.name || "未知项目";
    },
    [projects]
  );

  const handleImportTargets = () => {
    if (currentProject) {
      navigate(`/projects/${currentProject.id}/targets`);
    } else {
      navigate("/targets");
    }
  };

  const isCompletelyEmpty =
    !loading &&
    projects.length === 0 &&
    runs.length === 0 &&
    pendingFindings.length === 0;

  return (
    <div className="space-y-6">
      {/* 统计卡片 */}
      <div className="grid grid-cols-2 lg:grid-cols-4 gap-4">
        <StatCard
          title="总项目数"
          value={stats.totalProjects}
          loading={loading}
        />
        <StatCard
          title="活跃扫描"
          value={stats.activeRuns}
          loading={loading}
          active={stats.activeRuns > 0}
        />
        <StatCard
          title="待处理 Findings"
          value={stats.pendingFindings}
          loading={loading}
          active={stats.pendingFindings > 0}
        />
        <StatCard
          title="在线 Worker"
          value={stats.onlineWorkers}
          loading={loading}
          active={stats.onlineWorkers > 0}
        />
      </div>

      {/* 错误提示 */}
      {error && (
        <div className="rounded-lg bg-brand-danger/10 border border-brand-danger/20 text-brand-danger text-sm p-3 flex items-center justify-between">
          <span>加载数据失败：{error}</span>
          <button
            onClick={() => fetchDashboardData()}
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
          className="px-4 py-2 rounded-lg bg-brand-primary text-white text-sm font-medium hover:bg-brand-primary/90 transition-colors"
        >
          + 创建项目
        </button>
        <button
          onClick={handleImportTargets}
          className="px-4 py-2 rounded-lg liquid-glass text-sm font-medium text-text-secondary hover:text-text-primary transition-colors"
        >
          导入目标
        </button>
      </div>

      {/* 主体内容 */}
      {isCompletelyEmpty ? (
        <div className="liquid-glass rounded-xl p-8">
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
          <div className="liquid-glass rounded-xl p-4">
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

            {recentRuns.length === 0 ? (
              <div className="py-8">
                <EmptyState
                  title="暂无扫描活动"
                  description="开始扫描后，这里将显示最近的活动"
                />
              </div>
            ) : (
              <div className="space-y-2">
                {recentRuns.map((run) => (
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
                        {getProjectName(run.project_id)} ·{" "}
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
          <div className="liquid-glass rounded-xl p-4">
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

            {recentFindings.length === 0 ? (
              <div className="py-8">
                <EmptyState
                  title="暂无待处理 Findings"
                  description="扫描发现的安全问题将在这里显示"
                />
              </div>
            ) : (
              <div className="space-y-2">
                {recentFindings.map((finding) => (
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
                        {getProjectName(finding.project_id)}
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
}: {
  title: string;
  value: number;
  loading: boolean;
  active?: boolean;
}) {
  return (
    <div className="liquid-glass rounded-xl p-4">
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
