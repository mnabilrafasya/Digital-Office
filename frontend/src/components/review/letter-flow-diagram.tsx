"use client";

// ─────────────────────────────────────────────
// components/review/letter-flow-diagram.tsx
// Visualisasi Diagram Alur Perjalanan Surat (Node-Link Flowchart)
// Dilengkapi tombol navigasi geser (< >) & slider horizontal agar tidak memotong layout
// ─────────────────────────────────────────────

import { useState, useRef, useEffect, useCallback } from "react";
import {
  Check,
  Clock,
  RotateCcw,
  Send,
  XCircle,
  FileEdit,
  ArrowRight,
  MessageSquareWarning,
  ChevronLeft,
  ChevronRight,
} from "lucide-react";
import type {
  LetterDetail,
  LetterHistoryEntry,
} from "@/lib/api/letter-detail";
import {
  buildFlowDiagram,
  formatCompactTime,
  type FlowDiagramNode,
} from "./audit-trail-utils";
import { cn } from "@/lib/utils";

interface LetterFlowDiagramProps {
  letter: LetterDetail;
  history: LetterHistoryEntry[];
  compact?: boolean;
}

function nodeIcon(node: FlowDiagramNode) {
  if (node.state === "CANCELLED") return <XCircle size={13} />;
  if (node.state === "RETURNED") return <RotateCcw size={13} />;
  if (node.state === "DONE") {
    return node.roleCode === "FINAL" ? (
      <Send size={12} />
    ) : (
      <Check size={13} strokeWidth={2.5} />
    );
  }
  if (node.state === "ACTIVE") {
    return node.roleCode === "DRAFTER" ? (
      <FileEdit size={12} />
    ) : (
      <span className="w-2.5 h-2.5 rounded-full bg-white animate-pulse" />
    );
  }
  return <Clock size={12} />;
}

