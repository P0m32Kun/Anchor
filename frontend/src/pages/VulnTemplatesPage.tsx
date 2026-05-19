import { useState, useEffect, useCallback, useRef } from "react";
import { api, type FindingTemplate, API_BASE } from "../lib/api";
import { getApiToken } from "../lib/config";
import {
  useToast,
  Button,
  Badge,
  Card,
  Table,
  TableHeader,
  TableBody,
  TableRow,
  TableHead,
  TableCell,
  ConfirmDialog,
  EmptyState,
  TemplateEditor,
  type TemplateFormData,
} from "../components";
import { ShieldAlert, Plus, Trash2, Settings2, Download, RotateCcw, Lock } from "lucide-react";

function severityBadge(s: FindingTemplate["severity"]) {
  if (!s) return <Badge variant="outline">—</Badge>;
  const cls: Record<string, string> = {
    critical: "bg-rose-500/10 text-rose-400 border-rose-500/20",
    high: "bg-orange-500/10 text-orange-400 border-orange-500/20",
    medium: "bg-amber-500/10 text-amber-400 border-amber-500/20",
    low: "bg-cyan-500/10 text-cyan-400 border-cyan-500/20",
    info: "bg-muted/40 text-muted-foreground border-border/50",
  };
  const label: Record<string, string> = {
    critical: "严重",
    high: "高危",
    medium: "中危",
    low: "低危",
    info: "信息",
  };
  return <Badge className={cls[s]}>{label[s]}</Badge>;
}

