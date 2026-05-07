import { useState } from "react";
import Modal from "./Modal";
import { Button } from "./Button";
import { Input } from "./Input";
import { PipelineConfig, DEFAULT_PIPELINE_CONFIG, PORT_RANGE_PRESETS } from "../lib/api";

export type ScanMode = "external" | "internal";

const SCAN_CONFIG_STORAGE_KEY = "anchor.scanModal.config";
const SCAN_MODE_STORAGE_KEY = "anchor.scanModal.mode";

function loadStoredConfig(): PipelineConfig {
  try {
    const raw = localStorage.getItem(SCAN_CONFIG_STORAGE_KEY);
    if (!raw) return { ...DEFAULT_PIPELINE_CONFIG };
    const parsed = JSON.parse(raw) as Partial<PipelineConfig>;
    return { ...DEFAULT_PIPELINE_CONFIG, ...parsed };
  } catch {
    return { ...DEFAULT_PIPELINE_CONFIG };
  }
}

function loadStoredMode(): ScanMode {
  const raw = localStorage.getItem(SCAN_MODE_STORAGE_KEY);
  return raw === "internal" ? "internal" : "external";
}

interface ScanModalProps {
  open: boolean;
  onClose: () => void;
  onStart: (mode: ScanMode, config: PipelineConfig) => void;
  loading?: boolean;
}

const MODE_OPTIONS: {
  mode: ScanMode;
  label: string;
  description: string;
  tools: string[];
  icon: string;
}[] = [
  {
    mode: "external",
    label: "外网扫描",
    description: "面向互联网的资产发现与漏洞检测",
    tools: ["FOFA", "Subfinder", "DNSx", "CDNCheck", "Naabu", "Nerva", "HTTPX", "Nuclei"],
    icon: "🌐",
  },
  {
    mode: "internal",
    label: "内网扫描",
    description: "内网资产端口、服务与漏洞检测",
    tools: ["Naabu", "Nerva", "HTTPX", "Nuclei"],
    icon: "🏠",
  },
];

interface ToolField {
  key: keyof PipelineConfig;
  label: string;
  unit: string;
  min: number;
  max: number;
  recommended: number;
}

const BASE_TOOL_FIELDS: { group: string; fields: ToolField[] }[] = [
  {
    group: "Naabu",
    fields: [
      { key: "naabu_rate", label: "发包速率", unit: "pps", min: 100, max: 10000, recommended: 1000 },
      { key: "naabu_threads", label: "并发线程", unit: "", min: 10, max: 500, recommended: 100 },
      { key: "naabu_timeout", label: "超时", unit: "秒", min: 30, max: 3600, recommended: 600 },
    ],
  },
  {
    group: "Nerva",
    fields: [
      { key: "nerva_rate_limit", label: "速率限制", unit: "rps", min: 10, max: 1000, recommended: 100 },
      { key: "nerva_workers", label: "工作进程", unit: "", min: 10, max: 200, recommended: 50 },
      { key: "nerva_timeout", label: "超时", unit: "秒", min: 1, max: 60, recommended: 10 },
    ],
  },
  {
    group: "HTTPX",
    fields: [
      { key: "httpx_rate_limit", label: "速率限制", unit: "rps", min: 50, max: 1000, recommended: 150 },
      { key: "httpx_threads", label: "并发线程", unit: "", min: 10, max: 200, recommended: 50 },
    ],
  },
  {
    group: "Nuclei",
    fields: [
      { key: "nuclei_rate_limit", label: "速率限制", unit: "rps", min: 50, max: 1000, recommended: 100 },
      { key: "nuclei_rate_limit_per_min", label: "分钟限速", unit: "rpm", min: 0, max: 600, recommended: 0 },
      { key: "nuclei_concurrency", label: "并发数", unit: "", min: 1, max: 100, recommended: 25 },
    ],
  },
];

