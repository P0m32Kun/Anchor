import { useNavigate } from "react-router-dom";
import { useEffect, useState, useRef } from "react";
import type React from "react";
import { api } from "../lib/api";
import { useStore } from "../lib/store";
import { cn } from "../lib/utils";
import { 
  EmptyState, 
  StatusBadge, 
  SeverityBadge, 
  Button, 
  Card, 
  CardHeader, 
  CardTitle, 
  CardDescription,
  CardContent 
} from "../components";
import type { DashboardStats } from "../lib/api";
import { 
  Plus, 
  Upload, 
  ArrowRight, 
  Activity, 
  CheckCircle2, 
  AlertCircle, 
  Users 
} from "lucide-react";

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
    <div className="space-y-8 animate-in fade-in duration-500">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-3xl font-bold tracking-tight text-foreground">安全测试工作台</h1>
          <p className="text-muted-foreground mt-1">
            统一查看跨项目风险、扫描状态和待审核发现。
          </p>
        </div>
        <div className="flex gap-3">
          <Button onClick={() => navigate("/projects")} variant="primary" className="h-10">
            <Plus className="mr-2 h-4 w-4" />
            新建项目
          </Button>
          <Button onClick={() => navigate(projectPath("targets"))} variant="secondary" className="h-10">
            <Upload className="mr-2 h-4 w-4" />
            导入目标
          </Button>
        </div>
      </div>

      <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-4">
        <StatCard title="项目" value={stats?.total_projects ?? 0} loading={loading} icon={FilesIcon} />
        <StatCard title="运行中扫描" value={stats?.active_runs ?? 0} loading={loading} active={(stats?.active_runs ?? 0) > 0} icon={Activity} />
        <StatCard title="待审核发现" value={stats?.pending_findings ?? 0} loading={loading} active={(stats?.pending_findings ?? 0) > 0} icon={AlertCircle} />
        <StatCard title="在线 Worker" value={stats?.online_workers ?? 0} loading={loading} active={(stats?.online_workers ?? 0) > 0} icon={Users} />
      </div>

      {error && (
        <div className="rounded-lg bg-destructive/10 border border-destructive/20 text-destructive text-sm p-4 flex items-center justify-between">
          <span>加载数据失败：{error}</span>
          <Button variant="ghost" size="sm" onClick={() => fetchDashboard()} className="text-destructive hover:text-destructive hover:bg-destructive/10">
            重试
          </Button>
        </div>
      )}

      {isEmpty ? (
        <Card className="p-12 text-center border-dashed">
          <EmptyState
            title="还没有项目"
            description="先创建项目，再导入目标、运行扫描、审核发现并导出报告。"
            actionLabel="创建项目"
            onAction={() => navigate("/projects")}
          />
        </Card>
      ) : (
        <div className="grid grid-cols-1 lg:grid-cols-[1fr_320px] gap-8">
          <div className="space-y-8">
            <ActivitySection
              title="最近扫描活动"
              description="快速定位运行中、失败和刚完成的扫描"
              actionLabel="查看全部"
              onAction={() => navigate(projectPath("runs"))}
            >
              {!stats || !stats.recent_runs || stats.recent_runs.length === 0 ? (
                <div className="py-12 border rounded-xl border-dashed bg-muted/30">
                  <EmptyState title="暂无扫描活动" description="启动扫描后，这里会显示阶段状态和最近运行记录" />
                </div>
              ) : (
                <div className="space-y-3">
                  {stats.recent_runs.map((run) => (
                    <button
                      key={run.id}
                      onClick={() => navigate(projectPath("runs"))}
                      className="group flex w-full items-center justify-between gap-4 rounded-xl border border-border bg-card p-4 text-left transition-all hover:bg-accent/50 hover:shadow-md"
                    >
                      <div className="min-w-0">
                        <div className="text-sm font-semibold text-foreground group-hover:text-primary transition-colors">{run.name}</div>
                        <div className="text-xs text-muted-foreground mt-1 flex items-center gap-2">
                          <span className="font-medium">{run.project_name}</span>
                          <span className="text-border">|</span>
                          <span>{run.started_at ? formatRelativeTime(run.started_at) : "未开始"}</span>
                        </div>
                      </div>
                      <StatusBadge status={run.status} />
                    </button>
                  ))}
                </div>
              )}
            </ActivitySection>

            <ActivitySection
              title="待审核发现"
              description="先确认真实风险，再进入报告交付"
              actionLabel="进入队列"
              onAction={() => navigate(projectPath("findings"))}
            >
              {!stats || !stats.recent_findings || stats.recent_findings.length === 0 ? (
                <div className="py-12 border rounded-xl border-dashed bg-muted/30">
                  <EmptyState title="暂无待处理发现" description="扫描产生的 Finding 会进入审核队列" />
                </div>
              ) : (
                <div className="space-y-3">
                  {stats.recent_findings.map((finding) => (
                    <button
                      key={finding.id}
                      onClick={() => navigate(projectPath("findings"))}
                      className="group flex w-full items-center justify-between gap-4 rounded-xl border border-border bg-card p-4 text-left transition-all hover:bg-accent/50 hover:shadow-md"
                    >
                      <div className="min-w-0">
                        <div className="text-sm font-semibold text-foreground group-hover:text-primary transition-colors">{finding.title}</div>
                        <div className="text-xs text-muted-foreground mt-1">{finding.project_name}</div>
                      </div>
                      <SeverityBadge severity={finding.severity} className="shrink-0" />
                    </button>
                  ))}
                </div>
              )}
            </ActivitySection>
          </div>

          <aside className="space-y-6">
            <Card className="overflow-hidden border-border bg-muted/30">
              <CardHeader className="bg-card/50 border-b border-border/50">
                <CardTitle className="text-sm">推荐操作路径</CardTitle>
                <CardDescription className="text-xs">按安全测试交付流程推进</CardDescription>
              </CardHeader>
              <CardContent className="p-2 pt-2">
                <div className="space-y-1">
                  {[
                    ["1", "确认项目授权", projectPath("targets")],
                    ["2", "导入并归一化目标", projectPath("targets")],
                    ["3", "发现资产与服务", projectPath("assets")],
                    ["4", "运行扫描并观察阶段", projectPath("runs")],
                    ["5", "审核 Finding 并交付", projectPath("findings")],
                  ].map(([step, title, path]) => (
                    <button
                      key={step}
                      onClick={() => navigate(path)}
                      className="flex w-full items-center gap-3 rounded-md px-3 py-2.5 text-left text-sm transition-colors hover:bg-accent group"
                    >
                      <span className="flex h-5 w-5 shrink-0 items-center justify-center rounded bg-muted text-[10px] font-bold text-muted-foreground group-hover:bg-primary group-hover:text-primary-foreground transition-colors">
                        {step}
                      </span>
                      <span className="flex-1 font-medium text-foreground/80 group-hover:text-foreground">{title}</span>
                      <ArrowRight className="h-3.5 w-3.5 text-muted-foreground opacity-0 group-hover:opacity-100 transition-all -translate-x-2 group-hover:translate-x-0" />
                    </button>
                  ))}
                </div>
              </CardContent>
            </Card>

            <Card className="bg-primary/5 border-primary/20">
               <CardContent className="p-4 flex gap-3">
                 <div className="h-10 w-10 rounded-full bg-primary/10 flex items-center justify-center shrink-0">
                    <CheckCircle2 className="h-5 w-5 text-primary" />
                 </div>
                 <div>
                    <div className="text-sm font-semibold">自动化建议</div>
                    <div className="text-xs text-muted-foreground mt-1">您有 3 个已完成的扫描可以导出报告了。</div>
                    <Button variant="link" className="p-0 h-auto text-xs mt-2 text-primary">立即导出</Button>
                 </div>
               </CardContent>
            </Card>
          </aside>
        </div>
      )}
    </div>
  );
}

