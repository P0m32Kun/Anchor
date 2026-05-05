import { forwardRef } from "react";

export interface SelectOption {
  value: string;
  label: string;
}

export interface SelectProps {
  label?: string;
  value: string;
  options: SelectOption[];
  onChange: (value: string) => void;
  error?: string;
  placeholder?: string;
  disabled?: boolean;
  className?: string;
  id?: string;
  name?: string;
}

export const Select = forwardRef<HTMLSelectElement, SelectProps>(
  function Select(
    {
      label,
      value,
      options,
      onChange,
      error,
      placeholder,
      disabled,
      className = "",
      id,
      name,
    },
    ref
  ) {
    const selectBase =
      "w-full bg-slate-950/40 border rounded-lg px-3 py-2 pr-9 text-sm text-text-primary transition-colors duration-150 appearance-none";

    const selectState = error
      ? "border-brand-danger"
      : "border-white/[0.10] hover:border-white/[0.18] focus:border-brand-primary focus:shadow-[0_0_0_3px_rgba(14,165,233,0.18)] focus:outline-none";

    const selectDisabled = disabled
      ? "opacity-50 cursor-not-allowed"
      : "";

    const inputId = id ?? name;

    return (
      <div className={`space-y-1.5 ${className}`}>
        {label && (
          <label
            htmlFor={inputId}
            className="block text-sm font-medium text-text-secondary"
          >
            {label}
          </label>
        )}
        <div className="relative">
          <select
            ref={ref}
            id={inputId}
            name={name}
            value={value}
            onChange={(e) => onChange(e.target.value)}
            disabled={disabled}
            className={`${selectBase} ${selectState} ${selectDisabled}`}
            aria-invalid={!!error}
            aria-describedby={
              error
                ? `${inputId}-error`
                : undefined
            }
          >
            {placeholder && (
              <option value="" disabled>
                {placeholder}
              </option>
            )}
            {options.map((opt) => (
              <option key={opt.value} value={opt.value}>
                {opt.label}
              </option>
            ))}
          </select>
          {/* Custom chevron icon */}
          <svg
            className="pointer-events-none absolute right-3 top-1/2 -translate-y-1/2 h-4 w-4 text-text-tertiary"
            viewBox="0 0 24 24"
            fill="none"
            stroke="currentColor"
            strokeWidth="2"
            strokeLinecap="round"
            strokeLinejoin="round"
          >
            <polyline points="6 9 12 15 18 9" />
          </svg>
        </div>
        {error && (
          <p
            id={`${inputId}-error`}
            className="text-xs text-brand-danger"
            role="alert"
          >
            {error}
          </p>
        )}
      </div>
    );
  }
);