export function LetterFlowDiagram({
  letter,
  history,
  compact = false,
}: LetterFlowDiagramProps) {
  const { nodes, revisionCount, saveCount, totalMilestones } = buildFlowDiagram(
    letter,
    history
  );
  const [selectedNodeId, setSelectedNodeId] = useState<string | null>(null);

  // State & ref untuk kontrol gulir (scroll) dan slider horizontal
  const scrollRef = useRef<HTMLDivElement>(null);
  const [canScrollLeft, setCanScrollLeft] = useState(false);
  const [canScrollRight, setCanScrollRight] = useState(false);
  const [scrollPercent, setScrollPercent] = useState(0);
  const [hasOverflow, setHasOverflow] = useState(false);

  const updateScrollState = useCallback(() => {
    const el = scrollRef.current;
    if (!el) return;
    const maxScroll = el.scrollWidth - el.clientWidth;
    const overflow = maxScroll > 4;
    setHasOverflow(overflow);
    setCanScrollLeft(el.scrollLeft > 4);
    setCanScrollRight(el.scrollLeft < maxScroll - 4);
    setScrollPercent(
      overflow ? Math.round((el.scrollLeft / maxScroll) * 100) : 0
    );
  }, []);

  useEffect(() => {
    const el = scrollRef.current;
    if (!el) return;
    updateScrollState();
    const ro = new ResizeObserver(() => updateScrollState());
    ro.observe(el);
    return () => ro.disconnect();
  }, [nodes.length, updateScrollState]);

  function scrollByAmount(delta: number) {
    const el = scrollRef.current;
    if (!el) return;
    el.scrollBy({ left: delta, behavior: "smooth" });
  }

  function handleSliderChange(val: number) {
    const el = scrollRef.current;
    if (!el) return;
    const maxScroll = el.scrollWidth - el.clientWidth;
    el.scrollLeft = (val / 100) * maxScroll;
    setScrollPercent(val);
  }

  const selectedNode =
    nodes.find((n) => n.id === selectedNodeId) ??
    nodes.find((n) => n.latestReturnNote) ??
    null;

  return (
    <div className="w-full min-w-0 max-w-full overflow-hidden space-y-2.5">
      {/* Baris metrik ringkas + tombol geser kiri/kanan */}
      <div className="flex items-center justify-between gap-2 flex-wrap text-[11px]">
        <div className="flex items-center gap-1.5 flex-wrap">
          <span className="px-2 py-0.5 rounded-md bg-panel border border-border text-text font-medium">
            {nodes.filter((n) => n.state === "DONE").length}/{nodes.length}{" "}
            Tahap Selesai
          </span>
          {revisionCount > 0 && (
            <span className="inline-flex items-center gap-1 px-2 py-0.5 rounded-md bg-status-warn-bg text-status-warn-text border border-status-warn-text/20 font-semibold">
              <RotateCcw size={10} /> {revisionCount}× Siklus Revisi
            </span>
          )}
          {saveCount > 0 && (
            <span className="px-2 py-0.5 rounded-md bg-panel border border-border text-text-muted">
              Versi Aktif:{" "}
              <strong className="text-text">
                v{letter.currentVersion || 1}
              </strong>
            </span>
          )}
        </div>

        <div className="flex items-center gap-2 ml-auto">
          <span className="text-text-muted">
            {totalMilestones} tonggak riwayat
          </span>
          {hasOverflow && (
            <div className="inline-flex items-center gap-1">
              <button
                type="button"
                onClick={() => scrollByAmount(-220)}
                disabled={!canScrollLeft}
                title="Geser diagram ke kiri"
                className="w-6 h-6 rounded-md border border-border bg-surface flex items-center justify-center text-text-muted hover:text-text hover:bg-panel disabled:opacity-35 disabled:pointer-events-none transition-colors"
              >
                <ChevronLeft size={13} />
              </button>
              <button
                type="button"
                onClick={() => scrollByAmount(220)}
                disabled={!canScrollRight}
                title="Geser diagram ke kanan"
                className="w-6 h-6 rounded-md border border-border bg-surface flex items-center justify-center text-text-muted hover:text-text hover:bg-panel disabled:opacity-35 disabled:pointer-events-none transition-colors"
              >
                <ChevronRight size={13} />
              </button>
            </div>
          )}
        </div>
      </div>

      {/* Kanvas Diagram Alur Horizontal (Terkonstrain & Bisa Digeser) */}
      <div
        ref={scrollRef}
        onScroll={updateScrollState}
        className="w-full max-w-full overflow-x-auto pb-2 scroll-smooth"
      >
        <div className="inline-flex items-stretch gap-1.5 py-1 pr-2">
          {nodes.map((node, idx) => {
            const isDone = node.state === "DONE";
            const isActive = node.state === "ACTIVE";
            const isReturned = node.state === "RETURNED";
            const isCancelled = node.state === "CANCELLED";
            const isSelected = selectedNode?.id === node.id;

            return (
              <div key={node.id} className="flex items-center shrink-0">
                {/* Kartu Node Diagram */}
                <button
                  type="button"
                  onClick={() =>
                    setSelectedNodeId((prev) =>
                      prev === node.id ? null : node.id
                    )
                  }
                  className={cn(
                    "group relative text-left rounded-xl border transition-all flex flex-col justify-between shrink-0",
                    compact ? "w-40 p-2.5" : "w-44 p-3",
                    isDone &&
                      "bg-status-success-bg/35 border-status-success-text/30 hover:border-status-success-text",
                    isActive &&
                      "bg-primary/5 border-primary ring-2 ring-primary/15",
                    isReturned &&
                      "bg-status-warn-bg/50 border-status-warn-text/40 ring-2 ring-status-warn-text/15",
                    isCancelled &&
                      "bg-status-danger-bg/40 border-status-danger-text/30",
                    node.state === "PENDING" &&
                      "bg-panel/40 border-border opacity-80 hover:opacity-100",
                    isSelected && "ring-2 ring-primary"
                  )}
                >
                  <div>
                    <div className="flex items-center justify-between gap-1.5 mb-1.5">
                      <span
                        className={cn(
                          "w-6 h-6 rounded-full flex items-center justify-center shrink-0 text-white",
                          isDone && "bg-status-success-text",
                          isActive && "bg-primary",
                          isReturned && "bg-status-warn-text",
                          isCancelled && "bg-status-danger-text",
                          node.state === "PENDING" &&
                            "bg-surface border border-border text-text-muted"
                        )}
                      >
                        {nodeIcon(node)}
                      </span>
                      <span
                        className={cn(
                          "text-[10px] font-bold uppercase tracking-wide truncate",
                          isDone && "text-status-success-text",
                          isActive && "text-primary",
                          isReturned && "text-status-warn-text",
                          isCancelled && "text-status-danger-text",
                          node.state === "PENDING" && "text-text-muted"
                        )}
                      >
                        {node.roleLabel}
                      </span>
                    </div>

                    <p className="text-xs font-semibold text-text truncate">
                      {node.actorName}
                    </p>
                    <p className="text-[11px] text-text-muted truncate">
                      {node.actorPosition}
                    </p>
                  </div>

                  <div className="mt-2 pt-1.5 border-t border-border/60 flex items-center justify-between gap-1 flex-wrap">
                    <span className="text-[10px] text-text-muted truncate">
                      {node.lastActionAt
                        ? formatCompactTime(node.lastActionAt)
                        : isActive
                          ? "Giliran saat ini"
                          : "Menunggu"}
                    </span>
                    <div className="flex items-center gap-1">
                      {node.savedVersions.length > 0 && (
                        <span className="text-[9px] font-mono px-1 py-0.2 rounded bg-surface border border-border text-text-muted">
                          v{node.savedVersions[node.savedVersions.length - 1]}
                        </span>
                      )}
                      {node.returnCount > 0 && (
                        <span
                          title="Pernah mengembalikan naskah untuk revisi"
                          className="inline-flex items-center gap-0.5 text-[9px] font-semibold px-1.5 py-0.2 rounded-full bg-status-warn-bg text-status-warn-text"
                        >
                          <RotateCcw size={8} /> {node.returnCount}×
                        </span>
                      )}
                    </div>
                  </div>
                </button>

                {/* Panah Penghubung Antar-Node */}
                {idx < nodes.length - 1 && (
                  <div className="flex flex-col items-center justify-center px-1 shrink-0">
                    {nodes[idx + 1].returnCount > 0 && (
                      <span
                        title="Terdapat pengembalian revisi dari tahap berikutnya"
                        className="text-[9px] font-semibold text-status-warn-text bg-status-warn-bg px-1 rounded mb-0.5 inline-flex items-center gap-0.5"
                      >
                        <RotateCcw size={8} />
                      </span>
                    )}
                    <div className="flex items-center">
                      <div
                        className={cn(
                          "w-3 h-0.5",
                          isDone ? "bg-status-success-text" : "bg-border"
                        )}
                      />
                      <ArrowRight
                        size={13}
                        className={cn(
                          "-ml-1",
                          isDone ? "text-status-success-text" : "text-border"
                        )}
                      />
                    </div>
                  </div>
                )}
              </div>
            );
          })}
        </div>
      </div>

      {/* Slider Kontrol Geser Horizontal (Muncul otomatis bila alur panjang) */}
      {hasOverflow && (
        <div className="flex items-center gap-2.5 px-2.5 py-1.5 rounded-lg bg-panel/40 border border-border/70">
          <button
            type="button"
            onClick={() => scrollByAmount(-200)}
            disabled={!canScrollLeft}
            className="text-text-muted hover:text-text disabled:opacity-30 transition-colors"
            title="Geser kiri"
          >
            <ChevronLeft size={14} />
          </button>
          <input
            type="range"
            min={0}
            max={100}
            value={scrollPercent}
            onChange={(e) => handleSliderChange(Number(e.target.value))}
            aria-label="Geser diagram alur"
            className="w-full h-1.5 bg-border rounded-lg appearance-none cursor-pointer accent-primary"
          />
          <button
            type="button"
            onClick={() => scrollByAmount(200)}
            disabled={!canScrollRight}
            className="text-text-muted hover:text-text disabled:opacity-30 transition-colors"
            title="Geser kanan"
          >
            <ChevronRight size={14} />
          </button>
          <span className="text-[10px] font-medium text-text-muted shrink-0">
            Geser Alur
          </span>
        </div>
      )}

      {/* Panel Catatan Node Terpilih (atau Catatan Revisi Terakhir) */}
      {selectedNode && selectedNode.latestReturnNote && (
        <div className="rounded-lg bg-status-warn-bg/50 border border-status-warn-text/25 px-3 py-2 text-xs flex items-start gap-2">
          <MessageSquareWarning
            size={14}
            className="text-status-warn-text shrink-0 mt-0.5"
          />
          <div className="min-w-0 flex-1">
            <span className="font-semibold text-status-warn-text">
              Catatan Revisi ({selectedNode.roleLabel} —{" "}
              {selectedNode.actorName}):{" "}
            </span>
            <span className="text-text italic">
              &ldquo;{selectedNode.latestReturnNote}&rdquo;
            </span>
          </div>
        </div>
      )}
    </div>
  );
}
