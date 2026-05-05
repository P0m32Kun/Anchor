import { useEffect, useState, useRef, useCallback } from "react";
import { Link } from "react-router-dom";
import { api, type ImportResult, type DryRunResult, type Project, type Target, type ScopeConfirmationResponse, PAGE_ALL } from "../lib/api";
import { useStore } from "../lib/store";
import { useProjectId, ConfirmDialog, useToast, EmptyState, Table, Badge, SkeletonList } from "../components";

function ProjectInfo({ project }: { project: Project }) {
  const now = new Date();
  const start = project.start_time ? new Date(project.start_time) : null;
  const end = project.end_time ? new Date(project.end_time) : null;
  const isExpired = end && end < now;
  const isPending = start && start > now;
  const isActive = (!start || start <= now) && (!end || end >= now);

  return (
    <section className="panel p-4">
      <h2 className="font-semibold mb-2">项目信息</h2>
      <div className="text-sm space-y-1">
        <div className="text-zinc-400">
          <span className="font-medium">组织:</span> {project.organization || "—"}
        </div>
        {project.purpose && (
          <div className="text-zinc-400">
            <span className="font-medium">目的:</span> {project.purpose}
          </div>
        )}
        {start && end ? (
          <div className="flex items-center gap-2">
            <span className="text-zinc-400">
              时间窗口: {start.toLocaleDateString()} ~ {end.toLocaleDateString()}
            </span>
            {isExpired && (
              <span className="text-xs bg-brand-danger/15 text-brand-danger px-2 py-0.5 rounded font-medium">
                已过期
              </span>
            )}
            {isPending && (
              <span className="text-xs bg-accent-yellow/15 text-accent-yellow px-2 py-0.5 rounded font-medium">
                未开始
              </span>
            )}
            {isActive && (
              <span className="text-xs bg-brand-success/15 text-brand-success px-2 py-0.5 rounded font-medium">
                进行中
              </span>
            )}
          </div>
        ) : (
          <div className="text-zinc-500 text-xs">未配置时间窗口 (始终可用)</div>
        )}
        <div className="text-zinc-400">
          <span className="font-medium">速率限制:</span>{" "}
          {project.rate_limit !== undefined && project.rate_limit > 0
            ? `${project.rate_limit} 包/秒`
            : "无限制"}
        </div>
        {project.default_profile && (
          <div className="text-zinc-400">
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
  const fileRef = useRef<HTMLInputElement>(null);
  const toast = useToast();

  const handleFile = useCallback(
    async (file: File) => {
      if (!file.name.endsWith(".txt") && !file.name.endsWith(".csv")) {
        toast("仅支持 .txt 或 .csv 文件", "warning");
        return;
      }
      setImporting(true);
      setResult(null);
      try {
        const res = await api.importTargets(projectId, file);
        setResult(res);
        onImported();
        toast(`导入完成: 成功 ${res.imported} 个`, "success");
      } catch (err) {
        toast("导入失败: " + (err instanceof Error ? err.message : String(err)), "error");
      } finally {
        setImporting(false);
        setDragOver(false);
      }
    },
    [projectId, onImported, toast],
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
        className={`border border-dashed rounded-lg p-8 text-center cursor-pointer transition-colors ${
          importing
            ? "border-white/[0.08] bg-white/[0.03] pointer-events-none"
            : dragOver
            ? "border-sky-400 bg-sky-500/10"
            : "border-white/[0.12] bg-white/[0.025] hover:border-white/[0.24]"
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
          <div className="text-zinc-400">
            <div className="animate-pulse mb-2">⏳</div>
            正在导入...
          </div>
        ) : (
          <div className="text-zinc-400">
            <div className="text-2xl mb-2">📂</div>
            <div className="font-medium">点击上传 或将文件拖拽到此处</div>
            <div className="text-xs text-zinc-500 mt-1">支持 .txt / .csv 格式，每行一个目标</div>
          </div>
        )}
      </div>

      {result && (
        <div className="panel p-3 text-sm space-y-2">
          <div className="font-semibold">导入结果</div>
          <div className="flex gap-4 flex-wrap">
            <StatBadge label="成功导入" value={result.imported} color="green" />
            {result.expanded !== undefined && result.expanded > 0 && (
              <StatBadge label="展开后目标" value={result.expanded} color="blue" />
            )}
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
                  <li key={i} className="text-zinc-400">
                    <code className="bg-yellow-500/10 px-1 rounded text-accent-yellow">{d.value}</code>
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
    green: "bg-brand-success/15 text-brand-success",
    gray: "bg-white/[0.04] text-text-tertiary",
    yellow: "bg-accent-yellow/15 text-accent-yellow",
    red: "bg-brand-danger/15 text-brand-danger",
    blue: "bg-brand-primary/15 text-brand-primary",
  };
  return (
    <div className="flex items-center gap-1">
      <span className={`px-2 py-0.5 rounded text-xs font-medium ${colors[color] || colors.gray}`}>
        {value}
      </span>
      <span className="text-xs text-zinc-400">{label}</span>
    </div>
  );
}

export default function TargetPage() {
  const projectId = useProjectId();
  const targets = useStore((state) => state.targets) ?? [];
  const setTargets = useStore((state) => state.setTargets);
  const currentProject = useStore((state) => state.currentProject);
  const loading = useStore((state) => state.targetsLoading);
  const error = useStore((state) => state.targetsError);
  const setTargetsLoading = useStore((state) => state.setTargetsLoading);
  const setTargetsError = useStore((state) => state.setTargetsError);
  const [targetValue, setTargetValue] = useState("");
  const [targetType, setTargetType] = useState("auto");
  const [scopeAction, setScopeAction] = useState<"include" | "exclude">("include");
  const [scopeValue, setScopeValue] = useState("");
  const [dryRunResult, setDryRunResult] = useState<DryRunResult | null>(null);
  const [scopeConfirmOpen, setScopeConfirmOpen] = useState(false);
  const [pendingScopeConfirm, setPendingScopeConfirm] = useState<{
    message: string;
    suggested: ScopeConfirmationResponse["suggested_rule"];
    pendingType: string;
    pendingValue: string;
  } | null>(null);
  const [scopeConfirmLoading, setScopeConfirmLoading] = useState(false);

  const [addingTarget, setAddingTarget] = useState(false);
  const [addingScope, setAddingScope] = useState(false);
  const [dryRunLoading, setDryRunLoading] = useState(false);
  const [dryRunConfirmOpen, setDryRunConfirmOpen] = useState(false);


  const toast = useToast();

  const loadTargets = useCallback(async (signal?: AbortSignal) => {
    if (!projectId) return;
    setTargetsLoading(true);
    setTargetsError(null);
    try {
      const data = await api.listTargets(projectId, PAGE_ALL, signal);
      setTargets(data.data ?? []);
    } catch (err) {
      if (err instanceof DOMException && err.name === "AbortError") return;
      const msg = err instanceof Error ? err.message : String(err);
      setTargetsError(msg);
      toast("加载目标失败: " + msg, "error");
      console.error(err);
    } finally {
      setTargetsLoading(false);
    }
  }, [projectId, setTargets, setTargetsLoading, setTargetsError]);

  useEffect(() => {
    if (!projectId) return;
    const ctrl = new AbortController();
    loadTargets(ctrl.signal);
    return () => ctrl.abort();
  }, [projectId, loadTargets]);

  const addTarget = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!projectId || !targetValue.trim()) {
      toast("请输入目标值", "warning");
      return;
    }
    if (addingTarget) return;
    setAddingTarget(true);
    try {
      const res = await api.createTarget(projectId, { type: targetType, value: targetValue });
      if ("needs_scope_confirmation" in res && res.needs_scope_confirmation) {
        setPendingScopeConfirm({
          message: res.message,
          suggested: res.suggested_rule,
          pendingType: targetType,
          pendingValue: targetValue,
        });
        setScopeConfirmOpen(true);
        return;
      }
      setTargets([...targets, res as Target]);
      setTargetValue("");
      toast("目标已添加", "success");
    } catch (err) {
      toast("添加目标失败: " + (err instanceof Error ? err.message : String(err)), "error");
    } finally {
      setAddingTarget(false);
    }
  };

  const handleConfirmScope = async () => {
    if (!pendingScopeConfirm || !projectId) return;
    setScopeConfirmLoading(true);
    try {
      const { suggested, pendingType, pendingValue } = pendingScopeConfirm;
      await api.createScopeRule({
        project_id: projectId,
        action: suggested.action,
        type: suggested.type,
        value: suggested.value,
      });
      const t = await api.createTarget(projectId, { type: pendingType, value: pendingValue });
      if ("needs_scope_confirmation" in t && t.needs_scope_confirmation) {
        toast("Scope 规则已添加，但目标仍需要确认，请手动检查。", "warning");
      } else {
        setTargets([...targets, t as Target]);
        setTargetValue("");
        toast("目标已添加", "success");
      }
    } catch (err) {
      toast("添加 Scope 规则失败: " + (err instanceof Error ? err.message : String(err)), "error");
    } finally {
      setScopeConfirmLoading(false);
      setScopeConfirmOpen(false);
      setPendingScopeConfirm(null);
    }
  };

  const addScopeRule = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!projectId || !scopeValue.trim()) {
      toast("请输入 Scope 规则值", "warning");
      return;
    }
    if (addingScope) return;
    setAddingScope(true);
    try {
      await api.createScopeRule({
        project_id: projectId,
        action: scopeAction,
        type: "domain",
        value: scopeValue,
      });
      setScopeValue("");
      toast("规则已添加", "success");
    } catch (err) {
      toast("添加规则失败: " + (err instanceof Error ? err.message : String(err)), "error");
    } finally {
      setAddingScope(false);
    }
  };

  const runDryRun = async () => {
    if (!projectId || dryRunLoading) return;
    setDryRunLoading(true);
    try {
      const res = await api.dryRun(projectId);
      setDryRunResult(res);
      toast("授权检测完成", "success");
    } catch (err) {
      toast("授权检测失败: " + (err instanceof Error ? err.message : String(err)), "error");
    } finally {
      setDryRunLoading(false);
      setDryRunConfirmOpen(false);
    }
  };



  const targetColumns: { key: string; header: string; width?: string; render?: (row: Record<string, unknown>) => React.ReactNode }[] = [
    {
      key: "type",
      header: "类型",
      width: "120px",
      render: (row) => (
        <Badge variant="primary" size="sm">
          {String(row.type)}
        </Badge>
      ),
    },
    { key: "value", header: "目标值" },
    {
      key: "created_at",
      header: "创建时间",
      width: "200px",
      render: (row) => new Date(String(row.created_at)).toLocaleString(),
    },
  ];

  if (!currentProject) {
    return (
      <div className="page-shell space-y-6">
        <div>
          <h1 className="text-2xl font-bold">目标管理</h1>
          <p className="text-zinc-400 text-sm mt-1">管理项目目标、Scope 规则和批量导入</p>
        </div>
        <div className="panel p-8 text-center">
          <p className="text-zinc-400 mb-4">请先从 Dashboard 选择一个项目</p>
          <Link to="/" className="text-blue-600 hover:underline">前往 Dashboard</Link>
        </div>
      </div>
    );
  }

  return (
    <div className="page-shell space-y-6">
      <div className="page-header">
        <div>
          <div className="page-eyebrow">Step 1</div>
          <h1 className="page-title">目标与 Scope</h1>
          <p className="page-subtitle">先确认授权边界，再导入域名、URL、IP 或 CIDR；所有后续扫描都从这里开始。</p>
        </div>
      </div>

      <ProjectInfo project={currentProject} />

      {/* 操作区 */}
      <section className="panel p-4 space-y-4">
        <h2 className="font-semibold">添加目标</h2>
        <form onSubmit={addTarget} className="flex gap-2">
          <select
            className="rounded-lg border border-white/[0.10] bg-slate-950/40 px-2 text-zinc-200"
            value={targetType}
            onChange={(e) => setTargetType(e.target.value)}
          >
            <option value="auto">自动检测</option>
            <option value="domain">域名</option>
            <option value="url">URL</option>
            <option value="ip">IP</option>
            <option value="cidr">CIDR</option>
          </select>
          <input
            className="flex-1 rounded-lg border border-white/[0.10] bg-slate-950/40 px-3 py-2 text-zinc-200 placeholder-zinc-500"
            placeholder="example.com 或 192.168.1.1 或 10.0.0.0/24 或 192.168.0.1-10"
            value={targetValue}
            onChange={(e) => setTargetValue(e.target.value)}
          />
          <button
            type="submit"
            disabled={addingTarget}
            className="btn-cyber-primary disabled:opacity-50 disabled:cursor-not-allowed"
          >
            {addingTarget ? "添加中..." : "添加"}
          </button>
        </form>

        <h2 className="font-semibold">批量导入</h2>
        <FileImport projectId={currentProject.id} onImported={loadTargets} />
      </section>

      {/* 内容区 */}
      <section className="panel p-4">
        <h2 className="font-semibold mb-3">目标列表</h2>
        {loading ? (
          <SkeletonList count={3} />
        ) : error ? (
          <div className="py-12 text-center">
            <p className="text-brand-danger mb-2">加载失败: {error}</p>
            <button
              onClick={() => loadTargets()}
              className="text-sm text-blue-600 hover:underline"
            >
              重试
            </button>
          </div>
        ) : targets.length === 0 ? (
          <EmptyState
            title="暂无目标"
            description="当前项目还没有添加任何目标，请在上方添加或导入。"
          />
        ) : (
          <Table
            columns={targetColumns}
            data={targets as unknown as Record<string, unknown>[]}
            emptyText="暂无目标"
            maxHeight={480}
          />
        )}
      </section>

      {/* Scope 规则 */}
      <section className="panel p-4">
        <h2 className="font-semibold mb-3">Scope 规则</h2>
        <form onSubmit={addScopeRule} className="flex gap-2 mb-3">
          <select
            className="rounded-lg border border-white/[0.10] bg-slate-950/40 px-2 text-zinc-200"
            value={scopeAction}
            onChange={(e) => setScopeAction(e.target.value as any)}
          >
            <option value="include">包含</option>
            <option value="exclude">排除</option>
          </select>
          <input
            className="flex-1 rounded-lg border border-white/[0.10] bg-slate-950/40 px-3 py-2 text-zinc-200 placeholder-zinc-500"
            placeholder="域名规则，如 example.com"
            value={scopeValue}
            onChange={(e) => setScopeValue(e.target.value)}
          />
          <button type="submit" disabled={addingScope} className="btn-cyber-secondary disabled:opacity-50 disabled:cursor-not-allowed">
            {addingScope ? "添加中..." : "添加"}
          </button>
        </form>
      </section>

      {/* 操作 */}
      <section className="panel p-4">
        <h2 className="font-semibold mb-3">操作</h2>
        <div className="flex gap-3 flex-wrap">
          <button
            onClick={() => setDryRunConfirmOpen(true)}
            disabled={dryRunLoading}
            className="btn-cyber-primary disabled:opacity-50 disabled:cursor-not-allowed"
          >
            {dryRunLoading ? "检测中..." : "授权检测 (Scope Check)"}
          </button>
          <Link to={`/projects/${currentProject.id}/runs`} className="btn-cyber-secondary">
            前往扫描
          </Link>
          <Link to={`/projects/${currentProject.id}/assets`} className="btn-cyber-secondary">
            查看资产
          </Link>
          <Link to={`/projects/${currentProject.id}/findings`} className="btn-cyber-secondary">
            漏洞发现
          </Link>
          <Link to={`/projects/${currentProject.id}/reports`} className="btn-cyber-secondary">
            报告
          </Link>
        </div>

        {dryRunResult && (
          <div className="mt-4 panel p-3 text-sm space-y-2">
            <div className="font-semibold">授权检测结果 ({dryRunResult.mode})</div>
            <div className="flex gap-4 text-xs">
              <div>
                <span className="text-zinc-400">时间窗口:</span>{" "}
                {dryRunResult.time_window_valid === undefined
                  ? "—"
                  : dryRunResult.time_window_valid
                  ? <span className="text-green-600 font-medium">有效</span>
                  : <span className="text-red-600 font-medium">无效</span>}
              </div>
              <div>
                <span className="text-zinc-400">速率限制:</span>{" "}
                {dryRunResult.rate_limit !== undefined && dryRunResult.rate_limit > 0
                  ? `${dryRunResult.rate_limit} 包/秒`
                  : "无限制"}
              </div>
              {dryRunResult.estimated_duration_seconds !== undefined && (
                <div>
                  <span className="text-zinc-400">预计耗时:</span>{" "}
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
                <div className="text-xs font-medium text-zinc-400 mt-2 mb-1">
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

      <ConfirmDialog
        open={scopeConfirmOpen}
        onClose={() => {
          setScopeConfirmOpen(false);
          setPendingScopeConfirm(null);
        }}
        onConfirm={handleConfirmScope}
        title="Scope 确认"
        description={
          pendingScopeConfirm
            ? `${pendingScopeConfirm.message}\n\n建议添加 Scope 规则: [${pendingScopeConfirm.suggested.action}] ${pendingScopeConfirm.suggested.type} = ${pendingScopeConfirm.suggested.value}\n\n是否自动添加该规则并重新添加目标？`
            : ""
        }
        confirmText="添加规则并继续"
        cancelText="取消"
        loading={scopeConfirmLoading}
      />

      <ConfirmDialog
        open={dryRunConfirmOpen}
        onClose={() => setDryRunConfirmOpen(false)}
        onConfirm={runDryRun}
        title="授权检测"
        description="即将对当前项目的所有目标执行授权检测（Dry Run），确认时间窗口和 Scope 规则是否有效。是否继续？"
        confirmText="开始检测"
        cancelText="取消"
        loading={dryRunLoading}
      />


    </div>
  );
}