export default function VulnTemplatesPage() {
  const toast = useToast();
  const [templates, setTemplates] = useState<FindingTemplate[]>([]);
  const [loading, setLoading] = useState(false);

  const [editorOpen, setEditorOpen] = useState(false);
  const [editingId, setEditingId] = useState<string | null>(null);
  const [editInitial, setEditInitial] = useState<Partial<TemplateFormData>>({});

  const [confirmOpen, setConfirmOpen] = useState(false);
  const [confirmConfig, setConfirmConfig] = useState({ title: "", onConfirm: () => {} });

  const abortRef = useRef<AbortController | null>(null);

  const load = useCallback(async () => {
    abortRef.current?.abort();
    const ctrl = new AbortController();
    abortRef.current = ctrl;
    setLoading(true);
    try {
      const list = await api.listFindingTemplates(undefined, ctrl.signal);
      setTemplates(list || []);
    } catch (err: any) {
      if (err?.name === "AbortError") return;
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    load();
    return () => abortRef.current?.abort();
  }, [load]);

  function openCreate() {
    setEditingId(null);
    setEditInitial({});
    setEditorOpen(true);
  }

  function openEdit(t: FindingTemplate) {
    setEditingId(t.id);
    setEditInitial({
      source_tool: t.source_tool,
      match_keys: t.match_keys,
      title: t.title,
      severity: t.severity,
      summary: t.summary,
      remediation: t.remediation,
      enabled: t.enabled,
    });
    setEditorOpen(true);
  }

  async function handleSave(form: TemplateFormData) {
    if (!form.source_tool.trim() || form.match_keys.length === 0) {
      toast("检测工具与匹配键不能为空", "warning");
      return;
    }
    if (!form.summary.trim() && !form.remediation.trim() && !form.title.trim() && !form.severity) {
      toast("模板至少填写一项覆盖字段(标题/严重等级/描述/修复)", "warning");
      return;
    }
    try {
      const payload: Partial<Omit<FindingTemplate, "id" | "created_at" | "updated_at">> = {
        source_tool: form.source_tool.trim(),
        match_keys: form.match_keys,
        title: form.title.trim(),
        severity: form.severity as FindingTemplate["severity"],
        summary: form.summary,
        remediation: form.remediation,
        enabled: form.enabled,
      };
      if (editingId) {
        await api.patchFindingTemplate(editingId, payload);
        toast("模板已更新", "success");
      } else {
        await api.createFindingTemplate(payload);
        toast("模板已创建", "success");
      }
      setEditorOpen(false);
      load();
    } catch (err: any) {
      toast(err?.message || "保存失败", "error");
    }
  }

  function handleDelete(t: FindingTemplate) {
    const keys = t.match_keys.join(", ");
    setConfirmConfig({
      title: `删除模板 "${keys}"？`,
      onConfirm: async () => {
        try {
          await api.deleteFindingTemplate(t.id);
          toast("已删除", "success");
          setConfirmOpen(false);
          load();
        } catch (err: any) {
          toast(err?.message || "删除失败", "error");
        }
      },
    });
    setConfirmOpen(true);
  }

  async function toggleEnabled(t: FindingTemplate) {
    try {
      await api.patchFindingTemplate(t.id, { enabled: !t.enabled });
      toast(t.enabled ? "已禁用" : "已启用", "success");
      load();
    } catch (err: any) {
      toast(err?.message || "操作失败", "error");
    }
  }

  async function acceptUpstream(t: FindingTemplate) {
    try {
      await api.acceptFindingTemplateUpstream(t.id);
      toast("已应用上游版本", "success");
      load();
    } catch (err: any) {
      toast(err?.message || "操作失败", "error");
    }
  }

  function handleExportAll() {
    const token = getApiToken();
    const url = `${API_BASE}/finding-templates/export`;
    if (!token) {
      window.open(url, "_blank");
      return;
    }
    fetch(url, { headers: { Authorization: `Bearer ${token}` } })
      .then((r) => {
        if (!r.ok) throw new Error(`导出失败:${r.status}`);
        return r.blob();
      })
      .then((blob) => {
        const a = document.createElement("a");
        const u = URL.createObjectURL(blob);
        a.href = u;
        a.download = "vuln-templates.json";
        document.body.appendChild(a);
        a.click();
        document.body.removeChild(a);
        URL.revokeObjectURL(u);
        toast("已下载 JSON (match_keys 数组格式)", "success");
      })
      .catch((e) => toast(e.message || "导出失败", "error"));
  }

  return (
    <div className="space-y-8 animate-in fade-in duration-500 pb-20">
      <div className="flex items-start justify-between">
        <div>
          <div className="flex items-center gap-2 text-rose-400 font-bold text-xs uppercase tracking-widest mb-1.5">
            <ShieldAlert className="h-3.5 w-3.5" />
            Vulnerability Knowledge Base
          </div>
          <h1 className="text-3xl font-black tracking-tight text-foreground">漏洞模板</h1>
          <p className="text-muted-foreground mt-1">
            为高频漏洞维护中文标题、描述与修复建议；生成报告时按「检测工具 + 匹配键」精确套用，非空字段覆盖 finding 原值。
          </p>
        </div>
        <div className="flex gap-2">
          <Button variant="outline" onClick={handleExportAll} title="导出 JSON，match_keys 为数组格式">
            <Download className="mr-2 h-4 w-4" />
            导出 JSON
          </Button>
          <Button variant="secondary" onClick={openCreate}>
            <Plus className="mr-2 h-4 w-4" />
            新增模板
          </Button>
        </div>
      </div>

      <Card className="overflow-hidden">
        {loading && templates.length === 0 ? (
          <div className="p-20 text-center text-muted-foreground">加载中...</div>
        ) : templates.length === 0 ? (
          <div className="p-20 text-center">
            <EmptyState
              title="暂无模板"
              description="新增模板后，扫描产生的相同漏洞将在报告中自动套用您的标准描述与修复建议"
            />
          </div>
        ) : (
          <Table>
            <TableHeader className="bg-white/5">
              <TableRow>
                <TableHead>来源</TableHead>
                <TableHead>检测工具</TableHead>
                <TableHead>匹配键</TableHead>
                <TableHead>模板标题</TableHead>
                <TableHead>严重等级</TableHead>
                <TableHead>状态</TableHead>
                <TableHead className="text-right">操作</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {templates.map((t) => (
                <TableRow
                  key={t.id}
                  className="border-white/5 bg-white/[0.02] hover:bg-white/[0.04] transition-colors"
                >
                  <TableCell>
                    {t.is_builtin ? (
                      <div className="flex flex-col gap-1 items-start">
                        <Badge className="bg-indigo-500/10 text-indigo-400 border-indigo-500/20">
                          <Lock className="mr-1 h-3 w-3 inline" />
                          内置
                        </Badge>
                        {t.user_modified && (
                          <Badge className="bg-amber-500/10 text-amber-400 border-amber-500/20 text-[10px]">
                            本地已修改
                          </Badge>
                        )}
                      </div>
                    ) : (
                      <Badge className="bg-emerald-500/10 text-emerald-400 border-emerald-500/20">
                        自定义
                      </Badge>
                    )}
                  </TableCell>
                  <TableCell>
                    <Badge variant="outline" className="font-mono">{t.source_tool}</Badge>
                  </TableCell>
                  <TableCell>
                    <div className="flex flex-wrap gap-1">
                      {t.match_keys.map((k) => (
                        <span key={k} className="px-1.5 py-0.5 text-[10px] bg-primary/10 text-primary rounded">
                          {k}
                        </span>
                      ))}
                    </div>
                  </TableCell>
                  <TableCell>
                    <span className="text-sm">{t.title || <span className="text-muted-foreground">—</span>}</span>
                  </TableCell>
                  <TableCell>{severityBadge(t.severity)}</TableCell>
                  <TableCell>
                    <button
                      onClick={() => toggleEnabled(t)}
                      className={`text-[10px] font-bold uppercase tracking-widest px-2 py-1 rounded ${
                        t.enabled
                          ? "bg-emerald-500/10 text-emerald-400 hover:bg-emerald-500/20"
                          : "bg-muted/40 text-muted-foreground hover:bg-muted/60"
                      }`}
                    >
                      {t.enabled ? "已启用" : "已禁用"}
                    </button>
                  </TableCell>
                  <TableCell>
                    <div className="flex items-center justify-end gap-1">
                      {t.is_builtin && t.user_modified && (
                        <Button
                          variant="ghost"
                          size="sm"
                          className="h-8 px-2 text-amber-400 hover:text-amber-300 hover:bg-amber-500/10"
                          onClick={() => acceptUpstream(t)}
                          title="放弃本地修改，应用上游(仓库)版本"
                        >
                          <RotateCcw className="mr-1 h-3.5 w-3.5" />
                          <span className="text-[10px]">应用上游</span>
                        </Button>
                      )}
                      <Button
                        variant="ghost"
                        size="sm"
                        className="h-8 w-8 p-0"
                        onClick={() => openEdit(t)}
                        title="编辑"
                      >
                        <Settings2 className="h-3.5 w-3.5" />
                      </Button>
                      {!t.is_builtin && (
                        <Button
                          variant="ghost"
                          size="sm"
                          className="h-8 w-8 p-0 text-rose-400 hover:text-rose-300"
                          onClick={() => handleDelete(t)}
                          title="删除"
                        >
                          <Trash2 className="h-3.5 w-3.5" />
                        </Button>
                      )}
                    </div>
                  </TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        )}
      </Card>

      <TemplateEditor
        open={editorOpen}
        onClose={() => setEditorOpen(false)}
        onSave={handleSave}
        initialData={editInitial}
        title="模板"
      />

      <ConfirmDialog
        open={confirmOpen}
        onClose={() => setConfirmOpen(false)}
        title={confirmConfig.title}
        onConfirm={confirmConfig.onConfirm}
        confirmText="删除"
        variant="danger"
      />
    </div>
  );
}
