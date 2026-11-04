"use client";

// ─────────────────────────────────────────────
// components/ui/word-zoom-control.tsx
// Kontrol zoom interaktif gaya Microsoft Word untuk Rich Text Editor.
// Fitur:
//   - Tombol perkecil (-) & perbesar (+)
//   - Slider range interaktif 50% s/d 200% (sesuai limit DevExpress RichEdit)
//   - Indikator persentase & tombol reset cepat (100% / Fit 1 Halaman)
//   - Menu preset zoom cepat (50%, 75%, 100%, 125%, 150%, 200%)
//   - Shortcut keyboard Ctrl + / Ctrl - / Ctrl 0 interaktif pada editor:
//     * Mencegah zoom browser Chrome (full-page zoom)
//     * Mencegah DevExpress mengeksekusi format Subscript (Ctrl+=)
//     * Memperbesar/memperkecil layout lembar kertas surat & slider secara sinkron
//   - Mendukung Ctrl + MouseWheel / trackpad pinch zoom
// ─────────────────────────────────────────────

import { useState, useRef, useEffect, useCallback } from "react";
import {
  Minus,
  Plus,
  ChevronDown,
  RotateCcw,
} from "lucide-react";
import { cn } from "@/lib/utils";

export const MIN_ZOOM = 0.5; // 50% (min limit DevExpress RichEdit)
export const MAX_ZOOM = 2.0; // 200% (max limit DevExpress RichEdit)
export const STEP_ZOOM = 0.05; // 5%

export const ZOOM_PRESETS = [
  { label: "Fit 1 Halaman", value: 0.5 },
  { label: "75%", value: 0.75 },
  { label: "100%", value: 1.0 },
  { label: "125%", value: 1.25 },
  { label: "150%", value: 1.5 },
  { label: "200%", value: 2.0 },
] as const;

export function isZoomInKeyEvent(e: KeyboardEvent): boolean {
  return (
    e.key === "=" ||
    e.key === "+" ||
    e.code === "Equal" ||
    e.code === "NumpadAdd" ||
    e.keyCode === 187 ||
    e.keyCode === 107
  );
}

export function isZoomOutKeyEvent(e: KeyboardEvent): boolean {
  return (
    e.key === "-" ||
    e.key === "_" ||
    e.code === "Minus" ||
    e.code === "NumpadSubtract" ||
    e.keyCode === 189 ||
    e.keyCode === 109
  );
}

export function isZoomResetKeyEvent(e: KeyboardEvent): boolean {
  return (
    e.key === "0" ||
    e.code === "Digit0" ||
    e.code === "Numpad0" ||
    e.keyCode === 48 ||
    e.keyCode === 96
  );
}

export interface WordZoomControlProps {
  zoomLevel: number;
  onZoomChange: (level: number) => void;
  className?: string;
  showPresets?: boolean;
}

