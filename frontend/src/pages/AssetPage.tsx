import { useEffect, useState, useCallback, useMemo } from "react";
import { Link } from "react-router-dom";
import { api, PAGE_ALL } from "../lib/api";
import { useStore } from "../lib/store";
import { useProjectId, useToast, EmptyState, Table, SkeletonList } from "../components";

const ASSET_TYPES = ["all", "domain", "url", "ip", "cidr", "service"] as const;

function AssetTypeBadge({ type }: { type: string }) {
  const colors: Record<string, string> = {
    domain: "bg-brand-primary/15 text-brand-primary",
    ip: "bg-brand-purple/15 text-brand-purple",
    url: "bg-brand-success/15 text-brand-success",
    cidr: "bg-brand-warning/15 text-brand-warning",
    service: "bg-accent-teal/15 text-accent-teal",
  };
  return (
    <span className={`px-2 py-0.5 rounded text-xs font-medium ${colors[type] || "bg-white/[0.04] text-text-tertiary"}`}>
      {type}
    </span>
  );
}

function StatusCodeBadge({ code }: { code?: number }) {
  if (!code) return <span className="text-text-quaternary text-xs">—</span>;
  const color =
    code >= 200 && code < 300
      ? "bg-brand-success/15 text-brand-success"
      : code >= 300 && code < 400
      ? "bg-accent-yellow/15 text-accent-yellow"
      : "bg-brand-danger/15 text-brand-danger";
  return (
    <span className={`px-2 py-0.5 rounded text-xs font-medium ${color}`}>
      {code}
    </span>
  );
}

