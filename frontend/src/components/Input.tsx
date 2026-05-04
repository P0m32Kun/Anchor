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
      "w-full bg-black/30 border rounded-apple-sm px-3 py-2 text-sm text-text-primary placeholder:text-text-quaternary transition-all duration-150 appearance-none shadow-[inset_0_2px_4px_rgba(0,0,0,0.5)]";

    const inputState = error
      ? "border-brand-danger"
      : "border-white/[0.08] hover:border-white/[0.15] focus:border-brand-primary focus:shadow-[0_0_0_3px_rgba(47,129,247,0.15),inset_0_2px_4px_rgba(0,0,0,0.5)] focus:outline-none";

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
