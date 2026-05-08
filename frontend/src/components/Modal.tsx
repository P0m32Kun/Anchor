import { useEffect, useRef, type ReactNode } from "react";
import ReactDOM from "react-dom";
import { cn } from "../lib/utils";
import { X } from "lucide-react";
import { Button } from "./Button";

interface ModalProps {
  open: boolean;
  onClose: () => void;
  title?: string;
  description?: string;
  children: ReactNode;
  footer?: ReactNode;
  size?: "sm" | "md" | "lg" | "xl";
}

const sizeMap = { 
  sm: "max-w-sm", 
  md: "max-w-md", 
  lg: "max-w-2xl",
  xl: "max-w-4xl"
};

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
      className="fixed inset-0 z-[100] flex items-center justify-center p-4 animate-in fade-in duration-300"
      role="presentation"
    >
      {/* 背景遮罩：降低模糊强度，使用更暗的半透明黑，避免“割裂感” */}
      <div 
        className="fixed inset-0 bg-slate-950/60 backdrop-blur-[2px] transition-opacity" 
        onClick={onClose}
      />
      
      {/* 弹窗容器：固定大小，自身具备毛玻璃质感，边缘更清晰 */}
      <div
        ref={contentRef}
        tabIndex={-1}
        role="dialog"
        aria-modal="true"
        aria-labelledby={title ? "modal-title" : undefined}
        className={cn(
          "relative z-10 w-full rounded-2xl border border-white/10 bg-card/90 backdrop-blur-xl shadow-2xl animate-in zoom-in-95 duration-300 outline-none overflow-hidden",
          sizeMap[size]
        )}
      >
        {/* 顶部装饰条 */}
        <div className="absolute inset-x-0 top-0 h-px bg-gradient-to-r from-transparent via-primary/30 to-transparent" />

        {(title || description) && (
          <div className="px-6 pt-6 pb-2 flex items-start justify-between">
            <div className="space-y-1">
              {title && (
                <h2 id="modal-title" className="text-xl font-bold tracking-tight text-foreground">
                  {title}
                </h2>
              )}
              {description && (
                <p className="text-sm text-muted-foreground font-medium">{description}</p>
              )}
            </div>
            <Button variant="ghost" size="sm" className="h-8 w-8 p-0 rounded-full hover:bg-white/5" onClick={onClose}>
               <X className="h-4 w-4 text-muted-foreground" />
            </Button>
          </div>
        )}
        
        {!title && !description && (
            <div className="absolute right-4 top-4 z-20">
                <Button variant="ghost" size="sm" className="h-8 w-8 p-0 rounded-full hover:bg-white/5" onClick={onClose}>
                    <X className="h-4 w-4 text-muted-foreground" />
                </Button>
            </div>
        )}

        <div className="px-6 py-4">{children}</div>
        
        {footer && (
          <div className="px-6 py-4 bg-white/5 border-t border-white/5 flex justify-end gap-3">
            {footer}
          </div>
        )}
      </div>
    </div>,
    document.body
  );
}
