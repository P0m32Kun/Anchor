import { useEffect, useState } from "react";
import { api } from "../lib/api";
import { useToast, Button, Input } from "../components";

interface EngineForm {
  engine: string;
  apiKey: string;
  showKey: boolean;
  saving: boolean;
  deleting: boolean;
  exists: boolean;
}

const ENGINE_DEFS = [
  { key: "fofa", label: "FOFA" },
  { key: "hunter", label: "Hunter" },
  { key: "quake", label: "Quake" },
];

function maskKey(key: string): string {
  if (!key) return "";
  if (key.length <= 4) return "****";
  return key.slice(0, 4) + "****";
}

export default function EngineKeysPage() {
  const toast = useToast();
  const [loading, setLoading] = useState(true);
  const [forms, setForms] = useState<Record<string, EngineForm>>(() => {
    const init: Record<string, EngineForm> = {};
    for (const def of ENGINE_DEFS) {
      init[def.key] = {
        engine: def.key,
        apiKey: "",
        showKey: false,
        saving: false,
        deleting: false,
        exists: false,
      };
    }
    return init;
  });

  useEffect(() => {
    let mounted = true;
    api
      .listEngineCredentials()
      .then((creds) => {
        if (!mounted) return;
        setForms((prev) => {
          const next = { ...prev };
          for (const c of creds) {
            next[c.engine] = {
              ...next[c.engine],
              apiKey: maskKey(c.api_key),
              exists: true,
            };
          }
          return next;
        });
      })
      .catch((err) => {
        toast("加载凭证失败: " + (err?.message || String(err)), "error");
      })
      .finally(() => setLoading(false));
    return () => {
      mounted = false;
    };
  }, [toast]);

  function updateForm(engine: string, patch: Partial<EngineForm>) {
    setForms((prev) => ({
      ...prev,
      [engine]: { ...prev[engine], ...patch },
    }));
  }

  async function handleSave(engine: string) {
    const form = forms[engine];
    if (!form.apiKey.trim()) {
      toast("API Key 不能为空", "warning");
      return;
    }
    // If user hasn't changed the masked key, skip
    if (form.exists && form.apiKey.includes("****")) {
      toast("API Key 未修改", "warning");
      return;
    }

    updateForm(engine, { saving: true });
    try {
      await api.saveEngineCredential({
        engine,
        api_key: form.apiKey.trim(),
        email: form.email.trim() || undefined,
      });
      updateForm(engine, {
        exists: true,
        apiKey: maskKey(form.apiKey.trim()),
        showKey: false,
      });
      toast("保存成功", "success");
    } catch (err: any) {
      toast("保存失败: " + (err?.message || String(err)), "error");
    } finally {
      updateForm(engine, { saving: false });
    }
  }

  async function handleDelete(engine: string) {
    const form = forms[engine];
    if (!form.exists) {
      updateForm(engine, { apiKey: "", email: "" });
      return;
    }
    if (!window.confirm(`确认删除 ${ENGINE_DEFS.find((d) => d.key === engine)?.label} 的 API Key？`)) {
      return;
    }
    updateForm(engine, { deleting: true });
    try {
      await api.deleteEngineCredential(engine);
      updateForm(engine, {
        apiKey: "",
        email: "",
        exists: false,
        showKey: false,
      });
      toast("删除成功", "success");
    } catch (err: any) {
      toast("删除失败: " + (err?.message || String(err)), "error");
    } finally {
      updateForm(engine, { deleting: false });
    }
  }

  if (loading) {
    return (
      <div className="space-y-6">
        <div>
          <h1 className="text-xl font-semibold">API Key 配置</h1>
          <p className="text-sm text-text-tertiary mt-1">管理搜索引擎平台的 API 凭证</p>
        </div>
        <div className="cyber-glass p-5 animate-pulse">
          <div className="h-4 w-32 bg-white/[0.06] rounded mb-4" />
          <div className="h-10 bg-white/[0.06] rounded" />
        </div>
      </div>
    );
  }

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-xl font-semibold">API Key 配置</h1>
        <p className="text-sm text-text-tertiary mt-1">管理搜索引擎平台的 API 凭证</p>
      </div>

      <div className="grid gap-5 md:grid-cols-2 xl:grid-cols-3">
        {ENGINE_DEFS.map((def) => {
          const form = forms[def.key];
          return (
            <div key={def.key} className="cyber-glass p-5">
              <div className="flex items-center justify-between mb-4">
                <h3 className="text-base font-semibold text-text-primary">{def.label}</h3>
                {form.exists && (
                  <span className="text-[10px] font-medium text-brand-success bg-brand-success/10 ring-1 ring-brand-success/20 rounded-full px-2 py-0.5">
                    已配置
                  </span>
                )}
              </div>

              <div className="space-y-3">
                {def.needsEmail && (
                  <Input
                    label="Email"
                    placeholder="请输入 Email"
                    value={form.email}
                    onChange={(e) => updateForm(def.key, { email: e.target.value })}
                  />
                )}

                <div className="flex gap-2 items-start">
                  <Input
                    type={form.showKey ? "text" : "password"}
                    label="API Key"
                    placeholder="请输入 API Key"
                    value={form.apiKey}
                    onChange={(e) => updateForm(def.key, { apiKey: e.target.value })}
                    className="flex-1"
                  />
                  <button
                    onClick={() => updateForm(def.key, { showKey: !form.showKey })}
                    className="shrink-0 rounded-lg border border-white/[0.10] bg-white/[0.03] px-3 py-2 text-xs text-text-tertiary hover:bg-white/[0.06] transition-colors mt-7"
                    title={form.showKey ? "隐藏" : "显示"}
                  >
                    {form.showKey ? "隐藏" : "显示"}
                  </button>
                </div>

                <div className="flex gap-2 pt-2">
                  <Button
                    onClick={() => handleSave(def.key)}
                    disabled={form.saving}
                    className="flex-1"
                  >
                    {form.saving ? "保存中..." : "保存"}
                  </Button>
                  <Button
                    variant="secondary"
                    onClick={() => handleDelete(def.key)}
                    disabled={form.deleting}
                    className="flex-1"
                  >
                    {form.deleting ? "删除中..." : "删除"}
                  </Button>
                </div>
              </div>
            </div>
          );
        })}
      </div>
    </div>
  );
}
