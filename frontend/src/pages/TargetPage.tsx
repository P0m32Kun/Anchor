import { useEffect, useState, useRef, useCallback } from "react";
import { useParams, Link } from "react-router-dom";
import { api, type ImportResult, type DryRunResult, type Project } from "../lib/api";
import { useStore } from "../lib/store";

function ProjectInfo({ project }: { project: Project }) {
  const now = new Date();
  const start = project.start_time ? new Date(project.start_time) : null;
  const end = project.end_time ? new Date(project.end_time) : null;
  const isExpired = end && end < now;
  const isPending = start && start > now;
  const isActive = (!start || start <= now) && (!end || end >= now);

  return (
    <section className="bg-white p-4 rounded shadow">
      <h2 className="font-semibold mb-2">项目信息</h2>
      <div className="text-sm space-y-1">
        <div className="text-gray-500">
          <span className="font-medium">组织:</span> {project.organization || "—"}
        </div>
        {project.purpose && (
          <div className="text-gray-500">
            <span className="font-medium">目的:</span> {project.purpose}
          </div>
        )}
        {start && end ? (
          <div className="flex items-center gap-2">
            <span className="text-gray-500">
              时间窗口: {start.toLocaleDateString()} ~ {end.toLocaleDateString()}
            </span>
            {isExpired && (
              <span className="text-xs bg-red-100 text-red-700 px-2 py-0.5 rounded font-medium">
                已过期
              </span>
            )}
            {isPending && (
              <span className="text-xs bg-yellow-100 text-yellow-700 px-2 py-0.5 rounded font-medium">
                未开始
              </span>
            )}
            {isActive && (
              <span className="text-xs bg-green-100 text-green-700 px-2 py-0.5 rounded font-medium">
                进行中
              </span>
            )}
          </div>
        ) : (
          <div className="text-gray-400 text-xs">未配置时间窗口 (始终可用)</div>
        )}
        <div className="text-gray-500">
          <span className="font-medium">速率限制:</span>{" "}
          {project.rate_limit !== undefined && project.rate_limit > 0
            ? `${project.rate_limit} 包/秒`
            : "无限制"}
        </div>
        {project.default_profile && (
          <div className="text-gray-500">
            <span className="font-medium">默认 Profile:</span> {project.default_profile}
          </div>
        )}
      </div>
    </section>
  );
}

function FileImport({ projectId, onImported }: { projectId: string; onImported: () => void }) {
  const [dragOver, setDragOver] = useState(false);
  const [importing, setImporting] = useState(false);
  const [result, setResult] = useState<ImportResult | null>(null);
  const [error, setError] = useState<string | null>(null);
  const fileRef = useRef<HTMLInputElement>(null);

  const handleFile = useCallback(
    async (file: File) => {
      if (!file.name.endsWith(".txt") && !file.name.endsWith(".csv")) {
        setError("仅支持 .txt 或 .csv 文件");
        return;
      }
      setImporting(true);
      setError(null);
      setResult(null);
      try {
        const res = await api.importTargets(projectId, file);
        setResult(res);
        onImported();
      } catch (err) {
        setError(String(err));
      } finally {
        setImporting(false);
        setDragOver(false);
      }
    },
    [projectId, onImported],
  );

  const handleDrop = useCallback(
    (e: React.DragEvent) => {
      e.preventDefault();
      const file = e.dataTransfer.files[0];
      if (file) handleFile(file);
    },
    [handleFile],
  );

  return (
    <div className="space-y-3">
      <div
        onDragOver={(e) => {
          e.preventDefault();
          setDragOver(true);
        }}
        onDragLeave={() => setDragOver(false)}
        onDrop={handleDrop}
        onClick={() => fileRef.current?.click()}
        className={`border-2 border-dashed rounded-lg p-8 text-center cursor-pointer transition ${
          importing
            ? "border-gray-300 bg-gray-50 pointer-events-none"
            : dragOver
            ? "border-blue-400 bg-blue-50"
            : "border-gray-300 hover:border-gray-400"
        }`}
      >
        <input
          ref={fileRef}
          type="file"
          accept=".txt,.csv"
          className="hidden"
          onChange={(e) => {
            const file = e.target.files?.[0];
            if (file) handleFile(file);
            e.target.value = "";
          }}
        />
        {importing ? (
          <div className="text-gray-500">
            <div className="animate-pulse mb-2">⏳</div>
            正在导入...
          </div>
        ) : (
          <div className="text-gray-500">
            <div className="text-2xl mb-2">📂</div>
            <div className="font-medium">点击上传 或将文件拖拽到此处</div>
            <div className="text-xs text-gray-400 mt-1">支持 .txt / .csv 格式，每行一个目标</div>
          </div>
        )}
      </div>

      {error && (
        <div className="bg-red-50 text-red-700 px-3 py-2 rounded text-sm">{error}</div>
      )}

      {result && (
        <div className="bg-gray-50 p-3 rounded text-sm space-y-2">
          <div className="font-semibold">导入结果</div>
          <div className="flex gap-4">
            <StatBadge label="成功导入" value={result.imported} color="green" />
            <StatBadge label="重复跳过" value={result.duplicates} color="gray" />
            <StatBadge label="Scope 拒绝" value={result.denied} color="yellow" />
            <StatBadge label="错误" value={result.errors} color="red" />
          </div>
          {result.denied_targets.length > 0 && (
            <div>
              <div className="text-xs font-medium text-yellow-700 mb-1">
                被 Scope 拒绝的目标:
              </div>
              <ul className="text-xs space-y-0.5 max-h-32 overflow-auto">
                {result.denied_targets.map((d, i) => (
                  <li key={i} className="text-gray-600">
                    <code className="bg-yellow-50 px-1 rounded">{d.value}</code>
                    <span className="text-yellow-700 ml-2">— {d.reason}</span>
                  </li>
                ))}
              </ul>
            </div>
          )}
        </div>
      )}
    </div>
  );
}

