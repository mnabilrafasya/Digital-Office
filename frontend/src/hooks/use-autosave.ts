"use client";

// ─────────────────────────────────────────────
// hooks/use-autosave.ts
// Autosave berbasis "signature" (JSON dari seluruh isi form + lampiran).
// Dirty = signature sekarang berbeda dari signature terakhir yang tersimpan.
// ─────────────────────────────────────────────

import { useCallback, useEffect, useRef, useState } from "react";

interface Options {
  signature: string;
  save: () => Promise<void>;
  enabled?: boolean;
  delay?: number;
}

export function useAutosave({ signature, save, enabled = true, delay = 1500 }: Options) {
  const [savedSignature, setSavedSignature] = useState(signature);
  const [lastSaved, setLastSaved] = useState<Date | null>(null);
  const [isSaving, setIsSaving] = useState(false);
  const [saveFailed, setSaveFailed] = useState(false);

  // Ref agar timer selalu memakai save() dan signature terbaru
  const saveRef = useRef(save);
  const signatureRef = useRef(signature);
  useEffect(() => {
    saveRef.current = save;
    signatureRef.current = signature;
  });

  const isDirty = signature !== savedSignature;

  const markSaved = useCallback((sig?: string) => {
    setSavedSignature(sig ?? signatureRef.current);
    setLastSaved(new Date());
    setSaveFailed(false);
  }, []);

  const saveNow = useCallback(async () => {
    const sig = signatureRef.current;
    setIsSaving(true);
    try {
      await saveRef.current();
      setSavedSignature(sig);
      setLastSaved(new Date());
      setSaveFailed(false);
    } catch (err) {
      setSaveFailed(true);
      throw err;
    } finally {
      setIsSaving(false);
    }
  }, []);

  useEffect(() => {
    if (!enabled || !isDirty) return;
    const timer = setTimeout(() => {
      saveNow().catch(() => {}); // kegagalan ditampilkan lewat saveFailed
    }, delay);
    return () => clearTimeout(timer);
  }, [signature, isDirty, enabled, delay, saveNow]);

  return { isDirty, isSaving, lastSaved, saveFailed, saveNow, markSaved };
}
