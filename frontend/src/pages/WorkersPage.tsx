import { useState } from "react";

export default function WorkersPage() {
  const [localWorkerRunning, setLocalWorkerRunning] = useState(false);

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

      {/* 本地 Worker */}
      <div className="liquid-glass rounded-xl p-5">
        <div className="flex items-center justify-between mb-4">
          <div className="flex items-center gap-3">
            <div className={`w-2.5 h-2.5 rounded-full ${localWorkerRunning ? 'bg-green-400' : 'bg-text-tertiary'}`} />
            <div>
              <div className="text-sm font-medium">本地 Worker</div>
              <div className="text-xs text-text-tertiary">localhost</div>
            </div>
          </div>
          <button
            onClick={() => setLocalWorkerRunning(!localWorkerRunning)}
            className={`px-4 py-1.5 text-xs font-medium rounded-lg transition-colors ${
              localWorkerRunning
                ? 'bg-red-500/10 text-red-400 border border-red-500/20 hover:bg-red-500/20'
                : 'bg-brand-primary text-white hover:bg-brand-secondary'
            }`}
          >
            {localWorkerRunning ? '停止' : '启动'}
          </button>
        </div>

        {localWorkerRunning ? (
          <div className="grid grid-cols-5 gap-3">
            {['Subfinder', 'httpx', 'Naabu', 'Nuclei', 'Rod'].map((tool) => (
              <div key={tool} className="bg-white/[0.03] rounded-lg p-3 text-center">
                <div className="text-xs font-medium text-text-secondary">{tool}</div>
                <div className="text-[10px] text-green-400 mt-1">就绪</div>
              </div>
            ))}
          </div>
        ) : (
          <div className="text-sm text-text-tertiary py-4 text-center bg-white/[0.02] rounded-lg">
            Worker 未运行
            <div className="mt-2 text-xs text-text-tertiary">
              点击"启动"启动本地 Worker 进程
            </div>
          </div>
        )}
      </div>

      {/* 远程 Worker */}
      <div className="liquid-glass rounded-xl p-5">
        <div className="flex items-center justify-between mb-4">
          <div>
            <div className="text-sm font-medium">远程 Worker</div>
            <div className="text-xs text-text-tertiary mt-1">外网场景：在云主机上部署 Worker</div>
          </div>
        </div>
        <div className="text-sm text-text-tertiary py-6 text-center bg-white/[0.02] rounded-lg">
          暂无远程 Worker
          <div className="mt-2 text-xs text-text-tertiary">
            在 Workers 页面生成 token，复制到远程机器上启动
          </div>
        </div>
      </div>
    </div>
  );
}