function StatBadge({ label, value, color }: { label: string; value: number; color: string }) {
  const colors: Record<string, string> = {
    green: "bg-green-100 text-green-700",
    gray: "bg-gray-200 text-gray-600",
    yellow: "bg-yellow-100 text-yellow-700",
    red: "bg-red-100 text-red-700",
  };
  return (
    <div className="flex items-center gap-1">
      <span className={`px-2 py-0.5 rounded text-xs font-medium ${colors[color] || colors.gray}`}>
        {value}
      </span>
      <span className="text-xs text-gray-500">{label}</span>
    </div>
  );
}

export default function TargetPage() {
  const { id } = useParams<{ id: string }>();
  const { targets, setTargets, currentProject, setCurrentProject } = useStore();
  const [targetValue, setTargetValue] = useState("");
  const [targetType, setTargetType] = useState("domain");
  const [scopeAction, setScopeAction] = useState<"include" | "exclude">("include");
  const [scopeValue, setScopeValue] = useState("");
  const [dryRunResult, setDryRunResult] = useState<DryRunResult | null>(null);

  const loadTargets = useCallback(() => {
    if (!id) return;
    api.listTargets(id).then(setTargets).catch(console.error);
  }, [id, setTargets]);

  useEffect(() => {
    if (!id) return;
    api.getProject(id).then(setCurrentProject).catch(console.error);
    loadTargets();
  }, [id, setCurrentProject, loadTargets]);

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
    const domain = targets.find((t) => t.type === "domain");
    if (!domain) {
      alert("请先添加一个域名目标");
      return;
    }
    const task = await api.runTask({
      project_id: id,
      tool: "subfinder",
      target_id: domain.id,
      command: `subfinder -d ${domain.value} -oJ -o subfinder_output.jsonl`,
    });
    alert(`任务已启动: ${task.id}`);
  };

  if (!currentProject) return <div>加载中...</div>;

  return (
    <div className="max-w-4xl space-y-6">
      <h1 className="text-2xl font-bold">{currentProject.name}</h1>

      <ProjectInfo project={currentProject} />

      <section className="bg-white p-4 rounded shadow">
        <h2 className="font-semibold mb-3">批量导入目标</h2>
        <FileImport projectId={currentProject.id} onImported={loadTargets} />
      </section>

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
        <div className="flex gap-3 flex-wrap">
          <button onClick={runDryRun} className="bg-blue-600 text-white px-4 py-2 rounded hover:bg-blue-500">
            干运行 (Scope Check)
          </button>
          <button onClick={runSubfinder} className="bg-green-600 text-white px-4 py-2 rounded hover:bg-green-500">
            运行 Subfinder
          </button>
          <Link to={`/projects/${currentProject.id}/assets`} className="bg-purple-600 text-white px-4 py-2 rounded hover:bg-purple-500">
            查看资产
          </Link>
        </div>

        {dryRunResult && (
          <div className="mt-4 bg-gray-50 p-3 rounded text-sm space-y-2">
            <div className="font-semibold">干运行结果 ({dryRunResult.mode})</div>
            <div className="flex gap-4 text-xs">
              <div>
                <span className="text-gray-500">时间窗口:</span>{" "}
                {dryRunResult.time_window_valid === undefined
                  ? "—"
                  : dryRunResult.time_window_valid
                  ? <span className="text-green-600 font-medium">有效</span>
                  : <span className="text-red-600 font-medium">无效</span>}
              </div>
              <div>
                <span className="text-gray-500">速率限制:</span>{" "}
                {dryRunResult.rate_limit !== undefined && dryRunResult.rate_limit > 0
                  ? `${dryRunResult.rate_limit} 包/秒`
                  : "无限制"}
              </div>
              {dryRunResult.estimated_duration_seconds !== undefined && (
                <div>
                  <span className="text-gray-500">预计耗时:</span>{" "}
                  <span className="font-medium">
                    {dryRunResult.estimated_duration_seconds < 60
                      ? `${dryRunResult.estimated_duration_seconds} 秒`
                      : `${Math.round(dryRunResult.estimated_duration_seconds / 60)} 分钟`}
                  </span>
                </div>
              )}
            </div>
            {dryRunResult.results && dryRunResult.results.length > 0 && (
              <div>
                <div className="text-xs font-medium text-gray-500 mt-2 mb-1">
                  目标决策 ({dryRunResult.results.length}):
                </div>
                <ul className="space-y-0.5 max-h-64 overflow-auto">
                  {dryRunResult.results.map((r, i) => (
                    <li
                      key={i}
                      className={
                        r.decision === "allow"
                          ? "text-green-700"
                          : r.decision === "deny"
                          ? "text-red-700"
                          : "text-yellow-700"
                      }
                    >
                      [{r.decision}] {r.target} — {r.reason}
                    </li>
                  ))}
                </ul>
              </div>
            )}
          </div>
        )}
      </section>
    </div>
  );
}