const EXTERNAL_TOOL_FIELDS: { group: string; fields: ToolField[] }[] = [
  {
    group: "Subfinder",
    fields: [
      { key: "subfinder_rate_limit", label: "速率限制", unit: "rps", min: 10, max: 500, recommended: 50 },
      { key: "subfinder_threads", label: "并发线程", unit: "", min: 5, max: 100, recommended: 10 },
      { key: "subfinder_timeout", label: "超时", unit: "秒", min: 60, max: 1800, recommended: 300 },
    ],
  },
  {
    group: "DNSx",
    fields: [
      { key: "dnsx_rate_limit", label: "速率限制", unit: "rps", min: 50, max: 1000, recommended: 100 },
      { key: "dnsx_threads", label: "并发线程", unit: "", min: 10, max: 200, recommended: 50 },
      { key: "dnsx_timeout", label: "超时", unit: "秒", min: 1, max: 30, recommended: 5 },
    ],
  },
  ...BASE_TOOL_FIELDS,
];

const INTERNAL_TOOL_FIELDS = BASE_TOOL_FIELDS;

const SCAN_DEPTH_OPTIONS: {
  value: string;
  label: string;
  description: string;
}[] = [
  {
    value: "workflow",
    label: "精确扫描",
    description: "Workflow 驱动，先指纹再检测，误报低",
  },
  {
    value: "tags",
    label: "广度扫描",
    description: "Tag 匹配模板，覆盖面广（默认）",
  },
  {
    value: "both",
    label: "综合扫描",
    description: "Workflow + Tag 双重检测，耗时较长",
  },
];

