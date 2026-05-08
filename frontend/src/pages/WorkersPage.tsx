import { useEffect, useState } from "react";
import { api, WorkerNode } from "../lib/api";
import { 
  useToast, 
  Button,
  Card,
  CardContent,
  Badge,
  SkeletonList,
  EmptyState
} from "../components";
import { Cpu, Zap, Activity, HardDrive, Shield, RefreshCcw, Server, Network } from "lucide-react";
import { cn } from "../lib/utils";

export default function WorkersPage() {
  const [workers, setWorkers] = useState<WorkerNode[]>([]);
  const [loading, setLoading] = useState(true);
  const toast = useToast();

  const fetchWorkers = async (signal?: AbortSignal) => {
    try {
      const data = await api.listWorkers(signal);
      setWorkers(data);
    } catch (err: any) {
      if (err.name === "AbortError") return;
      toast("加载 Worker 列表失败: " + (err.message || String(err)), "error");
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    const ctrl = new AbortController();
    fetchWorkers(ctrl.signal);
    const timer = setInterval(() => fetchWorkers(), 10000);
    return () => { ctrl.abort(); clearInterval(timer); };
  }, []);

  return (
    <div className="space-y-8 animate-in fade-in duration-500 pb-20">
      <div className="flex items-start justify-between">
        <div>
          <div className="flex items-center gap-2 text-emerald-500 font-bold text-xs uppercase tracking-widest mb-1.5">
            <Server className="h-3.5 w-3.5" />
            Infrastructure
          </div>
          <h1 className="text-3xl font-black tracking-tight text-foreground">Worker 节点集群</h1>
          <p className="text-muted-foreground mt-1">监控分布式扫描节点的实时健康状态、任务负载与资源占用。</p>
        </div>
        <Button variant="outline" className="glass-panel" onClick={() => { setLoading(true); fetchWorkers(); }}>
           <RefreshCcw className={cn("mr-2 h-4 w-4", loading && "animate-spin")} />
           强制刷新
        </Button>
      </div>

      <div className="grid gap-4 md:grid-cols-3">
         <SummaryCard title="在线节点" value={workers.filter(w => w.status === 'online').length} total={workers.length} icon={Activity} color="text-emerald-400" />
         <SummaryCard title="活跃任务" value={workers.reduce((acc, w) => acc + (w.busy ? 1 : 0), 0)} total={workers.length} icon={Zap} color="text-amber-400" />
         <SummaryCard title="系统版本" value="v0.4.2" icon={Shield} color="text-blue-400" />
      </div>

      <div className="space-y-4">
          <div className="flex items-center justify-between px-1">
             <h2 className="text-xl font-bold">节点列表</h2>
             <Badge variant="secondary" className="font-mono">{workers.length} Nodes Registered</Badge>
          </div>

          {loading && workers.length === 0 ? (
            <SkeletonList count={3} />
          ) : workers.length === 0 ? (
            <Card className="p-20 text-center border-dashed">
               <EmptyState title="暂无注册节点" description="请启动 Worker 程序并确保其配置文件中的 Server 地址正确。" />
            </Card>
          ) : (
            <div className="grid gap-4">
              {workers.map((worker) => (
                <Card key={worker.id} className={cn(
                    "overflow-hidden transition-all duration-300",
                    worker.status === 'online' ? "border-emerald-500/20" : "opacity-60"
                )}>
                  <CardContent className="p-0">
                    <div className="flex flex-col md:flex-row">
                      {/* 节点状态侧栏 */}
                      <div className={cn(
                          "w-full md:w-48 p-6 flex flex-col items-center justify-center text-center gap-3 border-b md:border-b-0 md:border-r border-white/5",
                          worker.status === 'online' ? "bg-emerald-500/5" : "bg-white/5"
                      )}>
                        <div className="relative">
                           <div className={cn(
                               "h-12 w-12 rounded-2xl flex items-center justify-center border",
                               worker.status === 'online' ? "bg-emerald-500/20 border-emerald-500/30 text-emerald-400" : "bg-white/5 border-white/10 text-muted-foreground"
                           )}>
                               <Cpu className="h-6 w-6" />
                           </div>
                           {worker.status === 'online' && (
                             <span className="absolute -top-1 -right-1 h-3 w-3 rounded-full bg-emerald-500 border-2 border-slate-950 animate-pulse" />
                           )}
                        </div>
                        <div>
                           <div className="font-bold text-foreground">{worker.id.slice(0, 8)}</div>
                           <Badge variant={worker.status === 'online' ? 'success' : 'secondary'} className="mt-1 text-[9px]">
                                {worker.status.toUpperCase()}
                           </Badge>
                        </div>
                      </div>

                      {/* 详细信息 */}
                      <div className="flex-1 p-6 grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-6">
                         <InfoItem label="内网 IP" value={worker.ip || 'Unknown'} icon={Network} />
                         <InfoItem label="操作系统" value={worker.os || 'Linux'} icon={HardDrive} />
                         <InfoItem label="负载状态" value={worker.busy ? "BUSY (1/1)" : "IDLE (0/1)"} icon={Zap} highlight={worker.busy} />
                         <InfoItem label="最后心跳" value={new Date(worker.last_seen).toLocaleTimeString()} icon={Activity} />
                      </div>
                    </div>
                  </CardContent>
                </Card>
              ))}
            </div>
          )}
      </div>
    </div>
  );
}

function SummaryCard({ title, value, total, icon: Icon, color }: any) {
    return (
        <Card className="bg-white/[0.02] border-white/[0.05]">
            <CardContent className="p-5 flex items-center justify-between">
                <div>
                    <p className="text-[10px] font-black uppercase tracking-widest text-muted-foreground/60">{title}</p>
                    <div className="text-2xl font-black mt-1 flex items-baseline gap-1">
                        {value}
                        {total !== undefined && <span className="text-xs text-muted-foreground font-medium">/ {total}</span>}
                    </div>
                </div>
                <div className={cn("p-2.5 rounded-xl bg-white/5 border border-white/5", color)}>
                    <Icon className="h-5 w-5" />
                </div>
            </CardContent>
        </Card>
    )
}

function InfoItem({ label, value, icon: Icon, highlight }: any) {
    return (
        <div className="space-y-1.5">
            <div className="flex items-center gap-1.5 text-[10px] font-bold uppercase tracking-wider text-muted-foreground/50">
                <Icon className="h-3 w-3" />
                {label}
            </div>
            <div className={cn("text-sm font-bold truncate", highlight ? "text-primary" : "text-foreground/80")}>
                {value}
            </div>
        </div>
    )
}
