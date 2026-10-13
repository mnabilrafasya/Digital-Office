"use client";

// ─────────────────────────────────────────────
// components/ui/dialog.tsx
// Custom Dialog — Native HTML dialog element
// Tidak bergantung Base UI/Radix, lebih reliable
// ─────────────────────────────────────────────

import * as React from "react";
import { X } from "lucide-react";
import { cn } from "@/lib/utils";

interface DialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  children: React.ReactNode;
}

function Dialog({ open, onOpenChange, children }: DialogProps) {
  // Close on Escape
  React.useEffect(() => {
    if (!open) return;
    function onKey(e: KeyboardEvent) {
      if (e.key === "Escape") onOpenChange(false);
    }
    document.addEventListener("keydown", onKey);
    return () => document.removeEventListener("keydown", onKey);
  }, [open, onOpenChange]);

  // Prevent body scroll saat dialog terbuka
  React.useEffect(() => {
    document.body.style.overflow = open ? "hidden" : "";
    return () => { document.body.style.overflow = ""; };
  }, [open]);

  if (!open) return null;

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center">
      {/* Backdrop */}
      <div
        className="absolute inset-0 bg-black/40 backdrop-blur-sm"
        onClick={() => onOpenChange(false)}
      />
      {/* Content container */}
      <div className="relative z-10 w-full max-w-md mx-4">
        {children}
      </div>
    </div>
  );
}

function DialogContent({
  className,
  children,
  showCloseButton = true,
  onClose,
  ...props
}: React.HTMLAttributes<HTMLDivElement> & {
  showCloseButton?: boolean;
  onClose?: () => void;
}) {
  return (
    <div
      className={cn(
        "bg-surface rounded-xl border border-border shadow-2xl p-6 space-y-4",
        "animate-in fade-in-0 zoom-in-95 duration-150",
        className,
      )}
      {...props}
    >
      {showCloseButton && onClose && (
        <button
          type="button"
          onClick={onClose}
          className="absolute top-4 right-4 text-text-muted hover:text-text transition-colors"
          aria-label="Tutup"
        >
          <X size={16} />
        </button>
      )}
      {children}
    </div>
  );
}

function DialogHeader({ className, ...props }: React.HTMLAttributes<HTMLDivElement>) {
  return (
    <div
      className={cn("flex flex-col gap-1.5", className)}
      {...props}
    />
  );
}

function DialogTitle({ className, ...props }: React.HTMLAttributes<HTMLHeadingElement>) {
  return (
    <h3
      className={cn("text-base font-bold text-text leading-tight", className)}
      {...props}
    />
  );
}

function DialogDescription({ className, ...props }: React.HTMLAttributes<HTMLParagraphElement>) {
  return (
    <p
      className={cn("text-sm text-text-muted leading-relaxed", className)}
      {...props}
    />
  );
}

function DialogFooter({ className, ...props }: React.HTMLAttributes<HTMLDivElement>) {
  return (
    <div
      className={cn("flex items-center justify-end gap-2 pt-2", className)}
      {...props}
    />
  );
}

export {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogDescription,
  DialogFooter,
};
