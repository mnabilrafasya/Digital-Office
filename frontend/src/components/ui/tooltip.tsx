"use client";

// ─────────────────────────────────────────────
// components/ui/tooltip.tsx
// Custom Tooltip — CSS-only hover approach
// Tidak bergantung @base-ui/react/tooltip
// ─────────────────────────────────────────────

import * as React from "react";
import { cn } from "@/lib/utils";

interface TooltipProps {
  children: React.ReactNode;
  delayDuration?: number;
}

function TooltipProvider({ children }: { children: React.ReactNode; delayDuration?: number }) {
  return <>{children}</>;
}

function Tooltip({ children }: TooltipProps) {
  return (
    <div className="relative inline-flex group/tooltip">
      {children}
    </div>
  );
}

function TooltipTrigger({
  children,
  className,
  ...props
}: React.HTMLAttributes<HTMLDivElement>) {
  return (
    <div className={cn("inline-flex", className)} {...props}>
      {children}
    </div>
  );
}

function TooltipContent({
  children,
  side = "top",
  className,
  ...props
}: React.HTMLAttributes<HTMLDivElement> & {
  side?: "top" | "bottom" | "left" | "right";
}) {
  const sideClass = {
    top: "bottom-full left-1/2 -translate-x-1/2 mb-2",
    bottom: "top-full left-1/2 -translate-x-1/2 mt-2",
    left: "right-full top-1/2 -translate-y-1/2 mr-2",
    right: "left-full top-1/2 -translate-y-1/2 ml-2",
  }[side];

  return (
    <div
      role="tooltip"
      className={cn(
        "absolute z-50 pointer-events-none",
        "opacity-0 group-hover/tooltip:opacity-100",
        "scale-95 group-hover/tooltip:scale-100",
        "transition-all duration-150",
        sideClass,
        className,
      )}
      {...props}
    >
      <div className="bg-text text-white text-xs rounded-md px-2.5 py-1.5 whitespace-nowrap shadow-lg">
        {children}
      </div>
    </div>
  );
}

export { Tooltip, TooltipTrigger, TooltipContent, TooltipProvider };
