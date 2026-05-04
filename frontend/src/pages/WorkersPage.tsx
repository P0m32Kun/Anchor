import { useEffect, useRef, useState } from "react";
import { API_BASE, request } from "../lib/api";
import { getApiToken } from "../lib/config";
import { useStore } from "../lib/store";
import { EmptyState, SkeletonList, useToast } from "../components";

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
  const [offlineWorkers, setOfflineWorkers] = useState<Worker[]>([]);
  const loading = useStore((state) => state.workersLoading);
  const error = useStore((state) => state.workersError);
  const setWorkersLoading = useStore((state) => state.setWorkersLoading);
  const setWorkersError = useStore((state) => state.setWorkersError);
  const fetchingRef = useRef(false);
  const isInitialRef = useRef(true);
  const toast = useToast();
  const [deletingWorkerId, setDeletingWorkerId] = useState<string | null>(null);
  const [bulkDeleting, setBulkDeleting] = useState(false);

  async function handleDeleteWorker(w: Worker) {
    if (!window.confirm(`确认删除离线 Worker "${w.name}"？此操作不可恢复。`)) return;
    setDeletingWorkerId(w.id);
    try {
      await request(`/workers/${w.id}`, { method: "DELETE" });
      setOfflineWorkers((prev) => prev.filter((x) => x.id !== w.id));
      toast("Worker 已删除", "success");
    } catch (err) {
      toast("删除失败: " + (err instanceof Error ? err.message : String(err)), "error");
    } finally {
      setDeletingWorkerId(null);
    }
  }

  async function handleBulkDelete() {
    if (!window.confirm(`确认删除全部 ${offlineWorkers.length} 个离线 Worker？此操作不可恢复。`)) return;
    setBulkDeleting(true);
    try {
      const results = await Promise.allSettled(
        offlineWorkers.map((w) => request(`/workers/${w.id}`, { method: "DELETE" }))
      );
      const failed = results.filter((r) => r.status === "rejected");
      if (failed.length > 0) {
        toast(`清理完成，${failed.length} 个删除失败`, "warning");
      } else {
        toast("已清理全部离线 Worker", "success");
      }
      setOfflineWorkers([]);
    } catch (err) {
      toast("清理失败: " + (err instanceof Error ? err.message : String(err)), "error");
    } finally {
      setBulkDeleting(false);
    }
  }

  useEffect(() => {
    let ctrl = new AbortController();

    async function fetchWorkers() {
      if (fetchingRef.current) return; // Prevent request storm
      fetchingRef.current = true;

      if (isInitialRef.current) {
        setWorkersLoading(true);
      }
      try {
        const token = getApiToken();
        const headers: Record<string, string> = token ? { Authorization: `Bearer ${token}` } : {};
        const res = await fetch(`${API_BASE}/workers`, { signal: ctrl.signal, headers });
        if (!res.ok) throw new Error("fetch failed");
        const data: Worker[] = await res.json();

        const onlineOrBusy = data.filter(
          (w) => w.status === "online" || w.status === "busy"
        );
        const offline = data.filter((w) => w.status === "offline");

        setWorkers(onlineOrBusy);
        setOfflineWorkers(offline);
        setWorkersError(null);
      } catch (err) {
        if (err instanceof DOMException && err.name === "AbortError") return;
        setWorkersError("连接失败，请检查服务是否运行");
      } finally {
        if (isInitialRef.current) {
          setWorkersLoading(false);
          isInitialRef.current = false;
        }
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
      fetchingRef.current = false;
      clearInterval(interval);
    };
  }, [setWorkersLoading, setWorkersError]);

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

      <div className="cyber-glass p-5">
        <div className="flex items-center justify-between mb-4">
          <div>
            <div className="text-sm font-medium">已注册 Worker</div>
            <div className="text-xs text-text-tertiary mt-1">
              通过 Docker 部署 Worker，注册到 Server 后显示在此
            </div>
          </div>
        </div>

        {loading ? (
          <SkeletonList count={3} />
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
                className="flex items-center gap-2 text-xs text-amber-400 bg-amber-400/10 border border-amber-400/20 rounded-lg px-3 py-2"
                role="status"
              >
                <span className="flex-1">
                  ⚠️ 检测到 {offlineWorkers.length} 个 Worker 离线（{offlineWorkers.map(w => w.name).join("、")}）
                </span>
                <button
                  onClick={handleBulkDelete}
                  disabled={bulkDeleting}
                  className="text-xs text-red-400 border border-red-400/30 rounded px-2 py-0.5 hover:bg-red-400/10 disabled:opacity-50 disabled:cursor-not-allowed shrink-0"
                  title="删除全部离线 Worker"
                >
                  {bulkDeleting ? "清理中..." : "一键清理"}
                </button>
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
                className="flex items-center justify-between bg-white/[0.02] rounded-lg px-4 py-3 opacity-50 group"
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
                <div className="flex items-center gap-3">
                  <button
                    onClick={() => handleDeleteWorker(w)}
                    disabled={deletingWorkerId === w.id}
                    className="text-xs text-red-400 hover:text-red-300 hover:underline disabled:opacity-50 disabled:cursor-not-allowed opacity-0 group-hover:opacity-100 transition-opacity"
                    title="删除 Worker"
                  >
                    {deletingWorkerId === w.id ? "删除中..." : "删除"}
                  </button>
                  <div className="text-xs text-accent-red">
                    {statusLabel(w.status)}
                  </div>
                </div>
              </div>
            ))}
          </div>
        )}
      </div>
    </div>
  );
}
