"use client";

// components/inbox/redirect-modal.tsx
// Modal Pengalihan — alihkan ke satu orang

import { useState, useEffect, useCallback } from "react";
import { X, Loader2, Search, UserCheck, AlertCircle } from "lucide-react";
import { redirectInboxItem } from "@/lib/api/inbox";
import { cn } from "@/lib/utils";

interface UserCandidate {
  id: string;
  name: string;
  positionName: string;
  unitName: string;
}

interface Props {
  itemId: string;
  onSuccess: () => void;
  onClose: () => void;
}

export function RedirectModal({ itemId, onSuccess, onClose }: Props) {
  const [query, setQuery]           = useState("");
  const [users, setUsers]           = useState<UserCandidate[]>([]);
  const [selected, setSelected]     = useState<UserCandidate | null>(null);
  const [note, setNote]             = useState("");
  const [loading, setLoading]       = useState(false);
  const [searching, setSearching]   = useState(false);
  const [error, setError]           = useState<string | null>(null);

  const search = useCallback(async (q: string) => {
    setSearching(true);
    try {
      const res = await fetch(`/api/v1/users/candidates?type=REVIEWER&q=${encodeURIComponent(q)}&limit=20`);
      if (res.ok) {
        const json = await res.json() as { data: UserCandidate[] };
        setUsers(json.data);
      }
    } finally {
      setSearching(false);
    }
  }, []);

  useEffect(() => {
    const t = setTimeout(() => search(query), 300);
    return () => clearTimeout(t);
  }, [query, search]);

  async function handleSubmit() {
    if (!selected) { setError("Pilih satu orang tujuan."); return; }
    if (note.length > 1000) { setError("Catatan maksimal 1000 karakter."); return; }
    setLoading(true);
    setError(null);
    try {
      await redirectInboxItem(itemId, { userId: selected.id, note: note.trim() || undefined });
      onSuccess();
    } catch (e) {
      setError((e as Error).message);
    } finally {
      setLoading(false);
    }
  }

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50 backdrop-blur-sm p-4">
      <div className="bg-surface rounded-xl shadow-2xl w-full max-w-md border border-border overflow-hidden">
        <div className="flex items-center justify-between px-5 py-4 border-b border-border">
          <h2 className="text-base font-semibold text-text">Alihkan Surat</h2>
          <button type="button" onClick={onClose} className="text-text-muted hover:text-text"><X size={18} /></button>
        </div>

        <div className="p-5 space-y-4">
          <p className="text-xs text-text-muted">Surat akan dialihkan ke satu orang. Status item Anda akan menjadi &ldquo;Dialihkan&rdquo;.</p>

          <div className="relative">
            <Search size={14} className="absolute left-3 top-1/2 -translate-y-1/2 text-text-muted" />
            <input value={query} onChange={(e) => setQuery(e.target.value)}
              placeholder="Cari nama atau jabatan..."
              className="w-full pl-8 pr-3 py-2 text-sm border border-border rounded-lg bg-surface outline-none focus:border-primary" />
          </div>

          <div className="space-y-1 max-h-40 overflow-y-auto">
            {searching && <p className="text-xs text-text-muted text-center py-2">Mencari...</p>}
            {!searching && users.map((u) => (
              <button key={u.id} type="button" onClick={() => setSelected(u)}
                className={cn("w-full flex items-center gap-2 px-3 py-2 rounded-lg text-left text-sm transition-colors",
                  selected?.id === u.id ? "bg-primary/10 text-primary" : "hover:bg-panel text-text")}>
                <UserCheck size={13} className="shrink-0" />
                <span>{u.name} <span className="text-xs text-text-muted">— {u.positionName}</span></span>
              </button>
            ))}
          </div>

          {selected && (
            <div className="bg-panel/50 border border-border rounded-lg px-3 py-2 text-sm flex items-center gap-2">
              <UserCheck size={13} className="text-primary shrink-0" />
              <span className="text-text font-medium">{selected.name}</span>
              <button type="button" onClick={() => setSelected(null)} className="ml-auto text-text-muted hover:text-text">
                <X size={14} />
              </button>
            </div>
          )}

          <div>
            <label className="text-xs font-semibold text-text-muted block mb-1">Catatan (opsional)</label>
            <textarea rows={3} value={note} onChange={(e) => setNote(e.target.value)}
              placeholder="Tuliskan alasan atau instruksi pengalihan..."
              className="w-full text-sm border border-border rounded-lg p-2.5 outline-none resize-none bg-surface focus:border-primary" />
          </div>

          {error && (
            <div className="flex items-center gap-2 text-xs text-status-danger-text bg-status-danger-bg border border-status-danger-text/20 rounded-lg px-3 py-2">
              <AlertCircle size={13} /> {error}
            </div>
          )}
        </div>

        <div className="px-5 py-4 border-t border-border flex justify-end gap-2">
          <button type="button" onClick={onClose} disabled={loading}
            className="px-4 py-2 text-sm rounded-lg border border-border text-text hover:bg-panel transition-colors">Batal</button>
          <button type="button" onClick={handleSubmit} disabled={loading || !selected}
            className="flex items-center gap-1.5 px-4 py-2 text-sm font-medium rounded-lg bg-primary text-white hover:bg-primary-dark disabled:opacity-50 transition-colors">
            {loading ? <><Loader2 size={14} className="animate-spin" /> Mengirim...</> : "Alihkan"}
          </button>
        </div>
      </div>
    </div>
  );
}
