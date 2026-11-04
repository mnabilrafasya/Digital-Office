"use client";

// components/surat/letter-editor/use-select-on-focus.ts
// Meningkatkan UX Ribbon DevExpress agar sama seperti Microsoft Word:
// - Saat kotak Font Name, Font Size, atau input angka di Ribbon diklik / difokus,
//   seluruh teks di dalamnya langsung terblok (selected) supaya staf bisa langsung
//   mengetik untuk mencari font (misal ketik "Times" atau "12") tanpa harus blok/hapus manual.
// - Mendukung shortcut Ctrl+S di dalam area editor untuk memicu penyimpanan naskah/draf.

import { useEffect, type RefObject } from "react";

export function useSelectOnFocus(
  containerRef: RefObject<HTMLElement | null>,
  enabled: boolean,
  onCtrlSave?: () => void
) {
  useEffect(() => {
    const root = containerRef.current;
    if (!enabled || !root) return;

    const isRibbonInput = (t: EventTarget | null): t is HTMLInputElement =>
      t instanceof HTMLInputElement &&
      (t.classList.contains("dx-texteditor-input") ||
        t.getAttribute("role") === "combobox" ||
        t.getAttribute("role") === "spinbutton") &&
      !!t.closest(".dx-toolbar, .dxre-ribbon, .dx-ribbon, .dx-tabs");

    let justFocused: HTMLInputElement | null = null;

    const selectAllText = (input: HTMLInputElement) => {
      try {
        input.select();
        if (typeof input.setSelectionRange === "function") {
          input.setSelectionRange(0, input.value.length);
        }
      } catch {
        // Abaikan jika tipe input tidak mendukung setSelectionRange
      }
    };

    const onFocusIn = (e: FocusEvent) => {
      if (!isRibbonInput(e.target)) return;
      const input = e.target;
      justFocused = input;
      // Jalankan dua kali (microtask & setelah animasi dropdown DevExtreme) agar tidak tertimpa
      requestAnimationFrame(() => selectAllText(input));
      setTimeout(() => selectAllText(input), 40);
    };

    // Saat pengguna mengklik kotak yang sudah fokus atau baru fokus, cegah mouseup menghapus blok seleksi
    const onMouseUp = (e: MouseEvent) => {
      if (isRibbonInput(e.target) && e.target === justFocused) {
        e.preventDefault();
        selectAllText(e.target);
        justFocused = null;
      }
    };

    const onKeyDown = (e: KeyboardEvent) => {
      if ((e.ctrlKey || e.metaKey) && e.key.toLowerCase() === "s") {
        if (onCtrlSave) {
          e.preventDefault();
          e.stopPropagation();
          onCtrlSave();
        }
      }
    };

    root.addEventListener("focusin", onFocusIn, true);
    root.addEventListener("mouseup", onMouseUp, true);
    root.addEventListener("keydown", onKeyDown, true);
    return () => {
      root.removeEventListener("focusin", onFocusIn, true);
      root.removeEventListener("mouseup", onMouseUp, true);
      root.removeEventListener("keydown", onKeyDown, true);
    };
  }, [containerRef, enabled, onCtrlSave]);
}
