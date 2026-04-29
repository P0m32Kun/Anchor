import { useEffect, useState } from "react";
import { useParams } from "react-router-dom";
import { api } from "../lib/api";
import { useStore } from "../lib/store";
import type { Run, ScanTask, ToolTemplate } from "../lib/api";

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
  const { id } = useParams<{ id: string }>();
  const currentProject = useStore((s) => s.currentProject);
  const [runs, setRuns] = useState<Run[]>([]);
  const [templates, setTemplates] = useState<ToolTemplate[]>([]);
  const [selectedRun, setSelectedRun] = useState<string | null>(null);
  const [tasks, setTasks] = useState<ScanTask[]>([]);
  const [loading, setLoading] = useState(false);
  const [showCreate, setShowCreate] = useState(false);
  const [creating, setCreating] = useState(false);

  const projectId = id || currentProject?.id;

  useEffect(() => {
    if (!projectId) return;
    api.listRuns(projectId).then((data) => setRuns(data ?? [])).catch(console.error);
    api.listToolTemplates().then((data) => setTemplates(data ?? [])).catch(console.error);
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

  const handleCreate = async (templateId: string, name: string) => {
    if (!projectId) return;
    setCreating(true);
    try {
      await api.createRun(projectId, { tool_template_id: templateId, name });
      setShowCreate(false);
      const data = await api.listRuns(projectId);
      setRuns(data ?? []);
    } catch (err) {
      console.error(err);
    } finally {
      setCreating(false);
    }
  };

  return (
    <div className="max-w-5xl space-y-6">
      <h1 className="text-2xl font-bold text-zinc-100 tracking-tight">扫描执行</h1>

      <section className="bg-zinc-900/50 backdrop-blur-md border border-zinc-800/80 rounded-xl p-6">
        <div className="flex justify-between items-center mb-4">
          <h2 className="text-base font-medium text-zinc-200">执行历史</h2>
          <div className="flex items-center gap-3">
            <span className="text-zinc-500 text-sm">共 {runs.length} 次</span>
            {projectId && (
              <button
                onClick={() => setShowCreate(true)}
                className="bg-brand-primary text-white text-xs px-3 py-1.5 rounded-lg hover:opacity-90 transition-opacity"
              >
                新建扫描
              </button>
            )}
          </div>
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
            <div className="py-8 text-center text-zinc-500">
              {projectId ? "暂无扫描执行记录" : "请先选择一个项目"}
            </div>
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

      {/* Create Run Modal */}
      {showCreate && (
        <CreateRunModal
          templates={templates}
          onCreate={handleCreate}
          onClose={() => setShowCreate(false)}
          creating={creating}
        />
      )}
    </div>
  );
}

function CreateRunModal({
  templates,
  onCreate,
  onClose,
  creating,
}: {
  templates: ToolTemplate[];
  onCreate: (templateId: string, name: string) => void;
  onClose: () => void;
  creating: boolean;
}) {
  const [selectedTemplate, setSelectedTemplate] = useState("");
  const [name, setName] = useState("");

  return (
    <div className="fixed inset-0 bg-black/50 backdrop-blur-sm flex items-center justify-center z-50" onClick={onClose}>
      <div className="bg-zinc-900 border border-zinc-800 rounded-xl p-6 w-full max-w-md space-y-4" onClick={(e) => e.stopPropagation()}>
        <h2 className="text-lg font-semibold text-zinc-100">新建扫描</h2>
        <div className="space-y-3">
          <div>
            <label className="text-sm text-zinc-400 block mb-1">扫描名称</label>
            <input
              type="text"
              value={name}
              onChange={(e) => setName(e.target.value)}
              placeholder="例如：外网初筛"
              className="w-full bg-zinc-800 border border-zinc-700 rounded-lg px-3 py-2 text-sm text-zinc-100 focus:outline-none focus:border-brand-primary/50"
            />
          </div>
          <div>
            <label className="text-sm text-zinc-400 block mb-1">工具模板</label>
            <div className="space-y-2">
              {templates.map((t) => (
                <div
                  key={t.id}
                  onClick={() => setSelectedTemplate(t.id)}
                  className={`p-3 rounded-lg border cursor-pointer transition-all ${
                    selectedTemplate === t.id
                      ? "border-brand-primary bg-brand-primary/10"
                      : "border-zinc-700 hover:border-zinc-600"
                  }`}
                >
                  <div className="text-sm font-medium text-zinc-200">{t.name}</div>
                  <div className="text-xs text-zinc-500 mt-0.5">{t.description}</div>
                </div>
              ))}
            </div>
          </div>
        </div>
        <div className="flex gap-3 pt-2">
          <button
            onClick={onClose}
            className="flex-1 bg-zinc-800 text-zinc-300 text-sm py-2 rounded-lg hover:bg-zinc-700 transition-colors"
          >
            取消
          </button>
          <button
            onClick={() => onCreate(selectedTemplate, name || "未命名扫描")}
            disabled={!selectedTemplate || creating}
            className="flex-1 bg-brand-primary text-white text-sm py-2 rounded-lg hover:opacity-90 transition-opacity disabled:opacity-50"
          >
            {creating ? "创建中..." : "开始扫描"}
          </button>
        </div>
      </div>
    </div>
  );
}
