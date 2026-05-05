import { useState, useEffect, useRef } from "react";
import { Link } from "react-router-dom";
import { api, SearchResult } from "../lib/api";
import { useToast, EmptyState, Table, Button, Input } from "../components";

const ENGINES = [
  { key: "fofa", label: "FOFA", placeholder: 'domain="example.com"' },
  { key: "hunter", label: "Hunter", placeholder: 'ip.port=443' },
  { key: "quake", label: "Quake", placeholder: 'service:http' },
];

const PAGE_SIZE = 20;

export default function EnginesPage() {
  const [activeEngine, setActiveEngine] = useState("fofa");
  const [query, setQuery] = useState("");
  const [results, setResults] = useState<SearchResult[]>([]);
  const [total, setTotal] = useState(0);
  const [page, setPage] = useState(1);
  const [loading, setLoading] = useState(false);
  const [hasSearched, setHasSearched] = useState(false);
  const toast = useToast();
  const abortRef = useRef<AbortController | null>(null);

  const placeholder = ENGINES.find((e) => e.key === activeEngine)?.placeholder ?? "";

  useEffect(() => {
    return () => {
      if (abortRef.current) {
        abortRef.current.abort();
      }
    };
  }, []);

  async function handleSearch(resetPage = true, explicitPage?: number) {
    if (!query.trim()) {
      toast("请输入查询语句", "warning");
      return;
    }

    if (abortRef.current) {
      abortRef.current.abort();
    }
    const ctrl = new AbortController();
    abortRef.current = ctrl;

    const currentPage = resetPage ? 1 : (explicitPage ?? page);
    if (resetPage) setPage(1);

    setLoading(true);
    setHasSearched(true);

    try {
      const res = await api.searchEngine(
        {
          engine: activeEngine,
          query: query.trim(),
          page: currentPage,
          size: PAGE_SIZE,
        },
        ctrl.signal
      );
      setResults(res.data ?? []);
      setTotal(res.total ?? 0);
      if (explicitPage && explicitPage !== page) {
        setPage(explicitPage);
      }
    } catch (err: any) {
      if (err instanceof DOMException && err.name === "AbortError") return;
      toast("搜索失败: " + (err?.message || String(err)), "error");
    } finally {
      setLoading(false);
    }
  }

  function handleKeyDown(e: React.KeyboardEvent) {
    if (e.key === "Enter") {
      e.preventDefault();
      handleSearch(true);
    }
  }

  function handlePageChange(newPage: number) {
    handleSearch(false, newPage);
  }

  const totalPages = Math.max(1, Math.ceil(total / PAGE_SIZE));

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-xl font-semibold">互联网搜索引擎</h1>
          <p className="text-sm text-text-tertiary mt-1">
            集成 FOFA、Hunter、Quake 三大平台，快速发现互联网资产
          </p>
        </div>
        <Link
          to="/engines/keys"
          className="text-sm text-brand-primary hover:text-brand-secondary transition-colors"
        >
          配置 API Key →
        </Link>
      </div>

      {/* Engine Tabs */}
      <div className="flex gap-2">
        {ENGINES.map((e) => (
          <button
            key={e.key}
            onClick={() => {
              setActiveEngine(e.key);
              setResults([]);
              setHasSearched(false);
              setPage(1);
            }}
            className={`rounded-lg px-4 py-2 text-sm font-medium transition-colors ${
              activeEngine === e.key
                ? "bg-brand-primary/12 text-brand-primary ring-1 ring-brand-primary/25"
                : "text-text-tertiary hover:bg-white/[0.04] hover:text-text-secondary"
            }`}
          >
            {e.label}
          </button>
        ))}
      </div>

      {/* Search Input */}
      <div className="cyber-glass p-5">
        <div className="flex gap-3">
          <div className="flex-1">
            <Input
              placeholder={`输入查询语句 (${placeholder})`}
              value={query}
              onChange={(e) => setQuery(e.target.value)}
              onKeyDown={handleKeyDown}
              className="w-full"
            />
          </div>
          <Button onClick={() => handleSearch(true)} disabled={loading}>
            {loading ? "搜索中..." : "搜索"}
          </Button>
        </div>
      </div>

      {/* Results */}
      <div className="cyber-glass p-0 overflow-hidden">
        {hasSearched && results.length === 0 && !loading ? (
          <EmptyState title="未找到结果" description="尝试修改查询语句后重新搜索" />
        ) : (
          <Table<SearchResult>
            columns={[
              { key: "ip", header: "IP" },
              { key: "port", header: "端口", render: (r) => (r.port ? String(r.port) : "-") },
              { key: "domain", header: "域名", render: (r) => r.domain || "-" },
              { key: "title", header: "标题", render: (r) => r.title || "-" },
              { key: "service", header: "服务", render: (r) => r.service || "-" },
              { key: "protocol", header: "协议", render: (r) => r.protocol || "-" },
              { key: "location", header: "位置", render: (r) => r.location || "-" },
              { key: "os", header: "系统", render: (r) => r.os || "-" },
            ]}
            data={results}
            loading={loading}
            emptyText={hasSearched ? "暂无数据" : "输入查询语句并点击搜索"}
          />
        )}

        {/* Pagination */}
        {hasSearched && total > 0 && (
          <div className="flex items-center justify-between border-t border-white/[0.06] px-4 py-3">
            <div className="text-xs text-text-tertiary">
              共 {total} 条结果，第 {page} / {totalPages} 页
            </div>
            <div className="flex gap-2">
              <button
                onClick={() => handlePageChange(page - 1)}
                disabled={page <= 1 || loading}
                className="rounded-md border border-white/[0.10] bg-white/[0.03] px-3 py-1.5 text-xs text-text-secondary hover:bg-white/[0.06] disabled:opacity-40 disabled:cursor-not-allowed transition-colors"
              >
                上一页
              </button>
              <button
                onClick={() => handlePageChange(page + 1)}
                disabled={page >= totalPages || loading}
                className="rounded-md border border-white/[0.10] bg-white/[0.03] px-3 py-1.5 text-xs text-text-secondary hover:bg-white/[0.06] disabled:opacity-40 disabled:cursor-not-allowed transition-colors"
              >
                下一页
              </button>
            </div>
          </div>
        )}
      </div>
    </div>
  );
}
