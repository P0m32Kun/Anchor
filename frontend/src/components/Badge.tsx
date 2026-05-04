import React from "react";

type BadgeVariant =
  | "default"
  | "primary"
  | "success"
  | "warning"
  | "danger"
  | "info"
  | "critical";

// Dark mode translucent badge styles
const variantStyles: Record<
  BadgeVariant,
  { bg: string; text: string; border: string; glow?: string }
> = {
  default: {
    bg: "bg-white/[0.04]",
    text: "text-text-secondary",
    border: "border-white/[0.08]",
  },
  primary: {
    bg: "bg-brand-primary/10",
    text: "text-brand-primary",
    border: "border-brand-primary/20",
    glow: "shadow-[0_0_12px_rgba(47,129,247,0.15)]",
  },
  success: {
    bg: "bg-brand-success/10",
    text: "text-brand-success",
    border: "border-brand-success/20",
    glow: "shadow-[0_0_12px_rgba(63,185,80,0.15)]",
  },
  warning: {
    bg: "bg-brand-warning/10",
    text: "text-brand-warning",
    border: "border-brand-warning/20",
    glow: "shadow-[0_0_12px_rgba(210,153,34,0.1)]",
  },
  danger: {
    bg: "bg-brand-danger/10",
    text: "text-brand-danger",
    border: "border-brand-danger/20",
    glow: "shadow-[0_0_12px_rgba(248,81,73,0.15)]",
  },
  info: {
    bg: "bg-accent-teal/10",
    text: "text-accent-teal",
    border: "border-accent-teal/20",
  },
  critical: {
    bg: "bg-brand-danger/20",
    text: "text-brand-danger",
    border: "border-brand-danger/30",
    glow: "shadow-[0_0_15px_rgba(248,81,73,0.2)]",
  },
};

interface BadgeProps extends React.HTMLAttributes<HTMLSpanElement> {
  variant?: BadgeVariant;
  size?: "sm" | "md";
  dot?: boolean;
}

export function Badge({
  variant = "default",
  size = "sm",
  dot = false,
  children,
  className = "",
  ...props
}: BadgeProps & { ref?: React.Ref<HTMLSpanElement> }) {
  const styles = variantStyles[variant];
  const sizeCls = size === "sm" ? "px-2 py-0.5 text-[11px]" : "px-2.5 py-1 text-xs";

  return (
    <span
      className={`inline-flex items-center gap-1 rounded-lg font-medium border backdrop-blur-sm ${styles.bg} ${styles.text} ${styles.border} ${styles.glow || ""} ${sizeCls} ${className}`}
      {...props}
    >
      {dot && (
        <span
          className={`inline-block w-1.5 h-1.5 rounded-full ${styles.text.replace("text-", "bg-")}`}
        />
      )}
      {children}
    </span>
  );
}

/** Severity badge — maps severity string to Badge variant */
export function SeverityBadge({
  severity,
  className = "",
}: {
  severity: string;
  className?: string;
}) {
  const map: Record<string, BadgeVariant> = {
    critical: "critical",
    high: "danger",
    medium: "warning",
    low: "info",
    info: "default",
  };
  return (
    <Badge variant={map[severity] || "default"} className={className}>
      {severity}
    </Badge>
  );
}

/** Status badge */
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
    false_positive: "default",
    ignored: "default",
    active: "success",
    expired: "danger",
    pending: "warning",
    running: "info",
    failed: "danger",
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
    <Badge variant={map[status] || "default"} className={className}>
      {label[status] || status}
    </Badge>
  );
}
