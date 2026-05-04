import { useEffect, useRef, type ReactNode } from "react";
import ReactDOM from "react-dom";

interface ModalProps {
  open: boolean;
  onClose: () => void;
  title?: string;
  description?: string;
  children: ReactNode;
  footer?: ReactNode;
  size?: "sm" | "md" | "lg";
}

const sizeMap = { sm: "max-w-sm", md: "max-w-md", lg: "max-w-lg" };

export default function Modal({
  open,
  onClose,
  title,
  description,
  children,
  footer,
  size = "md",
}: ModalProps) {
  const contentRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    if (!open) return;
    const handler = (e: KeyboardEvent) => {
      if (e.key === "Escape") onClose();
    };
    document.addEventListener("keydown", handler);
    return () => document.removeEventListener("keydown", handler);
  }, [open, onClose]);

  useEffect(() => {
    if (open) {
      document.body.style.overflow = "hidden";
      contentRef.current?.focus();
    } else {
      document.body.style.overflow = "";
    }
    return () => {
      document.body.style.overflow = "";
    };
  }, [open]);

  if (!open) return null;

  return ReactDOM.createPortal(
    <div
      className="fixed inset-0 z-50 flex items-center justify-center p-4 animate-fade-in"
      onClick={(e) => {
        if (e.target === e.currentTarget) onClose();
      }}
      role="presentation"
    >
      <div className="fixed inset-0 bg-black/60 backdrop-blur-xl" />
      <div
        ref={contentRef}
        tabIndex={-1}
        role="dialog"
        aria-modal="true"
        aria-labelledby={title ? "modal-title" : undefined}
        className={`relative z-10 w-full ${sizeMap[size]} vision-glass-strong rounded-[20px] animate-scale-in outline-none`}
      >
        {(title || description) && (
          <div className="px-6 pt-5 pb-2">
            {title && (
              <h2 id="modal-title" className="text-lg font-semibold text-text-primary">
                {title}
              </h2>
            )}
            {description && (
              <p className="mt-1 text-sm text-text-secondary">{description}</p>
            )}
          </div>
        )}
        <div className="px-6 py-4">{children}</div>
        {footer && (
          <div className="px-6 py-4 border-t border-glass-border-light flex justify-end gap-2">
            {footer}
          </div>
        )}
      </div>
    </div>,
    document.body
  );
}
