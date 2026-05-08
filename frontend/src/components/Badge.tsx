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
  | "critical";

const variantStyles: Record<BadgeVariant, string> = {
  default: "border-transparent bg-primary text-primary-foreground hover:bg-primary/80",
  secondary: "border-transparent bg-secondary text-secondary-foreground hover:bg-secondary/80",
  outline: "text-foreground border-border hover:bg-accent",
  destructive: "border-transparent bg-destructive text-destructive-foreground hover:bg-destructive/80",
  success: "border-transparent bg-brand-success/15 text-brand-success",
  warning: "border-transparent bg-brand-warning/15 text-brand-warning",
  info: "border-transparent bg-brand-info/15 text-brand-info",
  critical: "border-transparent bg-destructive text-destructive-foreground animate-pulse",
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
        "inline-flex items-center rounded-full border px-2.5 py-0.5 text-xs font-semibold transition-colors focus:outline-none focus:ring-2 focus:ring-ring focus:ring-offset-2",
        variantStyles[variant],
        className
      )}
      {...props}
    >
      {dot && (
        <span
          className={cn(
            "mr-1.5 h-1.5 w-1.5 rounded-full bg-current"
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
      {severity.toUpperCase()}
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
    ignored: "outline",
    active: "success",
    expired: "destructive",
    pending: "secondary",
    running: "info",
    failed: "destructive",
    completed: "success",
  };
  const label: Record<string, string> = {
    confirmed: "已确认",
    accepted_risk: "已接受风险",
    pending_review: "待审核",
    false_positive: "误报",
    ignored: "已忽略",
    active: "进行中",
    expired: "已过期",
    pending: "未开始",
    running: "运行中",
    failed: "失败",
    completed: "已完成",
  };
  return (
    <Badge variant={map[status] || "outline"} className={className} dot={status === 'running'}>
      {label[status] || status}
    </Badge>
  );
}
