import { useEffect, useState, useCallback, useMemo } from "react";
import { Link } from "react-router-dom";
import { api, PAGE_ALL } from "../lib/api";
import { useStore } from "../lib/store";
import { useResource } from "../hooks";
import {
  useProjectId,
  useToast,
  EmptyState,
  Table,
  TableHeader,
  TableBody,
  TableRow,
  TableHead,
  TableCell,
  Badge,
  SkeletonList,
  Card,
  Button,
  Input
} from "../components";
import {
  Box,
  Globe,
  Network,
  Search,
  ChevronLeft,
  ArrowRight,
  ExternalLink,
  RefreshCcw,
  Layers,
  Terminal
} from "lucide-react";
import { cn } from "../lib/utils";

const ASSET_TYPES = ["all", "domain", "url", "ip", "cidr", "service"] as const;

function AssetTypeBadge({ type }: { type: string }) {
  const map: Record<string, string> = {
    domain: "bg-blue-500/10 text-blue-400 border-blue-500/20",
    ip: "bg-purple-500/10 text-purple-400 border-purple-500/20",
    url: "bg-emerald-500/10 text-emerald-400 border-emerald-500/20",
    cidr: "bg-amber-500/10 text-amber-400 border-amber-500/20",
    service: "bg-cyan-500/10 text-cyan-400 border-cyan-500/20",
  };
  return (
    <Badge variant="outline" className={cn("px-1.5 py-0 text-[10px] uppercase tracking-wider", map[type])}>
      {type}
    </Badge>
  );
}

function StatusCodeBadge({ code }: { code?: number }) {
  if (!code) return <span className="text-muted-foreground text-xs">—</span>;
  const variant =
    code >= 200 && code < 300
      ? "bg-emerald-500/10 text-emerald-400 border-emerald-500/20"
      : code >= 300 && code < 400
      ? "bg-amber-500/10 text-amber-400 border-amber-500/20"
      : "bg-rose-500/10 text-rose-400 border-rose-500/20";
  return (
    <Badge variant="outline" className={cn("px-1.5 py-0 text-[10px] font-bold font-mono", variant)}>
      {code}
    </Badge>
  );
}

