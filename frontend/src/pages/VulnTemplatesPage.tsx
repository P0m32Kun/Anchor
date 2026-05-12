import { useState, useEffect, useCallback, useRef } from "react";
import { api, type FindingTemplate, API_BASE } from "../lib/api";
import { getApiToken } from "../lib/config";
import {
  useToast,
  Button,
  Input,
  Card,
  Badge,
  Table,
  TableHeader,
  TableBody,
  TableRow,
  TableHead,
  TableCell,
  Modal,
  ConfirmDialog,
  EmptyState,
} from "../components";
import { ShieldAlert, Plus, Trash2, Settings2, Download, RotateCcw, Lock } from "lucide-react";

const SOURCE_TOOL_OPTIONS = [
  "nuclei",
  "sqlmap",
  "hydra",
  "httpx",
  "dnsx",
  "ffuf",
  "urlfinder",
  "其他",
];

const SEVERITY_OPTIONS: { value: FindingTemplate["severity"]; label: string }[] = [
  { value: "", label: "保留 finding 原值" },
  { value: "critical", label: "严重 critical" },
  { value: "high", label: "高危 high" },
  { value: "medium", label: "中危 medium" },
  { value: "low", label: "低危 low" },
  { value: "info", label: "信息 info" },
];

type TemplateForm = {
  source_tool: string;
  match_key: string;
  title: string;
  severity: FindingTemplate["severity"];
  summary: string;
  remediation: string;
  enabled: boolean;
};