export default function ScanModal({ open, onClose, onStart, loading }: ScanModalProps) {
  const [step, setStep] = useState<1 | 2>(1);
  const [mode, setMode] = useState<ScanMode>("external");
  const [config, setConfig] = useState<PipelineConfig>({ ...DEFAULT_PIPELINE_CONFIG });

  const handleReset = () => {
    setStep(1);
    setMode("external");
    setConfig({ ...DEFAULT_PIPELINE_CONFIG });
  };

  const handleClose = () => {
    handleReset();
    onClose();
  };

  const handleSelectMode = (m: ScanMode) => {
    setMode(m);
  };

  const handleNext = () => {
    setStep(2);
  };

  const handleStart = () => {
    onStart(mode, config);
  };

  const handleResetDefaults = () => {
    setConfig({ ...DEFAULT_PIPELINE_CONFIG });
  };

  const updateConfig = (key: keyof PipelineConfig, value: string) => {
    const num = parseInt(value, 10);
    if (isNaN(num)) return;
    setConfig((prev) => ({ ...prev, [key]: num }));
  };

  const toolFields = mode === "external" ? EXTERNAL_TOOL_FIELDS : INTERNAL_TOOL_FIELDS;

  return (
    <Modal open={open} onClose={handleClose} title="新建扫描" size="lg">
      {/* Step indicator */}
      <div className="flex items-center gap-2 mb-5">
        <div className={`flex-1 h-1 rounded-full ${step >= 1 ? "bg-brand-primary" : "bg-white/[0.08]"}`} />
        <div className={`flex-1 h-1 rounded-full ${step >= 2 ? "bg-brand-primary" : "bg-white/[0.08]"}`} />
      </div>

      {step === 1 && (
        <div className="space-y-4">
          <p className="text-sm text-text-tertiary">选择扫描场景</p>
          <div className="grid grid-cols-2 gap-3">
            {MODE_OPTIONS.map((opt) => (
              <button
                key={opt.mode}
                onClick={() => handleSelectMode(opt.mode)}
                disabled={loading}
                className={`text-left p-4 rounded-lg border transition-all duration-200 group disabled:opacity-50 ${
                  mode === opt.mode
                    ? "border-brand-primary/40 bg-brand-primary/[0.06] ring-1 ring-brand-primary/20"
                    : "border-white/[0.08] bg-white/[0.02] hover:bg-white/[0.05] hover:border-white/[0.15]"
                }`}
              >
                <div className="flex items-center gap-2 mb-2">
                  <span className="text-lg">{opt.icon}</span>
                  <span className="font-medium text-text-primary text-sm">{opt.label}</span>
                </div>
                <p className="text-xs text-text-secondary leading-relaxed mb-3">
                  {opt.description}
                </p>
                <div className="flex flex-wrap gap-1">
                  {opt.tools.map((t) => (
                    <span
                      key={t}
                      className="text-[10px] px-1.5 py-0.5 rounded bg-white/[0.06] text-text-tertiary"
                    >
                      {t}
                    </span>
                  ))}
                </div>
              </button>
            ))}
          </div>
          <div className="flex justify-end gap-2 pt-2">
            <Button variant="ghost" onClick={handleClose} disabled={loading}>
              取消
            </Button>
            <Button onClick={handleNext} disabled={!mode || loading}>
              下一步
            </Button>
          </div>
        </div>
      )}

      {step === 2 && (
        <div className="space-y-4 max-h-[60vh] overflow-y-auto pr-1">
          {/* Port Range */}
          <div className="cyber-glass p-4 rounded-lg">
            <div className="text-sm font-medium text-text-primary mb-2">端口范围</div>
            <div className="grid grid-cols-2 gap-2">
              {PORT_RANGE_PRESETS.map((preset) => (
                <button
                  key={preset.value}
                  onClick={() => setConfig((prev) => ({ ...prev, port_range: preset.value }))}
                  className={`min-w-0 text-left p-2.5 rounded-lg border text-xs transition-colors ${
                    config.port_range === preset.value
                      ? "border-brand-primary/40 bg-brand-primary/[0.06] ring-1 ring-brand-primary/20"
                      : "border-white/[0.08] bg-white/[0.02] hover:bg-white/[0.04]"
                  }`}
                >
                  <div className="font-medium text-text-primary break-words">{preset.label}</div>
                  <div className="text-text-tertiary mt-0.5 break-words">{preset.description}</div>
                </button>
              ))}
            </div>
          </div>

          {/* Nuclei Scan Depth */}
          <div className="cyber-glass p-4 rounded-lg">
            <div className="text-sm font-medium text-text-primary mb-2">Nuclei 扫描策略</div>
            <div className="grid grid-cols-3 gap-2">
              {SCAN_DEPTH_OPTIONS.map((opt) => (
                <button
                  key={opt.value}
                  onClick={() => setConfig((prev) => ({ ...prev, nuclei_scan_depth: opt.value }))}
                  className={`min-w-0 text-left p-2.5 rounded-lg border text-xs transition-colors ${
                    config.nuclei_scan_depth === opt.value
                      ? "border-brand-primary/40 bg-brand-primary/[0.06] ring-1 ring-brand-primary/20"
                      : "border-white/[0.08] bg-white/[0.02] hover:bg-white/[0.04]"
                  }`}
                >
                  <div className="font-medium text-text-primary break-words">{opt.label}</div>
                  <div className="text-text-tertiary mt-0.5 break-words">{opt.description}</div>
                </button>
              ))}
            </div>
          </div>

          {/* Tool Speed Panels */}
          {toolFields.map((group) => (
            <div key={group.group} className="cyber-glass p-4 rounded-lg">
              <div className="text-sm font-medium text-text-primary mb-3">{group.group}</div>
              <div className="grid grid-cols-3 gap-3">
                {group.fields.map((field) => (
                  <div key={field.key}>
                    <label className="block text-xs text-text-secondary mb-1">
                      {field.label}
                      {field.unit && <span className="text-text-tertiary ml-0.5">({field.unit})</span>}
                    </label>
                    <Input
                      type="number"
                      min={field.min}
                      max={field.max}
                      value={config[field.key] as number}
                      onChange={(e) => updateConfig(field.key, e.target.value)}
                      className="w-full"
                    />
                    <div className="text-[10px] text-text-quaternary mt-0.5">
                      推荐: {field.recommended} · 范围: {field.min}-{field.max}
                    </div>
                  </div>
                ))}
              </div>
            </div>
          ))}

          <div className="flex items-center justify-between pt-2">
            <button
              onClick={handleResetDefaults}
              className="text-xs text-brand-primary hover:text-brand-secondary transition-colors"
              type="button"
            >
              使用推荐值
            </button>
            <div className="flex gap-2">
              <Button variant="ghost" onClick={() => setStep(1)} disabled={loading}>
                上一步
              </Button>
              <Button onClick={handleStart} disabled={loading}>
                {loading ? "启动中..." : "开始扫描"}
              </Button>
            </div>
          </div>
        </div>
      )}
    </Modal>
  );
}
