import { useEffect, useRef, useState } from "react";
import { API_BASE } from "../lib/api";
import { EmptyState } from "../components";

interface Worker {
  id: string;
  name: string;
  mode: string;
  status: string;
  endpoint: string;
}

function statusColor(status: string): string {
  switch (status) {
    case "online":
      return "bg-brand-success";
    case "busy":
      return "bg-accent-yellow";
    case "offline":
    default:
      return "bg-text-tertiary";
  }
}

function statusLabel(status: string): string {
  switch (status) {
    case "online":
      return "在线";
    case "busy":
      return "运行中";
    case "offline":
      return "离线";
    default:
      return status;
  }
}

export default function WorkersPage() {
  const [workers, setWorkers] = useState<Worker[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [offlineWorkers, setOfflineWorkers] = useState<Worker[]>([]);
  const fetchingRef = useRef(false);

  useEffect(() => {
    let ctrl = new AbortController();

    async function fetchWorkers() {
      if (fetchingRef.current) return; // Prevent request storm
      fetchingRef.current = true;

      try {
        const res = await fetch(`${API_BASE}/workers`, { signal: ctrl.signal });
        if (!res.ok) throw new Error("fetch failed");
        const data: Worker[] = await res.json();

        const onlineOrBusy = data.filter(
          (w) => w.status === "online" || w.status === "busy"
        );
        const offline = data.filter((w) => w.status === "offline");

        setWorkers(onlineOrBusy);
        setOfflineWorkers(offline);
        setError(null);
      } catch (err) {
        if (err instanceof DOMException && err.name === "AbortError") return;
        setError("连接失败，请检查服务是否运行");
      } finally {
        setLoading(false);
        fetchingRef.current = false;
      }
    }

    fetchWorkers();
    const interval = setInterval(() => {
      // Abort any pending request before starting a new one to prevent overlap
      ctrl.abort();
      ctrl = new AbortController();
      fetchWorkers();
    }, 5000);

    return () => {
      ctrl.abort();
      clearInterval(interval);
    };
  }, []);

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-xl font-semibold">Workers</h1>
          <p className="text-sm text-text-tertiary mt-1">
            管理扫描节点，查看工具健康状态
          </p>
        </div>
      </div>

      <div className="liquid-glass rounded-xl p-5">
        <div className="flex items-center justify-between mb-4">
          <div>
            <div className="text-sm font-medium">已注册 Worker</div>
            <div className="text-xs text-text-tertiary mt-1">
              通过 Docker 部署 Worker，注册到 Server 后显示在此
            </div>
          </div>
        </div>

        {loading ? (
          <div className="text-sm text-text-tertiary py-6 text-center bg-white/[0.02] rounded-lg">
            加载中...
          </div>
        ) : error ? (
          <div className="text-sm text-red-400 py-6 text-center bg-white/[0.02] rounded-lg">
            {error}
          </div>
        ) : workers.length === 0 && offlineWorkers.length === 0 ? (
          <EmptyState
            title="暂无 Worker"
            description="通过 Docker 部署 Worker，使用 Server 生成的 token 注册后显示在此"
          />
        ) : (
          <div className="space-y-3">
            {offlineWorkers.length > 0 && (
              <div
                className="text-xs text-amber-400 bg-amber-400/10 border border-amber-400/20 rounded-lg px-3 py-2"
                role="status"
              >
                ⚠️ 检测到 {offlineWorkers.length} 个 Worker 离线（{offlineWorkers.map(w => w.name).join("、")}）
              </div>
            )}
            {workers.map((w) => (
              <div
                key={w.id}
                className="flex items-center justify-between bg-white/[0.02] rounded-lg px-4 py-3"
              >
                <div className="flex items-center gap-3">
                  <div
                    className={`w-2.5 h-2.5 rounded-full ${statusColor(w.status)}`}
                    aria-hidden="true"
                  />
                  <div>
                    <div className="text-sm font-medium">{w.name}</div>
                    <div className="text-xs text-text-tertiary mt-0.5">
                      {w.endpoint || w.id}
                    </div>
                  </div>
                </div>
                <div className="text-xs text-text-tertiary">
                  {statusLabel(w.status)}
                </div>
              </div>
            ))}
            {offlineWorkers.map((w) => (
              <div
                key={w.id}
                className="flex items-center justify-between bg-white/[0.02] rounded-lg px-4 py-3 opacity-50"
                title="该 Worker 当前离线"
              >
                <div className="flex items-center gap-3">
                  <div
                    className={`w-2.5 h-2.5 rounded-full ${statusColor(w.status)}`}
                    aria-hidden="true"
                  />
                  <div>
                    <div className="text-sm font-medium">{w.name}</div>
                    <div className="text-xs text-text-tertiary mt-0.5">
                      {w.endpoint || w.id}
                    </div>
                  </div>
                </div>
                <div className="text-xs text-accent-red">
                  {statusLabel(w.status)}
                </div>
              </div>
            ))}
          </div>
        )}
      </div>
    </div>
  );
}
