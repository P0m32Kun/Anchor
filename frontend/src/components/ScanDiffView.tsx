import { useState, useEffect } from "react";
import { api, ScanDiffResult, Asset, Finding, PipelineRun } from "../lib/api";
import { cn } from "../lib/utils";
import { Card, CardHeader, CardTitle, CardDescription, CardContent } from "./Card";
import { Badge, SeverityBadge } from "./Badge";
import { Button } from "./Button";
import { ArrowLeft, Plus, Minus, MinusCircle, PlusCircle, Layers } from "lucide-react";

interface Props {
  projectId: string;
  baseRun: PipelineRun;
  targetRun: PipelineRun;
  onClose: () => void;
}

type TabKey = "summary" | "assets" | "findings";

const TabHeaders: { key: TabKey; label: string }[] = [
  { key: "summary", label: "对比概览" },
  { key: "assets", label: "资产变化" },
  { key: "findings", label: "漏洞变化" },
];

export function ScanDiffView({ projectId, baseRun, targetRun, onClose }: Props) {
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [diff, setDiff] = useState<ScanDiffResult | null>(null);
  const [activeTab, setActiveTab] = useState<TabKey>("summary");

  useEffect(() => {
    let cancelled = false;
    setLoading(true);
    setError(null);
    api.getScanDiff(projectId, baseRun.id, targetRun.id)
      .then((result) => {
        if (!cancelled) setDiff(result);
      })
      .catch((err) => {
        if (!cancelled) setError(err?.message || "加载对比数据失败");
      })
      .finally(() => {
        if (!cancelled) setLoading(false);
      });
    return () => { cancelled = true; };
  }, [projectId, baseRun.id, targetRun.id]);

  if (loading) {
    return (
      <Card>
        <CardContent className="p-6">
          <div className="flex items-center justify-center py-12 text-sm text-muted-foreground">
            <svg className="animate-spin -ml-1 mr-2 h-4 w-4" fill="none" viewBox="0 0 24 24">
              <circle className="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" strokeWidth="4" />
              <path className="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4z" />
            </svg>
            加载对比数据...
          </div>
        </CardContent>
      </Card>
    );
  }

  if (error) {
    return (
      <Card>
        <CardContent className="p-6">
          <div className="py-8 text-center">
            <p className="text-sm text-destructive mb-3">{error}</p>
            <Button variant="outline" size="sm" onClick={onClose}>
              <ArrowLeft className="mr-1.5 h-3.5 w-3.5" />
              返回
            </Button>
          </div>
        </CardContent>
      </Card>
    );
  }

  if (!diff) return null;

  const tabs = (
    <div className="flex gap-1 border-b border-border pb-2 mb-4">
      {TabHeaders.map((t) => (
        <button
          key={t.key}
          onClick={() => setActiveTab(t.key)}
          className={cn(
            "px-3 py-1.5 text-xs font-medium rounded-md transition-colors",
            activeTab === t.key
              ? "bg-primary/10 text-primary"
              : "text-muted-foreground hover:text-foreground hover:bg-muted/50"
          )}
        >
          {t.label}
        </button>
      ))}
    </div>
  );

  return (
    <Card className="border-primary/20">
      <CardHeader className="bg-muted/30 pb-3">
        <div className="flex items-center justify-between">
          <CardTitle className="text-sm flex items-center gap-2">
            <Layers className="h-4 w-4" />
            扫描对比
          </CardTitle>
          <Button variant="ghost" size="sm" className="h-7 text-xs" onClick={onClose}>
            <ArrowLeft className="mr-1 h-3 w-3" />
            返回
          </Button>
        </div>
        <CardDescription className="text-xs">
          基准: {baseRun.id.slice(-8).toUpperCase()} ({new Date(baseRun.created_at).toLocaleString()})
          &nbsp;→&nbsp;
          目标: {targetRun.id.slice(-8).toUpperCase()} ({new Date(targetRun.created_at).toLocaleString()})
        </CardDescription>
      </CardHeader>
      <CardContent className="p-4">
        {tabs}
        {activeTab === "summary" && <SummaryTab diff={diff} />}
        {activeTab === "assets" && <AssetsTab diff={diff} />}
        {activeTab === "findings" && <FindingsTab diff={diff} />}
      </CardContent>
    </Card>
  );
}

function SummaryTab({ diff }: { diff: ScanDiffResult }) {
  const s = diff.summary;
  return (
    <div className="space-y-4">
      <div className="grid grid-cols-2 gap-3">
        <StatCard
          icon={<PlusCircle className="h-4 w-4 text-brand-success" />}
          label="新增资产"
          value={s.assets_added}
          variant="success"
        />
        <StatCard
          icon={<MinusCircle className="h-4 w-4 text-destructive" />}
          label="消失资产"
          value={s.assets_removed}
          variant="danger"
        />
        <StatCard
          icon={<PlusCircle className="h-4 w-4 text-brand-success" />}
          label="新增漏洞"
          value={s.findings_added}
          variant="success"
        />
        <StatCard
          icon={<MinusCircle className="h-4 w-4 text-destructive" />}
          label="已修复漏洞"
          value={s.findings_removed}
          variant="danger"
        />
      </div>
      <div className="rounded-lg border border-border p-3 text-xs space-y-1.5 bg-muted/20">
        <div className="flex justify-between">
          <span className="text-muted-foreground">基准扫描</span>
          <span className="font-medium">
            {diff.base_run.id.slice(-8).toUpperCase()} — {diff.base_run.mode}
            <span className="text-muted-foreground ml-2">{new Date(diff.base_run.created_at).toLocaleString()}</span>
          </span>
        </div>
        <div className="flex justify-between">
          <span className="text-muted-foreground">目标扫描</span>
          <span className="font-medium">
            {diff.target_run.id.slice(-8).toUpperCase()} — {diff.target_run.mode}
            <span className="text-muted-foreground ml-2">{new Date(diff.target_run.created_at).toLocaleString()}</span>
          </span>
        </div>
      </div>
    </div>
  );
}

