import { useState, useEffect, useRef } from "react";
import { Link } from "react-router-dom";
import { api, SearchResult } from "../lib/api";
import {
  useToast,
  EmptyState,
  Button,
  Input,
  Card,
  CardHeader,
  CardTitle,
  CardContent,
  Badge,
  Table,
  TableHeader,
  TableBody,
  TableRow,
  TableHead,
  TableCell
} from "../components";
import { Search, Key, Globe, Database, Info, Coins, WifiOff } from "lucide-react";
import { cn } from "../lib/utils";

const ENGINES = [
  { key: "fofa", label: "FOFA", placeholder: 'domain="example.com"', icon: Database, color: "text-blue-400" },
  { key: "hunter", label: "Hunter", placeholder: 'ip.port=443', icon: Database, color: "text-cyan-400" },
  { key: "quake", label: "Quake", placeholder: 'service:http', icon: Database, color: "text-indigo-400" },
];

const PAGE_SIZE = 20;

type QuotaData = { points: { name: string; value: number; unit: string }[] } | null | "offline";

export default function EnginesPage() {
  const [activeEngine, setActiveEngine] = useState("fofa");
  const [query, setQuery] = useState("");
  const [results, setResults] = useState<SearchResult[]>([]);
  const [total, setTotal] = useState(0);
  const [page, setPage] = useState(1);
  const [loading, setLoading] = useState(false);
  const [hasSearched, setHasSearched] = useState(false);
  const [quotas, setQuotas] = useState<Record<string, QuotaData>>({});
  const toast = useToast();
  const abortRef = useRef<AbortController | null>(null);
  const quotaAbortRef = useRef<AbortController | null>(null);

  const activeMeta = ENGINES.find((e) => e.key === activeEngine)!;

  async function fetchQuota(engine: string) {
    quotaAbortRef.current?.abort();
    const ctrl = new AbortController();
    quotaAbortRef.current = ctrl;

    try {
      const res = await api.getEngineQuota(engine, ctrl.signal);
      setQuotas(prev => ({ ...prev, [engine]: res.quota }));
    } catch (err: any) {
      if (err.name === "AbortError") return;
      // Network/offline error
      if (!err.status || err.message?.includes("fetch") || err.message?.includes("network")) {
        setQuotas(prev => ({ ...prev, [engine]: "offline" }));
      } else {
        setQuotas(prev => ({ ...prev, [engine]: null }));
      }
    }
  }

  // Initial load: fetch all quotas once
  useEffect(() => {
    const timer = setTimeout(() => {
      ENGINES.forEach(e => fetchQuota(e.key));
    }, 300);
    return () => {
      clearTimeout(timer);
      abortRef.current?.abort();
      quotaAbortRef.current?.abort();
    };
  }, []);

  async function handleSearch(resetPage = true, explicitPage?: number) {
    if (!query.trim()) {
      toast("请输入查询语句", "warning");
      return;
    }
    abortRef.current?.abort();
    const ctrl = new AbortController();
    abortRef.current = ctrl;

    const currentPage = resetPage ? 1 : (explicitPage ?? page);
    if (resetPage) setPage(1);

    setLoading(true);
    setHasSearched(true);
    try {
      const res = await api.searchEngine({ engine: activeEngine, query, page: currentPage, size: PAGE_SIZE }, ctrl.signal);
      setResults(res.data || []);
      setTotal(res.total || 0);
      // Refresh quota for the active engine after search
      fetchQuota(activeEngine);
    } catch (err: any) {
      if (err.name === "AbortError") return;
      toast("搜索失败: " + (err.message || String(err)), "error");
    } finally {
      setLoading(false);
    }
  }

  function renderQuota(engine: string) {
    const quota = quotas[engine];
    if (quota === "offline") {
      return (
        <span className="text-[11px] text-muted-foreground/50 flex items-center gap-1">
          <WifiOff className="h-3 w-3" />
          未联网
        </span>
      );
    }
    if (!quota || quota.points.length === 0) {
      return <span className="text-[11px] text-muted-foreground/50">--</span>;
    }
    return (
      <div className="flex items-center gap-1.5">
        {quota.points.map((p, i) => (
          <Badge
            key={i}
            variant="outline"
            className="font-mono text-[10px] bg-primary/5 border-primary/20 text-primary px-1.5 py-0"
            title={p.name}
          >
            {p.value.toLocaleString()}{p.unit}
          </Badge>
        ))}
      </div>
    );
  }

  return (
    <div className="space-y-8 animate-in fade-in duration-500 pb-20">
      <div className="flex items-start justify-between">
        <div>
          <div className="flex items-center gap-2 text-cyan-500 font-bold text-xs uppercase tracking-widest mb-1.5">
            <Globe className="h-3.5 w-3.5" />
            Intelligence Discovery
          </div>
          <h1 className="text-3xl font-black tracking-tight text-foreground">空间测绘引擎</h1>
          <p className="text-muted-foreground mt-1">跨平台检索 FOFA、Hunter、Quake 资产数据，一键导入目标范围。</p>
        </div>
        <Link to="/engines/keys">
          <Button variant="secondary" className="h-10 glass-panel">
            <Key className="mr-2 h-4 w-4" />
            凭证管理
          </Button>
        </Link>
      </div>

      <div className="grid gap-8 lg:grid-cols-[300px_1fr]">
        <aside className="space-y-6">
          <Card>
            <CardHeader className="pb-3">
              <CardTitle className="text-sm font-bold uppercase tracking-wider text-muted-foreground">选择数据源</CardTitle>
            </CardHeader>
            <CardContent className="p-2 space-y-1">
              {ENGINES.map((e) => (
                <button
                  key={e.key}
                  onClick={() => { setActiveEngine(e.key); setResults([]); setHasSearched(false); }}
                  className={cn(
                    "flex w-full items-center justify-between rounded-xl px-4 py-3 text-sm font-bold transition-all",
                    activeEngine === e.key
                        ? "bg-primary/10 text-primary border border-primary/20 shadow-[inset_0_0_12px_rgba(0,212,255,0.05)]"
                        : "text-muted-foreground hover:bg-white/5"
                  )}
                >
                  <div className="flex items-center gap-3">
                     <e.icon className={cn("h-4 w-4", activeEngine === e.key ? "text-primary" : "text-muted-foreground")} />
                     {e.label}
                  </div>
                  {activeEngine === e.key && <div className="h-1.5 w-1.5 rounded-full bg-primary" />}
                </button>
              ))}
            </CardContent>
          </Card>

          <Card className="bg-primary/5 border-primary/20">
             <CardContent className="p-5 flex flex-col gap-3">
                 <div className="h-10 w-10 rounded-xl bg-primary/20 flex items-center justify-center text-primary">
                    <Info className="h-5 w-5" />
                 </div>
                 <div>
                    <div className="text-sm font-bold">搜索小贴士</div>
                    <p className="text-[11px] text-muted-foreground mt-1 leading-relaxed">
                        使用标准语法如 <code className="text-primary font-mono">{activeMeta.placeholder}</code>。搜索结果可直接用于后续扫描流水线。
                    </p>
                 </div>
             </CardContent>
          </Card>

          <Card>
            <CardHeader className="pb-3">
              <CardTitle className="text-sm font-bold uppercase tracking-wider text-muted-foreground flex items-center gap-2">
                <Coins className="h-3.5 w-3.5" />
                额度状态
              </CardTitle>
            </CardHeader>
            <CardContent className="p-2 space-y-1">
              {ENGINES.map((e) => (
                <div key={e.key} className="flex items-center justify-between rounded-lg px-3 py-2 text-sm">
                  <div className="flex items-center gap-2">
                    <e.icon className={cn("h-3.5 w-3.5", e.color)} />
                    <span className="text-muted-foreground">{e.label}</span>
                  </div>
                  {renderQuota(e.key)}
                </div>
              ))}
            </CardContent>
          </Card>
        </aside>

        <section className="space-y-6">
          <Card className="p-1 overflow-hidden">
             <div className="flex gap-2 p-2">
                <div className="relative flex-1">
                    <Search className="absolute left-3 top-3 h-4 w-4 text-muted-foreground" />
                    <Input
                        value={query}
                        onChange={(e) => setQuery(e.target.value)}
                        placeholder={`在 ${activeMeta.label} 中检索: ${activeMeta.placeholder}`}
                        onKeyDown={(e) => e.key === "Enter" && handleSearch(true)}
                        className="pl-10 h-11 border-none bg-white/5 focus-visible:ring-1 focus-visible:ring-primary/50 text-base"
                    />
                </div>
                <Button onClick={() => handleSearch(true)} loading={loading} className="h-11 px-8 rounded-xl font-bold">
                    开始搜索
                </Button>
             </div>
          </Card>

          <div className="flex items-center justify-between px-1">
              <h2 className="text-xl font-bold flex items-center gap-2">
                检索结果
                {hasSearched && <Badge variant="secondary" className="font-mono">{total} Results</Badge>}
              </h2>
              {hasSearched && total > 0 && (
                <div className="flex items-center gap-2">
                    <Button variant="outline" size="sm" className="h-8 rounded-lg" disabled={page <= 1} onClick={() => { const p = page - 1; setPage(p); handleSearch(false, p); }}>上一页</Button>
                    <span className="text-xs font-bold font-mono px-2">{page} / {Math.ceil(total / PAGE_SIZE)}</span>
                    <Button variant="outline" size="sm" className="h-8 rounded-lg" disabled={page * PAGE_SIZE >= total} onClick={() => { const p = page + 1; setPage(p); handleSearch(false, p); }}>下一页</Button>
                </div>
              )}
          </div>

          {loading && results.length === 0 ? (
            <div className="grid gap-4"><Card className="p-20"><EmptyState title="正在检索..." description="正在从云端调取测绘数据，请稍后" /></Card></div>
          ) : !hasSearched ? (
            <Card className="p-24 text-center border-dashed bg-white/[0.01]">
                <div className="h-16 w-16 rounded-full bg-white/5 flex items-center justify-center mx-auto mb-6">
                    <Database className="h-8 w-8 text-muted-foreground opacity-30" />
                </div>
                <h3 className="text-xl font-bold text-foreground/50">准备就绪</h3>
                <p className="text-sm text-muted-foreground mt-2">输入查询语句并点击搜索开始发现资产</p>
            </Card>
          ) : results.length === 0 ? (
            <Card className="p-24 text-center">
                <EmptyState title="未找到匹配项" description="试试更通用的搜索词或检查语法是否正确" />
            </Card>
          ) : (
            <Card className="overflow-hidden">
                <Table>
                    <TableHeader className="bg-white/5">
                        <TableRow>
                            <TableHead>资产目标</TableHead>
                            <TableHead className="w-32">端口</TableHead>
                            <TableHead>服务协议</TableHead>
                            <TableHead className="text-right">地理位置</TableHead>
                        </TableRow>
                    </TableHeader>
                    <TableBody>
                        {results.map((r, i) => (
                        <TableRow key={i} className="group">
                            <TableCell>
                                <div className="flex flex-col gap-0.5">
                                    <span className="font-mono text-sm font-bold text-foreground group-hover:text-primary transition-colors">{r.url || r.host}</span>
                                    {r.ip && <span className="text-[10px] text-muted-foreground font-mono">{r.ip}</span>}
                                </div>
                            </TableCell>
                            <TableCell>
                                <Badge variant="outline" className="font-mono text-primary bg-primary/5 border-primary/20">{r.port}</Badge>
                            </TableCell>
                            <TableCell>
                                <div className="flex items-center gap-2">
                                    <span className="text-sm font-medium capitalize">{r.protocol || r.service || "unknown"}</span>
                                    {r.title && <span className="text-xs text-muted-foreground truncate max-w-[200px]">({r.title})</span>}
                                </div>
                            </TableCell>
                            <TableCell className="text-right">
                                <span className="text-xs text-muted-foreground">{r.location || "Global"}</span>
                            </TableCell>
                        </TableRow>
                        ))}
                    </TableBody>
                </Table>
            </Card>
          )}
        </section>
      </div>
    </div>
  );
}
