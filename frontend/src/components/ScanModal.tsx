import { useState } from "react";
import Modal from "./Modal";
import { Button } from "./Button";
import { Input } from "./Input";
import { PipelineConfig, DEFAULT_PIPELINE_CONFIG, PORT_RANGE_PRESETS } from "../lib/api";
import { cn } from "../lib/utils";
import { Zap, Globe, Shield, Gauge, Cpu, CheckCircle2, RotateCcw, ChevronRight, Search } from "lucide-react";

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
  icon: any;
  color: string;
}[] = [
  {
    mode: "external",
    label: "外网扫描",
    description: "面向互联网的资产发现与漏洞检测",
    tools: ["FOFA", "Subfinder", "DNSx", "CDNCheck", "Naabu", "nmap -sV", "HTTPX", "Nuclei"],
    icon: Globe,
    color: "text-blue-400",
  },
  {
    mode: "internal",
    label: "内网扫描",
    description: "内网资产端口、服务与漏洞检测",
    tools: ["Naabu", "Nerva", "HTTPX", "Nuclei"],
    icon: Shield,
    color: "text-emerald-400",
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

const BASE_TOOL_FIELDS: { group: string; fields: ToolField[]; icon: any }[] = [
  {
    group: "Naabu",
    icon: Zap,
    fields: [
      { key: "naabu_rate", label: "发包速率", unit: "pps", min: 100, max: 10000, recommended: 1000 },
      { key: "naabu_threads", label: "并发线程", unit: "", min: 10, max: 500, recommended: 100 },
      { key: "naabu_timeout", label: "超时", unit: "秒", min: 30, max: 3600, recommended: 600 },
    ],
  },
  {
    group: "Nerva",
    icon: Cpu,
    fields: [
      { key: "nerva_rate_limit", label: "速率限制", unit: "rps", min: 10, max: 1000, recommended: 100 },
      { key: "nerva_workers", label: "工作进程", unit: "", min: 10, max: 200, recommended: 50 },
      { key: "nerva_timeout", label: "超时", unit: "秒", min: 1, max: 60, recommended: 10 },
    ],
  },
  {
    group: "HTTPX",
    icon: Globe,
    fields: [
      { key: "httpx_rate_limit", label: "速率限制", unit: "rps", min: 50, max: 1000, recommended: 150 },
      { key: "httpx_threads", label: "并发线程", unit: "", min: 10, max: 200, recommended: 50 },
    ],
  },
  {
    group: "Nuclei",
    icon: Shield,
    fields: [
      { key: "nuclei_rate_limit", label: "速率限制", unit: "rps", min: 50, max: 1000, recommended: 100 },
      { key: "nuclei_rate_limit_per_min", label: "分钟限速", unit: "rpm", min: 0, max: 600, recommended: 0 },
      { key: "nuclei_concurrency", label: "并发数", unit: "", min: 1, max: 100, recommended: 25 },
    ],
  },
];

const EXTERNAL_TOOL_FIELDS: { group: string; fields: ToolField[]; icon: any }[] = [
  {
    group: "Subfinder",
    icon: Search,
    fields: [
      { key: "subfinder_rate_limit", label: "速率限制", unit: "rps", min: 10, max: 500, recommended: 50 },
      { key: "subfinder_threads", label: "并发线程", unit: "", min: 5, max: 100, recommended: 10 },
      { key: "subfinder_timeout", label: "超时", unit: "秒", min: 60, max: 1800, recommended: 300 },
    ],
  },
  {
    group: "DNSx",
    icon: Globe,
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
    description: "Workflow 驱动，先指纹再检测",
  },
  {
    value: "tags",
    label: "广度扫描",
    description: "Tag 匹配，覆盖面广 (默认)",
  },
  {
    value: "both",
    label: "综合扫描",
    description: "Workflow + Tag 双重检测",
  },
];

export default function ScanModal({ open, onClose, onStart, loading }: ScanModalProps) {
  const [step, setStep] = useState<1 | 2>(1);
  const [mode, setMode] = useState<ScanMode>(() => loadStoredMode());
  const [config, setConfig] = useState<PipelineConfig>(() => loadStoredConfig());

  const handleReset = () => {
    setStep(1);
  };

  const handleClose = () => {
    handleReset();
    onClose();
  };

  const handleSelectMode = (m: ScanMode) => {
    setMode(m);
    localStorage.setItem(SCAN_MODE_STORAGE_KEY, m);
  };

  const handleNext = () => {
    setStep(2);
  };

  const handleStart = () => {
    localStorage.setItem(SCAN_CONFIG_STORAGE_KEY, JSON.stringify(config));
    localStorage.setItem(SCAN_MODE_STORAGE_KEY, mode);
    onStart(mode, config);
  };

  const handleResetDefaults = () => {
    setConfig({ ...DEFAULT_PIPELINE_CONFIG });
    localStorage.removeItem(SCAN_CONFIG_STORAGE_KEY);
  };

  const updateConfig = (key: keyof PipelineConfig, value: string) => {
    const num = parseInt(value, 10);
    if (isNaN(num)) return;
    setConfig((prev) => ({ ...prev, [key]: num }));
  };

  const toolFields = mode === "external" ? EXTERNAL_TOOL_FIELDS : INTERNAL_TOOL_FIELDS;

  return (
    <Modal open={open} onClose={handleClose} title="新建扫描流水线" size="lg">
      {/* 步骤指示器 */}
      <div className="flex items-center gap-3 mb-8 px-1">
        {[1, 2].map((s) => (
            <div key={s} className="flex-1 flex items-center gap-2">
                <div className={cn(
                    "h-6 w-6 rounded-full flex items-center justify-center text-[10px] font-black border transition-all duration-500",
                    step >= s ? "bg-primary border-primary text-slate-950 shadow-[0_0_15px_rgba(0,212,255,0.3)]" : "bg-white/5 border-white/10 text-muted-foreground"
                )}>
                    {step > s ? <CheckCircle2 className="h-3 w-3" /> : s}
                </div>
                <div className={cn(
                    "flex-1 h-0.5 rounded-full transition-all duration-500",
                    step > s ? "bg-primary" : "bg-white/5"
                )} />
            </div>
        ))}
      </div>

      {step === 1 && (
        <div className="space-y-6 animate-in fade-in slide-in-from-right-4 duration-300">
          <div>
              <p className="text-xs font-black uppercase tracking-[0.2em] text-muted-foreground/60 mb-4">选择执行环境</p>
              <div className="grid grid-cols-2 gap-4">
                {MODE_OPTIONS.map((opt) => (
                  <button
                    key={opt.mode}
                    onClick={() => handleSelectMode(opt.mode)}
                    disabled={loading}
                    className={cn(
                        "text-left p-5 rounded-2xl border transition-all duration-300 group disabled:opacity-50 relative overflow-hidden",
                        mode === opt.mode
                            ? "border-primary/40 bg-primary/5 ring-1 ring-primary/20 shadow-xl"
                            : "border-white/5 bg-white/[0.02] hover:bg-white/[0.05] hover:border-white/10"
                    )}
                  >
                    <div className={cn(
                        "h-10 w-10 rounded-xl flex items-center justify-center border mb-4 transition-colors",
                        mode === opt.mode ? "bg-primary/20 border-primary/30 text-primary" : "bg-white/5 border-white/5 text-muted-foreground"
                    )}>
                      <opt.icon className="h-5 w-5" />
                    </div>
                    <div className="font-bold text-foreground mb-1">{opt.label}</div>
                    <p className="text-xs text-muted-foreground leading-relaxed mb-4 line-clamp-2">
                      {opt.description}
                    </p>
                    <div className="flex flex-wrap gap-1.5">
                      {opt.tools.slice(0, 4).map((t) => (
                        <span key={t} className="text-[9px] font-bold px-1.5 py-0.5 rounded bg-white/5 text-muted-foreground/70 uppercase">
                          {t}
                        </span>
                      ))}
                      {opt.tools.length > 4 && <span className="text-[9px] font-bold text-muted-foreground/40">+{opt.tools.length - 4} More</span>}
                    </div>
                    {mode === opt.mode && (
                        <div className="absolute top-2 right-2 h-2 w-2 rounded-full bg-primary shadow-[0_0_8px_rgba(0,212,255,0.8)]" />
                    )}
                  </button>
                ))}
              </div>
          </div>
          <div className="flex justify-end gap-3 pt-4 border-t border-white/5">
            <Button variant="ghost" onClick={handleClose} disabled={loading} className="rounded-xl">
              取消
            </Button>
            <Button onClick={handleNext} disabled={!mode || loading} className="rounded-xl px-8 font-bold">
              配置参数
              <ChevronRight className="ml-2 h-4 w-4" />
            </Button>
          </div>
        </div>
      )}

      {step === 2 && (
        <div className="space-y-6 max-h-[65vh] overflow-y-auto pr-2 custom-scrollbar animate-in fade-in slide-in-from-left-4 duration-300">
          {/* 端口与策略 */}
          <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
              <div className="space-y-3">
                 <label className="text-[10px] font-black uppercase tracking-widest text-muted-foreground/60 flex items-center gap-2">
                    <Zap className="h-3 w-3" />
                    端口探测范围
                 </label>
                 <div className="grid grid-cols-1 gap-2">
                    {PORT_RANGE_PRESETS.slice(0, 4).map((preset) => (
                        <button
                        key={preset.value}
                        onClick={() => setConfig((prev) => ({ ...prev, port_range: preset.value }))}
                        className={cn(
                            "text-left p-3 rounded-xl border text-xs transition-all duration-200",
                            config.port_range === preset.value
                                ? "border-primary/40 bg-primary/5 ring-1 ring-primary/20"
                                : "border-white/5 bg-white/[0.02] hover:bg-white/[0.04]"
                        )}
                        >
                        <div className="font-bold text-foreground">{preset.label}</div>
                        <div className="text-[10px] text-muted-foreground mt-0.5">{preset.description}</div>
                        </button>
                    ))}
                 </div>
              </div>

              <div className="space-y-3">
                 <label className="text-[10px] font-black uppercase tracking-widest text-muted-foreground/60 flex items-center gap-2">
                    <Shield className="h-3 w-3" />
                    Nuclei 扫描策略
                 </label>
                 <div className="grid grid-cols-1 gap-2">
                    {SCAN_DEPTH_OPTIONS.map((opt) => (
                        <button
                        key={opt.value}
                        onClick={() => setConfig((prev) => ({ ...prev, nuclei_scan_depth: opt.value }))}
                        className={cn(
                            "text-left p-3 rounded-xl border text-xs transition-all duration-200",
                            config.nuclei_scan_depth === opt.value
                                ? "border-primary/40 bg-primary/5 ring-1 ring-primary/20"
                                : "border-white/5 bg-white/[0.02] hover:bg-white/[0.04]"
                        )}
                        >
                        <div className="font-bold text-foreground">{opt.label}</div>
                        <div className="text-[10px] text-muted-foreground mt-0.5">{opt.description}</div>
                        </button>
                    ))}
                 </div>
              </div>
          </div>

          {/* 性能调优 */}
          <div className="space-y-4">
             <label className="text-[10px] font-black uppercase tracking-widest text-muted-foreground/60 flex items-center gap-2">
                <Gauge className="h-3 w-3" />
                性能与并发调优
             </label>
             <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                {toolFields.map((group) => (
                    <div key={group.group} className="p-4 rounded-2xl border border-white/5 bg-white/[0.01] space-y-4">
                        <div className="flex items-center gap-2 mb-1">
                            <group.icon className="h-3.5 w-3.5 text-primary" />
                            <span className="text-xs font-bold uppercase tracking-wider">{group.group}</span>
                        </div>
                        <div className="grid grid-cols-1 gap-3">
                            {group.fields.map((field) => (
                            <div key={field.key} className="space-y-1.5">
                                <div className="flex justify-between items-center">
                                    <label className="text-[10px] font-medium text-muted-foreground">
                                        {field.label}
                                    </label>
                                    <span className="text-[9px] text-muted-foreground/40 font-mono italic">
                                        REC: {field.recommended}{field.unit}
                                    </span>
                                </div>
                                <div className="relative">
                                    <Input
                                        type="number"
                                        min={field.min}
                                        max={field.max}
                                        value={config[field.key] as number}
                                        onChange={(e) => updateConfig(field.key, e.target.value)}
                                        className="h-8 bg-white/5 border-white/5 text-xs focus-visible:ring-primary/30"
                                    />
                                    {field.unit && <span className="absolute right-2 top-1.5 text-[9px] font-bold text-muted-foreground/30">{field.unit}</span>}
                                </div>
                            </div>
                            ))}
                        </div>
                    </div>
                ))}
             </div>
          </div>

          <div className="flex items-center justify-between pt-6 border-t border-white/5">
            <button
              onClick={handleResetDefaults}
              className="text-[10px] font-black uppercase tracking-widest text-muted-foreground hover:text-primary transition-colors flex items-center gap-1.5"
              type="button"
            >
              <RotateCcw className="h-3 w-3" />
              恢复出厂默认值
            </button>
            <div className="flex gap-3">
              <Button variant="ghost" onClick={() => setStep(1)} disabled={loading} className="rounded-xl">
                上一步
              </Button>
              <Button onClick={handleStart} loading={loading} className="rounded-xl px-8 font-black shadow-lg shadow-primary/20">
                立即启动扫描
              </Button>
            </div>
          </div>
        </div>
      )}
    </Modal>
  );
}
