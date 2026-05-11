import React, { createContext, useContext, useState, useCallback, useRef, useEffect } from "react";

interface ToastItem {
  id: string;
  message: string;
  type: "success" | "warning" | "error";
}

interface ToastContextValue {
  toast: (message: string, type?: "success" | "warning" | "error") => void;
}

const ToastContext = createContext<ToastContextValue | null>(null);

export function useToast() {
  const ctx = useContext(ToastContext);
  if (!ctx) throw new Error("useToast must be used within ToastProvider");
  return ctx.toast;
}

const MAX_TOASTS = 3;

const DURATION: Record<ToastItem["type"], number> = {
  success: 3000,
  warning: 4000,
  error: 5000,
};

export function ToastProvider({ children }: { children: React.ReactNode }) {
  const [toasts, setToasts] = useState<ToastItem[]>([]);
  const timersRef = useRef<Map<string, number>>(new Map());

  useEffect(() => {
    return () => {
      timersRef.current.forEach(clearTimeout);
    };
  }, []);

  const scheduleRemove = useCallback((id: string, type: ToastItem["type"]) => {
    const timer = window.setTimeout(() => {
      setToasts((prev) => prev.filter((t) => t.id !== id));
      timersRef.current.delete(id);
    }, DURATION[type]);
    timersRef.current.set(id, timer);
  }, []);

  const toast = useCallback(
    (message: string, type: ToastItem["type"] = "success") => {
      const id = crypto.randomUUID();
      setToasts((prev) => {
        const next = [...prev, { id, message, type }];
        // Evict oldest if over limit
        if (next.length > MAX_TOASTS) {
          const evicted = next[0];
          const timer = timersRef.current.get(evicted.id);
          if (timer) {
            clearTimeout(timer);
            timersRef.current.delete(evicted.id);
          }
          return next.slice(1);
        }
        return next;
      });
      scheduleRemove(id, type);
    },
    [scheduleRemove]
  );

  const remove = useCallback((id: string) => {
    const timer = timersRef.current.get(id);
    if (timer) {
      clearTimeout(timer);
      timersRef.current.delete(id);
    }
    setToasts((prev) => prev.filter((t) => t.id !== id));
  }, []);

  const iconMap = {
    success: (
      <svg className="w-4 h-4 text-accent-green" viewBox="0 0 20 20" fill="currentColor">
        <path fillRule="evenodd" d="M10 18a8 8 0 100-16 8 8 0 000 16zm3.707-9.293a1 1 0 00-1.414-1.414L9 10.586 7.707 9.293a1 1 0 00-1.414 1.414l2 2a1 1 0 001.414 0l4-4z" clipRule="evenodd" />
      </svg>
    ),
    warning: (
      <svg className="w-4 h-4 text-accent-yellow" viewBox="0 0 20 20" fill="currentColor">
        <path fillRule="evenodd" d="M8.257 3.099c.765-1.36 2.722-1.36 3.486 0l5.58 9.92c.75 1.334-.213 2.98-1.742 2.98H4.42c-1.53 0-2.493-1.646-1.743-2.98l5.58-9.92zM11 13a1 1 0 11-2 0 1 1 0 012 0zm-1-8a1 1 0 00-1 1v3a1 1 0 002 0V6a1 1 0 00-1-1z" clipRule="evenodd" />
      </svg>
    ),
    error: (
      <svg className="w-4 h-4 text-accent-red" viewBox="0 0 20 20" fill="currentColor">
        <path fillRule="evenodd" d="M10 18a8 8 0 100-16 8 8 0 000 16zM8.707 7.293a1 1 0 00-1.414 1.414L8.586 10l-1.293 1.293a1 1 0 101.414 1.414L10 11.414l1.293 1.293a1 1 0 001.414-1.414L11.414 10l1.293-1.293a1 1 0 00-1.414-1.414L10 8.586 8.707 7.293z" clipRule="evenodd" />
      </svg>
    ),
  };

  const borderMap = {
    success: "border-l-accent-green",
    warning: "border-l-accent-yellow",
    error: "border-l-accent-red",
  };

  return (
    <ToastContext.Provider value={{ toast }}>
      {children}
      <div
        role="alert"
        aria-live="polite"
        className="fixed top-4 left-1/2 -translate-x-1/2 z-[100] flex flex-col gap-2 pointer-events-none"
      >
        {toasts.map((t) => (
          <div
            key={t.id}
            className={`pointer-events-auto flex items-center gap-2.5 px-4 py-2.5 rounded-lg border border-l-[3px] ${borderMap[t.type]} animate-slide-down`}
            style={{
              background: "linear-gradient(180deg, rgba(18,35,62,0.94), rgba(10,22,40,0.96))",
              backdropFilter: "saturate(180%) blur(28px)",
              WebkitBackdropFilter: "saturate(180%) blur(28px)",
              boxShadow:
                "inset 0 1px 0 rgba(255,255,255,0.06), inset 0 0 18px rgba(0,212,255,0.04), 0 8px 24px rgba(0,0,0,0.34)",
            }}
          >
            {iconMap[t.type]}
            <span className="text-sm font-medium text-text-primary">{t.message}</span>
            <button
              onClick={() => remove(t.id)}
              className="ml-1 text-text-quaternary hover:text-text-secondary transition-colors"
              aria-label="关闭通知"
            >
              <svg className="w-3.5 h-3.5" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2.5">
                <path d="M18 6L6 18M6 6l12 12" />
              </svg>
            </button>
          </div>
        ))}
      </div>
    </ToastContext.Provider>
  );
}