export default function AssetPage() {
  const projectId = useProjectId();
  const currentProject = useStore((state) => state.currentProject);
  const assets = useStore((state) => state.assets) ?? [];
  const setAssets = useStore((state) => state.setAssets);
  const webEndpoints = useStore((state) => state.webEndpoints) ?? [];
  const setWebEndpoints = useStore((state) => state.setWebEndpoints);
  const ports = useStore((state) => state.ports);
  const setPorts = useStore((state) => state.setPorts);
  const services = useStore((state) => state.services);
  const setServices = useStore((state) => state.setServices);

  const [activeTab, setActiveTab] = useState<"assets" | "web" | "ports">("assets");
  const loading = useStore((state) => state.assetsLoading);
  const error = useStore((state) => state.assetsError);
  const setAssetsLoading = useStore((state) => state.setAssetsLoading);
  const setAssetsError = useStore((state) => state.setAssetsError);
  const [selectedAsset, setSelectedAsset] = useState<string | null>(null);
  const toast = useToast();
  const [portsAllLoading, setPortsAllLoading] = useState(false);
  const [portPage, setPortPage] = useState(1);
  const PORT_PAGE_SIZE = 20;

  const loadAssets = useCallback(async (signal?: AbortSignal) => {
    if (!projectId) return;
    setAssetsLoading(true);
    setAssetsError(null);
    try {
      const data = await api.listAssets(projectId, PAGE_ALL, signal);
      setAssets(data.data ?? []);
    } catch (err) {
      if (err instanceof DOMException && err.name === "AbortError") return;
      const msg = err instanceof Error ? err.message : String(err);
      setAssetsError(msg);
      console.error(err);
    } finally {
      setAssetsLoading(false);
    }
  }, [projectId, setAssets, setAssetsLoading, setAssetsError]);

  const loadWebEndpoints = useCallback((signal?: AbortSignal) => {
    if (!projectId) return;
    api.listWebEndpoints(projectId, PAGE_ALL, signal)
      .then((res) => setWebEndpoints(res.data ?? []))
      .catch((err) => {
        if (err instanceof DOMException && err.name === "AbortError") return;
        console.error(err);
      });
  }, [projectId, setWebEndpoints]);

  const loadPorts = useCallback(
    (assetId: string, signal?: AbortSignal) => {
      api.listPorts(assetId, signal)
        .then((p) => setPorts(assetId, p))
        .catch((err) => {
          if (err instanceof DOMException && err.name === "AbortError") return;
          console.error(err);
        });
    },
    [setPorts]
  );

  useEffect(() => {
    if (!projectId) return;
    const ctrl = new AbortController();
    loadAssets(ctrl.signal);
    loadWebEndpoints(ctrl.signal);
    return () => ctrl.abort();
  }, [projectId, loadAssets, loadWebEndpoints]);

  const startDiscovery = async () => {
    if (!projectId) return;
    setAssetsLoading(true);
    try {
      await api.startAssetDiscovery(projectId);
      toast("资产发现工作流已启动", "success");
    } catch (err) {
      toast("启动失败: " + String(err), "error");
    } finally {
      setAssetsLoading(false);
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

  const filteredPorts = useMemo(() => {
    if (!selectedAsset || !ports[selectedAsset]) return [];
    return ports[selectedAsset].filter((p) => {
      if (filterPort && !String(p.port).includes(filterPort)) return false;
      return true;
    });
  }, [ports, selectedAsset, filterPort]);

  const assetColumns: { key: string; header: string; width?: string; render?: (row: Record<string, unknown>) => React.ReactNode }[] = [
    {
      key: "type",
      header: "类型",
      width: "100px",
      render: (row) => <AssetTypeBadge type={String(row.type)} />,
    },
    { key: "value", header: "资产值" },
    {
      key: "normalized_value",
      header: "归一化值",
      render: (row) => (
        <span className="text-text-quaternary text-xs">{String(row.normalized_value)}</span>
      ),
    },
    {
      key: "source_tools",
      header: "来源工具",
      width: "180px",
      render: (row) => (
        <span className="text-text-quaternary text-xs">
          {Array.isArray(row.source_tools) ? row.source_tools.join(", ") : "—"}
        </span>
      ),
    },
    {
      key: "first_seen",
      header: "首次发现",
      width: "180px",
      render: (row) => new Date(String(row.first_seen)).toLocaleString(),
    },
  ];

  if (!currentProject) {
    return (
      <div className="page-shell space-y-6">
        <div>
          <h1 className="text-2xl font-bold">资产清单</h1>
          <p className="text-text-tertiary text-sm mt-1">查看和管理项目发现的资产、Web 端点和端口信息</p>
        </div>
        <div className="panel p-8 text-center">
          <p className="text-text-tertiary mb-4">请先从 Dashboard 选择一个项目</p>
          <Link to="/" className="link-cyber">前往 Dashboard</Link>
        </div>
      </div>
    );
  }

  return (
    <div className="page-shell space-y-6">
      <div className="page-header">
        <div>
          <div className="page-eyebrow text-accent-teal">Step 2</div>
          <h1 className="page-title">资产清单</h1>
          <p className="page-subtitle">按资产、Web 端点和开放端口查看发现结果，为后续指纹驱动扫描做准备。</p>
        </div>
        <button
          onClick={startDiscovery}
          disabled={loading}
          className="btn-cyber-primary disabled:opacity-50"
        >
          {loading ? "启动中..." : "资产发现"}
        </button>
      </div>

      <div className="flex gap-2 border-b border-white/[0.08]">
        {[
          { key: "assets", label: `资产 (${assets.length})` },
          { key: "web", label: `Web 端点 (${webEndpoints.length})` },
          { key: "ports", label: "端口" },
        ].map((t) => (
          <button
            key={t.key}
            onClick={() => setActiveTab(t.key as any)}
            className={`px-4 py-2 text-sm font-medium border-b-2 transition ${
              activeTab === t.key
                ? "border-brand-primary text-text-primary"
                : "border-transparent text-text-tertiary hover:text-text-secondary"
            }`}
          >
            {t.label}
          </button>
        ))}
      </div>

      {activeTab === "assets" && (
        <section className="panel p-4">
          <div className="flex flex-col gap-3 mb-4">
            <div className="flex flex-wrap gap-2">
              {ASSET_TYPES.map((t) => (
                <button
                  key={t}
                  onClick={() => setFilterType(t)}
                  className={`px-3 py-1 rounded text-xs font-medium border transition ${
                    filterType === t
                      ? "filter-pill-active"
                      : "filter-pill"
                  }`}
                >
                  {t === "all" ? "全部" : t}
                </button>
              ))}
            </div>
            <div className="flex items-center gap-2">
              <input
                type="text"
                placeholder="筛选资产值..."
                value={filterAsset}
                onChange={(e) => setFilterAsset(e.target.value)}
                className="input-dark w-48 !py-1.5"
              />
              <button
                onClick={() => { setFilterType("all"); setFilterAsset(""); }}
                className="text-text-quaternary text-sm hover:text-text-secondary px-2"
              >
                清除
              </button>
              <span className="text-text-quaternary text-xs ml-auto">共 {filteredAssets.length} 个资产</span>
            </div>
          </div>
          {loading ? (
            <SkeletonList count={5} />
          ) : error ? (
            <div className="py-12 text-center">
              <p className="text-brand-danger mb-2">加载失败: {error}</p>
              <button onClick={() => loadAssets()} className="text-sm link-cyber">
                重试
              </button>
            </div>
          ) : filteredAssets.length === 0 ? (
            <EmptyState
              title="暂无资产"
              description="当前项目还没有发现任何资产，点击右上角「资产发现」开始扫描。"
            />
          ) : (
            <Table
              columns={assetColumns}
              data={filteredAssets as unknown as Record<string, unknown>[]}
              emptyText="暂无匹配的资产"
              maxHeight={480}
            />
          )}
        </section>
      )}

      {activeTab === "web" && (
        <section className="panel p-4">
          <div className="flex flex-wrap items-center gap-2 mb-4">
            <input
              type="text"
              placeholder="筛选标题..."
              value={filterTitle}
              onChange={(e) => setFilterTitle(e.target.value)}
              className="input-dark w-48 !py-1.5"
            />
            <input
              type="text"
              placeholder="搜索技术栈..."
              value={filterTech}
              onChange={(e) => setFilterTech(e.target.value)}
              className="input-dark w-48 !py-1.5"
            />
            <button
              onClick={() => { setFilterTitle(""); setFilterTech(""); }}
              className="text-text-quaternary text-sm hover:text-text-secondary px-2"
            >
              清除
            </button>
            <span className="text-text-quaternary text-xs ml-auto">共 {filteredWeb.length} 个端点</span>
          </div>
          {filteredWeb.length > 0 ? (
            <div className="max-h-[480px] overflow-auto">
            <table className="table-cyber text-sm">
              <thead>
                <tr>
                  <th className="pb-2">URL</th>
                  <th className="pb-2">状态码</th>
                  <th className="pb-2">Title</th>
                  <th className="pb-2">技术栈</th>
                </tr>
              </thead>
              <tbody>
                {filteredWeb.map((we) => (
                  <tr key={we.id}>
                    <td className="py-2 font-mono text-xs">
                      <a href={we.url} target="_blank" rel="noreferrer" className="link-cyber">
                        {we.url}
                      </a>
                    </td>
                    <td className="py-2">
                      <StatusCodeBadge code={we.status_code} />
                    </td>
                    <td className="py-2 text-text-secondary">{we.title || "—"}</td>
                    <td className="py-2">
                      <div className="flex flex-wrap gap-1">
                        {(we.technologies || []).map((t) => (
                          <span key={t} className="px-1.5 py-0.5 bg-brand-primary/10 text-brand-primary rounded text-xs border border-brand-primary/20">
                            {t}
                          </span>
                        ))}
                        {!(we.technologies || []).length && <span className="text-text-quaternary">—</span>}
                      </div>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
            </div>
          ) : (
            <EmptyState title="暂无 Web 端点" description="当前项目还没有发现任何 Web 端点" />
          )}
        </section>
      )}

      {activeTab === "ports" && (
        <div className="space-y-4">
          <section className="panel p-4">
            <h3 className="font-semibold mb-2 text-sm text-text-tertiary">选择 IP 资产查看端口</h3>
            <div className="flex flex-wrap gap-2">
              {assets.filter((a) => a.type === "ip").map((a) => (
                <button
                  key={a.id}
                  onClick={() => {
                    setSelectedAsset(a.id);
                    loadPorts(a.id);
                  }}
                  className={`px-3 py-1 rounded text-sm border ${
                    selectedAsset === a.id
                      ? "filter-pill-active"
                      : "filter-pill"
                  }`}
                >
                  {a.value}
                </button>
              ))}
            </div>
          </section>

          {selectedAsset && ports[selectedAsset] && (
            <section className="panel p-4">
              <div className="flex items-center justify-between mb-2">
                <h3 className="font-semibold">端口</h3>
                <div className="flex items-center gap-2">
                  <input
                    type="text"
                    placeholder="筛选端口..."
                    value={filterPort}
                    onChange={(e) => setFilterPort(e.target.value)}
                    className="input-dark w-32 !py-1"
                  />
                  <button
                    onClick={() => setFilterPort("")}
                    className="text-text-quaternary text-sm hover:text-text-secondary px-2"
                  >
                    清除
                  </button>
                </div>
              </div>
              <p className="text-text-quaternary text-xs mb-2">共 {filteredPorts.length} 个端口</p>
              {filteredPorts.length > 0 ? (
                <div className="grid grid-cols-4 gap-2">
                  {filteredPorts.map((p) => (
                    <div
                      key={p.id}
                      className="surface-item p-2 text-center text-sm"
                    >
                      <div className="font-mono font-semibold text-lg">{p.port}</div>
                      <div className="text-text-quaternary text-xs">
                        {p.protocol} / {p.state}
                      </div>
                    </div>
                  ))}
                </div>
              ) : (
                <EmptyState title="暂无端口数据" description="当前 IP 资产还没有扫描到任何开放端口" />
              )}
            </section>
          )}
        </div>
      )}
    </div>
  );
}
