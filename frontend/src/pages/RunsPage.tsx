import { useEffect, useState } from "react";
import { useParams } from "react-router-dom";
import { api } from "../lib/api";
import type { Run, ScanTask } from "../lib/api";

const statusColors: Record<string, string> = {
  pending: "bg-yellow-500/15 text-yellow-300",
  running: "bg-blue-500/15 text-blue-300",
  completed: "bg-green-500/15 text-green-300",
  failed: "bg-red-500/15 text-red-300",
  cancelled: "bg-zinc-800/60 text-zinc-400",
};

const statusLabels: Record<string, string> = {
  pending: "待启动",
  running: "运行中",
  completed: "已完成",
  failed: "失败",
  cancelled: "已取消",
};

const taskStatusColors: Record<string, string> = {
  created: "bg-zinc-800/60 text-zinc-400",
  queued: "bg-yellow-500/15 text-yellow-300",
  running: "bg-blue-500/15 text-blue-300",
  completed: "bg-green-500/15 text-green-300",
  failed: "bg-red-500/15 text-red-300",
};

export default function RunsPage() {
  const { id: projectId } = useParams<{ id: string }>();
  const [runs, setRuns] = useState<Run[]>([]);
  const [selectedRun, setSelectedRun] = useState<string | null>(null);
  const [tasks, setTasks] = useState<ScanTask[]>([]);
  const [loading, setLoading] = useState(false);

  useEffect(() => {
    if (!projectId) return;
    api.listRuns(projectId).then((data) => setRuns(data ?? [])).catch(console.error);
  }, [projectId]);

  const loadTasks = async (runId: string) => {
    setSelectedRun(runId);
    setLoading(true);
    try {
      const data = await api.getRunTasks(runId);
      setTasks(data ?? []);
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="max-w-5xl space-y-6">
      <h1 className="text-2xl font-bold text-zinc-100 tracking-tight">扫描执行</h1>

      <section className="bg-zinc-900/50 backdrop-blur-md border border-zinc-800/80 rounded-xl p-6">
        <div className="flex justify-between items-center mb-4">
          <h2 className="text-base font-medium text-zinc-200">执行历史</h2>
          <span className="text-zinc-500 text-sm">共 {runs.length} 次</span>
        </div>
        <div className="divide-y divide-zinc-800/60">
          {runs.map((run) => (
            <div
              key={run.id}
              onClick={() => loadTasks(run.id)}
              className={`py-3 px-2 flex items-center justify-between text-sm cursor-pointer transition-all rounded-lg ${
                selectedRun === run.id ? "bg-zinc-800/60" : "hover:bg-zinc-800/40"
              }`}
            >
              <div className="flex items-center gap-4">
                <span className="font-medium text-zinc-200">{run.name}</span>
                <span className="text-zinc-500 text-xs font-mono">{run.id.slice(-8)}</span>
              </div>
              <div className="flex items-center gap-3">
                <span className={`px-2 py-0.5 rounded text-xs font-medium ${statusColors[run.status] || "bg-zinc-800/60 text-zinc-400"}`}>
                  {statusLabels[run.status] || run.status}
                </span>
                <span className="text-zinc-500 text-xs">
                  {run.created_at ? new Date(run.created_at).toLocaleString("zh-CN") : "—"}
                </span>
              </div>
            </div>
          ))}
          {runs.length === 0 && (
            <div className="py-8 text-center text-zinc-500">暂无扫描执行记录</div>
          )}
        </div>
      </section>

      {selectedRun && (
        <section className="bg-zinc-900/50 backdrop-blur-md border border-zinc-800/80 rounded-xl p-6">
          <div className="flex justify-between items-center mb-4">
            <h2 className="text-base font-medium text-zinc-200">任务详情</h2>
            {loading && <span className="text-zinc-500 text-sm">加载中...</span>}
          </div>
          <div className="divide-y divide-zinc-800/60">
            {tasks.map((task) => (
              <div key={task.id} className="py-3 px-2 flex items-center justify-between text-sm">
                <div className="flex items-center gap-4">
                  <span className="font-medium text-zinc-200 w-20">{task.tool}</span>
                  <span className="font-mono text-zinc-500 text-xs">{task.id.slice(-8)}</span>
                </div>
                <span className={`px-2 py-0.5 rounded text-xs font-medium ${taskStatusColors[task.status] || "bg-zinc-800/60 text-zinc-400"}`}>
                  {task.status}
                </span>
              </div>
            ))}
            {tasks.length === 0 && !loading && (
              <div className="py-8 text-center text-zinc-500">暂无任务</div>
            )}
          </div>
        </section>
      )}
    </div>
  );
}
