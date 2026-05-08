import React from "react";
import { cn } from "../lib/utils";

type BadgeVariant =
  | "default"
  | "secondary"
  | "outline"
  | "destructive"
  | "success"
  | "warning"
  | "info"
  | "critical"
  | "purple";

const variantStyles: Record<BadgeVariant, string> = {
  default: "bg-primary/20 text-primary border-primary/30",
  secondary: "bg-slate-500/10 text-slate-400 border-slate-500/20",
  outline: "text-foreground border-white/10 bg-white/5",
  destructive: "bg-rose-500/20 text-rose-400 border-rose-500/30",
  success: "bg-emerald-500/20 text-emerald-400 border-emerald-500/30",
  warning: "bg-amber-500/20 text-amber-400 border-amber-500/30",
  info: "bg-cyan-500/20 text-cyan-400 border-cyan-500/30",
  purple: "bg-violet-500/20 text-violet-400 border-violet-500/30",
  critical: "bg-rose-600 text-white border-rose-400 shadow-[0_0_15px_rgba(225,29,72,0.4)] animate-pulse",
};

interface BadgeProps extends React.HTMLAttributes<HTMLSpanElement> {
  variant?: BadgeVariant;
  dot?: boolean;
}

export function Badge({
  variant = "default",
  dot = false,
  children,
  className = "",
  ...props
}: BadgeProps) {
  return (
    <span
      className={cn(
        "inline-flex items-center rounded-full border px-2.5 py-0.5 text-[10px] font-bold uppercase tracking-wider transition-all",
        variantStyles[variant],
        className
      )}
      {...props}
    >
      {dot && (
        <span
          className={cn(
            "mr-1.5 h-1 w-1 rounded-full bg-current animate-pulse"
          )}
        />
      )}
      {children}
    </span>
  );
}

export function SeverityBadge({
  severity,
  className = "",
}: {
  severity: string;
  className?: string;
}) {
  const map: Record<string, BadgeVariant> = {
    critical: "critical",
    high: "destructive",
    medium: "warning",
    low: "info",
    info: "outline",
  };
  return (
    <Badge variant={map[severity] || "outline"} className={className}>
      {severity}
    </Badge>
  );
}

export function StatusBadge({
  status,
  className = "",
}: {
  status: string;
  className?: string;
}) {
  const map: Record<string, BadgeVariant> = {
    confirmed: "success",
    accepted_risk: "info",
    pending_review: "warning",
    false_positive: "outline",
    ignored: "secondary",
    active: "success",
    expired: "destructive",
    pending: "secondary",
    running: "purple",
    failed: "destructive",
    completed: "success",
  };
  const label: Record<string, string> = {
    confirmed: "已确认",
    accepted_risk: "接受风险",
    pending_review: "待审核",
    false_positive: "误报",
    ignored: "忽略",
    active: "进行中",
    expired: "已过期",
    pending: "未开始",
    running: "运行中",
    failed: "失败",
    completed: "已完成",
  };
  return (
    <Badge variant={map[status] || "outline"} className={cn("px-3 py-1", className)} dot={status === 'running'}>
      {label[status] || status}
    </Badge>
  );
}
