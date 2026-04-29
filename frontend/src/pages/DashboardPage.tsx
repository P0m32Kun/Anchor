import { useNavigate } from "react-router-dom";
import { useCallback, useEffect, useState } from "react";
import { API_BASE } from "../lib/api";
import { useStore } from "../lib/store";

interface Worker {
  id: string;
  name: string;
  mode: string;
  status: string;
}

export default function DashboardPage() {
  const navigate = useNavigate();
  const currentProject = useStore((state) => state.currentProject);
  const [onlineWorkers, setOnlineWorkers] = useState(0);
  const [isLoading, setIsLoading] = useState(true);
  const [connectionError, setConnectionError] = useState<string | null>(null);

  const fetchWorkers = useCallback(async () => {
    setIsLoading(true);
    setConnectionError(null);
    try {
      const res = await fetch(`${API_BASE}/workers`);
      if (!res.ok) {
        throw new Error(`HTTP ${res.status}`);
      }
      const data: Worker[] = await res.json();
      const count = data.filter(
        (w) => w.status === "online" || w.status === "busy"
      ).length;
      setOnlineWorkers(count);
      setConnectionError(null);
    } catch (err) {
      const message = err instanceof Error ? err.message : "网络连接失败";
      setConnectionError(message);
      setOnlineWorkers(0);
    } finally {
      setIsLoading(false);
    }
  }, []);

  useEffect(() => {
    let cancelled = false;

    async function poll() {
      if (cancelled) return;
      await fetchWorkers();
    }

    poll();
    const interval = setInterval(poll, 5000);
    return () => {
      cancelled = true;
      clearInterval(interval);
    };
  }, [fetchWorkers]);

  return (
    <div className="space-y-6">
      {/* 顶部状态条 */}
      <div className="liquid-glass rounded-xl p-4">
        <div className="flex items-center justify-between mb-3">
          <h2 className="text-sm font-semibold text-text-secondary">项目状态</h2>
        </div>
        <div className="grid grid-cols-4 gap-4">
          <div>
            <div className="text-xs text-text-tertiary mb-1">当前项目</div>
            {currentProject ? (
              <button
                onClick={() => navigate(`/projects/${currentProject.id}`)}
                className="text-sm font-medium text-brand-primary hover:text-brand-secondary transition-colors"
              >
                {currentProject.name} →
              </button>
            ) : (
              <button
                onClick={() => navigate("/projects")}
                className="text-sm font-medium text-brand-primary hover:text-brand-secondary transition-colors"
              >
                前往创建 →
              </button>
            )}
          </div>
          <div>
            <div className="text-xs text-text-tertiary mb-1">在线 Worker</div>
            {isLoading ? (
              <div className="text-sm font-medium text-text-tertiary animate-pulse">
                加载中...
              </div>
            ) : connectionError ? (
              <div className="space-y-1">
                <div className="text-sm font-medium text-red-400">
                  连接失败
                </div>
                <button
                  onClick={fetchWorkers}
                  className="text-xs text-brand-primary hover:text-brand-secondary transition-colors"
                  title={`错误: ${connectionError}`}
                >
                  重试
                </button>
              </div>
            ) : (
              <div
                className={`text-sm font-medium ${
                  onlineWorkers > 0 ? "text-green-400" : "text-text-tertiary"
                }`}
              >
                {onlineWorkers}
              </div>
            )}
          </div>
          <div>
            <div className="text-xs text-text-tertiary mb-1">运行中任务</div>
            <div className="text-sm font-medium">0</div>
          </div>
          <div>
            <div className="text-xs text-text-tertiary mb-1">Scope 状态</div>
            <div className="text-sm font-medium text-yellow-400">未配置</div>
          </div>
        </div>
      </div>

      {/* 核心区域 */}
      <div className="grid grid-cols-3 gap-6">
        {/* 左侧：高优先级资产 */}
        <div className="liquid-glass rounded-xl p-4">
          <h3 className="text-sm font-semibold text-text-secondary mb-3">高优先级资产</h3>
          <div className="text-sm text-text-tertiary py-8 text-center">
            暂无资产
            <div className="mt-2">
              <button
                onClick={() => navigate("/targets")}
                className="text-xs text-brand-primary hover:text-brand-secondary transition-colors"
              >
                导入目标 →
              </button>
            </div>
          </div>
        </div>

        {/* 中间：待验证 Finding */}
        <div className="liquid-glass rounded-xl p-4">
          <h3 className="text-sm font-semibold text-text-secondary mb-3">待验证 Finding</h3>
          <div className="text-sm text-text-tertiary py-8 text-center">
            暂无 Finding
            <div className="mt-2">
              <button
                onClick={() => navigate("/findings")}
                className="text-xs text-brand-primary hover:text-brand-secondary transition-colors"
              >
                查看全部 →
              </button>
            </div>
          </div>
        </div>

        {/* 右侧：活动流 */}
        <div className="liquid-glass rounded-xl p-4">
          <h3 className="text-sm font-semibold text-text-secondary mb-3">最近活动</h3>
          <div className="space-y-3">
            <div className="flex items-start gap-2 text-xs">
              <div className="w-1.5 h-1.5 rounded-full bg-brand-primary mt-1.5 shrink-0" />
              <div>
                <span className="text-text-secondary">应用启动</span>
                <div className="text-text-tertiary mt-0.5">刚刚</div>
              </div>
            </div>
            <div className="flex items-start gap-2 text-xs">
              <div className="w-1.5 h-1.5 rounded-full bg-text-tertiary mt-1.5 shrink-0" />
              <div>
                <span className="text-text-secondary">v0.2 初始化完成</span>
                <div className="text-text-tertiary mt-0.5">4 个工具模板已加载</div>
              </div>
            </div>
          </div>
        </div>
      </div>

      {/* 失败任务 + 复测队列 */}
      <div className="grid grid-cols-2 gap-6">
        <div className="liquid-glass rounded-xl p-4">
          <h3 className="text-sm font-semibold text-text-secondary mb-3">失败/部分成功任务</h3>
          <div className="text-sm text-text-tertiary py-4 text-center">无</div>
        </div>
        <div className="liquid-glass rounded-xl p-4">
          <h3 className="text-sm font-semibold text-text-secondary mb-3">复测队列</h3>
          <div className="text-sm text-text-tertiary py-4 text-center">无</div>
        </div>
      </div>
    </div>
  );
}
