import React, { forwardRef } from "react";

interface CardProps extends React.HTMLAttributes<HTMLDivElement> {
  hover?: boolean;
  padding?: "sm" | "md" | "lg";
  glass?: boolean;
}

export const Card = forwardRef<HTMLDivElement, CardProps>(
  function Card(
    { hover = true, padding = "md", glass = true, children, className = "", ...props },
    ref
  ) {
    const paddings = {
      sm: "p-4",
      md: "p-5",
      lg: "p-6",
    };

    const baseClass = glass
      ? "vision-glass"
      : "bg-surface-elevated rounded-apple-xl border border-glass-border border-t-white/[0.05]";

    const hoverCls = hover
      ? "vision-glass-hover cursor-pointer"
      : "";

    return (
      <div
        ref={ref}
        className={`${baseClass} ${paddings[padding]} ${hoverCls} ${className}`}
        {...props}
      >
        {children}
      </div>
    );
  }
);

export const CardHeader = forwardRef<HTMLDivElement, React.HTMLAttributes<HTMLDivElement>>(
  function CardHeader({ children, className = "", ...props }, ref) {
    return (
      <div ref={ref} className={`flex items-center justify-between mb-4 ${className}`} {...props}>
        {children}
      </div>
    );
  }
);

export const CardTitle = forwardRef<HTMLHeadingElement, React.HTMLAttributes<HTMLHeadingElement>>(
  function CardTitle({ children, className = "", ...props }, ref) {
    return (
      <h3 ref={ref} className={`text-base font-semibold text-text-primary ${className}`} {...props}>
        {children}
      </h3>
    );
  }
);

export const CardDescription = forwardRef<HTMLParagraphElement, React.HTMLAttributes<HTMLParagraphElement>>(
  function CardDescription({ children, className = "", ...props }, ref) {
    return (
      <p ref={ref} className={`text-sm text-text-tertiary mt-0.5 ${className}`} {...props}>
        {children}
      </p>
    );
  }
);
