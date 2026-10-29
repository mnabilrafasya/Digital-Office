"use client";

// components/inbox/disposisi-modal.tsx
// Modal Disposisi — kirim ke user/unit dengan catatan

import { useState, useEffect, useCallback } from "react";
import { X, Loader2, Search, UserCheck, Building2, AlertCircle } from "lucide-react";
import { disposeInboxItem } from "@/lib/api/inbox";
import { cn } from "@/lib/utils";

interface UserCandidate {
  id: string;
  name: string;
  positionName: string;
  unitName: string;
}

interface UnitItem {
  id: string;
  code: string;
  name: string;
}

interface Props {
  itemId: string;
  onSuccess: () => void;
  onClose: () => void;
}

const PRESETS = [
  "Tindak lanjuti segera",
  "Pelajari dan laporkan hasilnya",
  "Koordinasikan dengan unit terkait",
  "Untuk diketahui & arsip",
  "Wakili / hadiri kegiatan",
];

export function DisposisiModal({ itemId, onSuccess, onClose }: Props) {
  const [tab, setTab]               = useState<"user" | "unit">("user");
  const [query, setQuery]           = useState("");
  const [users, setUsers]           = useState<UserCandidate[]>([]);
  const [units, setUnits]           = useState<UnitItem[]>([]);
  const [selectedUsers, setSelectedUsers] = useState<UserCandidate[]>([]);
  const [selectedUnits, setSelectedUnits] = useState<UnitItem[]>([]);
  const [note, setNote]             = useState("");
  const [loading, setLoading]       = useState(false);
  const [searching, setSearching]   = useState(false);
  const [error, setError]           = useState<string | null>(null);

  const totalSelected = selectedUsers.length + selectedUnits.length;

  const search = useCallback(async (q: string) => {
    setSearching(true);
    try {
      if (tab === "user") {
        const res = await fetch(`/api/v1/users/candidates?type=REVIEWER&q=${encodeURIComponent(q)}&limit=20`);
        if (res.ok) {
          const json = await res.json() as { data: UserCandidate[] };
          setUsers(json.data);
        }
      } else {
        const res = await fetch(`/api/v1/units?q=${encodeURIComponent(q)}`);
        if (res.ok) {
          const json = await res.json() as { data: UnitItem[] };
          setUnits(json.data);
        }
      }
    } finally {
      setSearching(false);
    }
  }, [tab]);

  useEffect(() => {
    const t = setTimeout(() => search(query), 300);
    return () => clearTimeout(t);
  }, [query, search]);

  async function handleSubmit() {
    if (totalSelected === 0) {
      setError("Pilih minimal satu tujuan.");
      return;
    }
    if (note.length > 1000) {
      setError("Catatan maksimal 1000 karakter.");
      return;
    }
    setLoading(true);
    setError(null);
    try {
      await disposeInboxItem(itemId, {
        userIds: selectedUsers.map((u) => u.id),
        unitIds: selectedUnits.map((u) => u.id),
        note: note.trim() || undefined,
      });
      onSuccess();
    } catch (e) {
      setError((e as Error).message);
    } finally {
      setLoading(false);
    }
  }

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50 backdrop-blur-sm p-4">
      <div className="bg-surface rounded-xl shadow-2xl w-full max-w-lg border border-border overflow-hidden flex flex-col max-h-[90vh]">
        {/* Header */}
        <div className="flex items-center justify-between px-5 py-4 border-b border-border shrink-0">
          <h2 className="text-base font-semibold text-text">Disposisi Surat</h2>
          <button type="button" onClick={onClose} className="text-text-muted hover:text-text">
            <X size={18} />
          </button>
        </div>

        {/* Body */}
        <div className="flex-1 overflow-y-auto p-5 space-y-4">
          {/* Tab */}
          <div className="flex gap-2 border-b border-border">
            {(["user", "unit"] as const).map((t) => (
              <button key={t} type="button" onClick={() => { setTab(t); setQuery(""); }}
                className={cn("px-4 py-2 text-sm font-medium border-b-2 -mb-px transition-colors",
                  tab === t ? "border-primary text-primary" : "border-transparent text-text-muted hover:text-text")}>
                {t === "user" ? "Orang" : "Unit"}
              </button>
            ))}
          </div>

          {/* Search */}
          <div className="relative">
            <Search size={14} className="absolute left-3 top-1/2 -translate-y-1/2 text-text-muted" />
            <input
              value={query}
              onChange={(e) => setQuery(e.target.value)}
              placeholder={tab === "user" ? "Cari nama atau jabatan..." : "Cari nama unit..."}
              className="w-full pl-8 pr-3 py-2 text-sm border border-border rounded-lg bg-surface outline-none focus:border-primary"
            />
          </div>

          {/* Hasil pencarian */}
          <div className="space-y-1 max-h-40 overflow-y-auto">
            {searching && <p className="text-xs text-text-muted text-center py-2">Mencari...</p>}
            {!searching && tab === "user" && users.map((u) => {
              const sel = selectedUsers.some((s) => s.id === u.id);
              return (
                <button key={u.id} type="button"
                  onClick={() => setSelectedUsers(sel ? selectedUsers.filter((s) => s.id !== u.id) : [...selectedUsers, u])}
                  className={cn("w-full flex items-center gap-2 px-3 py-2 rounded-lg text-left text-sm transition-colors",
                    sel ? "bg-primary/10 text-primary" : "hover:bg-panel text-text")}>
                  <UserCheck size={13} className="shrink-0" />
                  <span>{u.name} <span className="text-xs text-text-muted">— {u.positionName}</span></span>
                </button>
              );
            })}
            {!searching && tab === "unit" && units.map((u) => {
              const sel = selectedUnits.some((s) => s.id === u.id);
              return (
                <button key={u.id} type="button"
                  onClick={() => setSelectedUnits(sel ? selectedUnits.filter((s) => s.id !== u.id) : [...selectedUnits, u])}
                  className={cn("w-full flex items-center gap-2 px-3 py-2 rounded-lg text-left text-sm transition-colors",
                    sel ? "bg-primary/10 text-primary" : "hover:bg-panel text-text")}>
                  <Building2 size={13} className="shrink-0" />
                  <span>{u.name}</span>
                </button>
              );
            })}
          </div>

          {/* Terpilih */}
          {totalSelected > 0 && (
            <div className="bg-panel/50 border border-border rounded-lg p-3">
              <p className="text-xs font-semibold text-text-muted mb-1.5">Tujuan dipilih ({totalSelected}/20):</p>
              <div className="flex flex-wrap gap-1">
                {selectedUsers.map((u) => (
                  <span key={u.id} className="inline-flex items-center gap-1 text-xs bg-primary/10 text-primary px-2 py-0.5 rounded-full">
                    {u.name}
                    <button type="button" onClick={() => setSelectedUsers(selectedUsers.filter((s) => s.id !== u.id))}><X size={10} /></button>
                  </span>
                ))}
                {selectedUnits.map((u) => (
                  <span key={u.id} className="inline-flex items-center gap-1 text-xs bg-panel text-text-muted border border-border px-2 py-0.5 rounded-full">
                    {u.name}
                    <button type="button" onClick={() => setSelectedUnits(selectedUnits.filter((s) => s.id !== u.id))}><X size={10} /></button>
                  </span>
                ))}
              </div>
            </div>
          )}

          {/* Preset catatan */}
          <div>
            <p className="text-xs font-semibold text-text-muted mb-1.5">Instruksi cepat:</p>
            <div className="flex flex-wrap gap-1.5">
              {PRESETS.map((p) => (
                <button key={p} type="button" onClick={() => setNote(p)}
                  className={cn("px-2.5 py-1 rounded-lg text-xs border transition-colors",
                    note === p ? "bg-primary/10 border-primary text-primary" : "border-border text-text-muted hover:text-text")}>
                  {p}
                </button>
              ))}
            </div>
          </div>

          {/* Catatan tambahan */}
          <div>
            <label className="text-xs font-semibold text-text-muted block mb-1">Catatan tambahan (opsional)</label>
            <textarea rows={3} value={note} onChange={(e) => setNote(e.target.value)}
              placeholder="Tuliskan instruksi atau arahan tambahan..."
              className="w-full text-sm border border-border rounded-lg p-2.5 outline-none resize-none bg-surface focus:border-primary" />
            <p className="text-[11px] text-text-muted text-right">{note.length}/1000</p>
          </div>

          {error && (
            <div className="flex items-center gap-2 text-xs text-status-danger-text bg-status-danger-bg border border-status-danger-text/20 rounded-lg px-3 py-2">
              <AlertCircle size={13} /> {error}
            </div>
          )}
        </div>

        {/* Footer */}
        <div className="px-5 py-4 border-t border-border flex justify-end gap-2 shrink-0">
          <button type="button" onClick={onClose} disabled={loading}
            className="px-4 py-2 text-sm rounded-lg border border-border text-text hover:bg-panel transition-colors">
            Batal
          </button>
          <button type="button" onClick={handleSubmit} disabled={loading || totalSelected === 0}
            className="flex items-center gap-1.5 px-4 py-2 text-sm font-medium rounded-lg bg-primary text-white hover:bg-primary-dark disabled:opacity-50 transition-colors">
            {loading ? <><Loader2 size={14} className="animate-spin" /> Mengirim...</> : "Disposisikan"}
          </button>
        </div>
      </div>
    </div>
  );
}
