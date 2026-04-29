import { useEffect, useState } from "react";
import { API_BASE } from "../lib/api";

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
      return "bg-green-400";
    case "busy":
      return "bg-yellow-400";
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

  useEffect(() => {
    let cancelled = false;

    async function fetchWorkers() {
      try {
        const res = await fetch(`${API_BASE}/workers`);
        if (!res.ok) throw new Error("fetch failed");
        const data: Worker[] = await res.json();
        if (!cancelled) {
          setWorkers(data.filter((w) => w.status === "online" || w.status === "busy"));
          setError(null);
        }
      } catch {
        if (!cancelled) setError("连接失败，请检查服务是否运行");
      } finally {
        if (!cancelled) setLoading(false);
      }
    }

    fetchWorkers();
    const interval = setInterval(fetchWorkers, 5000);
    return () => {
      cancelled = true;
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
        ) : workers.length === 0 ? (
          <div className="text-sm text-text-tertiary py-6 text-center bg-white/[0.02] rounded-lg">
            暂无 Worker
            <div className="mt-2 text-xs text-text-tertiary">
              通过 Docker 部署 Worker，使用 Server 生成的 token 注册
            </div>
          </div>
        ) : (
          <div className="space-y-3">
            {workers.map((w) => (
              <div
                key={w.id}
                className="flex items-center justify-between bg-white/[0.02] rounded-lg px-4 py-3"
              >
                <div className="flex items-center gap-3">
                  <div className={`w-2.5 h-2.5 rounded-full ${statusColor(w.status)}`} />
                  <div>
                    <div className="text-sm font-medium">{w.name}</div>
                    <div className="text-xs text-text-tertiary mt-0.5">{w.endpoint || w.id}</div>
                  </div>
                </div>
                <div className="text-xs text-text-tertiary">
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
