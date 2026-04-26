import { useEffect, useState } from "react";
import { api } from "../lib/api";

interface TaskWithArtifacts {
  task: any;
  artifacts: any[];
}

export default function RunsPage() {
  const [tasks, setTasks] = useState<TaskWithArtifacts[]>([]);
  const [health, setHealth] = useState<any[]>([]);

  useEffect(() => {
    api.listToolHealth().then(setHealth).catch(console.error);
  }, []);

  const refreshHealth = async () => {
    const h = await api.runHealthCheck();
    setHealth(h);
  };

  const statusColor = (s: string) => {
    switch (s) {
      case "completed": return "text-green-600";
      case "running": return "text-blue-600";
      case "failed": return "text-red-600";
      case "cancelled": return "text-gray-500";
      default: return "text-gray-600";
    }
  };

  return (
    <div className="max-w-4xl space-y-6">
      <h1 className="text-2xl font-bold">运行状态</h1>

      <section className="bg-white p-4 rounded shadow">
        <div className="flex justify-between items-center mb-3">
          <h2 className="font-semibold">工具健康检查</h2>
          <button onClick={refreshHealth} className="text-sm bg-slate-700 text-white px-3 py-1 rounded">
            刷新
          </button>
        </div>
        <div className="grid grid-cols-2 gap-3">
          {health.map((h) => (
            <div key={h.tool} className="border rounded p-3 text-sm">
              <div className="font-semibold">{h.tool}</div>
              <div className="text-gray-500">{h.version || "—"}</div>
              <div className={h.dns_available ? "text-green-600" : "text-red-600"}>
                DNS: {h.dns_available ? "可用" : "不可用"}
              </div>
            </div>
          ))}
          {health.length === 0 && (
            <div className="text-gray-400 col-span-2">未检测到工具，请安装后点击刷新</div>
          )}
        </div>
      </section>

      <section className="bg-white p-4 rounded shadow">
        <h2 className="font-semibold mb-3">任务列表</h2>
        <p className="text-gray-400 text-sm">任务历史将在实现持久化后显示</p>
      </section>
    </div>
  );
}
