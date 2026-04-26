import { useEffect, useState } from "react";
import { useParams } from "react-router-dom";
import { api } from "../lib/api";
import { useStore } from "../lib/store";

export default function TargetPage() {
  const { id } = useParams<{ id: string }>();
  const { targets, setTargets, currentProject, setCurrentProject } = useStore();
  const [targetValue, setTargetValue] = useState("");
  const [targetType, setTargetType] = useState("domain");
  const [scopeAction, setScopeAction] = useState<"include" | "exclude">("include");
  const [scopeValue, setScopeValue] = useState("");
  const [dryRunResult, setDryRunResult] = useState<any>(null);

  useEffect(() => {
    if (!id) return;
    api.getProject(id).then(setCurrentProject).catch(console.error);
    api.listTargets(id).then(setTargets).catch(console.error);
  }, [id, setCurrentProject, setTargets]);

  const addTarget = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!id || !targetValue.trim()) return;
    const t = await api.createTarget(id, { type: targetType, value: targetValue });
    setTargets([...targets, t]);
    setTargetValue("");
  };

  const addScopeRule = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!id || !scopeValue.trim()) return;
    await api.createScopeRule({
      project_id: id,
      action: scopeAction,
      type: "domain",
      value: scopeValue,
    });
    setScopeValue("");
    alert("规则已添加");
  };

  const runDryRun = async () => {
    if (!id) return;
    const res = await api.dryRun(id);
    setDryRunResult(res);
  };

  const runSubfinder = async () => {
    if (!id) return;
    const domain = targets.find((t) => t.type === "domain")?.value;
    if (!domain) {
      alert("请先添加一个域名目标");
      return;
    }
    const task = await api.runTask({
      project_id: id,
      tool: "subfinder",
      command: `subfinder -d ${domain} -oJ -o subfinder_output.jsonl`,
    });
    alert(`任务已启动: ${task.id}`);
  };

  if (!currentProject) return <div>加载中...</div>;

  return (
    <div className="max-w-4xl space-y-6">
      <h1 className="text-2xl font-bold">{currentProject.name}</h1>

      <section className="bg-white p-4 rounded shadow">
        <h2 className="font-semibold mb-3">目标</h2>
        <form onSubmit={addTarget} className="flex gap-2 mb-3">
          <select
            className="border rounded px-2"
            value={targetType}
            onChange={(e) => setTargetType(e.target.value)}
          >
            <option value="domain">域名</option>
            <option value="url">URL</option>
            <option value="ip">IP</option>
            <option value="cidr">CIDR</option>
          </select>
          <input
            className="flex-1 border rounded px-3 py-2"
            placeholder="目标值"
            value={targetValue}
            onChange={(e) => setTargetValue(e.target.value)}
          />
          <button type="submit" className="bg-slate-700 text-white px-4 py-2 rounded">
            添加
          </button>
        </form>
        <ul className="space-y-1">
          {targets.map((t) => (
            <li key={t.id} className="text-sm bg-gray-50 px-3 py-2 rounded">
              [{t.type}] {t.value}
            </li>
          ))}
        </ul>
      </section>

      <section className="bg-white p-4 rounded shadow">
        <h2 className="font-semibold mb-3">Scope 规则</h2>
        <form onSubmit={addScopeRule} className="flex gap-2 mb-3">
          <select
            className="border rounded px-2"
            value={scopeAction}
            onChange={(e) => setScopeAction(e.target.value as any)}
          >
            <option value="include">包含</option>
            <option value="exclude">排除</option>
          </select>
          <input
            className="flex-1 border rounded px-3 py-2"
            placeholder="域名规则，如 example.com"
            value={scopeValue}
            onChange={(e) => setScopeValue(e.target.value)}
          />
          <button type="submit" className="bg-slate-700 text-white px-4 py-2 rounded">
            添加
          </button>
        </form>
      </section>

      <section className="bg-white p-4 rounded shadow">
        <h2 className="font-semibold mb-3">操作</h2>
        <div className="flex gap-3">
          <button onClick={runDryRun} className="bg-blue-600 text-white px-4 py-2 rounded hover:bg-blue-500">
            干运行 (Scope Check)
          </button>
          <button onClick={runSubfinder} className="bg-green-600 text-white px-4 py-2 rounded hover:bg-green-500">
            运行 Subfinder
          </button>
        </div>

        {dryRunResult && (
          <div className="mt-4 bg-gray-50 p-3 rounded text-sm">
            <div className="font-semibold mb-2">干运行结果 ({dryRunResult.mode})</div>
            <ul className="space-y-1">
              {dryRunResult.results?.map((r: any, i: number) => (
                <li key={i} className={r.decision === "allow" ? "text-green-700" : "text-red-700"}>
                  [{r.decision}] {r.target} — {r.reason}
                </li>
              ))}
            </ul>
          </div>
        )}
      </section>
    </div>
  );
}
