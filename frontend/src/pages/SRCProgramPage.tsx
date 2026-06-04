import { useState, useEffect } from "react";
import { useParams } from "react-router-dom";
import { srcApi, SRCProgram, CreateSRCProgramRequest } from "../lib/api";
import { Button, Input, Switch, Select, Card, CardHeader, CardTitle, CardContent, useToast } from "../components";
import { Loader2, Save, Trash2, ExternalLink } from "lucide-react";

const PLATFORMS = [
  { value: "hackerone", label: "HackerOne" },
  { value: "bugcrowd", label: "Bugcrowd" },
  { value: "intigriti", label: "Intigriti" },
  { value: "yeswehack", label: "YesWeHack" },
  { value: "360src", label: "360SRC" },
  { value: "butian", label: "补天" },
  { value: "qianxin", label: "奇安信" },
  { value: "topsec", label: "天融信" },
  { value: "dbappsecurity", label: "安恒信息" },
  { value: "disijie", label: "迪思杰" },
  { value: "other", label: "其他" },
];

const VULN_TYPES = [
  "rce", "sqli", "ssrf", "idor", "file_read", "file_upload",
  "auth_bypass", "secret_leak", "info_disclosure", "xss", "csrf",
  "open_redirect", "default_password", "weak_password",
];

export default function SRCProgramPage() {
  const { projectId } = useParams<{ projectId: string }>();
  const toast = useToast();
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const [program, setProgram] = useState<SRCProgram | null>(null);
  const [form, setForm] = useState<CreateSRCProgramRequest>({
    name: "",
    platform: "other",
    program_url: "",
    rules_url: "",
    allow_automation: false,
    allow_dir_brute: false,
    allow_weak_password: false,
    allow_authenticated_test: false,
    max_rps: 5,
    max_concurrency: 3,
    preferred_vuln_types: [],
    notes: "",
  });

  useEffect(() => {
    if (projectId) {
      loadProgram();
    }
  }, [projectId]);

  const loadProgram = async () => {
    try {
      setLoading(true);
      const data = await srcApi.getProgram(projectId!);
      if (data) {
        setProgram(data);
        setForm({
          name: data.name,
          platform: data.platform,
          program_url: data.program_url,
          rules_url: data.rules_url,
          allow_automation: data.allow_automation,
          allow_dir_brute: data.allow_dir_brute,
          allow_weak_password: data.allow_weak_password,
          allow_authenticated_test: data.allow_authenticated_test,
          max_rps: data.max_rps,
          max_concurrency: data.max_concurrency,
          preferred_vuln_types: data.preferred_vuln_types,
          notes: data.notes,
        });
      }
    } catch (error) {
      console.error("Failed to load program:", error);
    } finally {
      setLoading(false);
    }
  };

  const handleSave = async () => {
    try {
      setSaving(true);
      if (program) {
        await srcApi.updateProgram(projectId!, form);
        toast("SRC 程序配置已更新", "success");
      } else {
        await srcApi.createProgram(projectId!, form);
        toast("SRC 程序配置已创建", "success");
      }
      await loadProgram();
    } catch (error) {
      toast("请检查输入并重试", "error");
    } finally {
      setSaving(false);
    }
  };

  const handleDelete = async () => {
    if (!confirm("确定要删除此 SRC 程序配置吗？")) return;
    try {
      await srcApi.deleteProgram(projectId!);
      setProgram(null);
      setForm({
        name: "",
        platform: "other",
        program_url: "",
        rules_url: "",
        allow_automation: false,
        allow_dir_brute: false,
        allow_weak_password: false,
        allow_authenticated_test: false,
        max_rps: 5,
        max_concurrency: 3,
        preferred_vuln_types: [],
        notes: "",
      });
      toast("SRC 程序配置已删除", "success");
    } catch (error) {
      toast("请重试", "error");
    }
  };

  const toggleVulnType = (type: string) => {
    setForm((prev) => ({
      ...prev,
      preferred_vuln_types: prev.preferred_vuln_types?.includes(type)
        ? prev.preferred_vuln_types.filter((t) => t !== type)
        : [...(prev.preferred_vuln_types || []), type],
    }));
  };

  if (loading) {
    return (
      <div className="flex items-center justify-center h-64">
        <Loader2 className="h-8 w-8 animate-spin" />
      </div>
    );
  }

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold">SRC 程序配置</h1>
          <p className="text-muted-foreground">
            配置安全应急响应中心的扫描规则和参数
          </p>
        </div>
        <div className="flex gap-2">
          {program && (
            <Button variant="danger" onClick={handleDelete}>
              <Trash2 className="h-4 w-4 mr-2" />
              删除
            </Button>
          )}
          <Button onClick={handleSave} disabled={saving}>
            {saving ? <Loader2 className="h-4 w-4 mr-2 animate-spin" /> : <Save className="h-4 w-4 mr-2" />}
            {program ? "更新" : "创建"}
          </Button>
        </div>
      </div>

      <div className="grid gap-6 md:grid-cols-2">
        {/* 基本信息 */}
        <Card>
          <CardHeader>
            <CardTitle>基本信息</CardTitle>
          </CardHeader>
          <CardContent className="space-y-4">
            <div className="space-y-2">
              <label className="text-sm font-medium">程序名称</label>
              <Input
                value={form.name}
                onChange={(e) => setForm({ ...form, name: e.target.value })}
                placeholder="例如：360SRC"
              />
            </div>

            <div className="space-y-2">
              <label className="text-sm font-medium">平台</label>
              <Select
                value={form.platform}
                onChange={(e) => setForm({ ...form, platform: e.target.value })}
              >
                {PLATFORMS.map((p) => (
                  <option key={p.value} value={p.value}>
                    {p.label}
                  </option>
                ))}
              </Select>
            </div>

            <div className="space-y-2">
              <label className="text-sm font-medium">程序 URL</label>
              <div className="flex gap-2">
                <Input
                  value={form.program_url}
                  onChange={(e) => setForm({ ...form, program_url: e.target.value })}
                  placeholder="https://example.com/src"
                />
                {form.program_url && (
                  <a
                    href={form.program_url}
                    target="_blank"
                    rel="noopener noreferrer"
                    className="inline-flex items-center justify-center h-9 px-3 border rounded-md hover:bg-muted"
                  >
                    <ExternalLink className="h-4 w-4" />
                  </a>
                )}
              </div>
            </div>

            <div className="space-y-2">
              <label className="text-sm font-medium">规则 URL</label>
              <div className="flex gap-2">
                <Input
                  value={form.rules_url}
                  onChange={(e) => setForm({ ...form, rules_url: e.target.value })}
                  placeholder="https://example.com/src/rules"
                />
                {form.rules_url && (
                  <a
                    href={form.rules_url}
                    target="_blank"
                    rel="noopener noreferrer"
                    className="inline-flex items-center justify-center h-9 px-3 border rounded-md hover:bg-muted"
                  >
                    <ExternalLink className="h-4 w-4" />
                  </a>
                )}
              </div>
            </div>

            <div className="space-y-2">
              <label className="text-sm font-medium">备注</label>
              <Input
                value={form.notes}
                onChange={(e) => setForm({ ...form, notes: e.target.value })}
                placeholder="可选备注..."
              />
            </div>
          </CardContent>
        </Card>

        {/* 扫描配置 */}
        <Card>
          <CardHeader>
            <CardTitle>扫描配置</CardTitle>
          </CardHeader>
          <CardContent className="space-y-4">
            <div className="grid grid-cols-2 gap-4">
              <div className="space-y-2">
                <label className="text-sm font-medium">最大 RPS</label>
                <Input
                  type="number"
                  min={1}
                  max={100}
                  value={form.max_rps}
                  onChange={(e) => setForm({ ...form, max_rps: parseInt(e.target.value) || 5 })}
                />
              </div>
              <div className="space-y-2">
                <label className="text-sm font-medium">最大并发</label>
                <Input
                  type="number"
                  min={1}
                  max={50}
                  value={form.max_concurrency}
                  onChange={(e) => setForm({ ...form, max_concurrency: parseInt(e.target.value) || 3 })}
                />
              </div>
            </div>

            <div className="space-y-3">
              <label className="text-sm font-medium">权限开关</label>
              <div className="space-y-2">
                <div className="flex items-center justify-between">
                  <label className="text-sm">允许自动化</label>
                  <Switch
                    checked={form.allow_automation ?? false}
                    onCheckedChange={(checked) => setForm({ ...form, allow_automation: checked })}
                  />
                </div>
                <div className="flex items-center justify-between">
                  <label className="text-sm">允许目录爆破</label>
                  <Switch
                    checked={form.allow_dir_brute ?? false}
                    onCheckedChange={(checked) => setForm({ ...form, allow_dir_brute: checked })}
                  />
                </div>
                <div className="flex items-center justify-between">
                  <label className="text-sm">允许弱口令测试</label>
                  <Switch
                    checked={form.allow_weak_password ?? false}
                    onCheckedChange={(checked) => setForm({ ...form, allow_weak_password: checked })}
                  />
                </div>
                <div className="flex items-center justify-between">
                  <label className="text-sm">允许认证测试</label>
                  <Switch
                    checked={form.allow_authenticated_test ?? false}
                    onCheckedChange={(checked) => setForm({ ...form, allow_authenticated_test: checked })}
                  />
                </div>
              </div>
            </div>
          </CardContent>
        </Card>

        {/* 偏好漏洞类型 */}
        <Card className="md:col-span-2">
          <CardHeader>
            <CardTitle>偏好漏洞类型</CardTitle>
          </CardHeader>
          <CardContent>
            <div className="flex flex-wrap gap-2">
              {VULN_TYPES.map((type) => (
                <Button
                  key={type}
                  variant={form.preferred_vuln_types?.includes(type) ? "primary" : "outline"}
                  size="sm"
                  onClick={() => toggleVulnType(type)}
                >
                  {type}
                </Button>
              ))}
            </div>
          </CardContent>
        </Card>
      </div>
    </div>
  );
}
