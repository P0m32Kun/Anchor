import { useState, useEffect, useRef } from "react";
import { Link } from "react-router-dom";
import { api, SearchResult } from "../lib/api";
import { useToast, EmptyState, Button, Input } from "../components";
import { Table, TableHeader, TableBody, TableRow, TableHead, TableCell } from "../components/Table";

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
    <div className="page-shell space-y-6">
      <div className="page-header">
        <div>
          <div className="page-eyebrow">Discovery</div>
          <h1 className="page-title">互联网搜索引擎</h1>
          <p className="page-subtitle">
            集成 FOFA、Hunter、Quake 三大平台，快速发现互联网资产
          </p>
        </div>
        <Link
          to="/engines/keys"
          className="text-sm link-cyber"
        >
          配置 API Key -&gt;
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
                ? "filter-pill-active"
                : "filter-pill"
            }`}
          >
            {e.label}
          </button>
        ))}
      </div>

      {/* Search Input */}
      <div className="panel p-5">
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
      <div className="panel p-0 overflow-hidden">
        {hasSearched && results.length === 0 && !loading ? (
          <EmptyState title="未找到结果" description="尝试修改查询语句后重新搜索" />
        ) : (
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>IP</TableHead>
                <TableHead>端口</TableHead>
                <TableHead>域名</TableHead>
                <TableHead>标题</TableHead>
                <TableHead>服务</TableHead>
                <TableHead>协议</TableHead>
                <TableHead>位置</TableHead>
                <TableHead>系统</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {loading ? (
                <TableRow>
                  <TableCell colSpan={8} className="text-center py-8 text-muted-foreground">
                    加载中...
                  </TableCell>
                </TableRow>
              ) : results.length === 0 ? (
                <TableRow>
                  <TableCell colSpan={8} className="text-center py-8 text-muted-foreground">
                    {hasSearched ? "暂无数据" : "输入查询语句并点击搜索"}
                  </TableCell>
                </TableRow>
              ) : (
                results.map((r, i) => (
                  <TableRow key={i}>
                    <TableCell>{r.ip}</TableCell>
                    <TableCell>{r.port ? String(r.port) : "-"}</TableCell>
                    <TableCell>{r.domain || "-"}</TableCell>
                    <TableCell>{r.title || "-"}</TableCell>
                    <TableCell>{r.service || "-"}</TableCell>
                    <TableCell>{r.protocol || "-"}</TableCell>
                    <TableCell>{r.location || "-"}</TableCell>
                    <TableCell>{r.os || "-"}</TableCell>
                  </TableRow>
                ))
              )}
            </TableBody>
          </Table>
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
