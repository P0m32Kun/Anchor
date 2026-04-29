import { useEffect, useState, useCallback } from "react";
import { Link } from "react-router-dom";
import { api } from "../lib/api";
import { useStore } from "../lib/store";
import { useProjectId, useToast, EmptyState, Table } from "../components";

function AssetTypeBadge({ type }: { type: string }) {
  const colors: Record<string, string> = {
    domain: "bg-brand-primary/15 text-brand-primary",
    ip: "bg-purple-100 text-purple-700",
    url: "bg-brand-success/15 text-brand-success",
  };
  return (
    <span className={`px-2 py-0.5 rounded text-xs font-medium ${colors[type] || "bg-zinc-800/60 text-zinc-400"}`}>
      {type}
    </span>
  );
}

function StatusCodeBadge({ code }: { code?: number }) {
  if (!code) return <span className="text-zinc-500 text-xs">—</span>;
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

  const [activeTab, setActiveTab] = useState<"assets" | "web" | "ports">("assets");
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [selectedAsset, setSelectedAsset] = useState<string | null>(null);
  const toast = useToast();

  const loadAssets = useCallback(async () => {
    if (!projectId) return;
    setLoading(true);
    setError(null);
    try {
      const data = await api.listAssets(projectId);
      setAssets(data ?? []);
    } catch (err) {
      const msg = String(err);
      setError(msg);
      console.error(err);
    } finally {
      setLoading(false);
    }
  }, [projectId, setAssets]);

  const loadWebEndpoints = useCallback(() => {
    if (!projectId) return;
    api.listWebEndpoints(projectId).then((data) => setWebEndpoints(data ?? [])).catch(console.error);
  }, [projectId, setWebEndpoints]);

  const loadPorts = useCallback(
    (assetId: string) => {
      api.listPorts(assetId).then((p) => setPorts(assetId, p)).catch(console.error);
    },
    [setPorts]
  );

  useEffect(() => {
    if (!projectId) return;
    loadAssets();
    loadWebEndpoints();
  }, [projectId, loadAssets, loadWebEndpoints]);

  const startDiscovery = async () => {
    if (!projectId) return;
    setLoading(true);
    try {
      await api.startAssetDiscovery(projectId);
      toast("资产发现工作流已启动", "success");
    } catch (err) {
      toast("启动失败: " + String(err), "error");
    } finally {
      setLoading(false);
    }
  };

  const [filterTitle, setFilterTitle] = useState("");
  const [filterAsset, setFilterAsset] = useState("");

  const filteredAssets = assets.filter((a) => {
    if (filterAsset && !a.value.toLowerCase().includes(filterAsset.toLowerCase())) return false;
    return true;
  });

  const filteredWeb = webEndpoints.filter((ep) => {
    if (filterTitle && !ep.title?.toLowerCase().includes(filterTitle.toLowerCase())) return false;
    return true;
  });

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
        <span className="text-zinc-500 text-xs">{String(row.normalized_value)}</span>
      ),
    },
    {
      key: "source_tools",
      header: "来源工具",
      width: "180px",
      render: (row) => (
        <span className="text-zinc-500 text-xs">
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
      <div className="max-w-5xl space-y-6">
        <div>
          <h1 className="text-2xl font-bold">资产清单</h1>
          <p className="text-zinc-400 text-sm mt-1">查看和管理项目发现的资产、Web 端点和端口信息</p>
        </div>
        <div className="bg-zinc-900/50 backdrop-blur-md border border-zinc-800/80 rounded-xl p-8 text-center">
          <p className="text-zinc-400 mb-4">请先从 Dashboard 选择一个项目</p>
          <Link to="/" className="text-blue-600 hover:underline">前往 Dashboard</Link>
        </div>
      </div>
    );
  }

  return (
    <div className="max-w-5xl space-y-6">
      {/* 标题区 */}
      <div>
        <h1 className="text-2xl font-bold">资产清单</h1>
        <p className="text-zinc-400 text-sm mt-1">查看和管理项目发现的资产、Web 端点和端口信息</p>
      </div>

      {/* 操作区 */}
      <div className="flex items-center justify-between">
        <Link to={`/projects/${currentProject.id}`} className="text-sm text-blue-600 hover:underline">
          ← 返回目标管理
        </Link>
        <button
          onClick={startDiscovery}
          disabled={loading}
          className="bg-green-600 text-white px-4 py-2 rounded hover:bg-green-500 disabled:opacity-50"
        >
          {loading ? "启动中..." : "资产发现"}
        </button>
      </div>

      <div className="flex gap-2 border-b">
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
                ? "border-green-600 text-green-700"
                : "border-transparent text-zinc-400 hover:text-zinc-300"
            }`}
          >
            {t.label}
          </button>
        ))}
      </div>

      {activeTab === "assets" && (
        <section className="bg-zinc-900/50 backdrop-blur-md border border-zinc-800/80 rounded-xl p-4">
          <div className="flex gap-2 mb-4">
            <input
              type="text"
              placeholder="筛选资产值..."
              value={filterAsset}
              onChange={(e) => setFilterAsset(e.target.value)}
              className="bg-zinc-800/60 border border-zinc-700/60 rounded-lg px-3 py-1.5 text-sm text-zinc-200 placeholder-zinc-500 focus:outline-none focus:border-green-600/50 w-48"
            />
            <button
              onClick={() => { setFilterAsset(""); }}
              className="text-zinc-500 text-sm hover:text-zinc-300 px-2"
            >
              清除
            </button>
          </div>
          {loading ? (
            <div className="py-12 text-center text-zinc-400">加载中...</div>
          ) : error ? (
            <div className="py-12 text-center">
              <p className="text-brand-danger mb-2">加载失败: {error}</p>
              <button onClick={loadAssets} className="text-sm text-blue-600 hover:underline">
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
            />
          )}
        </section>
      )}

      {activeTab === "web" && (
        <section className="bg-zinc-900/50 backdrop-blur-md border border-zinc-800/80 rounded-xl p-4 ">
          <div className="flex gap-2 mb-4">
            <input
              type="text"
              placeholder="筛选标题..."
              value={filterTitle}
              onChange={(e) => setFilterTitle(e.target.value)}
              className="bg-zinc-800/60 border border-zinc-700/60 rounded-lg px-3 py-1.5 text-sm text-zinc-200 placeholder-zinc-500 focus:outline-none focus:border-green-600/50 w-48"
            />
          </div>
          {filteredWeb.length > 0 ? (
            <table className="w-full text-sm">
              <thead>
                <tr className="text-left text-zinc-400 border-b">
                  <th className="pb-2">URL</th>
                  <th className="pb-2">状态码</th>
                  <th className="pb-2">Title</th>
                  <th className="pb-2">技术栈</th>
                </tr>
              </thead>
              <tbody>
                {filteredWeb.map((we) => (
                  <tr key={we.id} className="border-b last:border-0 hover:bg-zinc-800/40">
                    <td className="py-2 font-mono text-xs">
                      <a href={we.url} target="_blank" rel="noreferrer" className="text-blue-600 hover:underline">
                        {we.url}
                      </a>
                    </td>
                    <td className="py-2">
                      <StatusCodeBadge code={we.status_code} />
                    </td>
                    <td className="py-2 text-zinc-300">{we.title || "—"}</td>
                    <td className="py-2">
                      <div className="flex flex-wrap gap-1">
                        {(we.technologies || []).map((t) => (
                          <span key={t} className="px-1.5 py-0.5 bg-slate-100 text-slate-600 rounded text-xs">
                            {t}
                          </span>
                        ))}
                        {!(we.technologies || []).length && <span className="text-zinc-500">—</span>}
                      </div>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          ) : (
            <div className="text-zinc-500 text-sm text-center py-8">暂无匹配的 Web 端点</div>
          )}
        </section>
      )}

      {activeTab === "ports" && (
        <div className="space-y-4">
          <section className="bg-zinc-900/50 backdrop-blur-md border border-zinc-800/80 rounded-xl p-4 ">
            <h3 className="font-semibold mb-2 text-sm text-zinc-400">选择 IP 资产查看端口</h3>
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
                      ? "bg-purple-100 border-purple-300 text-purple-700"
                      : "bg-zinc-900/50 backdrop-blur-md border border-zinc-800/80 rounded-xl border-zinc-800 text-zinc-400 hover:bg-zinc-800/40"
                  }`}
                >
                  {a.value}
                </button>
              ))}
            </div>
          </section>

          {selectedAsset && ports[selectedAsset] && (
            <section className="bg-zinc-900/50 backdrop-blur-md border border-zinc-800/80 rounded-xl p-4 ">
              <h3 className="font-semibold mb-2">
                端口 ({ports[selectedAsset].length})
              </h3>
              {ports[selectedAsset].length > 0 ? (
                <div className="grid grid-cols-4 gap-2">
                  {ports[selectedAsset].map((p) => (
                    <div
                      key={p.id}
                      className="border rounded p-2 text-center text-sm hover:bg-zinc-800/40"
                    >
                      <div className="font-mono font-semibold text-lg">{p.port}</div>
                      <div className="text-zinc-500 text-xs">
                        {p.protocol} / {p.state}
                      </div>
                    </div>
                  ))}
                </div>
              ) : (
                <div className="text-zinc-500 text-sm text-center py-4">暂无端口数据</div>
              )}
            </section>
          )}
        </div>
      )}
    </div>
  );
}