function ActivitySection({
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
    <section className="space-y-4">
      <div className="flex items-center justify-between">
        <div>
          <h2 className="text-xl font-semibold tracking-tight">{title}</h2>
          <p className="text-sm text-muted-foreground">{description}</p>
        </div>
        <Button variant="ghost" size="sm" onClick={onAction} className="text-primary hover:text-primary hover:bg-primary/10">
          {actionLabel}
          <ArrowRight className="ml-1 h-3.5 w-3.5" />
        </Button>
      </div>
      <div>{children}</div>
    </section>
  );
}

function FilesIcon(props: any) {
  return (
    <svg
      {...props}
      xmlns="http://www.w3.org/2000/svg"
      width="24"
      height="24"
      viewBox="0 0 24 24"
      fill="none"
      stroke="currentColor"
      strokeWidth="2"
      strokeLinecap="round"
      strokeLinejoin="round"
    >
      <path d="M4 22h14a2 2 0 0 0 2-2V7.5L14.5 2H6a2 2 0 0 0-2 2v4" />
      <polyline points="14 2 14 8 20 8" />
      <path d="M2 15h10" />
      <path d="m9 18 3-3-3-3" />
    </svg>
  );
}

function StatCard({
  title,
  value,
  loading,
  active,
  icon: Icon,
}: {
  title: string;
  value: number;
  loading: boolean;
  active?: boolean;
  icon: React.ElementType;
}) {
  return (
    <Card className={cn("relative overflow-hidden", active && "border-primary/50")}>
      <CardContent className="p-6">
        <div className="flex items-center justify-between space-y-0 pb-2">
          <p className="text-sm font-medium text-muted-foreground">{title}</p>
          <Icon className="h-4 w-4 text-muted-foreground" />
        </div>
        {loading && value === 0 ? (
          <div className="h-9 w-16 animate-pulse rounded bg-muted" />
        ) : (
          <div className="text-2xl font-bold">{value}</div>
        )}
        {active && (
          <div className="absolute top-0 right-0 h-1 w-full bg-primary/20">
            <div className="h-full bg-primary animate-in slide-in-from-left duration-1000" style={{ width: '40%' }} />
          </div>
        )}
      </CardContent>
    </Card>
  );
}