export function WordZoomControl({
  zoomLevel,
  onZoomChange,
  className,
  showPresets = true,
}: WordZoomControlProps) {
  const [showMenu, setShowMenu] = useState(false);
  const menuRef = useRef<HTMLDivElement>(null);

  const percent = Math.round(zoomLevel * 100);

  const handleSliderChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    const nextVal = Number(e.target.value) / 100;
    onZoomChange(nextVal);
  };

  const handleZoomOut = () => {
    const next = Math.max(MIN_ZOOM, Math.round((zoomLevel - 0.1) * 100) / 100);
    onZoomChange(next);
  };

  const handleZoomIn = () => {
    const next = Math.min(MAX_ZOOM, Math.round((zoomLevel + 0.1) * 100) / 100);
    onZoomChange(next);
  };

  // Tutup menu preset jika klik di luar
  useEffect(() => {
    if (!showMenu) return;
    const handleClickOutside = (e: MouseEvent) => {
      if (menuRef.current && !menuRef.current.contains(e.target as Node)) {
        setShowMenu(false);
      }
    };
    document.addEventListener("mousedown", handleClickOutside);
    return () => document.removeEventListener("mousedown", handleClickOutside);
  }, [showMenu]);

  return (
    <div
      className={cn(
        "inline-flex items-center gap-1.5 bg-surface border border-border rounded-lg p-1 text-xs select-none shadow-2xs",
        className
      )}
    >
      {/* Preset Cepat: Fit 1 Halaman & 100% */}
      {showPresets && (
        <div className="flex items-center gap-0.5 border-r border-border pr-1 mr-0.5">
          <button
            type="button"
            onClick={() => onZoomChange(0.5)}
            className={cn(
              "px-2 py-0.5 rounded text-[11px] font-medium transition-colors",
              percent === 50
                ? "bg-primary text-white shadow-2xs font-semibold"
                : "text-text-muted hover:text-text hover:bg-bg"
            )}
            title="Tampilkan 1 lembar halaman penuh utuh (50%)"
          >
            Fit 1 Halaman
          </button>
          <button
            type="button"
            onClick={() => onZoomChange(1.0)}
            className={cn(
              "px-2 py-0.5 rounded text-[11px] font-medium transition-colors",
              percent === 100
                ? "bg-primary text-white shadow-2xs font-semibold"
                : "text-text-muted hover:text-text hover:bg-bg"
            )}
            title="Ukuran standar naskah (100%)"
          >
            100%
          </button>
        </div>
      )}

      {/* Tombol Zoom Out (-) */}
      <button
        type="button"
        disabled={zoomLevel <= MIN_ZOOM}
        onClick={handleZoomOut}
        className={cn(
          "p-1 rounded text-text-muted hover:text-text hover:bg-bg transition-colors",
          "disabled:opacity-40 disabled:cursor-not-allowed"
        )}
        title="Perkecil naskah (Ctrl + -)"
        aria-label="Perkecil zoom"
      >
        <Minus size={13} />
      </button>

      {/* Slider Gaya Word */}
      <div className="relative flex items-center px-1">
        <input
          type="range"
          min={50}
          max={200}
          step={5}
          value={percent}
          onChange={handleSliderChange}
          className={cn(
            "w-20 sm:w-28 h-1.5 bg-border rounded-lg appearance-none cursor-pointer",
            "accent-primary hover:accent-primary-dark transition-all",
            "focus:outline-none focus:ring-1 focus:ring-primary/40"
          )}
          title={`Zoom naskah: ${percent}% (Geser untuk mengatur)`}
          aria-label="Zoom slider"
        />
        {/* Titik penanda 100% di tengah slider */}
        <span
          className="absolute top-1/2 -translate-y-1/2 pointer-events-none w-0.5 h-2 bg-text-muted/40 rounded-full"
          style={{ left: "calc(33.33% + 2px)" }} // (100-50)/(200-50) = 33.33%
          title="Titik 100%"
        />
      </div>

      {/* Tombol Zoom In (+) */}
      <button
        type="button"
        disabled={zoomLevel >= MAX_ZOOM}
        onClick={handleZoomIn}
        className={cn(
          "p-1 rounded text-text-muted hover:text-text hover:bg-bg transition-colors",
          "disabled:opacity-40 disabled:cursor-not-allowed"
        )}
        title="Perbesar naskah (Ctrl + +)"
        aria-label="Perbesar zoom"
      >
        <Plus size={13} />
      </button>

      {/* Label Persentase & Menu Dropdown Preset */}
      <div className="relative" ref={menuRef}>
        <button
          type="button"
          onClick={() => setShowMenu((prev) => !prev)}
          className={cn(
            "flex items-center gap-1 min-w-12.5 justify-between px-1.5 py-0.5 rounded border border-border/60",
            "text-[11px] font-semibold text-text hover:bg-bg transition-colors"
          )}
          title="Klik untuk memilih preset ukuran zoom"
        >
          <span>{percent}%</span>
          <ChevronDown size={11} className="text-text-muted" />
        </button>

        {/* Dropdown Menu Preset */}
        {showMenu && (
          <div className="absolute right-0 bottom-full mb-1 sm:bottom-auto sm:top-full sm:mt-1 z-50 min-w-35 bg-surface border border-border rounded-lg shadow-lg py-1 text-xs">
            <div className="px-2.5 py-1 text-[10px] font-semibold text-text-muted uppercase tracking-wider border-b border-border/50">
              Pilihan Ukuran
            </div>
            {ZOOM_PRESETS.map((preset) => (
              <button
                key={preset.value}
                type="button"
                onClick={() => {
                  onZoomChange(preset.value);
                  setShowMenu(false);
                }}
                className={cn(
                  "w-full text-left px-2.5 py-1.5 text-xs flex items-center justify-between hover:bg-bg transition-colors",
                  percent === Math.round(preset.value * 100)
                    ? "font-bold text-primary bg-primary/5"
                    : "text-text"
                )}
              >
                <span>{preset.label}</span>
                {percent === Math.round(preset.value * 100) && (
                  <span className="size-1.5 rounded-full bg-primary shrink-0" />
                )}
              </button>
            ))}
            <div className="border-t border-border/50 pt-1 mt-1 px-1">
              <button
                type="button"
                onClick={() => {
                  onZoomChange(1.0);
                  setShowMenu(false);
                }}
                className="w-full text-left px-2 py-1 text-[11px] text-text-muted hover:text-text flex items-center gap-1.5 rounded hover:bg-bg"
              >
                <RotateCcw size={11} />
                Reset ke 100%
              </button>
            </div>
          </div>
        )}
      </div>
    </div>
  );
}