function StatCard({ icon, label, value, variant }: { icon: React.ReactNode; label: string; value: number; variant: "success" | "danger" }) {
  return (
    <div className={cn(
      "rounded-lg border p-3 flex items-center gap-3",
      variant === "success" ? "border-brand-success/20 bg-brand-success/[0.03]" : "border-destructive/20 bg-destructive/[0.03]"
    )}>
      <div className={cn(
        "h-8 w-8 rounded-full flex items-center justify-center",
        variant === "success" ? "bg-brand-success/10" : "bg-destructive/10"
      )}>
        {icon}
      </div>
      <div>
        <div className="text-xs text-muted-foreground">{label}</div>
        <div className={cn(
          "text-lg font-bold",
          variant === "success" ? "text-brand-success" : "text-destructive"
        )}>{value}</div>
      </div>
    </div>
  );
}

function AssetsTab({ diff }: { diff: ScanDiffResult }) {
  return (
    <div className="space-y-4 max-h-[500px] overflow-y-auto">
      {diff.assets.added.length > 0 && (
        <AssetList title="新增资产" items={diff.assets.added} icon={<Plus className="h-3.5 w-3.5 text-brand-success" />} variant="success" />
      )}
      {diff.assets.removed.length > 0 && (
        <AssetList title="消失资产" items={diff.assets.removed} icon={<Minus className="h-3.5 w-3.5 text-destructive" />} variant="danger" />
      )}
      {diff.assets.unchanged.length > 0 && (
        <AssetList title="未变化资产" items={diff.assets.unchanged} icon={<Layers className="h-3.5 w-3.5 text-muted-foreground" />} variant="default" />
      )}
      {diff.assets.added.length === 0 && diff.assets.removed.length === 0 && diff.assets.unchanged.length === 0 && (
        <p className="text-xs text-muted-foreground text-center py-8">暂无资产数据</p>
      )}
    </div>
  );
}

function AssetList({ title, items, icon, variant }: { title: string; items: { asset: Asset }[]; icon: React.ReactNode; variant: "success" | "danger" | "default" }) {
  return (
    <div>
      <h4 className="text-xs font-semibold text-foreground/70 mb-2 flex items-center gap-1.5">
        {icon}
        {title}
        <span className="text-muted-foreground font-normal">({items.length})</span>
      </h4>
      <div className="space-y-1">
        {items.map((item) => (
          <div key={item.asset.id} className={cn(
            "flex items-center gap-2 px-2.5 py-1.5 rounded-md text-xs border",
            variant === "success" ? "bg-brand-success/[0.03] border-brand-success/10" :
            variant === "danger" ? "bg-destructive/[0.03] border-destructive/10" :
            "bg-muted/20 border-border/50"
          )}>
            <Badge variant="secondary" className="h-4 px-1 text-[9px] uppercase">{item.asset.type}</Badge>
            <span className="font-mono text-[11px] truncate flex-1">{item.asset.value}</span>
            {item.asset.tags?.ip && (
              <span className="text-[10px] text-muted-foreground">{item.asset.tags.ip}</span>
            )}
          </div>
        ))}
      </div>
    </div>
  );
}

function FindingsTab({ diff }: { diff: ScanDiffResult }) {
  return (
    <div className="space-y-4 max-h-[500px] overflow-y-auto">
      {diff.findings.added.length > 0 && (
        <FindingsList title="新增漏洞" items={diff.findings.added} variant="danger" />
      )}
      {diff.findings.removed.length > 0 && (
        <FindingsList title="已修复漏洞" items={diff.findings.removed} variant="success" />
      )}
      {diff.findings.unchanged.length > 0 && (
        <FindingsList title="未变化漏洞" items={diff.findings.unchanged} variant="default" />
      )}
      {diff.findings.added.length === 0 && diff.findings.removed.length === 0 && diff.findings.unchanged.length === 0 && (
        <p className="text-xs text-muted-foreground text-center py-8">暂无漏洞数据</p>
      )}
    </div>
  );
}

function FindingsList({ title, items, variant }: { title: string; items: { finding: Finding }[]; icon?: React.ReactNode; variant: "success" | "danger" | "default" }) {
  return (
    <div>
      <h4 className="text-xs font-semibold text-foreground/70 mb-2 flex items-center gap-1.5">
        {variant === "danger" ? <Plus className="h-3.5 w-3.5 text-destructive" /> :
         variant === "success" ? <Minus className="h-3.5 w-3.5 text-brand-success" /> :
         <Layers className="h-3.5 w-3.5 text-muted-foreground" />}
        {title}
        <span className="text-muted-foreground font-normal">({items.length})</span>
      </h4>
      <div className="space-y-1">
        {items.map((item) => (
          <div key={item.finding.id} className={cn(
            "flex items-center gap-2 px-2.5 py-1.5 rounded-md text-xs border",
            variant === "success" ? "bg-brand-success/[0.03] border-brand-success/10" :
            variant === "danger" ? "bg-destructive/[0.03] border-destructive/10" :
            "bg-muted/20 border-border/50"
          )}>
            <SeverityBadge severity={item.finding.severity} className="h-4 text-[9px]" />
            <span className="truncate flex-1">{item.finding.title}</span>
            {item.finding.source_tool && (
              <span className="text-[10px] text-muted-foreground font-mono">{item.finding.source_tool}</span>
            )}
          </div>
        ))}
      </div>
    </div>
  );
}