export default function AssetPage() {
  const projectId = useProjectId();
  const currentProject = useStore((state) => state.currentProject);
  const assets = useStore((state) => state.assets) ?? [];
  const setAssets = useStore((state) => state.setAssets);
  const webEndpoints = useStore((state) => state.webEndpoints) ?? [];
  const setWebEndpoints = useStore((state) => state.setWebEndpoints);
  const [activeTab, setActiveTab] = useState<"assets" | "web" | "ports">("assets");
  const toast = useToast();

  const servicePorts = useStore((state) => state.servicePorts);
  const servicePortsLoading = useStore((state) => state.servicePortsLoading);
  const setServicePorts = useStore((state) => state.setServicePorts);
  const setServicePortsLoading = useStore((state) => state.setServicePortsLoading);
  const [portPage, setPortPage] = useState(1);
  const PORT_PAGE_SIZE = 20;

  const {
    loading,
    error,
    reload: loadAssets,
  } = useResource(
    async (signal) => {
      if (!projectId) return;
      const data = await api.listAssets(projectId, PAGE_ALL, signal);
      setAssets(data.data ?? []);
    },
    [projectId],
    undefined
  );

  const loadWebEndpoints = useCallback((signal?: AbortSignal) => {
    if (!projectId) return;
    api.listWebEndpoints(projectId, PAGE_ALL, signal)
      .then((res) => setWebEndpoints(res.data ?? []))
      .catch((err) => {
        if (err instanceof DOMException && err.name === "AbortError") return;
        console.error(err);
      });
  }, [projectId, setWebEndpoints]);

  const loadServicePorts = useCallback(
    async (signal?: AbortSignal) => {
      if (!projectId) return;
      setServicePortsLoading(true);
      try {
        const data = await api.listServicePorts(projectId, signal);
        setServicePorts(data);
      } catch (err) {
        if (err instanceof DOMException && err.name === "AbortError") return;
        console.error(err);
      } finally {
        setServicePortsLoading(false);
      }
    },
    [projectId, setServicePorts, setServicePortsLoading]
  );

  useEffect(() => {
    if (!projectId) return;
    const ctrl = new AbortController();
    loadWebEndpoints(ctrl.signal);
    return () => ctrl.abort();
  }, [projectId, loadWebEndpoints]);

  useEffect(() => {
    if (activeTab !== "ports" || !projectId) return;
    if (servicePorts.length > 0) return;
    const ctrl = new AbortController();
    loadServicePorts(ctrl.signal);
    return () => ctrl.abort();
  }, [activeTab, projectId, servicePorts.length, loadServicePorts]);

  const [discoveryLoading, setDiscoveryLoading] = useState(false);

  const startDiscovery = async () => {
    if (!projectId) return;
    setDiscoveryLoading(true);
    try {
      await api.startAssetDiscovery(projectId);
      toast("资产发现工作流已启动", "success");
    } catch (err) {
      toast("启动失败: " + String(err), "error");
    } finally {
      setDiscoveryLoading(false);
    }
  };

  const [filterTitle, setFilterTitle] = useState("");
  const [filterAsset, setFilterAsset] = useState("");
  const [filterType, setFilterType] = useState<string>("all");
  const [filterTech, setFilterTech] = useState("");
  const [filterPort, setFilterPort] = useState("");

  const filteredAssets = useMemo(() => {
    return assets.filter((a) => {
      if (filterType !== "all" && a.type !== filterType) return false;
      if (filterAsset && !a.value.toLowerCase().includes(filterAsset.toLowerCase())) return false;
      return true;
    });
  }, [assets, filterType, filterAsset]);

  const filteredWeb = useMemo(() => {
    return webEndpoints.filter((ep) => {
      if (filterTitle && !ep.title?.toLowerCase().includes(filterTitle.toLowerCase())) return false;
      if (filterTech) {
        const techs = (ep.technologies || []).join(" ").toLowerCase();
        if (!techs.includes(filterTech.toLowerCase())) return false;
      }
      return true;
    });
  }, [webEndpoints, filterTitle, filterTech]);

  const filteredPortRows = useMemo(() => {
    if (!filterPort) return servicePorts;
    const q = filterPort.toLowerCase();
    return servicePorts.filter((row) =>
      String(row.port).includes(q) ||
      row.ip.toLowerCase().includes(q) ||
      row.service_name.toLowerCase().includes(q) ||
      row.title.toLowerCase().includes(q) ||
      (row.technologies || []).join(" ").toLowerCase().includes(q)
    );
  }, [servicePorts, filterPort]);

  const totalPortPages = Math.max(1, Math.ceil(filteredPortRows.length / PORT_PAGE_SIZE));
  const paginatedPortRows = useMemo(() => {
    const start = (portPage - 1) * PORT_PAGE_SIZE;
    return filteredPortRows.slice(start, start + PORT_PAGE_SIZE);
  }, [filteredPortRows, portPage]);

  useEffect(() => {
    setPortPage(1);
  }, [filterPort]);

  if (!currentProject) {
    return (
      <div className="flex flex-col items-center justify-center py-20 text-center">
        <div className="h-16 w-16 rounded-full bg-muted flex items-center justify-center mb-4">
            <Box className="h-8 w-8 text-muted-foreground opacity-50" />
        </div>
        <h2 className="text-xl font-bold">资产清单</h2>
        <p className="text-muted-foreground mt-1 mb-6">请先从左侧菜单或总览选择一个项目</p>
        <Link to="/">
            <Button variant="primary">前往总览</Button>
        </Link>
      </div>
    );
  }

  return (
    <div className="space-y-8 animate-in fade-in duration-500">
      <div className="flex items-start justify-between">
        <div>
          <div className="flex items-center gap-2 text-cyan-500 font-bold text-xs uppercase tracking-widest mb-1.5">
            <Layers className="h-3.5 w-3.5" />
            Step 2: Asset Discovery
          </div>
          <h1 className="text-3xl font-bold tracking-tight text-foreground">资产清单</h1>
          <p className="text-muted-foreground mt-1">汇总管理域名、IP、Web 端点及开放端口。</p>
        </div>
        <Button variant="primary" onClick={startDiscovery} loading={discoveryLoading}>
          <RefreshCcw className={cn("mr-2 h-4 w-4", discoveryLoading && "animate-spin")} />
          资产发现
        </Button>
      </div>

      <div className="flex items-center gap-1 border-b border-border w-full">
        {[
          { key: "assets", label: "基础资产", count: assets.length, icon: Box },
          { key: "web", label: "Web 端点", count: webEndpoints.length, icon: Globe },
          { key: "ports", label: "开放端口", count: servicePorts.length, icon: Network },
        ].map((t) => (
          <button
            key={t.key}
            onClick={() => setActiveTab(t.key as any)}
            className={cn(
              "group relative flex items-center gap-2 px-4 py-3 text-sm font-medium transition-all",
              activeTab === t.key
                ? "text-primary"
                : "text-muted-foreground hover:text-foreground hover:bg-muted/50 rounded-t-md"
            )}
          >
            <t.icon className={cn("h-4 w-4", activeTab === t.key ? "text-primary" : "text-muted-foreground group-hover:text-foreground")} />
            {t.label}
            <Badge variant="secondary" className="h-5 px-1.5 text-[10px] ml-1">{t.count}</Badge>
            {activeTab === t.key && (
              <div className="absolute bottom-0 left-0 h-0.5 w-full bg-primary" />
            )}
          </button>
        ))}
      </div>

      {activeTab === "assets" && (
        <section className="space-y-4">
          <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-4">
            <div className="flex flex-wrap gap-1.5">
              {ASSET_TYPES.map((t) => (
                <Button
                  key={t}
                  variant={filterType === t ? "primary" : "outline"}
                  size="sm"
                  onClick={() => setFilterType(t)}
                  className="h-7 px-3 text-xs capitalize"
                >
                  {t === "all" ? "全部类型" : t}
                </Button>
              ))}
            </div>
            <div className="relative">
              <Search className="absolute left-3 top-2.5 h-3.5 w-3.5 text-muted-foreground" />
              <Input
                placeholder="搜索资产值..."
                value={filterAsset}
                onChange={(e) => setFilterAsset(e.target.value)}
                className="pl-9 h-9 w-64"
              />
            </div>
          </div>

          <Card>
            {loading ? (
              <div className="p-6">
                <SkeletonList count={8} />
              </div>
            ) : error ? (
              <div className="py-20 text-center">
                <p className="text-destructive font-medium mb-4">加载失败: {error}</p>
                <Button variant="outline" size="sm" onClick={() => loadAssets()}>重试</Button>
              </div>
            ) : filteredAssets.length === 0 ? (
              <div className="py-20 text-center">
                <EmptyState title="未找到资产" description="当前筛选条件下没有资产数据。" />
              </div>
            ) : (
              <Table>
                <TableHeader>
                  <TableRow>
                    <TableHead className="w-24">类型</TableHead>
                    <TableHead>资产值</TableHead>
                    <TableHead className="hidden md:table-cell">归一化</TableHead>
                    <TableHead>来源</TableHead>
                    <TableHead className="text-right">发现时间</TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {filteredAssets.map((a) => (
                    <TableRow key={a.id}>
                      <TableCell><AssetTypeBadge type={a.type} /></TableCell>
                      <TableCell className="font-mono text-sm font-medium">{a.value}</TableCell>
                      <TableCell className="hidden md:table-cell font-mono text-xs text-muted-foreground truncate max-w-[150px]">
                        {a.normalized_value}
                      </TableCell>
                      <TableCell>
                        <div className="flex gap-1 flex-wrap">
                            {a.source_tools?.map(s => (
                                <Badge key={s} variant="secondary" className="px-1 py-0 text-[9px] uppercase font-bold text-muted-foreground">{s}</Badge>
                            ))}
                        </div>
                      </TableCell>
                      <TableCell className="text-right text-xs text-muted-foreground">
                        {new Date(a.first_seen).toLocaleDateString()}
                      </TableCell>
                    </TableRow>
                  ))}
                </TableBody>
              </Table>
            )}
          </Card>
        </section>
      )}

      {activeTab === "web" && (
        <section className="space-y-4">
          <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-4">
            <div className="flex flex-wrap gap-2 flex-1">
               <div className="relative flex-1 max-w-sm">
                  <Search className="absolute left-3 top-2.5 h-3.5 w-3.5 text-muted-foreground" />
                  <Input
                    placeholder="搜索标题或 URL..."
                    value={filterTitle}
                    onChange={(e) => setFilterTitle(e.target.value)}
                    className="pl-9 h-9"
                  />
               </div>
               <div className="relative flex-1 max-w-sm">
                  <Terminal className="absolute left-3 top-2.5 h-3.5 w-3.5 text-muted-foreground" />
                  <Input
                    placeholder="搜索技术栈 (React, Nginx...)"
                    value={filterTech}
                    onChange={(e) => setFilterTech(e.target.value)}
                    className="pl-9 h-9"
                  />
               </div>
            </div>
            <div className="text-xs text-muted-foreground">匹配到 {filteredWeb.length} 个端点</div>
          </div>

          <Card>
            {filteredWeb.length > 0 ? (
              <Table>
                <TableHeader>
                  <TableRow>
                    <TableHead>端点 URL</TableHead>
                    <TableHead className="w-20">状态</TableHead>
                    <TableHead>页面标题</TableHead>
                    <TableHead>技术栈</TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {filteredWeb.map((we) => (
                    <TableRow key={we.id}>
                      <TableCell className="font-mono text-xs">
                        <a href={we.url} target="_blank" rel="noreferrer" className="text-primary hover:underline flex items-center gap-1">
                          {we.url}
                          <ExternalLink className="h-3 w-3" />
                        </a>
                      </TableCell>
                      <TableCell>
                        <StatusCodeBadge code={we.status_code} />
                      </TableCell>
                      <TableCell className="text-sm font-medium text-foreground max-w-[200px] truncate">
                        {we.title || <span className="text-muted-foreground italic">—</span>}
                      </TableCell>
                      <TableCell>
                        <div className="flex flex-wrap gap-1">
                          {we.technologies?.map((t) => (
                            <Badge key={t} variant="outline" className="h-5 px-1.5 text-[10px] bg-primary/5 text-primary border-primary/20">
                              {t}
                            </Badge>
                          ))}
                        </div>
                      </TableCell>
                    </TableRow>
                  ))}
                </TableBody>
              </Table>
            ) : (
              <div className="py-20 text-center">
                <EmptyState title="未找到 Web 端点" description="当前筛选条件下没有 Web 数据。" />
              </div>
            )}
          </Card>
        </section>
      )}

      {activeTab === "ports" && (
        <section className="space-y-4">
          <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-4">
            <div className="relative flex-1 max-w-md">
                <Search className="absolute left-3 top-2.5 h-3.5 w-3.5 text-muted-foreground" />
                <Input
                    placeholder="搜索 IP, 端口, 服务名或技术栈..."
                    value={filterPort}
                    onChange={(e) => setFilterPort(e.target.value)}
                    className="pl-9 h-9"
                />
            </div>
            <div className="flex items-center gap-4 text-xs text-muted-foreground">
                <span>共 {filteredPortRows.length} 个端口</span>
                {totalPortPages > 1 && (
                    <div className="flex items-center gap-1.5">
                        <Button variant="outline" size="sm" className="h-7 w-7 p-0" onClick={() => setPortPage(p => Math.max(1, p-1))} disabled={portPage <= 1}>
                            <ChevronLeft className="h-4 w-4" />
                        </Button>
                        <span>{portPage} / {totalPortPages}</span>
                        <Button variant="outline" size="sm" className="h-7 w-7 p-0" onClick={() => setPortPage(p => Math.min(totalPortPages, p+1))} disabled={portPage >= totalPortPages}>
                            <ArrowRight className="h-4 w-4" />
                        </Button>
                    </div>
                )}
            </div>
          </div>

          <Card>
            {servicePortsLoading ? (
              <div className="p-6">
                <SkeletonList count={8} />
              </div>
            ) : servicePorts.length === 0 ? (
              <div className="py-20 text-center">
                <EmptyState title="未探测到端口" description="目前没有任何端口发现记录。" />
              </div>
            ) : (
              <Table>
                <TableHeader>
                  <TableRow>
                    <TableHead>IP 地址</TableHead>
                    <TableHead className="w-20">端口</TableHead>
                    <TableHead className="w-20">状态</TableHead>
                    <TableHead>服务</TableHead>
                    <TableHead>指纹信息 / 标题</TableHead>
                    <TableHead>技术栈</TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {paginatedPortRows.map((row, i) => (
                    <TableRow key={i}>
                      <TableCell className="font-mono text-xs">{row.ip}</TableCell>
                      <TableCell>
                        <span className="font-mono font-bold text-primary">{row.port}</span>
                      </TableCell>
                      <TableCell>
                        <Badge variant={row.state === 'open' ? 'success' : 'secondary'} className="px-1.5 py-0 text-[10px] font-bold uppercase tracking-wider">
                            {row.state}
                        </Badge>
                      </TableCell>
                      <TableCell>
                        <span className="text-sm font-semibold capitalize">{row.service_name}</span>
                      </TableCell>
                      <TableCell className="max-w-[240px]">
                        <div className="flex flex-col">
                            {row.title && <span className="text-sm font-medium text-foreground truncate">{row.title}</span>}
                            {row.url && (
                                <a href={row.url} target="_blank" rel="noreferrer" className="text-xs text-primary hover:underline truncate">
                                    {row.url}
                                </a>
                            )}
                            {!row.title && !row.url && <span className="text-muted-foreground text-xs italic">No banner data</span>}
                        </div>
                      </TableCell>
                      <TableCell>
                        <div className="flex flex-wrap gap-1">
                            {row.technologies?.map((t: string) => (
                                <Badge key={t} variant="outline" className="h-5 px-1.5 text-[10px] bg-primary/5 text-primary border-primary/20">
                                    {t}
                                </Badge>
                            ))}
                        </div>
                      </TableCell>
                    </TableRow>
                  ))}
                </TableBody>
              </Table>
            )}
          </Card>
        </section>
      )}
    </div>
  );
}
