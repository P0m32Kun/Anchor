import { useEffect, useState } from "react";
import { useNavigate } from "react-router-dom";
import { api } from "../lib/api";
import { useStore } from "../lib/store";

export default function ProjectPage() {
  const navigate = useNavigate();
  const { projects, setProjects } = useStore();
  const [name, setName] = useState("");
  const [org, setOrg] = useState("");
  const [loading, setLoading] = useState(false);

  useEffect(() => {
    api.listProjects().then(setProjects).catch(console.error);
  }, [setProjects]);

  const handleCreate = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!name.trim()) return;
    setLoading(true);
    try {
      const p = await api.createProject({ name, organization: org || undefined });
      setProjects([p, ...projects]);
      setName("");
      setOrg("");
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
        <input
          className="w-full border rounded px-3 py-2"
          placeholder="组织/客户"
          value={org}
          onChange={(e) => setOrg(e.target.value)}
        />
        <button
          type="submit"
          disabled={loading}
          className="bg-slate-700 text-white px-4 py-2 rounded hover:bg-slate-600 disabled:opacity-50"
        >
          {loading ? "创建中..." : "创建项目"}
        </button>
      </form>

      <div className="space-y-3">
        {projects.map((p) => (
          <div
            key={p.id}
            onClick={() => navigate(`/projects/${p.id}`)}
            className="bg-white p-4 rounded shadow cursor-pointer hover:shadow-md transition"
          >
            <div className="font-semibold">{p.name}</div>
            <div className="text-sm text-gray-500">
              {p.organization || "—"} · {new Date(p.created_at).toLocaleString()}
            </div>
          </div>
        ))}
        {projects.length === 0 && (
          <div className="text-gray-400">暂无项目，请创建第一个项目</div>
        )}
      </div>
    </div>
  );
}