const EMPTY_FORM: TemplateForm = {
  source_tool: "nuclei",
  match_key: "",
  title: "",
  severity: "",
  summary: "",
  remediation: "",
  enabled: true,
};

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
  const [form, setForm] = useState<TemplateForm>(EMPTY_FORM);
  const [saving, setSaving] = useState(false);

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
    setForm(EMPTY_FORM);
    setEditorOpen(true);
  }

  function openEdit(t: FindingTemplate) {
    setEditingId(t.id);
    setForm({
      source_tool: t.source_tool,
      match_key: t.match_key,
      title: t.title,
      severity: t.severity,
      summary: t.summary,
      remediation: t.remediation,
      enabled: t.enabled,
    });
    setEditorOpen(true);
  }

  async function handleSave() {
    if (!form.source_tool.trim() || !form.match_key.trim()) {
      toast("检测工具与匹配键不能为空", "warning");
      return;
    }
    if (!form.summary.trim() && !form.remediation.trim() && !form.title.trim() && !form.severity) {
      toast("模板至少填写一项覆盖字段(标题/严重等级/描述/修复)", "warning");
      return;
    }
    setSaving(true);
    try {
      const payload = {
        source_tool: form.source_tool.trim(),
        match_key: form.match_key.trim(),
        title: form.title.trim(),
        severity: form.severity,
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
    } finally {
      setSaving(false);
    }
  }

  function handleDelete(t: FindingTemplate) {
    setConfirmConfig({
      title: `删除模板 "${t.match_key}"？`,
      onConfirm: async () => {
        try {
          await api.deleteFindingTemplate(t.id);
          toast("已删除", "success");
          setConfirmOpen(false);
          load();
        } catch (err: any) {
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
    }
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
            为高频漏洞维护中文标题、描述与修复建议;生成报告时按「检测工具 + 匹配键」精确套用,非空字段覆盖 finding 原值。
          </p>
        </div>
        <Button variant="secondary" onClick={openCreate}>
          <Plus className="mr-2 h-4 w-4" />
          新增模板
        </Button>
      </div>

      <Card className="overflow-hidden">
        {loading && templates.length === 0 ? (
          <div className="p-20 text-center text-muted-foreground">加载中...</div>
        ) : templates.length === 0 ? (
          <div className="p-20 text-center">
            <EmptyState
              title="暂无模板"
              description="新增模板后,扫描产生的相同漏洞将在报告中自动套用您的标准描述与修复建议"
            />
          </div>
        ) : (
          <Table>
            <TableHeader className="bg-white/5">
              <TableRow>
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
                    <Badge variant="outline" className="font-mono">{t.source_tool}</Badge>
                  </TableCell>
                  <TableCell>
                    <span className="font-mono text-xs">{t.match_key}</span>
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
                      <Button
                        variant="ghost"
                        size="sm"
                        className="h-8 w-8 p-0"
                        onClick={() => openEdit(t)}
                        title="编辑"
                      >
                        <Settings2 className="h-3.5 w-3.5" />
                      </Button>
                      <Button
                        variant="ghost"
                        size="sm"
                        className="h-8 w-8 p-0 text-rose-400 hover:text-rose-300"
                        onClick={() => handleDelete(t)}
                        title="删除"
                      >
                        <Trash2 className="h-3.5 w-3.5" />
                      </Button>
                    </div>
                  </TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        )}
      </Card>

      <Modal
        open={editorOpen}
        onClose={() => setEditorOpen(false)}
        title={editingId ? "编辑模板" : "新增模板"}
        description="匹配键可填:nuclei 的 template-id、其他工具的规则 ID,或漏洞标题精确文本"
        footer={
          <>
            <Button variant="secondary" onClick={() => setEditorOpen(false)} disabled={saving}>
              取消
            </Button>
            <Button onClick={handleSave} loading={saving}>{editingId ? "保存" : "创建"}</Button>
          </>
        }
      >
        <div className="space-y-4">
          <div className="grid grid-cols-2 gap-3">
            <div>
              <label className="text-sm font-medium mb-1.5 block">检测工具 *</label>
              <select
                value={form.source_tool}
                onChange={(e) => setForm((p) => ({ ...p, source_tool: e.target.value }))}
                className="w-full rounded-lg border border-white/10 bg-white/5 px-3 py-2 text-sm focus:outline-none focus:ring-1 focus:ring-primary/50"
              >
                {SOURCE_TOOL_OPTIONS.map((v) => (
                  <option key={v} value={v}>{v}</option>
                ))}
              </select>
            </div>
            <div>
              <label className="text-sm font-medium mb-1.5 block">严重等级</label>
              <select
                value={form.severity}
                onChange={(e) => setForm((p) => ({ ...p, severity: e.target.value as FindingTemplate["severity"] }))}
                className="w-full rounded-lg border border-white/10 bg-white/5 px-3 py-2 text-sm focus:outline-none focus:ring-1 focus:ring-primary/50"
              >
                {SEVERITY_OPTIONS.map((opt) => (
                  <option key={opt.value} value={opt.value}>{opt.label}</option>
                ))}
              </select>
            </div>
          </div>

          <div>
            <label className="text-sm font-medium mb-1.5 block">匹配键 *</label>
            <Input
              value={form.match_key}
              onChange={(e) => setForm((p) => ({ ...p, match_key: e.target.value }))}
              placeholder="例如:exposed-git 或 SQL 注入漏洞 (登录接口)"
            />
            <p className="text-[10px] text-muted-foreground mt-1">
              生成报告时按 finding 的 source_rule_id → matched_template → title 顺序与此键精确匹配,命中即用。
            </p>
          </div>

          <div>
            <label className="text-sm font-medium mb-1.5 block">模板标题</label>
            <Input
              value={form.title}
              onChange={(e) => setForm((p) => ({ ...p, title: e.target.value }))}
              placeholder="留空则保留 finding 原标题"
            />
          </div>

          <div>
            <label className="text-sm font-medium mb-1.5 block">漏洞描述</label>
            <textarea
              value={form.summary}
              onChange={(e) => setForm((p) => ({ ...p, summary: e.target.value }))}
              placeholder="留空则保留 finding 原描述"
              rows={4}
              className="w-full rounded-lg border border-white/10 bg-white/5 px-3 py-2 text-sm focus:outline-none focus:ring-1 focus:ring-primary/50 resize-y"
            />
          </div>

          <div>
            <label className="text-sm font-medium mb-1.5 block">修复建议</label>
            <textarea
              value={form.remediation}
              onChange={(e) => setForm((p) => ({ ...p, remediation: e.target.value }))}
              placeholder="留空则保留 finding 原修复建议"
              rows={4}
              className="w-full rounded-lg border border-white/10 bg-white/5 px-3 py-2 text-sm focus:outline-none focus:ring-1 focus:ring-primary/50 resize-y"
            />
          </div>

          <label className="flex items-center gap-2 text-sm cursor-pointer select-none">
            <input
              type="checkbox"
              checked={form.enabled}
              onChange={(e) => setForm((p) => ({ ...p, enabled: e.target.checked }))}
              className="h-4 w-4"
            />
            启用该模板(禁用后报告不会套用)
          </label>
        </div>
      </Modal>

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
