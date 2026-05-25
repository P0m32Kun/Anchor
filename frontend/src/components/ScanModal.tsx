import { useState, useEffect } from "react";
import Modal from "./Modal";
import { Button } from "./Button";
import { Input } from "./Input";
import {
  PipelineConfig,
  DEFAULT_PIPELINE_CONFIG,
  DEFAULT_EXTERNAL_PIPELINE_CONFIG,
  DEFAULT_HIGH_RISK_PORTS,
  TP_PRESET_VALUES,
  TP_PRESET_LABELS,
  TpPresetValue,
  Dictionary,
  api,
} from "../lib/api";
import { cn } from "../lib/utils";
import { Zap, Globe, Shield, Gauge, Cpu, CheckCircle2, RotateCcw, ChevronRight, Search } from "lucide-react";

export type ScanMode = "external" | "internal";

const SCAN_CONFIG_STORAGE_KEY = "anchor.scanModal.config";
const SCAN_MODE_STORAGE_KEY = "anchor.scanModal.mode";

function loadStoredConfig(mode: ScanMode): PipelineConfig {
  const base = mode === "external" ? DEFAULT_EXTERNAL_PIPELINE_CONFIG : DEFAULT_PIPELINE_CONFIG;
  try {
    const raw = localStorage.getItem(SCAN_CONFIG_STORAGE_KEY);
    if (!raw) return { ...base };
    const parsed = JSON.parse(raw) as Partial<PipelineConfig>;
    return { ...base, ...parsed };
  } catch {
    return { ...base };
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
  projectId?: string;
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
    tools: ["FOFA", "Subfinder", "DNSx", "CDNCheck", "nmap alive", "Naabu", "nmap -sV", "HTTPX", "Nuclei", "Ffuf", "URLFinder"],
    icon: Globe,
    color: "text-blue-400",
  },
  {
    mode: "internal",
    label: "内网扫描",
    description: "内网资产端口、服务与漏洞检测",
    tools: ["nmap alive", "Naabu", "nmap -sV", "HTTPX", "Nuclei", "Ffuf", "URLFinder"],
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
      { key: "naabu_timeout", label: "超时", unit: "毫秒", min: 1000, max: 60000, recommended: 5000 },
    ],
  },
  {
    group: "nmap -sV",
    icon: Cpu,
    fields: [
      { key: "nmap_service_timeout", label: "超时", unit: "秒", min: 30, max: 1800, recommended: 180 },
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
      { key: "subfinder_timeout", label: "超时", unit: "秒", min: 10, max: 600, recommended: 30 },
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

type PortMode = "tp" | "p";

// Derive UI state from the persisted port_range string. The backend (see
// internal/worker/commands.go:BuildNaabuCommand) accepts top-N presets, the
// magic "high-risk" alias, or a raw comma-separated port list — we normalize
// those into the two-mode UI here.
function decodePortRange(raw: string): {
  mode: PortMode;
  tpValue: TpPresetValue;
  pValue: string;
} {
  const normalized = raw.trim().toLowerCase();
  switch (normalized) {
    case "":
    case "top100":
    case "tp100":
    case "top-100":
      return { mode: "tp", tpValue: "top100", pValue: DEFAULT_HIGH_RISK_PORTS };
    case "top1000":
    case "tp1000":
    case "top-1000":
      return { mode: "tp", tpValue: "top1000", pValue: DEFAULT_HIGH_RISK_PORTS };
    case "full":
    case "tpfull":
    case "topfull":
    case "top-full":
      return { mode: "tp", tpValue: "full", pValue: DEFAULT_HIGH_RISK_PORTS };
    case "high-risk":
    case "highrisk":
    case "hr":
      return { mode: "p", tpValue: "top1000", pValue: DEFAULT_HIGH_RISK_PORTS };
    default:
      return { mode: "p", tpValue: "top1000", pValue: raw };
  }
}

export default function ScanModal({ open, onClose, onStart, loading, projectId }: ScanModalProps) {
  const [step, setStep] = useState<1 | 2>(1);
  const [mode, setMode] = useState<ScanMode>(() => loadStoredMode());
  const [config, setConfig] = useState<PipelineConfig>(() => loadStoredConfig(loadStoredMode()));

  // Merge project pipeline config on mount so saved settings (e.g.
  // nuclei_scan_depth) are honoured even when localStorage is stale.
  useEffect(() => {
    if (!projectId || !open) return;
    let cancelled = false;
    (async () => {
      try {
        const projectCfg = await api.getPipelineConfig(projectId);
        if (cancelled) return;
        setConfig((prev) => ({ ...prev, ...projectCfg }));
      } catch {
        // Project has no saved config — keep localStorage baseline.
      }
    })();
    return () => { cancelled = true; };
  }, [projectId, open]);

  const initialPort = decodePortRange(config.port_range);
  const [portMode, setPortMode] = useState<PortMode>(initialPort.mode);
  const [tpValue, setTpValue] = useState<TpPresetValue>(initialPort.tpValue);
  const [pValue, setPValue] = useState<string>(initialPort.pValue);

  const handlePortModeSwitch = (next: PortMode) => {
    setPortMode(next);
    setConfig((prev) => ({
      ...prev,
      port_range: next === "tp" ? tpValue : pValue,
    }));
  };

  const handleTpChange = (next: TpPresetValue) => {
    setTpValue(next);
    setConfig((prev) => ({ ...prev, port_range: next }));
  };

  const handlePChange = (next: string) => {
    setPValue(next);
    setConfig((prev) => ({ ...prev, port_range: next }));
  };

  const handleResetHighRiskPorts = () => {
    handlePChange(DEFAULT_HIGH_RISK_PORTS);
  };

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
    // When switching mode, reload config from the mode-appropriate defaults.
    setConfig(loadStoredConfig(m));
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
    const base = mode === "external" ? DEFAULT_EXTERNAL_PIPELINE_CONFIG : DEFAULT_PIPELINE_CONFIG;
    setConfig({ ...base });
    const reset = decodePortRange(base.port_range);
    setPortMode(reset.mode);
    setTpValue(reset.tpValue);
    setPValue(reset.pValue);
    localStorage.removeItem(SCAN_CONFIG_STORAGE_KEY);
  };

  const updateConfig = (key: keyof PipelineConfig, value: string) => {
    const num = parseInt(value, 10);
    if (isNaN(num)) return;
    setConfig((prev) => ({ ...prev, [key]: num }));
  };

  const toolFields = mode === "external" ? EXTERNAL_TOOL_FIELDS : INTERNAL_TOOL_FIELDS;

  const [dictionaries, setDictionaries] = useState<Dictionary[]>([]);
  useEffect(() => {
    if (config.enable_ffuf && dictionaries.length === 0) {
      api.listDictionaries("dirscan").then(setDictionaries).catch(() => {});
    }
  }, [config.enable_ffuf]);

  // Fix 3 frontend guard: ffuf without a dictionary would silently skip
  // server-side. Disable the start button and flag the dictionary select so
  // the user fixes it instead of losing ffuf coverage to a misconfiguration.
  const ffufMisconfigured = config.enable_ffuf && !config.ffuf_dictionary_id;

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

                 {/* Mode switch: -tp 预设 vs -p 自定义 */}
                 <div className="grid grid-cols-2 gap-2 p-1 rounded-xl bg-white/[0.02] border border-white/5">
                    {[
                       { value: "tp" as PortMode, label: "-tp 预设", hint: "Naabu Top-N" },
                       { value: "p" as PortMode, label: "-p 自定义", hint: "指定端口列表" },
                    ].map((opt) => (
                       <button
                          key={opt.value}
                          aria-label={`端口模式 ${opt.label}`}
                          aria-pressed={portMode === opt.value}
                          onClick={() => handlePortModeSwitch(opt.value)}
                          className={cn(
                             "px-3 py-2 rounded-lg text-xs font-bold transition-all duration-200 flex flex-col items-start gap-0.5",
                             portMode === opt.value
                                ? "bg-primary/10 text-primary ring-1 ring-primary/30"
                                : "text-muted-foreground hover:text-foreground hover:bg-white/[0.03]"
                          )}
                       >
                          <span className="font-mono">{opt.label}</span>
                          <span className="text-[9px] font-normal text-muted-foreground/60 normal-case tracking-normal">
                             {opt.hint}
                          </span>
                       </button>
                    ))}
                 </div>

                 {portMode === "tp" && (
                    <div className="space-y-2 animate-in fade-in duration-200">
                       <div className="relative">
                          <select
                             aria-label="Top-N 端口预设"
                             value={tpValue}
                             onChange={(e) => handleTpChange(e.target.value as TpPresetValue)}
                             className="w-full h-10 px-3 pr-9 rounded-xl bg-white/5 border border-white/5 text-xs font-bold text-foreground focus:outline-none focus:ring-1 focus:ring-primary/30 appearance-none"
                          >
                             {TP_PRESET_VALUES.map((v) => (
                                <option key={v} value={v} className="bg-slate-900">
                                   {TP_PRESET_LABELS[v]}
                                </option>
                             ))}
                          </select>
                          <ChevronRight className="absolute right-3 top-3 h-4 w-4 rotate-90 text-muted-foreground/50 pointer-events-none" />
                       </div>
                       <p className="text-[10px] text-muted-foreground/60 leading-relaxed">
                          Naabu 默认 Top 100 仅覆盖高频服务；Top 1000 更广但可能漏掉 Redis/ES/K8s；全端口最慢。
                       </p>
                    </div>
                 )}

                 {portMode === "p" && (
                    <div className="space-y-2 animate-in fade-in duration-200">
                       <textarea
                          aria-label="自定义端口列表"
                          value={pValue}
                          onChange={(e) => handlePChange(e.target.value)}
                          rows={4}
                          spellCheck={false}
                          placeholder="例如：80,443,8080,1-1000"
                          className="w-full px-3 py-2 rounded-xl bg-white/5 border border-white/5 text-[11px] font-mono text-foreground placeholder:text-muted-foreground/30 focus:outline-none focus:ring-1 focus:ring-primary/30 resize-y leading-relaxed"
                       />
                       <div className="flex items-center justify-between gap-2">
                          <span className="text-[10px] text-muted-foreground/60">
                             逗号分隔，支持范围如 1-1000；默认填充高危端口
                          </span>
                          <button
                             type="button"
                             onClick={handleResetHighRiskPorts}
                             className="text-[10px] font-bold text-muted-foreground hover:text-primary transition-colors flex items-center gap-1 shrink-0"
                          >
                             <RotateCcw className="h-2.5 w-2.5" />
                             恢复高危端口
                          </button>
                       </div>
                    </div>
                 )}
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

          {/* 后台慢速扫描 */}
          <div className="space-y-4">
            <label className="text-[10px] font-black uppercase tracking-widest text-muted-foreground/60 flex items-center gap-2">
              <Search className="h-3 w-3" />
              后台慢速扫描
            </label>

            {/* Toggle buttons */}
            <div className="grid grid-cols-1 gap-4">
              {/* Ffuf toggle */}
              <div className="space-y-3">
                <button
                  onClick={() => setConfig((prev) => ({ ...prev, enable_ffuf: !prev.enable_ffuf }))}
                  className={cn(
                    "w-full text-left p-3 rounded-xl border text-xs transition-all duration-200",
                    config.enable_ffuf
                      ? "border-primary/40 bg-primary/5 ring-1 ring-primary/20"
                      : "border-white/5 bg-white/[0.02] hover:bg-white/[0.04]"
                  )}
                >
                  <div className="font-bold text-foreground">Ffuf</div>
                  <div className="text-[10px] text-muted-foreground mt-0.5">目录与文件爆破</div>
                </button>

                {config.enable_ffuf && (
                  <div className="p-4 rounded-2xl border border-white/5 bg-white/[0.01] space-y-4">
                    <div className="space-y-1.5">
                      <div className="flex justify-between items-center">
                        <label className="text-[10px] font-medium text-muted-foreground">速率限制</label>
                        <span className="text-[9px] text-muted-foreground/40 font-mono italic">REC: 6rps</span>
                      </div>
                      <div className="relative">
                        <Input
                          type="number"
                          min={1}
                          max={60}
                          value={config.ffuf_rate_limit}
                          onChange={(e) => updateConfig("ffuf_rate_limit", e.target.value)}
                          className="h-8 bg-white/5 border-white/5 text-xs focus-visible:ring-primary/30"
                        />
                        <span className="absolute right-2 top-1.5 text-[9px] font-bold text-muted-foreground/30">rps</span>
                      </div>
                    </div>
                    <div className="space-y-1.5">
                      <div className="flex justify-between items-center">
                        <label className="text-[10px] font-medium text-muted-foreground">超时</label>
                        <span className="text-[9px] text-muted-foreground/40 font-mono italic">REC: 30秒</span>
                      </div>
                      <div className="relative">
                        <Input
                          type="number"
                          min={10}
                          max={300}
                          value={config.ffuf_timeout}
                          onChange={(e) => updateConfig("ffuf_timeout", e.target.value)}
                          className="h-8 bg-white/5 border-white/5 text-xs focus-visible:ring-primary/30"
                        />
                        <span className="absolute right-2 top-1.5 text-[9px] font-bold text-muted-foreground/30">秒</span>
                      </div>
                    </div>
                    {/* Dictionary selector */}
                    <div className="space-y-1.5">
                      <label className="text-[10px] font-medium text-muted-foreground">字典</label>
                      {dictionaries.length > 0 ? (
                        <select
                          value={config.ffuf_dictionary_id}
                          onChange={(e) => setConfig((prev) => ({ ...prev, ffuf_dictionary_id: e.target.value }))}
                          aria-invalid={ffufMisconfigured || undefined}
                          className={cn(
                            "w-full h-8 px-2 rounded-md bg-white/5 border text-xs text-foreground focus:outline-none focus:ring-1 focus:ring-primary/30",
                            ffufMisconfigured ? "border-destructive/60 ring-1 ring-destructive/30" : "border-white/5"
                          )}
                        >
                          <option value="" className="bg-slate-900">请选择字典</option>
                          {dictionaries.map((d) => (
                            <option key={d.id} value={d.id} className="bg-slate-900">
                              {d.name} ({d.line_count} 行)
                            </option>
                          ))}
                        </select>
                      ) : (
                        <div className="text-[10px] text-muted-foreground/50 py-1.5">请先上传字典</div>
                      )}
                      {ffufMisconfigured && (
                        <div className="text-[10px] text-destructive">请选择 Ffuf 字典,或关闭 Ffuf</div>
                      )}
                    </div>
                  </div>
                )}
              </div>

              {/* URLFinder toggle */}
              <div className="space-y-3">
                <button
                  onClick={() => setConfig((prev) => ({ ...prev, enable_urlfinder: !prev.enable_urlfinder }))}
                  className={cn(
                    "w-full text-left p-3 rounded-xl border text-xs transition-all duration-200",
                    config.enable_urlfinder
                      ? "border-primary/40 bg-primary/5 ring-1 ring-primary/20"
                      : "border-white/5 bg-white/[0.02] hover:bg-white/[0.04]"
                  )}
                >
                  <div className="font-bold text-foreground">URLFinder</div>
                  <div className="text-[10px] text-muted-foreground mt-0.5">从页面中提取 URL 与 JS 链接,扩展攻击面</div>
                </button>

                {config.enable_urlfinder && (
                  <div className="p-4 rounded-2xl border border-white/5 bg-white/[0.01] space-y-4">
                    <div className="space-y-1.5">
                      <div className="flex justify-between items-center">
                        <label className="text-[10px] font-medium text-muted-foreground">线程数</label>
                        <span className="text-[9px] text-muted-foreground/40 font-mono italic">REC: 20</span>
                      </div>
                      <div className="relative">
                        <Input
                          type="number"
                          min={1}
                          max={100}
                          value={config.urlfinder_threads}
                          onChange={(e) => updateConfig("urlfinder_threads", e.target.value)}
                          className="h-8 bg-white/5 border-white/5 text-xs focus-visible:ring-primary/30"
                        />
                        <span className="absolute right-2 top-1.5 text-[9px] font-bold text-muted-foreground/30">threads</span>
                      </div>
                    </div>
                    <div className="space-y-1.5">
                      <div className="flex justify-between items-center">
                        <label className="text-[10px] font-medium text-muted-foreground">超时</label>
                        <span className="text-[9px] text-muted-foreground/40 font-mono italic">REC: 10秒</span>
                      </div>
                      <div className="relative">
                        <Input
                          type="number"
                          min={1}
                          max={120}
                          value={config.urlfinder_timeout}
                          onChange={(e) => updateConfig("urlfinder_timeout", e.target.value)}
                          className="h-8 bg-white/5 border-white/5 text-xs focus-visible:ring-primary/30"
                        />
                        <span className="absolute right-2 top-1.5 text-[9px] font-bold text-muted-foreground/30">秒</span>
                      </div>
                    </div>
                  </div>
                )}
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
              <Button
                onClick={handleStart}
                loading={loading}
                disabled={loading || ffufMisconfigured}
                title={ffufMisconfigured ? "请选择 Ffuf 字典或关闭 Ffuf" : undefined}
                className="rounded-xl px-8 font-black shadow-lg shadow-primary/20"
              >
                立即启动扫描
              </Button>
            </div>
          </div>
        </div>
      )}
    </Modal>
  );
}
