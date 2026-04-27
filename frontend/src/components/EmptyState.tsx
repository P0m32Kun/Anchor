import React from "react";
import { Button } from "./Button";

interface EmptyStateProps {
  icon?: React.ReactNode;
  title: string;
  description?: string;
  actionLabel?: string;
  onAction?: () => void;
}

export function EmptyState({
  icon,
  title,
  description,
  actionLabel,
  onAction,
}: EmptyStateProps) {
  return (
    <div className="flex flex-col items-center justify-center py-16 text-center animate-fade-in">
      {icon ? (
        <div className="w-12 h-12 rounded-apple bg-white/[0.06] flex items-center justify-center text-text-quaternary mb-4 border border-white/[0.06]">
          {icon}
        </div>
      ) : (
        <svg
          className="w-12 h-12 text-text-quaternary mb-4"
          viewBox="0 0 24 24"
          fill="none"
          stroke="currentColor"
          strokeWidth="1.5"
          strokeLinecap="round"
          strokeLinejoin="round"
        >
          <rect x="3" y="3" width="18" height="18" rx="2" />
          <path d="M3 9h18" />
          <path d="M9 21V9" />
        </svg>
      )}
      <h3 className="text-sm font-semibold text-text-secondary mb-1">{title}</h3>
      {description && (
        <p className="text-sm text-text-tertiary max-w-xs mb-4">{description}</p>
      )}
      {actionLabel && onAction && (
        <Button variant="primary" size="sm" onClick={onAction}>
          {actionLabel}
        </Button>
      )}
    </div>
  );
}
