import { useEffect, useState } from "react";
import { useNavigate } from "react-router-dom";
import { api } from "../lib/api";
import { useStore } from "../lib/store";

export default function ProjectPage() {
  const navigate = useNavigate();
  const { projects, setProjects } = useStore();
  const [name, setName] = useState("");
  const [org, setOrg] = useState("");
  const [purpose, setPurpose] = useState("");
  const [startTime, setStartTime] = useState("");
  const [endTime, setEndTime] = useState("");
  const [rateLimit, setRateLimit] = useState(0);
  const [loading, setLoading] = useState(false);

  useEffect(() => {
    api.listProjects().then(setProjects).catch(console.error);
  }, [setProjects]);

  const handleCreate = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!name.trim()) return;
    setLoading(true);
    try {
      const p = await api.createProject({
        name,
        organization: org || undefined,
        purpose: purpose || undefined,
        start_time: startTime ? new Date(startTime).toISOString() : undefined,
        end_time: endTime ? new Date(endTime).toISOString() : undefined,
        rate_limit: rateLimit,
      });
      setProjects([p, ...projects]);
      setName("");
      setOrg("");
      setPurpose("");
      setStartTime("");
      setEndTime("");
      setRateLimit(0);
    } catch (err) {
      alert(String(err));
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="max-w-3xl">
      <h1 className="text-2xl font-bold mb-6">项目管理</h1>

      <form onSubmit={handleCreate} className="bg-white p-4 rounded shadow mb-6 space-y-3">
        <h2 className="font-semibold">新建项目</h2>
        <input
          className="w-full border rounded px-3 py-2"
          placeholder="项目名称 *"
          value={name}
          onChange={(e) => setName(e.target.value)}
          required
        />
        <div className="flex gap-3">
          <input
            className="flex-1 border rounded px-3 py-2"
            placeholder="组织/客户"
            value={org}
            onChange={(e) => setOrg(e.target.value)}
          />
          <input
            className="flex-1 border rounded px-3 py-2"
            placeholder="目的/描述"
            value={purpose}
            onChange={(e) => setPurpose(e.target.value)}
          />
        </div>
        <div className="flex gap-3">
          <div className="flex-1">
            <label className="block text-xs text-gray-500 mb-1">开始时间</label>
            <input
              type="datetime-local"
              className="w-full border rounded px-3 py-2"
              value={startTime}
              onChange={(e) => setStartTime(e.target.value)}
            />
          </div>
          <div className="flex-1">
            <label className="block text-xs text-gray-500 mb-1">结束时间</label>
            <input
              type="datetime-local"
              className="w-full border rounded px-3 py-2"
              value={endTime}
              onChange={(e) => setEndTime(e.target.value)}
            />
          </div>
          <div>
            <label className="block text-xs text-gray-500 mb-1">速率限制 (包/秒)</label>
            <input
              type="number"
              min={0}
              className="w-28 border rounded px-3 py-2"
              value={rateLimit}
              onChange={(e) => setRateLimit(Math.max(0, Number(e.target.value)))}
            />
          </div>
        </div>
        <button
          type="submit"
          disabled={loading}
          className="bg-slate-700 text-white px-4 py-2 rounded hover:bg-slate-600 disabled:opacity-50"
        >
          {loading ? "创建中..." : "创建项目"}
        </button>
      </form>

      <div className="space-y-3">
        {projects.map((p) => {
          const now = new Date();
          const start = p.start_time ? new Date(p.start_time) : null;
          const end = p.end_time ? new Date(p.end_time) : null;
          const isActive = (!start || start <= now) && (!end || end >= now);
          const isExpired = end && end < now;
          const isPending = start && start > now;

          return (
            <div
              key={p.id}
              onClick={() => navigate(`/projects/${p.id}`)}
              className="bg-white p-4 rounded shadow cursor-pointer hover:shadow-md transition"
            >
              <div className="flex items-center gap-2">
                <div className="font-semibold">{p.name}</div>
                {isExpired && (
                  <span className="text-xs bg-red-100 text-red-700 px-2 py-0.5 rounded">已过期</span>
                )}
                {isPending && (
                  <span className="text-xs bg-yellow-100 text-yellow-700 px-2 py-0.5 rounded">未开始</span>
                )}
                {isActive && start && end && (
                  <span className="text-xs bg-green-100 text-green-700 px-2 py-0.5 rounded">进行中</span>
                )}
              </div>
              <div className="text-sm text-gray-500 mt-1 space-y-0.5">
                <div>{p.organization || "—"} · {p.purpose || ""}</div>
                <div>创建: {new Date(p.created_at).toLocaleString()}</div>
                {start && end && (
                  <div>
                    时间窗口: {start.toLocaleDateString()} ~ {end.toLocaleDateString()}
                  </div>
                )}
                {p.rate_limit !== undefined && p.rate_limit > 0 && (
                  <div>速率限制: {p.rate_limit} 包/秒</div>
                )}
              </div>
            </div>
          );
        })}
        {projects.length === 0 && (
          <div className="text-gray-400">暂无项目，请创建第一个项目</div>
        )}
      </div>
    </div>
  );
}
