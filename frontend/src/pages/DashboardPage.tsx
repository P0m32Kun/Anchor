import { useNavigate } from "react-router-dom";
import { useEffect, useState, useRef } from "react";
import type React from "react";
import { api } from "../lib/api";
import { useStore } from "../lib/store";
import { 
  EmptyState, 
  StatusBadge, 
  SeverityBadge, 
  Button, 
  Card, 
  CardHeader, 
  CardTitle, 
  CardContent
} from "../components";
import type { DashboardStats } from "../lib/api";
import { 
  Plus, 
  Upload, 
  ArrowRight, 
  Activity, 
  CheckCircle2,
  Users,
  Zap,
  LayoutDashboard,
  Box,
  Flame,
  ChevronRight
} from "lucide-react";
import { cn } from "../lib/utils";

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
  return date.toLocaleDateString();
}

export default function DashboardPage() {
  const navigate = useNavigate();
  const currentProject = useStore((state) => state.currentProject);
  const [stats, setStats] = useState<DashboardStats | null>(null);
  const [loading, setLoading] = useState(true);
  const [_error, setError] = useState<string | null>(null);
  const abortRef = useRef<AbortController | null>(null);

  const fetchDashboard = async (signal?: AbortSignal) => {
    setLoading(true);
    try {
      const data = await api.getDashboardStats(signal);
      setStats(data);
    } catch (err) {
      if (err instanceof DOMException && err.name === "AbortError") return;
      setError(err instanceof Error ? err.message : "加载数据失败");
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    abortRef.current = new AbortController();
    fetchDashboard(abortRef.current.signal);
    return () => abortRef.current?.abort();
  }, []);

  const isEmpty = !loading && stats && stats.total_projects === 0;
  const projectPath = (suffix: string) => currentProject ? `/projects/${currentProject.id}/${suffix}` : "/projects";

  return (
    <div className="space-y-10 animate-in fade-in slide-in-from-bottom-2 duration-700">
      {/* 顶部欢迎区 - 增加背景光晕 */}
      <div className="relative flex flex-col md:flex-row md:items-end justify-between gap-6 pb-2">
        <div className="relative z-10">
          <div className="flex items-center gap-2 text-primary font-black uppercase tracking-[0.2em] text-[10px] mb-2">
            <LayoutDashboard className="h-3 w-3" />
            Command Center
          </div>
          <h1 className="text-4xl font-black tracking-tighter text-foreground leading-none">
            安全工作台
          </h1>
          <p className="text-muted-foreground mt-3 max-w-md font-medium">
            实时监控全线项目的授权状态、扫描活跃度及风险暴露面。
          </p>
        </div>
        <div className="flex gap-3 z-10">
          <Button onClick={() => navigate("/projects")} variant="primary" className="h-11 px-6 shadow-lg shadow-primary/20 font-bold">
            <Plus className="mr-2 h-4 w-4 stroke-[3px]" />
            新建项目
          </Button>
          <Button onClick={() => navigate(projectPath("targets"))} variant="secondary" className="h-11 px-6 glass-panel font-bold">
            <Upload className="mr-2 h-4 w-4" />
            快速导入
          </Button>
        </div>
        <div className="absolute -top-24 -left-20 h-64 w-64 bg-primary/10 rounded-full blur-[100px] pointer-events-none" />
      </div>

      {/* 核心指标统计 - 赋予颜色活力 */}
      <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-4">
        <StatCard 
            title="总览项目" 
            value={stats?.total_projects ?? 0} 
            loading={loading} 
            icon={Box} 
            color="blue"
            description="Active engagements"
        />
        <StatCard 
            title="运行中扫描" 
            value={stats?.active_runs ?? 0} 
            loading={loading} 
            active={(stats?.active_runs ?? 0) > 0} 
            icon={Zap} 
            color="emerald"
            description="Real-time execution"
        />
        <StatCard 
            title="待审核漏洞" 
            value={stats?.pending_findings ?? 0} 
            loading={loading} 
            active={(stats?.pending_findings ?? 0) > 0} 
            icon={Flame} 
            color="rose"
            description="Requires attention"
        />
        <StatCard 
            title="在线节点" 
            value={stats?.online_workers ?? 0} 
            loading={loading} 
            active={(stats?.online_workers ?? 0) > 0} 
            icon={Users} 
            color="indigo"
            description="Processing power"
        />
      </div>

      {isEmpty ? (
        <Card className="p-20 text-center border-dashed border-white/10 bg-white/[0.02]">
          <EmptyState
            title="初始化工作空间"
            description="当前没有任何项目数据。请先创建一个新项目以开始安全测试生命周期。"
            actionLabel="创建第一个项目"
            onAction={() => navigate("/projects")}
          />
        </Card>
      ) : (
        <div className="grid grid-cols-1 lg:grid-cols-[1fr_340px] gap-10">
          <div className="space-y-10">
            <ActivitySection
              title="最近扫描活动"
              description="流水线执行进度与实时状态反馈"
              actionLabel="查看历史"
              onAction={() => navigate(projectPath("runs"))}
            >
              {!stats || !stats.recent_runs || stats.recent_runs.length === 0 ? (
                <div className="py-16 border rounded-2xl border-dashed border-white/10 bg-white/[0.01] flex flex-col items-center justify-center text-center">
                   <Activity className="h-8 w-8 text-muted-foreground/30 mb-4" />
                   <p className="text-sm text-muted-foreground font-medium text-balance max-w-xs">暂无活跃的扫描流水线，启动后将在此处实时同步状态。</p>
                </div>
              ) : (
                <div className="grid gap-3">
                  {stats.recent_runs.map((run) => (
                    <Card
                      key={run.id}
                      hover
                      onClick={() => navigate(projectPath("runs"))}
                      className="group p-1 overflow-hidden"
                    >
                      <div className="flex items-center justify-between p-4 bg-transparent">
                        <div className="flex items-center gap-4 min-w-0">
                           <div className={cn(
                               "h-10 w-10 rounded-xl flex items-center justify-center border transition-all",
                               run.status === 'running' ? 'bg-primary/20 border-primary/30 text-primary animate-pulse' : 'bg-white/5 border-white/5 text-muted-foreground'
                           )}>
                               <Activity className="h-5 w-5" />
                           </div>
                           <div className="min-w-0">
                             <div className="text-sm font-bold text-foreground truncate group-hover:text-primary transition-colors">{run.name}</div>
                             <div className="text-[11px] text-muted-foreground mt-1.5 flex items-center gap-2 font-medium">
                               <span className="px-1.5 py-0.5 rounded bg-white/5 uppercase text-[9px] tracking-wider">{run.project_name}</span>
                               <span className="opacity-30">•</span>
                               <span>{run.started_at ? formatRelativeTime(run.started_at) : "未开始"}</span>
                             </div>
                           </div>
                        </div>
                        <StatusBadge status={run.status} className="h-7 text-[10px]" />
                      </div>
                    </Card>
                  ))}
                </div>
              )}
            </ActivitySection>

            <ActivitySection
              title="待处理 Finding"
              description="新发现的高置信度漏洞风险审核"
              actionLabel="进入审核队列"
              onAction={() => navigate(projectPath("findings"))}
            >
              {!stats || !stats.recent_findings || stats.recent_findings.length === 0 ? (
                <div className="py-16 border rounded-2xl border-dashed border-white/10 bg-white/[0.01] flex flex-col items-center justify-center text-center">
                   <CheckCircle2 className="h-8 w-8 text-emerald-500/30 mb-4" />
                   <p className="text-sm text-muted-foreground font-medium text-balance max-w-xs">当前没有需要审核的新漏洞发现。所有风险已处理。</p>
                </div>
              ) : (
                <div className="grid gap-3">
                  {stats.recent_findings.map((finding) => (
                    <Card
                      key={finding.id}
                      hover
                      onClick={() => navigate(projectPath("findings"))}
                      className="group p-4 flex items-center justify-between gap-4"
                    >
                      <div className="min-w-0 flex-1">
                        <div className="text-sm font-bold text-foreground group-hover:text-primary transition-colors truncate">{finding.title}</div>
                        <div className="text-[11px] text-muted-foreground mt-1.5 font-medium">{finding.project_name}</div>
                      </div>
                      <SeverityBadge severity={finding.severity} className="h-7 px-3 text-[10px]" />
                    </Card>
                  ))}
                </div>
              )}
            </ActivitySection>
          </div>

          <aside className="space-y-8">
            <Card className="overflow-hidden border-none bg-gradient-to-b from-white/[0.04] to-transparent shadow-none">
              <CardHeader className="pb-3 border-b border-white/5">
                <CardTitle className="text-xs font-black uppercase tracking-[0.15em] text-muted-foreground/70">快速导航指南</CardTitle>
              </CardHeader>
              <CardContent className="p-2 pt-4">
                <div className="space-y-1 px-2">
                  {[
                    ["1", "项目授权", projectPath("targets"), "emerald"],
                    ["2", "目标导入", projectPath("targets"), "cyan"],
                    ["3", "资产枚举", projectPath("assets"), "blue"],
                    ["4", "启动扫描", projectPath("runs"), "indigo"],
                    ["5", "结果审计", projectPath("findings"), "rose"],
                  ].map(([step, title, path, color]) => (
                    <button
                      key={step}
                      onClick={() => navigate(path as string)}
                      className="flex w-full items-center gap-4 rounded-xl px-4 py-3 text-left text-sm transition-all hover:bg-white/5 group"
                    >
                      <span className={cn(
                          "flex h-6 w-6 shrink-0 items-center justify-center rounded-lg text-[10px] font-black transition-all",
                          `bg-${color}-500/10 text-${color}-400 group-hover:bg-${color}-500 group-hover:text-white`
                      )}>
                        {step}
                      </span>
                      <span className="flex-1 font-bold text-foreground/70 group-hover:text-foreground transition-colors">{title as string}</span>
                      <ChevronRight className="h-4 w-4 text-muted-foreground opacity-0 group-hover:opacity-100 transition-all -translate-x-2 group-hover:translate-x-0" />
                    </button>
                  ))}
                </div>
              </CardContent>
            </Card>

            <Card className="bg-primary/10 border-primary/20 relative overflow-hidden group">
               <div className="absolute top-0 right-0 h-32 w-32 bg-primary/20 rounded-full blur-3xl -mr-16 -mt-16 group-hover:bg-primary/30 transition-colors" />
               <CardContent className="p-6 relative z-10 flex flex-col gap-4">
                 <div className="h-12 w-12 rounded-2xl bg-primary/20 flex items-center justify-center text-primary border border-primary/20">
                    <CheckCircle2 className="h-6 w-6" />
                 </div>
                 <div>
                    <div className="text-lg font-bold tracking-tight">自动化建议</div>
                    <div className="text-sm font-medium text-muted-foreground mt-2 leading-relaxed">
                        您有 <span className="text-foreground font-black">3</span> 个已完成的扫描可以导出报告了。系统已为您自动排版。
                    </div>
                    <Button variant="primary" size="sm" className="mt-5 h-10 px-5 rounded-xl font-bold">
                        立即生成
                        <ArrowRight className="ml-2 h-4 w-4" />
                    </Button>
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
    <section className="space-y-6">
      <div className="flex items-end justify-between px-1">
        <div>
          <h2 className="text-2xl font-black tracking-tight">{title}</h2>
          <p className="text-sm text-muted-foreground mt-1.5 font-medium">{description}</p>
        </div>
        <Button variant="ghost" size="sm" onClick={onAction} className="text-primary font-bold hover:bg-primary/10 rounded-lg">
          {actionLabel}
          <ArrowRight className="ml-2 h-4 w-4" />
        </Button>
      </div>
      <div>{children}</div>
    </section>
  );
}

function StatCard({
  title,
  value,
  loading,
  active,
  icon: Icon,
  color = "blue",
  description
}: {
  title: string;
  value: number;
  loading: boolean;
  active?: boolean;
  icon: React.ElementType;
  color?: "blue" | "emerald" | "rose" | "indigo" | "amber";
  description?: string;
}) {
  const colorMap = {
    blue: "text-blue-400 bg-blue-500/10 border-blue-500/20",
    emerald: "text-emerald-400 bg-emerald-500/10 border-emerald-500/20",
    rose: "text-rose-400 bg-rose-500/10 border-rose-500/20",
    indigo: "text-indigo-400 bg-indigo-500/10 border-indigo-500/20",
    amber: "text-amber-400 bg-amber-500/10 border-amber-500/20",
  };

  return (
    <Card className={cn(
        "relative overflow-hidden group transition-all duration-500 border-white/[0.03]", 
        active && "border-white/10 shadow-2xl shadow-black/20"
    )}>
      <CardContent className="p-6">
        <div className="flex items-start justify-between">
          <div className="space-y-1">
            <p className="text-[10px] font-black uppercase tracking-widest text-muted-foreground/60">{title}</p>
            {loading && value === 0 ? (
                <div className="h-10 w-20 animate-pulse rounded-lg bg-white/5 mt-2" />
            ) : (
                <div className="text-4xl font-black tracking-tighter tabular-nums mt-1">{value}</div>
            )}
            <p className="text-[11px] font-medium text-muted-foreground/50 mt-1">{description}</p>
          </div>
          <div className={cn("p-3 rounded-2xl border transition-all duration-500 group-hover:scale-110", colorMap[color])}>
            <Icon className="h-5 w-5" strokeWidth={2.5} />
          </div>
        </div>
        
        {active && (
          <div className="absolute bottom-0 left-0 h-1 w-full bg-white/5 overflow-hidden">
            <div 
                className={cn("h-full animate-pulse", 
                    color === 'blue' ? 'bg-blue-500' : 
                    color === 'emerald' ? 'bg-emerald-500' : 
                    color === 'rose' ? 'bg-rose-500' : 'bg-primary'
                )} 
                style={{ width: '100%' }} 
            />
          </div>
        )}
      </CardContent>
    </Card>
  );
}
