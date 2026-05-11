import { useEffect, useState } from "react";
import { api } from "../lib/api";
import { 
  useToast, 
  Button, 
  Input,
  Card,
  CardHeader,
  CardTitle,
  CardContent,
  Badge
} from "../components";
import { ShieldCheck, Save, Trash2, Database, AlertCircle } from "lucide-react";
import { cn } from "../lib/utils";

interface EngineForm {
  engine: string;
  apiKey: string;
  showKey: boolean;
  saving: boolean;
  deleting: boolean;
  exists: boolean;
}

const ENGINE_DEFS = [
  { key: "fofa", label: "FOFA", color: "from-blue-500/20 to-blue-600/5", icon: Database },
  { key: "hunter", label: "Hunter", color: "from-cyan-500/20 to-cyan-600/5", icon: Database },
  { key: "quake", label: "Quake", color: "from-indigo-500/20 to-indigo-600/5", icon: Database },
];

function maskKey(key: string): string {
  if (!key) return "";
  if (key.length <= 4) return "****";
  return key.slice(0, 4) + "****";
}

export default function EngineKeysPage() {
  const toast = useToast();
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
    api.listEngineCredentials()
      .then((creds) => {
        if (!mounted) return;
        setForms((prev) => {
          const next = { ...prev };
          for (const c of creds) {
            if (next[c.engine]) {
                next[c.engine] = {
                ...next[c.engine],
                apiKey: maskKey(c.api_key),
                exists: true,
                };
            }
          }
          return next;
        });
      })
      .catch(() => {});
    return () => { mounted = false; };
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
    if (form.exists && form.apiKey.includes("****")) {
      toast("API Key 未修改", "warning");
      return;
    }

    updateForm(engine, { saving: true });
    try {
      await api.saveEngineCredential({ engine, api_key: form.apiKey.trim() });
      updateForm(engine, { exists: true, apiKey: maskKey(form.apiKey.trim()), showKey: false });
      toast("保存成功", "success");
    } catch (err: any) {
    } finally {
      updateForm(engine, { saving: false });
    }
  }

  async function handleDelete(engine: string) {
    const form = forms[engine];
    if (!form.exists) {
      updateForm(engine, { apiKey: "" });
      return;
    }
    if (!window.confirm(`确认删除 ${ENGINE_DEFS.find((d) => d.key === engine)?.label} 的 API Key？`)) return;
    updateForm(engine, { deleting: true });
    try {
      await api.deleteEngineCredential(engine);
      updateForm(engine, { apiKey: "", exists: false, showKey: false });
      toast("删除成功", "success");
    } catch (err: any) {
    } finally {
      updateForm(engine, { deleting: false });
    }
  }

  return (
    <div className="space-y-8 animate-in fade-in duration-500">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-3xl font-black tracking-tight text-foreground">API 凭证管理</h1>
          <p className="text-muted-foreground mt-1">配置空间测绘引擎的访问密钥，这些密钥将安全地存储在后端。</p>
        </div>
      </div>

      <div className="grid gap-6 md:grid-cols-3">
        {ENGINE_DEFS.map((def) => {
          const form = forms[def.key];
          return (
            <Card key={def.key} className={cn("relative overflow-hidden group border-white/[0.03]", form.exists && "border-emerald-500/20")}>
              <div className={cn("absolute top-0 left-0 w-full h-1 bg-gradient-to-r", def.color)} />
              
              <CardHeader className="pb-4">
                <div className="flex items-center justify-between">
                   <div className="flex items-center gap-3">
                      <div className="h-10 w-10 rounded-xl bg-white/5 flex items-center justify-center border border-white/5">
                        <def.icon className="h-5 w-5 text-muted-foreground" />
                      </div>
                      <CardTitle className="text-xl">{def.label}</CardTitle>
                   </div>
                   {form.exists ? (
                     <Badge variant="success" dot>Active</Badge>
                   ) : (
                     <Badge variant="secondary">Missing</Badge>
                   )}
                </div>
              </CardHeader>

              <CardContent className="space-y-5">
                <div className="space-y-2">
                   <div className="flex items-center justify-between">
                      <label className="text-[10px] font-black uppercase tracking-widest text-muted-foreground/60">Provider API Key</label>
                      <button 
                        onClick={() => updateForm(def.key, { showKey: !form.showKey })}
                        className="text-[10px] font-bold text-primary hover:underline"
                      >
                        {form.showKey ? "HIDE" : "SHOW"}
                      </button>
                   </div>
                   <div className="relative">
                      <Input
                        type={form.showKey ? "text" : "password"}
                        placeholder="Paste your key here..."
                        value={form.apiKey}
                        onChange={(e) => updateForm(def.key, { apiKey: e.target.value })}
                        className="font-mono bg-white/5 border-white/5 h-10 pr-10 focus-visible:ring-primary/30"
                      />
                      <div className="absolute right-3 top-2.5">
                         {form.exists ? <ShieldCheck className="h-4 w-4 text-emerald-500" /> : <AlertCircle className="h-4 w-4 text-amber-500" />}
                      </div>
                   </div>
                </div>

                <div className="flex gap-2 pt-2">
                  <Button
                    onClick={() => handleSave(def.key)}
                    disabled={form.saving}
                    variant={form.exists ? "secondary" : "primary"}
                    className="flex-1 h-10 font-bold"
                  >
                    <Save className="mr-2 h-4 w-4" />
                    {form.saving ? "Saving..." : form.exists ? "Update" : "Save Key"}
                  </Button>
                  {form.exists && (
                    <Button
                        variant="ghost"
                        onClick={() => handleDelete(def.key)}
                        disabled={form.deleting}
                        className="w-10 h-10 p-0 text-muted-foreground hover:text-rose-500 hover:bg-rose-500/10"
                    >
                        <Trash2 className="h-4 w-4" />
                    </Button>
                  )}
                </div>
              </CardContent>
            </Card>
          );
        })}
      </div>

      <Card className="bg-amber-500/5 border-amber-500/20">
         <CardContent className="p-6 flex items-start gap-4">
            <div className="h-10 w-10 rounded-xl bg-amber-500/10 flex items-center justify-center shrink-0">
                <AlertCircle className="h-6 w-6 text-amber-500" />
            </div>
            <div>
                <h4 className="text-sm font-bold text-amber-500">安全声明</h4>
                <p className="text-xs text-muted-foreground mt-1 leading-relaxed">
                    所有 API Key 均会通过加密通道传输并存储在您的私有数据库中。系统不会将这些密钥发送至任何第三方（搜索引擎官方 API 除外）。请确保您的环境授权合规。
                </p>
            </div>
         </CardContent>
      </Card>
    </div>
  );
}
