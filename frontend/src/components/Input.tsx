import React, { forwardRef } from "react";

export interface InputProps extends React.InputHTMLAttributes<HTMLInputElement> {
  label?: string;
  error?: string;
  helperText?: string;
}

export const Input = forwardRef<HTMLInputElement, InputProps>(
  function Input(
    { label, error, helperText, disabled, className = "", ...props },
    ref
  ) {
    const inputBase =
      "w-full bg-surface-elevated border rounded-apple-sm px-3 py-2 text-sm text-text-primary placeholder:text-text-quaternary transition-colors duration-200 appearance-none";

    const inputState = error
      ? "border-brand-danger"
      : "border-glass-border hover:border-white/[0.18]";

    const inputDisabled = disabled
      ? "opacity-50 cursor-not-allowed"
      : "";

    return (
      <div className={`space-y-1.5 ${className}`}>
        {label && (
          <label className="block text-sm font-medium text-text-secondary">
            {label}
          </label>
        )}
        <input
          ref={ref}
          className={`${inputBase} ${inputState} ${inputDisabled}`}
          disabled={disabled}
          aria-invalid={!!error}
          aria-describedby={
            error
              ? `${props.id ?? props.name}-error`
              : helperText
              ? `${props.id ?? props.name}-help`
              : undefined
          }
          {...props}
        />
        {error && (
          <p
            id={`${props.id ?? props.name}-error`}
            className="text-xs text-brand-danger"
            role="alert"
          >
            {error}
          </p>
        )}
        {!error && helperText && (
          <p
            id={`${props.id ?? props.name}-help`}
            className="text-xs text-text-tertiary"
          >
            {helperText}
          </p>
        )}
      </div>
    );
  }
);