// ─────────────────────────────────────────────
// Hook untuk menambahkan interaktivitas zoom lengkap:
// - Intersep DevExpress internal keyDown (mencegah format Subscript Ctrl+=)
// - Capture event listeners pada window, wrapper, dan internal DevExpress iframe
// - Mencegah browser Chrome zoom in/out
// - Memperbarui state zoomLevel layout kertas & slider secara sinkron
// - Floating Feedback Toast saat zoom diubah via shortcut
// ─────────────────────────────────────────────
export interface UseWordZoomInteractivityOptions {
  wrapperRef: React.RefObject<HTMLDivElement | null>;
  containerRef?: React.RefObject<HTMLDivElement | null>;
  zoomLevel: number;
  onZoomChange: (level: number) => void;
  isReady?: boolean;
}

export function useWordZoomInteractivity({
  wrapperRef,
  containerRef,
  zoomLevel,
  onZoomChange,
  isReady = true,
}: UseWordZoomInteractivityOptions) {
  const [feedbackPercent, setFeedbackPercent] = useState<number | null>(null);
  const feedbackTimerRef = useRef<NodeJS.Timeout | null>(null);
  const zoomLevelRef = useRef(zoomLevel);
  useEffect(() => { zoomLevelRef.current = zoomLevel; }, [zoomLevel]);

  const showFeedback = useCallback((pct: number) => {
    setFeedbackPercent(pct);
    if (feedbackTimerRef.current) clearTimeout(feedbackTimerRef.current);
    feedbackTimerRef.current = setTimeout(() => {
      setFeedbackPercent(null);
    }, 1200);
  }, []);

  const zoomIn = useCallback(() => {
    const current = zoomLevelRef.current;
    const next = Math.min(MAX_ZOOM, Math.round((current + 0.1) * 100) / 100);
    onZoomChange(next);
    showFeedback(Math.round(next * 100));
  }, [onZoomChange, showFeedback]);

  const zoomOut = useCallback(() => {
    const current = zoomLevelRef.current;
    const next = Math.max(MIN_ZOOM, Math.round((current - 0.1) * 100) / 100);
    onZoomChange(next);
    showFeedback(Math.round(next * 100));
  }, [onZoomChange, showFeedback]);

  const zoomReset = useCallback(() => {
    onZoomChange(1.0);
    showFeedback(100);
  }, [onZoomChange, showFeedback]);

  // Handler khusus untuk dipasang pada options.events.keyDown DevExpress RichEdit.
  // Menyetel args.handled = true mencegah RichEdit mengeksekusi Subscript (Ctrl+=).
  const handleDevExpressKeyDown = useCallback(
    (_sender: unknown, args: { handled: boolean; htmlEvent: KeyboardEvent }) => {
      const evt = args?.htmlEvent;
      if (!evt || (!evt.ctrlKey && !evt.metaKey)) return;

      if (isZoomInKeyEvent(evt)) {
        evt.preventDefault();
        evt.stopPropagation();
        args.handled = true; // Hentikan DevExpress agar tidak mensubscript kata
        zoomIn();
      } else if (isZoomOutKeyEvent(evt)) {
        evt.preventDefault();
        evt.stopPropagation();
        args.handled = true; // Hentikan DevExpress
        zoomOut();
      } else if (isZoomResetKeyEvent(evt)) {
        evt.preventDefault();
        evt.stopPropagation();
        args.handled = true;
        zoomReset();
      }
    },
    [zoomIn, zoomOut, zoomReset]
  );

  // Helper untuk mengecek apakah elemen target berada di dalam area editor
  const isTargetInEditorArea = useCallback((target: EventTarget | null): boolean => {
    if (!target) return false;
    const wrapper = wrapperRef.current;
    if (!wrapper) return false;
    if (wrapper.contains(target as Node)) return true;

    // Cek iframe internal DevExpress
    const iframes = wrapper.querySelectorAll("iframe");
    for (let i = 0; i < iframes.length; i++) {
      try {
        const doc = iframes[i].contentDocument;
        if (doc && (doc === target || doc.contains(target as Node))) {
          return true;
        }
      } catch {
        // Abaikan cross-origin
      }
    }
    return false;
  }, [wrapperRef]);

  // Capture event listener untuk keyboard (menangkap sebelum browser / DevExpress)
  const handleKeyCapture = useCallback(
    (e: KeyboardEvent) => {
      if (!e.ctrlKey && !e.metaKey) return;

      const activeEl = document.activeElement;
      const isInputOutside =
        activeEl &&
        (activeEl.tagName === "INPUT" ||
          activeEl.tagName === "TEXTAREA" ||
          activeEl.tagName === "SELECT") &&
        !isTargetInEditorArea(activeEl);

      // Jangan cegah jika user sedang mengetik di input form luar
      if (isInputOutside) return;

      const isInside =
        isTargetInEditorArea(e.target) ||
        isTargetInEditorArea(activeEl) ||
        (wrapperRef.current && wrapperRef.current.matches(":hover"));

      if (!isInside) return;

      if (isZoomInKeyEvent(e)) {
        e.preventDefault();
        e.stopPropagation();
        e.stopImmediatePropagation();
        zoomIn();
      } else if (isZoomOutKeyEvent(e)) {
        e.preventDefault();
        e.stopPropagation();
        e.stopImmediatePropagation();
        zoomOut();
      } else if (isZoomResetKeyEvent(e)) {
        e.preventDefault();
        e.stopPropagation();
        e.stopImmediatePropagation();
        zoomReset();
      }
    },
    [isTargetInEditorArea, wrapperRef, zoomIn, zoomOut, zoomReset]
  );

  // Capture event listener untuk mouse wheel (Ctrl + Wheel)
  const handleWheelCapture = useCallback(
    (e: WheelEvent) => {
      if (!e.ctrlKey && !e.metaKey) return;

      const isInside =
        isTargetInEditorArea(e.target) ||
        (wrapperRef.current && wrapperRef.current.matches(":hover"));

      if (!isInside) return;

      e.preventDefault();
      e.stopPropagation();
      e.stopImmediatePropagation();

      const current = zoomLevelRef.current;
      const delta = e.deltaY < 0 ? STEP_ZOOM : -STEP_ZOOM;
      const next = Math.min(
        MAX_ZOOM,
        Math.max(MIN_ZOOM, Math.round((current + delta) * 100) / 100)
      );

      if (next !== current) {
        onZoomChange(next);
        showFeedback(Math.round(next * 100));
      }
    },
    [isTargetInEditorArea, wrapperRef, onZoomChange, showFeedback]
  );

  // Pasang capture listeners pada window dan periksa kemunculan iframe DevExpress
  useEffect(() => {
    if (!isReady) return;

    window.addEventListener("keydown", handleKeyCapture, { capture: true });
    window.addEventListener("wheel", handleWheelCapture, {
      capture: true,
      passive: false,
    });

    const wrapper = wrapperRef.current;
    if (wrapper) {
      wrapper.addEventListener("keydown", handleKeyCapture, { capture: true });
      wrapper.addEventListener("wheel", handleWheelCapture, {
        capture: true,
        passive: false,
      });
    }

    // Fungsi untuk menyematkan listener ke iframe dokumen editable DevExpress
    const attachToIframes = () => {
      const root = containerRef?.current || wrapperRef.current;
      if (!root) return;
      const iframes = root.querySelectorAll("iframe");
      iframes.forEach((iframe) => {
        try {
          const doc = iframe.contentDocument || iframe.contentWindow?.document;
          const win = iframe.contentWindow;
          // eslint-disable-next-line @typescript-eslint/no-explicit-any
          if (doc && !(doc as any).__wordZoomAttached) {
            // eslint-disable-next-line @typescript-eslint/no-explicit-any
            (doc as any).__wordZoomAttached = true;
            doc.addEventListener("keydown", handleKeyCapture, {
              capture: true,
            });
            doc.addEventListener("wheel", handleWheelCapture, {
              capture: true,
              passive: false,
            });
          }
          // eslint-disable-next-line @typescript-eslint/no-explicit-any
          if (win && !(win as any).__wordZoomAttached) {
            // eslint-disable-next-line @typescript-eslint/no-explicit-any
            (win as any).__wordZoomAttached = true;
            win.addEventListener("keydown", handleKeyCapture, {
              capture: true,
            });
            win.addEventListener("wheel", handleWheelCapture, {
              capture: true,
              passive: false,
            });
          }
        } catch {
          // Abaikan cross-origin
        }
      });
    };

    attachToIframes();
    const timer = setInterval(attachToIframes, 500);

    return () => {
      clearInterval(timer);
      window.removeEventListener("keydown", handleKeyCapture, {
        capture: true,
      });
      window.removeEventListener("wheel", handleWheelCapture, {
        capture: true,
      });
      if (wrapper) {
        wrapper.removeEventListener("keydown", handleKeyCapture, {
          capture: true,
        });
        wrapper.removeEventListener("wheel", handleWheelCapture, {
          capture: true,
        });
      }
      if (feedbackTimerRef.current) clearTimeout(feedbackTimerRef.current);
    };
  }, [
    isReady,
    wrapperRef,
    containerRef,
    handleKeyCapture,
    handleWheelCapture,
  ]);

  return {
    feedbackPercent,
    handleDevExpressKeyDown,
    zoomIn,
    zoomOut,
    zoomReset,
  };
}
