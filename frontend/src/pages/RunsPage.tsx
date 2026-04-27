import { useEffect, useState } from "react";
import { api } from "../lib/api";
import { Button } from "../components/Button";
import { Badge } from "../components/Badge";

export default function RunsPage() {
  const [health, setHealth] = useState<any[]>([]);

  useEffect(() => {
    api.listToolHealth().then((h) => setHealth(h ?? [])).catch(console.error);
  }, []);

  const refreshHealth = async () => {
    const h = await api.runHealthCheck();
    setHealth(h ?? []);
  };

  return (
    <div className="max-w-4xl space-y-6">
      <h1 className="text-2xl font-bold text-zinc-100 tracking-tight">运行状态</h1>

      <section className="bg-zinc-900/50 backdrop-blur-md border border-zinc-800/80 rounded-xl p-6">
        <div className="flex justify-between items-center mb-4">
          <h2 className="text-base font-medium text-zinc-200">工具健康检查</h2>
          <Button variant="secondary" size="sm" onClick={refreshHealth}>
            刷新
          </Button>
        </div>
        <div className="divide-y divide-zinc-800/60">
          {health.map((h) => (
            <div key={h.tool} className="py-3 px-2 flex items-center justify-between text-sm hover:bg-zinc-800/40 transition-all rounded-lg">
              <div className="flex items-center gap-4">
                <span className="font-medium text-zinc-200 w-20">{h.tool}</span>
                <span className="font-mono text-zinc-400 text-xs">{h.version || "—"}</span>
              </div>
              <div className="flex items-center gap-3">
                <Badge variant={h.dns_available ? "success" : "danger"}>
                  DNS {h.dns_available ? "可用" : "不可用"}
                </Badge>
              </div>
            </div>
          ))}
          {health.length === 0 && (
            <div className="py-8 text-center text-zinc-500">未检测到工具，请安装后点击刷新</div>
          )}
        </div>
      </section>

      <section className="bg-zinc-900/50 backdrop-blur-md border border-zinc-800/80 rounded-xl p-6">
        <h2 className="text-base font-medium text-zinc-200 mb-3">任务列表</h2>
        <p className="text-zinc-500 text-sm">任务历史将在实现持久化后显示</p>
      </section>
    </div>
  );
}
