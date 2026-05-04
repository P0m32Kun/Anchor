import { useEffect, useState, useCallback } from "react";
import {
  Button,
  Card,
  CardDescription,
  CardHeader,
  CardTitle,
  Input,
  Select,
  Skeleton,
  useProjectId,
} from "../components";
import { useToast } from "../components/Toast";
import {
  api,
  DEFAULT_PIPELINE_CONFIG,
  PORT_RANGE_PRESETS,
  type PipelineConfig,
} from "../lib/api";

const PRESET_VALUES = new Set(PORT_RANGE_PRESETS.map((p) => p.value).filter((v) => v !== "custom"));

function isPresetPort(value: string): boolean {
  return PRESET_VALUES.has(value);
}

export default function ScanConfigPage() {
  const projectId = useProjectId();
  const toast = useToast();

  const [config, setConfig] = useState<PipelineConfig>(DEFAULT_PIPELINE_CONFIG);
  const [originalConfig, setOriginalConfig] = useState<PipelineConfig>(DEFAULT_PIPELINE_CONFIG);
  const [portMode, setPortMode] = useState<string>("top1000");
  const [customPorts, setCustomPorts] = useState<string>("");
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const load = useCallback(
    async (signal?: AbortSignal) => {
      if (!projectId) return;
      setLoading(true);
      setError(null);
      try {
        const data = await api.getPipelineConfig(projectId, signal);
        setConfig(data);
        setOriginalConfig(data);
        if (isPresetPort(data.port_range)) {
          setPortMode(data.port_range);
          setCustomPorts("");
        } else {
          setPortMode("custom");
          setCustomPorts(data.port_range);
        }
      } catch (err) {
        if (err instanceof DOMException && err.name === "AbortError") return;
        const msg = err instanceof Error ? err.message : "加载扫描配置失败";
        setError(msg);
      } finally {
        setLoading(false);
      }
    },
    [projectId]
  );

  useEffect(() => {
    if (!projectId) return;
    const ctrl = new AbortController();
    load(ctrl.signal);
    return () => ctrl.abort();
  }, [projectId, load]);

  const handlePortModeChange = (value: string) => {
    setPortMode(value);
    if (value !== "custom") {
      setConfig((c) => ({ ...c, port_range: value }));
    } else {
      setConfig((c) => ({ ...c, port_range: customPorts }));
    }
  };

  const handleCustomPortsChange = (value: string) => {
    setCustomPorts(value);
    if (portMode === "custom") {
      setConfig((c) => ({ ...c, port_range: value }));
    }
  };

  const updateField = <K extends keyof PipelineConfig>(key: K, value: PipelineConfig[K]) => {
    setConfig((c) => ({ ...c, [key]: value }));
  };

  const validate = (): string | null => {
    if (portMode === "custom" && !customPorts.trim()) {
      return "请填写自定义端口";
    }
    if (portMode === "custom") {
      const trimmed = customPorts.trim();
      if (!/^[0-9,\-\s]+$/.test(trimmed)) {
        return "自定义端口格式错误，仅支持数字、逗号、连字符";
      }
    }
    if (config.fofa_result_limit < 1 || config.fofa_result_limit > 10000) {
      return "FOFA 结果上限必须在 1-10000 之间";
    }
    if (config.fofa_concurrency < 1 || config.fofa_concurrency > 100) {
      return "FOFA 并发必须在 1-100 之间";
    }
    if (config.subfinder_timeout < 30 || config.subfinder_timeout > 3600) {
      return "Subfinder 超时必须在 30-3600 秒之间";
    }
    if (config.dns_concurrency < 1 || config.dns_concurrency > 1000) {
      return "DNS 并发必须在 1-1000 之间";
    }
    if (config.dns_timeout < 1 || config.dns_timeout > 60) {
      return "DNS 超时必须在 1-60 秒之间";
    }
    if (config.port_scan_timeout < 30 || config.port_scan_timeout > 7200) {
      return "端口扫描超时必须在 30-7200 秒之间";
    }
    if (config.port_scan_concurrency < 1 || config.port_scan_concurrency > 1000) {
      return "端口扫描并发必须在 1-1000 之间";
    }
    if (config.nerva_timeout < 1 || config.nerva_timeout > 300) {
      return "Nerva 超时必须在 1-300 秒之间";
    }
    if (config.nerva_concurrency < 1 || config.nerva_concurrency > 500) {
      return "Nerva 并发必须在 1-500 之间";
    }
    if (config.nuclei_rate_limit < 1 || config.nuclei_rate_limit > 5000) {
      return "Nuclei 速率限制必须在 1-5000 之间";
    }
    if (config.nuclei_concurrency < 1 || config.nuclei_concurrency > 200) {
      return "Nuclei 并发必须在 1-200 之间";
    }
    return null;
  };

  const handleSave = async () => {
    if (!projectId || saving) return;
    const validationError = validate();
    if (validationError) {
      toast(validationError, "error");
      return;
    }
    setSaving(true);
    try {
      await api.updatePipelineConfig(projectId, config);
      toast("扫描配置已保存", "success");
      setOriginalConfig(config);
    } catch (err) {
      const msg = err instanceof Error ? err.message : "保存失败";
      toast(msg, "error");
    } finally {
      setSaving(false);
    }
  };

  const handleReset = () => {
    setConfig(DEFAULT_PIPELINE_CONFIG);
    if (isPresetPort(DEFAULT_PIPELINE_CONFIG.port_range)) {
      setPortMode(DEFAULT_PIPELINE_CONFIG.port_range);
      setCustomPorts("");
    } else {
      setPortMode("custom");
      setCustomPorts(DEFAULT_PIPELINE_CONFIG.port_range);
    }
    toast("已重置为默认配置（尚未保存）", "warning");
  };

  const isDirty = JSON.stringify(config) !== JSON.stringify(originalConfig);

  if (!projectId) {
    return (
      <div className="max-w-4xl">
        <h1 className="text-2xl font-bold text-zinc-100 tracking-tight">扫描配置</h1>
        <p className="text-sm text-text-tertiary mt-2">请先选择一个项目</p>
      </div>
    );
  }

  if (loading) {
    return (
      <div className="max-w-4xl space-y-6">
        <h1 className="text-2xl font-bold text-zinc-100 tracking-tight">扫描配置</h1>
        <Skeleton className="h-32" />
        <Skeleton className="h-48" />
        <Skeleton className="h-48" />
      </div>
    );
  }

  if (error) {
    return (
      <div className="max-w-4xl space-y-4">
        <h1 className="text-2xl font-bold text-zinc-100 tracking-tight">扫描配置</h1>
        <div className="p-4 rounded-lg bg-brand-danger/10 border border-brand-danger/20 text-brand-danger text-sm">
          加载失败：{error}
        </div>
        <Button variant="secondary" onClick={() => load()}>
          重试
        </Button>
      </div>
    );
  }

  const portPresetOptions = PORT_RANGE_PRESETS.map((p) => ({ value: p.value, label: p.label }));
  const selectedPreset = PORT_RANGE_PRESETS.find((p) => p.value === portMode);

  return (
    <div className="max-w-4xl space-y-6 pb-12">
      <div className="flex items-start justify-between">
        <div>
          <h1 className="text-2xl font-bold text-zinc-100 tracking-tight">扫描配置</h1>
          <p className="text-sm text-text-tertiary mt-1">
            调整流水线各阶段的开关、端口范围、并发与超时
          </p>
        </div>
        <div className="flex gap-2">
          <Button variant="secondary" size="sm" onClick={handleReset} disabled={saving}>
            重置默认
          </Button>
          <Button
            variant="primary"
            size="sm"
            onClick={handleSave}
            disabled={!isDirty || saving}
          >
            {saving ? "保存中..." : isDirty ? "保存配置" : "已保存"}
          </Button>
        </div>
      </div>

      {/* Port range — most important */}
      <Card hover={false} padding="lg">
        <CardHeader>
          <div>
            <CardTitle>端口范围</CardTitle>
            <CardDescription>
              选择端口预设或自定义端口列表，影响 Naabu 扫描范围
            </CardDescription>
          </div>
        </CardHeader>
        <div className="space-y-4">
          <Select
            label="预设"
            value={portMode}
            options={portPresetOptions}
            onChange={handlePortModeChange}
          />
          {selectedPreset && (
            <p className="text-xs text-text-tertiary -mt-2">
              {selectedPreset.description}
            </p>
          )}
          {portMode === "custom" && (
            <Input
              label="自定义端口"
              value={customPorts}
              onChange={(e) => handleCustomPortsChange(e.target.value)}
              placeholder="例如 80,443,8080 或 1-1000"
              helperText="支持单端口、逗号分隔、范围（1-1000）"
            />
          )}
        </div>
      </Card>

      {/* Stage toggles */}
      <Card hover={false} padding="lg">
        <CardHeader>
          <div>
            <CardTitle>流水线阶段</CardTitle>
            <CardDescription>开启/关闭各扫描阶段</CardDescription>
          </div>
        </CardHeader>
        <div className="space-y-3">
          <StageToggle
            label="FOFA 资产搜集"
            description="使用 FOFA API 搜集资产（需在项目中配置 FOFA 凭证）"
            enabled={config.enable_fofa}
            onChange={(v) => updateField("enable_fofa", v)}
          />
          <StageToggle
            label="Subfinder 子域名爆破"
            description="被动子域枚举"
            enabled={config.enable_subfinder}
            onChange={(v) => updateField("enable_subfinder", v)}
          />
          <StageToggle
            label="CDN 过滤"
            description="使用 cdncheck 过滤掉 CDN/WAF IP，避免误扫"
            enabled={config.enable_cdn_filter}
            onChange={(v) => updateField("enable_cdn_filter", v)}
          />
          <StageToggle
            label="Nerva 服务指纹"
            description="对开放端口做服务/产品识别"
            enabled={config.enable_nerva}
            onChange={(v) => updateField("enable_nerva", v)}
          />
          <StageToggle
            label="Nuclei 漏洞扫描"
            description="基于模板的 POC 扫描"
            enabled={config.enable_nuclei}
            onChange={(v) => updateField("enable_nuclei", v)}
          />
        </div>
      </Card>

      {/* FOFA tuning */}
      {config.enable_fofa && (
        <Card hover={false} padding="lg">
          <CardHeader>
            <div>
              <CardTitle>FOFA 调优</CardTitle>
              <CardDescription>结果上限与并发</CardDescription>
            </div>
          </CardHeader>
          <div className="grid grid-cols-2 gap-4">
            <Input
              label="结果上限"
              type="number"
              value={config.fofa_result_limit}
              onChange={(e) => updateField("fofa_result_limit", Number(e.target.value))}
              min={1}
              max={10000}
            />
            <Input
              label="并发数"
              type="number"
              value={config.fofa_concurrency}
              onChange={(e) => updateField("fofa_concurrency", Number(e.target.value))}
              min={1}
              max={100}
            />
          </div>
        </Card>
      )}

      {/* DNS / Subfinder tuning */}
      <Card hover={false} padding="lg">
        <CardHeader>
          <div>
            <CardTitle>DNS / 子域</CardTitle>
            <CardDescription>Subfinder 超时与 DNS 解析参数</CardDescription>
          </div>
        </CardHeader>
        <div className="grid grid-cols-3 gap-4">
          <Input
            label="Subfinder 超时（秒）"
            type="number"
            value={config.subfinder_timeout}
            onChange={(e) => updateField("subfinder_timeout", Number(e.target.value))}
            min={30}
            max={3600}
            disabled={!config.enable_subfinder}
          />
          <Input
            label="DNS 并发"
            type="number"
            value={config.dns_concurrency}
            onChange={(e) => updateField("dns_concurrency", Number(e.target.value))}
            min={1}
            max={1000}
          />
          <Input
            label="DNS 超时（秒）"
            type="number"
            value={config.dns_timeout}
            onChange={(e) => updateField("dns_timeout", Number(e.target.value))}
            min={1}
            max={60}
          />
        </div>
      </Card>

      {/* Port scan tuning */}
      <Card hover={false} padding="lg">
        <CardHeader>
          <div>
            <CardTitle>端口扫描调优</CardTitle>
            <CardDescription>Naabu 超时与并发</CardDescription>
          </div>
        </CardHeader>
        <div className="grid grid-cols-2 gap-4">
          <Input
            label="超时（秒）"
            type="number"
            value={config.port_scan_timeout}
            onChange={(e) => updateField("port_scan_timeout", Number(e.target.value))}
            min={30}
            max={7200}
          />
          <Input
            label="并发数"
            type="number"
            value={config.port_scan_concurrency}
            onChange={(e) => updateField("port_scan_concurrency", Number(e.target.value))}
            min={1}
            max={1000}
          />
        </div>
      </Card>

      {/* Nerva tuning */}
      {config.enable_nerva && (
        <Card hover={false} padding="lg">
          <CardHeader>
            <div>
              <CardTitle>Nerva 调优</CardTitle>
              <CardDescription>服务指纹超时与并发</CardDescription>
            </div>
          </CardHeader>
          <div className="grid grid-cols-2 gap-4">
            <Input
              label="超时（秒）"
              type="number"
              value={config.nerva_timeout}
              onChange={(e) => updateField("nerva_timeout", Number(e.target.value))}
              min={1}
              max={300}
            />
            <Input
              label="并发数"
              type="number"
              value={config.nerva_concurrency}
              onChange={(e) => updateField("nerva_concurrency", Number(e.target.value))}
              min={1}
              max={500}
            />
          </div>
        </Card>
      )}

      {/* Nuclei tuning */}
      {config.enable_nuclei && (
        <Card hover={false} padding="lg">
          <CardHeader>
            <div>
              <CardTitle>Nuclei 调优</CardTitle>
              <CardDescription>速率限制与并发</CardDescription>
            </div>
          </CardHeader>
          <div className="grid grid-cols-2 gap-4">
            <Input
              label="速率限制（rps）"
              type="number"
              value={config.nuclei_rate_limit}
              onChange={(e) => updateField("nuclei_rate_limit", Number(e.target.value))}
              min={1}
              max={5000}
            />
            <Input
              label="并发数"
              type="number"
              value={config.nuclei_concurrency}
              onChange={(e) => updateField("nuclei_concurrency", Number(e.target.value))}
              min={1}
              max={200}
            />
          </div>
        </Card>
      )}

      {/* Sticky save bar at bottom for long page */}
      {isDirty && (
        <div className="sticky bottom-4 z-10">
          <div className="liquid-glass rounded-apple px-4 py-3 flex items-center justify-between border border-brand-primary/30">
            <span className="text-sm text-text-secondary">配置有未保存的修改</span>
            <div className="flex gap-2">
              <Button variant="secondary" size="sm" onClick={() => load()} disabled={saving}>
                丢弃
              </Button>
              <Button variant="primary" size="sm" onClick={handleSave} disabled={saving}>
                {saving ? "保存中..." : "保存配置"}
              </Button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}

interface StageToggleProps {
  label: string;
  description: string;
  enabled: boolean;
  onChange: (enabled: boolean) => void;
}

function StageToggle({ label, description, enabled, onChange }: StageToggleProps) {
  return (
    <div className="flex items-center justify-between gap-4 py-2">
      <div className="flex-1 min-w-0">
        <div className="text-sm font-medium text-text-primary">{label}</div>
        <div className="text-xs text-text-tertiary mt-0.5">{description}</div>
      </div>
      <button
        type="button"
        role="switch"
        aria-checked={enabled}
        aria-label={`${label} ${enabled ? "已开启" : "已关闭"}`}
        onClick={() => onChange(!enabled)}
        className={`relative w-10 h-5 rounded-full transition-colors shrink-0 ${
          enabled ? "bg-brand-primary" : "bg-white/[0.10]"
        }`}
      >
        <span
          className={`absolute top-0.5 w-4 h-4 bg-white rounded-full transition-transform ${
            enabled ? "translate-x-[22px]" : "translate-x-0.5"
          }`}
        />
      </button>
    </div>
  );
}
