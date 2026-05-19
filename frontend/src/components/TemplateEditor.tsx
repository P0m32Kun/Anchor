import { useState, useEffect } from "react";
import { Button } from "./ui/button";
import { Input } from "./Input";
import { X, Plus } from "lucide-react";

export interface TemplateFormData {
  source_tool: string;
  match_keys: string[];
  title: string;
  severity: string;
  summary: string;
  remediation: string;
  enabled: boolean;
}

interface TemplateEditorProps {
  open: boolean;
  onClose: () => void;
  onSave: (data: TemplateFormData) => void;
  initialData?: Partial<TemplateFormData>;
  title?: string;
}

const SOURCE_TOOL_OPTIONS = [
  "nuclei",
  "sqlmap",
  "hydra",
  "httpx",
  "dnsx",
  "ffuf",
  "其他",
];

export default function TemplateEditor({
  open,
  onClose,
  onSave,
  initialData,
  title = "词条",
}: TemplateEditorProps) {
  const [form, setForm] = useState<TemplateFormData>({
    source_tool: "nuclei",
    match_keys: [],
    title: "",
    severity: "",
    summary: "",
    remediation: "",
    enabled: true,
  });
  const [newKey, setNewKey] = useState("");

  useEffect(() => {
    if (open) {
      setForm({
        source_tool: initialData?.source_tool || "nuclei",
        match_keys: initialData?.match_keys || [],
        title: initialData?.title || "",
        severity: initialData?.severity || "",
        summary: initialData?.summary || "",
        remediation: initialData?.remediation || "",
        enabled: initialData?.enabled ?? true,
      });
      setNewKey("");
    }
  }, [open, initialData]);

  const addKey = () => {
    const k = newKey.trim();
    if (k && !form.match_keys.includes(k)) {
      setForm({ ...form, match_keys: [...form.match_keys, k] });
      setNewKey("");
    }
  };

  const removeKey = (k: string) => {
    setForm({ ...form, match_keys: form.match_keys.filter((x) => x !== k) });
  };

  if (!open) return null;

  const isValid = form.source_tool.trim() && form.match_keys.length > 0;

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center p-4 bg-background/80 backdrop-blur-sm">
      <div className="w-full max-w-lg bg-card border rounded-lg shadow-xl p-6 space-y-4 max-h-[90vh] overflow-y-auto">
        <div className="flex items-center justify-between">
          <h2 className="text-lg font-bold">
            {initialData?.title || initialData?.match_keys?.length
              ? `编辑${title}`
              : `新建${title}`}
          </h2>
          <button onClick={onClose} className="p-1 hover:bg-muted rounded">
            <X className="h-4 w-4" />
          </button>
        </div>

        <div className="space-y-3">
          <div className="grid grid-cols-2 gap-3">
            <div>
              <label className="text-sm font-medium">检测工具 *</label>
              <select
                value={form.source_tool}
                onChange={(e) =>
                  setForm({ ...form, source_tool: e.target.value })
                }
                className="w-full h-9 px-3 rounded-md border bg-background text-sm mt-1"
              >
                {SOURCE_TOOL_OPTIONS.map((v) => (
                  <option key={v} value={v}>
                    {v}
                  </option>
                ))}
              </select>
            </div>
            <div>
              <label className="text-sm font-medium">严重等级</label>
              <select
                value={form.severity}
                onChange={(e) =>
                  setForm({ ...form, severity: e.target.value })
                }
                className="w-full h-9 px-3 rounded-md border bg-background text-sm mt-1"
              >
                <option value="">保留 finding 原值</option>
                <option value="critical">严重 critical</option>
                <option value="high">高危 high</option>
                <option value="medium">中危 medium</option>
                <option value="low">低危 low</option>
                <option value="info">信息 info</option>
              </select>
            </div>
          </div>

          <div>
            <label className="text-sm font-medium">匹配键 (可多个) *</label>
            <div className="flex gap-2 mt-1">
              <Input
                value={newKey}
                onChange={(e) => setNewKey(e.target.value)}
                placeholder="输入匹配键按回车添加"
                onKeyDown={(e) => {
                  if (e.key === "Enter") {
                    e.preventDefault();
                    addKey();
                  }
                }}
              />
              <Button type="button" size="sm" onClick={addKey}>
                <Plus className="h-4 w-4" />
              </Button>
            </div>
            <div className="flex flex-wrap gap-1 mt-2">
              {form.match_keys.map((k) => (
                <span
                  key={k}
                  className="inline-flex items-center gap-1 px-2 py-0.5 text-xs bg-primary/10 text-primary rounded-full"
                >
                  {k}
                  <button
                    onClick={() => removeKey(k)}
                    className="hover:text-primary-foreground"
                  >
                    <X className="h-3 w-3" />
                  </button>
                </span>
              ))}
            </div>
            <p className="text-[10px] text-muted-foreground mt-1">
              生成报告时按 finding 的 source_rule_id / matched_template /
              title 与这些键逐一匹配，命中即用。
            </p>
          </div>

          <div>
            <label className="text-sm font-medium">模板标题</label>
            <Input
              value={form.title}
              onChange={(e) => setForm({ ...form, title: e.target.value })}
              placeholder="留空则保留 finding 原标题"
              className="mt-1"
            />
          </div>

          <div>
            <label className="text-sm font-medium">漏洞描述</label>
            <textarea
              value={form.summary}
              onChange={(e) => setForm({ ...form, summary: e.target.value })}
              placeholder="留空则保留 finding 原描述"
              rows={3}
              className="w-full rounded-md border bg-background px-3 py-2 text-sm mt-1 resize-y focus-visible:outline-none focus-visible:ring-1 focus-visible:ring-ring"
            />
          </div>

          <div>
            <label className="text-sm font-medium">修复建议</label>
            <textarea
              value={form.remediation}
              onChange={(e) =>
                setForm({ ...form, remediation: e.target.value })
              }
              placeholder="留空则保留 finding 原修复建议"
              rows={3}
              className="w-full rounded-md border bg-background px-3 py-2 text-sm mt-1 resize-y focus-visible:outline-none focus-visible:ring-1 focus-visible:ring-ring"
            />
          </div>

          <label className="flex items-center gap-2 text-sm cursor-pointer select-none">
            <input
              type="checkbox"
              checked={form.enabled}
              onChange={(e) =>
                setForm({ ...form, enabled: e.target.checked })
              }
              className="h-4 w-4"
            />
            启用该模板（禁用后报告不会套用）
          </label>
        </div>

        <div className="flex justify-end gap-2 pt-2">
          <Button variant="ghost" onClick={onClose}>
            取消
          </Button>
          <Button onClick={() => onSave(form)} disabled={!isValid}>
            保存
          </Button>
        </div>
      </div>
    </div>
  );
}
