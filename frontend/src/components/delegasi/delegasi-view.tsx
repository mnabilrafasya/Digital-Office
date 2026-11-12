"use client";

import { useState, useEffect, useCallback } from "react";
import Link from "next/link";
import {
  UserCheck,
  Plus,
  Calendar,
  AlertCircle,
  CheckCircle2,
  Pencil,
  Trash2,
  ChevronRight,
  Search,
  X,
  ArrowRight,
  Users,
  Info,
  Loader2,
  ShieldCheck,
} from "lucide-react";
import { format } from "date-fns";
import { id as idLocale } from "date-fns/locale";
import type { AuthUser } from "@/lib/auth";
import type { CandidateItem } from "@/lib/api/form";
import {
  getMineDelegations,
  createMineDelegation,
  updateMineDelegation,
  deleteMineDelegation,
  getAdminDelegations,
  createAdminDelegation,
  updateAdminDelegation,
  deleteAdminDelegation,
  searchDelegateCandidates,
  type DelegationItem,
} from "@/lib/api/delegasi";
import { cn } from "@/lib/utils";

function formatDateID(dateStr: string): string {
  if (!dateStr) return "—";
  const parts = dateStr.split("-");
  if (parts.length !== 3) return dateStr;
  const d = new Date(Number(parts[0]), Number(parts[1]) - 1, Number(parts[2]));
  if (!Number.isFinite(d.getTime())) return dateStr;
  return format(d, "d MMM yyyy", { locale: idLocale });
}

function getTodayYMD(): string {
  const now = new Date();
  const y = now.getFullYear();
  const m = String(now.getMonth() + 1).padStart(2, "0");
  const d = String(now.getDate()).padStart(2, "0");
  return `${y}-${m}-${d}`;
}

function StatusBadge({ status }: { status: DelegationItem["status"] }) {
  const cfg =
    status === "ACTIVE"
      ? {
          label: "Berlaku",
          cls: "bg-status-success-bg text-status-success-text border-status-success-text/20",
          dot: "bg-status-success-text",
        }
      : status === "UPCOMING"
        ? {
            label: "Akan datang",
            cls: "bg-status-info-bg text-status-info-text border-status-info-text/20",
            dot: "bg-status-info-text",
          }
        : {
            label: "Selesai",
            cls: "bg-panel text-text-muted border-border",
            dot: "bg-text-muted",
          };

  return (
    <span
      className={cn(
        "inline-flex items-center gap-1.5 px-2.5 py-0.5 rounded-full text-xs font-semibold border whitespace-nowrap",
        cfg.cls
      )}
    >
      <span className={cn("w-1.5 h-1.5 rounded-full", cfg.dot)} />
      {cfg.label}
    </span>
  );
}

// ─── Komponen Pemilihan Pegawai / Pengganti ───────────────────

function CandidateSelector({
  label,
  required = false,
  selected,
  onSelect,
  excludeIds = [],
  placeholder = "Cari nama, jabatan, atau unit...",
  disabled = false,
}: {
  label: string;
  required?: boolean;
  selected: { id: string; name: string; positionName?: string } | null;
  onSelect: (person: { id: string; name: string; positionName: string } | null) => void;
  excludeIds?: string[];
  placeholder?: string;
  disabled?: boolean;
}) {
  const [open, setOpen] = useState(false);
  const [query, setQuery] = useState("");
  const [items, setItems] = useState<CandidateItem[]>([]);
  const [loading, setLoading] = useState(false);

  useEffect(() => {
    if (!open || disabled) return;
    let cancelled = false;
    const timer = setTimeout(() => {
      setLoading(true);
      searchDelegateCandidates(query.trim() || undefined)
        .then((res) => {
          if (!cancelled) setItems(res);
        })
        .catch(() => {
          if (!cancelled) setItems([]);
        })
        .finally(() => {
          if (!cancelled) setLoading(false);
        });
    }, 250);
    return () => {
      cancelled = true;
      clearTimeout(timer);
    };
  }, [open, query, disabled]);

  const filtered = items.filter((item) => !excludeIds.includes(String(item.id)));

  return (
    <div className="space-y-1.5">
      <label className="block text-xs font-semibold text-text">
        {label} {required && <span className="text-status-danger-text">*</span>}
      </label>

      {selected ? (
        <div className="flex items-center justify-between gap-2 rounded-lg border border-primary/30 bg-primary/5 px-3 py-2">
          <div className="min-w-0">
            <p className="text-sm font-semibold text-text truncate">{selected.name}</p>
            {selected.positionName && (
              <p className="text-xs text-text-muted truncate">{selected.positionName}</p>
            )}
          </div>
          {!disabled && (
            <button
              type="button"
              onClick={() => {
                onSelect(null);
                setOpen(true);
              }}
              className="text-xs font-medium text-primary hover:underline shrink-0"
            >
              Ganti
            </button>
          )}
        </div>
      ) : (
        <div className="relative">
          <div className="relative">
            <Search
              size={14}
              className="absolute left-3 top-1/2 -translate-y-1/2 text-text-muted pointer-events-none"
            />
            <input
              type="text"
              value={query}
              onFocus={() => setOpen(true)}
              onChange={(e) => {
                setQuery(e.target.value);
                setOpen(true);
              }}
              placeholder={placeholder}
              disabled={disabled}
              className="w-full rounded-lg border border-border bg-surface pl-9 pr-3 py-2 text-sm text-text placeholder:text-text-muted/70 focus:outline-none focus:ring-2 focus:ring-primary/30 focus:border-primary/50"
            />
          </div>

          {open && !disabled && (
            <div className="mt-1 rounded-xl border border-border bg-surface shadow-lg max-h-52 overflow-y-auto divide-y divide-border/60 z-20">
              {loading ? (
                <div className="p-3 flex items-center gap-2 text-xs text-text-muted">
                  <Loader2 size={14} className="animate-spin text-primary" />
                  <span>Mencari pegawai...</span>
                </div>
              ) : filtered.length === 0 ? (
                <div className="p-3 text-xs text-text-muted text-center">
                  Pegawai tidak ditemukan.
                </div>
              ) : (
                filtered.map((c) => (
                  <button
                    key={c.id}
                    type="button"
                    onClick={() => {
                      onSelect({
                        id: String(c.id),
                        name: c.name,
                        positionName: c.positionName,
                      });
                      setQuery("");
                      setOpen(false);
                    }}
                    className="w-full text-left px-3 py-2 hover:bg-panel/60 transition-colors"
                  >
                    <p className="text-xs font-semibold text-text">{c.name}</p>
                    <p className="text-[11px] text-text-muted">
                      {c.positionName} · {c.unitName}
                    </p>
                  </button>
                ))
              )}
            </div>
          )}
        </div>
      )}
    </div>
  );
}

// ─── Modal Form Tambah / Ubah Delegasi ────────────────────────

function DelegationFormModal({
  mode,
  initial,
  currentUserId,
  onClose,
  onSuccess,
}: {
  mode: "create-mine" | "edit-mine" | "create-admin" | "edit-admin";
  initial?: DelegationItem;
  currentUserId: string;
  onClose: () => void;
  onSuccess: (msg: string) => void;
}) {
  const isAdminMode = mode === "create-admin" || mode === "edit-admin";
  const isEdit = mode === "edit-mine" || mode === "edit-admin";
  const today = getTodayYMD();

  const [targetUser, setTargetUser] = useState<{
    id: string;
    name: string;
    positionName: string;
  } | null>(
    initial
      ? {
          id: initial.user.id,
          name: initial.user.name,
          positionName: initial.user.positionName,
        }
      : null
  );
  const [substitute, setSubstitute] = useState<{
    id: string;
    name: string;
    positionName: string;
  } | null>(
    initial
      ? {
          id: initial.delegate.id,
          name: initial.delegate.name,
          positionName: initial.delegate.positionName,
        }
      : null
  );
  const [fromDate, setFromDate] = useState(initial?.from ?? today);
  const [untilDate, setUntilDate] = useState(initial?.until ?? today);
  const [note, setNote] = useState(initial?.note ?? "");
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault();
    setError("");

    if (mode === "create-admin" && !targetUser) {
      setError("Pegawai yang mendelegasikan wajib dipilih.");
      return;
    }
    if (!substitute) {
      setError("Pengganti wajib dipilih.");
      return;
    }
    if (!fromDate || !untilDate) {
      setError("Tanggal mulai dan tanggal selesai wajib diisi.");
      return;
    }
    if (fromDate > untilDate) {
      setError("Tanggal mulai tidak boleh setelah tanggal selesai.");
      return;
    }
    if (untilDate < today) {
      setError("Tanggal selesai tidak boleh sebelum hari ini.");
      return;
    }

    setBusy(true);
    try {
      const cleanNote = note.trim() || undefined;
      if (mode === "create-mine") {
        await createMineDelegation({
          substituteId: substitute.id,
          from: fromDate,
          until: untilDate,
          note: cleanNote,
        });
        onSuccess("Jadwal delegasi Anda berhasil dibuat.");
      } else if (mode === "edit-mine" && initial) {
        await updateMineDelegation(initial.id, {
          substituteId: substitute.id,
          from: fromDate,
          until: untilDate,
          note: cleanNote,
        });
        onSuccess("Jadwal delegasi berhasil diperbarui.");
      } else if (mode === "create-admin" && targetUser) {
        await createAdminDelegation({
          userId: targetUser.id,
          substituteId: substitute.id,
          from: fromDate,
          until: untilDate,
          note: cleanNote,
        });
        onSuccess(`Jadwal delegasi untuk ${targetUser.name} berhasil dibuat.`);
      } else if (mode === "edit-admin" && initial) {
        await updateAdminDelegation(initial.id, {
          substituteId: substitute.id,
          from: fromDate,
          until: untilDate,
          note: cleanNote,
        });
        onSuccess(
          `Jadwal delegasi untuk ${initial.user.name} berhasil diperbarui.`
        );
      }
      onClose();
    } catch (err) {
      setError(
        err instanceof Error ? err.message : "Gagal menyimpan jadwal delegasi."
      );
    } finally {
      setBusy(false);
    }
  }

  const excludedForSubstitute = isAdminMode
    ? targetUser
      ? [targetUser.id]
      : []
    : [currentUserId];

  return (
    <div className="fixed inset-0 z-50 bg-black/50 backdrop-blur-[1px] flex items-center justify-center p-4 overflow-y-auto">
      <div className="w-full max-w-lg rounded-2xl border border-border bg-surface shadow-2xl overflow-hidden">
        <div className="flex items-center justify-between px-5 py-4 border-b border-border bg-panel/40">
          <div className="flex items-center gap-2.5">
            <div className="p-2 rounded-xl bg-primary/10 text-primary">
              <UserCheck size={18} />
            </div>
            <div>
              <h2 className="text-base font-bold text-text">
                {isEdit ? "Ubah Jadwal Delegasi" : "Tambah Jadwal Delegasi"}
              </h2>
              <p className="text-xs text-text-muted">
                {isAdminMode
                  ? "Pengelolaan jadwal delegasi tugas pegawai perusahaan."
                  : "Limpahkan tugas review, approval, dan tembusan inbox sementara."}
              </p>
            </div>
          </div>
          <button
            type="button"
            onClick={onClose}
            className="p-1.5 rounded-lg text-text-muted hover:text-text hover:bg-panel transition-colors"
            aria-label="Tutup modal"
          >
            <X size={16} />
          </button>
        </div>

        <form onSubmit={handleSubmit} className="p-5 space-y-4">
          {error && (
            <div
              role="alert"
              className="flex items-start gap-2 rounded-xl bg-status-danger-bg border border-status-danger-text/30 px-3.5 py-3 text-xs text-status-danger-text"
            >
              <AlertCircle size={15} className="shrink-0 mt-0.5" />
              <span>{error}</span>
            </div>
          )}

          {isAdminMode && (
            <CandidateSelector
              label="Pegawai yang Mendelegasikan"
              required
              selected={targetUser}
              onSelect={(person) => {
                setTargetUser(person);
                if (person && substitute?.id === person.id) {
                  setSubstitute(null);
                }
              }}
              excludeIds={substitute ? [substitute.id] : []}
              disabled={isEdit}
            />
          )}

          <CandidateSelector
            label="Pengganti (Yang Mewakili)"
            required
            selected={substitute}
            onSelect={setSubstitute}
            excludeIds={excludedForSubstitute}
          />

          <div className="grid grid-cols-1 sm:grid-cols-2 gap-3.5">
            <div className="space-y-1.5">
              <label className="block text-xs font-semibold text-text">
                Tanggal Mulai <span className="text-status-danger-text">*</span>
              </label>
              <input
                type="date"
                required
                value={fromDate}
                onChange={(e) => {
                  const val = e.target.value;
                  setFromDate(val);
                  if (untilDate < val) setUntilDate(val);
                }}
                className="w-full rounded-lg border border-border bg-surface px-3 py-2 text-sm text-text focus:outline-none focus:ring-2 focus:ring-primary/30 focus:border-primary/50"
              />
            </div>

            <div className="space-y-1.5">
              <div className="flex items-center justify-between gap-2">
                <label className="block text-xs font-semibold text-text">
                  Tanggal Selesai <span className="text-status-danger-text">*</span>
                </label>
                {isEdit && untilDate !== today && (
                  <button
                    type="button"
                    onClick={() => setUntilDate(today)}
                    className="text-[11px] font-semibold text-primary hover:underline"
                  >
                    Akhiri Hari Ini
                  </button>
                )}
              </div>
              <input
                type="date"
                required
                min={fromDate > today ? fromDate : today}
                value={untilDate}
                onChange={(e) => setUntilDate(e.target.value)}
                className="w-full rounded-lg border border-border bg-surface px-3 py-2 text-sm text-text focus:outline-none focus:ring-2 focus:ring-primary/30 focus:border-primary/50"
              />
            </div>
          </div>

          <div className="space-y-1.5">
            <div className="flex items-center justify-between">
              <label className="block text-xs font-semibold text-text">
                Keterangan (Opsional)
              </label>
              <span className="text-[11px] text-text-muted">
                {note.length} / 200
              </span>
            </div>
            <textarea
              rows={2}
              maxLength={200}
              value={note}
              onChange={(e) => setNote(e.target.value)}
              placeholder="Contoh: Dinas luar kota / Cuti tahunan"
              className="w-full rounded-lg border border-border bg-surface px-3 py-2 text-sm text-text placeholder:text-text-muted/70 focus:outline-none focus:ring-2 focus:ring-primary/30 focus:border-primary/50 resize-none"
            />
          </div>

          <div className="flex items-center justify-end gap-2 pt-2 border-t border-border">
            <button
              type="button"
              onClick={onClose}
              disabled={busy}
              className="px-4 py-2 rounded-lg border border-border bg-surface text-xs font-semibold text-text hover:bg-panel transition-colors"
            >
              Batal
            </button>
            <button
              type="submit"
              disabled={busy}
              className="inline-flex items-center gap-1.5 px-4 py-2 rounded-lg bg-primary text-white text-xs font-semibold hover:bg-primary-dark transition-colors disabled:opacity-50"
            >
              {busy ? (
                <>
                  <Loader2 size={14} className="animate-spin" />
                  Menyimpan...
                </>
              ) : isEdit ? (
                "Simpan Perubahan"
              ) : (
                "Simpan Delegasi"
              )}
            </button>
          </div>
        </form>
      </div>
    </div>
  );
}

// ─── Komponen Utama Halaman Delegasi ──────────────────────────

export function DelegasiView({ user }: { user: AuthUser }) {
  const canManageAll = user.permissions.includes("USER_MANAGE");
  const [activeTab, setActiveTab] = useState<"mine" | "admin">("mine");

  // Data saya
  const [mineData, setMineData] = useState<{
    delegations: DelegationItem[];
    representing: DelegationItem[];
  }>({ delegations: [], representing: [] });
  const [loadingMine, setLoadingMine] = useState(true);
  const [mineError, setMineError] = useState("");

  // Data admin
  const [adminStatus, setAdminStatus] = useState<string>("");
  const [adminUserFilter, setAdminUserFilter] = useState<{
    id: string;
    name: string;
    positionName: string;
  } | null>(null);
  const [adminPage, setAdminPage] = useState(1);
  const [adminItems, setAdminItems] = useState<DelegationItem[]>([]);
  const [adminTotal, setAdminTotal] = useState(0);
  const [loadingAdmin, setLoadingAdmin] = useState(false);
  const [adminError, setAdminError] = useState("");

  // Modal & Notifikasi
  const [modalState, setModalState] = useState<{
    mode: "create-mine" | "edit-mine" | "create-admin" | "edit-admin";
    item?: DelegationItem;
  } | null>(null);
  const [deleteTarget, setDeleteTarget] = useState<{
    scope: "mine" | "admin";
    item: DelegationItem;
  } | null>(null);
  const [deleting, setDeleting] = useState(false);
  const [deleteError, setDeleteError] = useState("");
  const [notice, setNotice] = useState("");

  const [mineRefreshKey, setMineRefreshKey] = useState(0);
  const [adminRefreshKey, setAdminRefreshKey] = useState(0);

  useEffect(() => {
    let cancelled = false;
    getMineDelegations()
      .then((res) => {
        if (!cancelled) {
          setMineData({
            delegations: res.delegations ?? [],
            representing: res.representing ?? [],
          });
          setMineError("");
        }
      })
      .catch((err) => {
        if (!cancelled) {
          setMineError(
            err instanceof Error ? err.message : "Gagal memuat data delegasi."
          );
        }
      })
      .finally(() => {
        if (!cancelled) setLoadingMine(false);
      });
    return () => {
      cancelled = true;
    };
  }, [mineRefreshKey]);

  useEffect(() => {
    if (!canManageAll || activeTab !== "admin") return;
    let cancelled = false;
    getAdminDelegations({
      status: adminStatus || undefined,
      userId: adminUserFilter?.id || undefined,
      page: adminPage,
      pageSize: 15,
    })
      .then((res) => {
        if (!cancelled) {
          setAdminItems(res.data ?? []);
          setAdminTotal(res.meta?.total ?? 0);
          setAdminError("");
        }
      })
      .catch((err) => {
        if (!cancelled) {
          setAdminError(
            err instanceof Error
              ? err.message
              : "Gagal memuat daftar delegasi perusahaan."
          );
        }
      })
      .finally(() => {
        if (!cancelled) setLoadingAdmin(false);
      });
    return () => {
      cancelled = true;
    };
  }, [
    canManageAll,
    activeTab,
    adminStatus,
    adminUserFilter,
    adminPage,
    adminRefreshKey,
  ]);

  const refreshMine = useCallback(() => {
    setLoadingMine(true);
    setMineRefreshKey((k) => k + 1);
  }, []);

  const refreshAdmin = useCallback(() => {
    setLoadingAdmin(true);
    setAdminRefreshKey((k) => k + 1);
  }, []);

  async function handleConfirmDelete() {
    if (!deleteTarget) return;
    setDeleting(true);
    setDeleteError("");
    try {
      if (deleteTarget.scope === "mine") {
        await deleteMineDelegation(deleteTarget.item.id);
      } else {
        await deleteAdminDelegation(deleteTarget.item.id);
      }
      setNotice("Jadwal delegasi berhasil dicabut/dihapus.");
      setDeleteTarget(null);
      refreshMine();
      if (canManageAll && activeTab === "admin") {
        refreshAdmin();
      }
    } catch (err) {
      setDeleteError(
        err instanceof Error ? err.message : "Gagal menghapus delegasi."
      );
    } finally {
      setDeleting(false);
    }
  }

  const activeSelfDelegation = mineData.delegations.find(
    (d) => d.status === "ACTIVE"
  );
  const adminTotalPages = Math.max(1, Math.ceil(adminTotal / 15));

  return (
    <div className="space-y-6 text-text">
      {/* Header & Breadcrumb */}
      <header className="space-y-3">
        <nav
          aria-label="Breadcrumb"
          className="flex items-center gap-1.5 text-xs text-text-muted"
        >
          <Link href="/dashboard" className="hover:text-primary transition-colors">
            Beranda
          </Link>
          <ChevronRight size={12} />
          <span className="text-text font-semibold">Manajemen Delegasi</span>
        </nav>

        <div className="flex flex-wrap items-start justify-between gap-4">
          <div className="space-y-1">
            <h1 className="text-2xl font-bold tracking-tight">
              Manajemen Delegasi Tugas
            </h1>
            <p className="text-sm text-text-muted max-w-2xl">
              Selama periode delegasi, tugas review dan approval Anda akan dikerjakan
              oleh pengganti atas nama Anda, serta surat masuk ke Inbox ditembuskan ke
              pengganti.
            </p>
          </div>

          <button
            type="button"
            onClick={() =>
              setModalState({
                mode: activeTab === "admin" ? "create-admin" : "create-mine",
              })
            }
            className="inline-flex items-center gap-2 rounded-xl bg-primary px-4 py-2.5 text-xs font-semibold text-white hover:bg-primary-dark transition-colors shadow-xs shrink-0"
          >
            <Plus size={15} />
            {activeTab === "admin"
              ? "Tambah Delegasi Pegawai"
              : "Tambah Delegasi Saya"}
          </button>
        </div>
      </header>

      {/* Notifikasi Sukses */}
      {notice && (
        <div
          role="status"
          className="flex items-center justify-between gap-3 rounded-xl border border-status-success-text/20 bg-status-success-bg px-4 py-3 text-xs font-medium text-status-success-text"
        >
          <div className="flex items-center gap-2">
            <CheckCircle2 size={16} />
            <span>{notice}</span>
          </div>
          <button
            type="button"
            onClick={() => setNotice("")}
            className="opacity-75 hover:opacity-100"
          >
            <X size={14} />
          </button>
        </div>
      )}

      {/* Banner Status Delegasi Aktif Pribadi */}
      {activeSelfDelegation && (
        <div className="flex flex-wrap items-center justify-between gap-3 rounded-xl border border-status-warn-text/30 bg-status-warn-bg/60 px-4 py-3.5 text-xs text-status-warn-text">
          <div className="flex items-start gap-2.5">
            <AlertCircle size={17} className="shrink-0 mt-0.5" />
            <div>
              <p className="font-bold text-sm">
                Tugas Anda sedang didelegasikan ke{" "}
                {activeSelfDelegation.delegate.name}
              </p>
              <p className="mt-0.5 text-text">
                Berlaku mulai{" "}
                <strong>{formatDateID(activeSelfDelegation.from)}</strong> sampai{" "}
                <strong>{formatDateID(activeSelfDelegation.until)}</strong>
                {activeSelfDelegation.note
                  ? ` · Keterangan: "${activeSelfDelegation.note}"`
                  : ""}
              </p>
            </div>
          </div>
          <button
            type="button"
            onClick={() =>
              setModalState({ mode: "edit-mine", item: activeSelfDelegation })
            }
            className="inline-flex items-center gap-1.5 rounded-lg border border-status-warn-text/30 bg-surface px-3 py-1.5 text-xs font-semibold text-text hover:border-primary/40 hover:text-primary transition-colors"
          >
            <Pencil size={12} />
            Ubah / Akhiri Lebih Awal
          </button>
        </div>
      )}

      {/* Kartu Saya Mewakili */}
      {mineData.representing.length > 0 && (
        <section className="space-y-3">
          <div className="flex items-center gap-2">
            <Users size={16} className="text-primary" />
            <h2 className="text-sm font-bold text-text">
              Saya Mewakili ({mineData.representing.length} Rekan Kerja)
            </h2>
          </div>

          <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-3.5">
            {mineData.representing.map((rep) => (
              <div
                key={rep.id}
                className="rounded-xl border border-primary/25 bg-primary/5 p-4 space-y-2.5"
              >
                <div className="flex items-start justify-between gap-2">
                  <div className="min-w-0">
                    <p className="text-sm font-bold text-text truncate">
                      {rep.user.name}
                    </p>
                    <p className="text-xs text-text-muted truncate">
                      {rep.user.positionName}
                    </p>
                  </div>
                  <StatusBadge status={rep.status} />
                </div>

                <div className="flex items-center gap-1.5 text-xs text-text-muted pt-1">
                  <Calendar size={13} className="text-primary shrink-0" />
                  <span>
                    {formatDateID(rep.from)} – {formatDateID(rep.until)}
                  </span>
                </div>

                {rep.note && (
                  <p className="text-xs text-text-muted bg-surface/70 rounded-lg px-2.5 py-1.5 border border-border/60 italic">
                    &ldquo;{rep.note}&rdquo;
                  </p>
                )}

                <p className="text-[11px] text-primary font-medium flex items-center gap-1 pt-0.5">
                  <Info size={12} className="shrink-0" />
                  Tugas &amp; tembusan surat masuk tersedia di menu Review / Approval / Inbox.
                </p>
              </div>
            ))}
          </div>
        </section>
      )}

      {/* Tabs untuk Admin (USER_MANAGE) */}
      {canManageAll && (
        <div className="flex items-center gap-2 border-b border-border">
          <button
            type="button"
            onClick={() => setActiveTab("mine")}
            className={cn(
              "px-4 py-2.5 text-xs font-semibold border-b-2 transition-colors flex items-center gap-1.5",
              activeTab === "mine"
                ? "border-primary text-primary"
                : "border-transparent text-text-muted hover:text-text"
            )}
          >
            <UserCheck size={14} />
            Jadwal Delegasi Saya
          </button>
          <button
            type="button"
            onClick={() => {
              setActiveTab("admin");
              setLoadingAdmin(true);
            }}
            className={cn(
              "px-4 py-2.5 text-xs font-semibold border-b-2 transition-colors flex items-center gap-1.5",
              activeTab === "admin"
                ? "border-primary text-primary"
                : "border-transparent text-text-muted hover:text-text"
            )}
          >
            <ShieldCheck size={14} />
            Semua Delegasi Perusahaan ({user.company.code})
          </button>
        </div>
      )}

      {/* Konten Tab 1: Jadwal Delegasi Saya */}
      {activeTab === "mine" && (
        <section className="rounded-xl border border-border bg-surface overflow-hidden shadow-2xs">
          <div className="px-5 py-3.5 border-b border-border bg-panel/40 flex flex-wrap items-center justify-between gap-2">
            <div>
              <h2 className="text-sm font-bold text-text">
                Jadwal Delegasi Saya (Berlaku &amp; Akan Datang)
              </h2>
              <p className="text-xs text-text-muted">
                Daftar pelimpahan tugas Anda ke rekan kerja pengganti.
              </p>
            </div>
          </div>

          {loadingMine ? (
            <div className="p-10 flex items-center justify-center gap-2 text-xs text-text-muted">
              <Loader2 size={16} className="animate-spin text-primary" />
              Memuat jadwal delegasi...
            </div>
          ) : mineError ? (
            <div className="p-8 text-center space-y-2">
              <p className="text-sm text-status-danger-text font-medium">
                {mineError}
              </p>
              <button
                type="button"
                onClick={refreshMine}
                className="text-xs text-primary underline font-semibold"
              >
                Coba lagi
              </button>
            </div>
          ) : mineData.delegations.length === 0 ? (
            <div className="p-12 text-center space-y-3">
              <div className="w-12 h-12 rounded-2xl bg-panel border border-border flex items-center justify-center mx-auto text-text-muted">
                <UserCheck size={22} />
              </div>
              <div>
                <p className="text-sm font-semibold text-text">
                  Belum ada jadwal delegasi aktif atau mendatang
                </p>
                <p className="text-xs text-text-muted mt-1 max-w-md mx-auto">
                  Jika Anda akan dinas luar kota atau cuti, buat jadwal delegasi agar
                  alur persuratan tidak tertunda.
                </p>
              </div>
              <button
                type="button"
                onClick={() => setModalState({ mode: "create-mine" })}
                className="inline-flex items-center gap-1.5 rounded-lg border border-primary/30 bg-primary/10 px-3.5 py-2 text-xs font-semibold text-primary hover:bg-primary hover:text-white transition-colors"
              >
                <Plus size={14} />
                Buat Jadwal Delegasi
              </button>
            </div>
          ) : (
            <div className="overflow-x-auto">
              <table className="w-full text-sm text-left">
                <thead className="bg-panel/60 text-[11px] uppercase tracking-wider text-text-muted border-b border-border">
                  <tr>
                    <th className="px-4 py-3 font-semibold">Pengganti</th>
                    <th className="px-4 py-3 font-semibold">Periode</th>
                    <th className="px-4 py-3 font-semibold">Status</th>
                    <th className="px-4 py-3 font-semibold">Keterangan</th>
                    <th className="px-4 py-3 font-semibold">Diisi Oleh</th>
                    <th className="px-4 py-3 font-semibold text-right">Tindakan</th>
                  </tr>
                </thead>
                <tbody className="divide-y divide-border">
                  {mineData.delegations.map((item) => {
                    const editable =
                      item.status === "ACTIVE" || item.status === "UPCOMING";
                    return (
                      <tr key={item.id} className="hover:bg-panel/30 transition-colors">
                        <td className="px-4 py-3">
                          <p className="font-semibold text-text">
                            {item.delegate.name}
                          </p>
                          <p className="text-xs text-text-muted">
                            {item.delegate.positionName}
                          </p>
                        </td>
                        <td className="px-4 py-3 whitespace-nowrap text-xs">
                          <span className="font-medium text-text">
                            {formatDateID(item.from)}
                          </span>
                          <span className="text-text-muted mx-1.5">s.d.</span>
                          <span className="font-medium text-text">
                            {formatDateID(item.until)}
                          </span>
                        </td>
                        <td className="px-4 py-3 whitespace-nowrap">
                          <StatusBadge status={item.status} />
                        </td>
                        <td className="px-4 py-3 text-xs text-text-muted max-w-xs">
                          {item.note || <span className="italic opacity-60">—</span>}
                        </td>
                        <td className="px-4 py-3 text-xs text-text-muted">
                          {item.createdBy.name}
                        </td>
                        <td className="px-4 py-3 text-right whitespace-nowrap">
                          {editable && (
                            <div className="inline-flex items-center gap-1.5">
                              <button
                                type="button"
                                onClick={() =>
                                  setModalState({ mode: "edit-mine", item })
                                }
                                className="inline-flex items-center gap-1 rounded-lg border border-border bg-surface px-2.5 py-1.5 text-xs font-semibold text-text hover:border-primary/40 hover:text-primary transition-colors"
                              >
                                <Pencil size={12} />
                                Ubah
                              </button>
                              <button
                                type="button"
                                onClick={() => {
                                  setDeleteError("");
                                  setDeleteTarget({ scope: "mine", item });
                                }}
                                className="inline-flex items-center gap-1 rounded-lg border border-border bg-surface px-2.5 py-1.5 text-xs font-semibold text-text-muted hover:border-status-danger-text/50 hover:bg-status-danger-bg/50 hover:text-status-danger-text transition-colors"
                              >
                                <Trash2 size={12} />
                                Cabut
                              </button>
                            </div>
                          )}
                        </td>
                      </tr>
                    );
                  })}
                </tbody>
              </table>
            </div>
          )}
        </section>
      )}

      {/* Konten Tab 2: Admin - Semua Delegasi Perusahaan */}
      {canManageAll && activeTab === "admin" && (
        <section className="rounded-xl border border-border bg-surface overflow-hidden shadow-2xs">
          <div className="p-4 border-b border-border bg-panel/40 flex flex-wrap items-center justify-between gap-3">
            {/* Filter Status */}
            <div className="flex flex-wrap items-center gap-1.5">
              {[
                { key: "", label: "Semua" },
                { key: "ACTIVE", label: "Berlaku" },
                { key: "UPCOMING", label: "Akan datang" },
                { key: "ENDED", label: "Selesai" },
              ].map((st) => (
                <button
                  key={st.key}
                  type="button"
                  onClick={() => {
                    setAdminStatus(st.key);
                    setAdminPage(1);
                  }}
                  className={cn(
                    "px-3 py-1.5 rounded-lg text-xs font-semibold border transition-colors",
                    adminStatus === st.key
                      ? "bg-primary text-white border-primary"
                      : "bg-surface text-text-muted border-border hover:text-text"
                  )}
                >
                  {st.label}
                </button>
              ))}
            </div>

            {/* Filter Pegawai */}
            <div className="w-full sm:w-72">
              {adminUserFilter ? (
                <div className="flex items-center justify-between gap-2 rounded-lg border border-primary/30 bg-primary/5 px-3 py-1.5 text-xs">
                  <span className="truncate font-medium text-text">
                    Filter: <strong>{adminUserFilter.name}</strong>
                  </span>
                  <button
                    type="button"
                    onClick={() => {
                      setAdminUserFilter(null);
                      setAdminPage(1);
                    }}
                    className="text-primary hover:underline font-semibold shrink-0"
                  >
                    Reset
                  </button>
                </div>
              ) : (
                <CandidateSelector
                  label=""
                  selected={null}
                  onSelect={(p) => {
                    setAdminUserFilter(p);
                    setAdminPage(1);
                  }}
                  placeholder="Saring pegawai / pengganti..."
                />
              )}
            </div>
          </div>

          {loadingAdmin ? (
            <div className="p-10 flex items-center justify-center gap-2 text-xs text-text-muted">
              <Loader2 size={16} className="animate-spin text-primary" />
              Memuat seluruh jadwal delegasi perusahaan...
            </div>
          ) : adminError ? (
            <div className="p-8 text-center space-y-2">
              <p className="text-sm text-status-danger-text font-medium">
                {adminError}
              </p>
              <button
                type="button"
                onClick={refreshAdmin}
                className="text-xs text-primary underline font-semibold"
              >
                Coba lagi
              </button>
            </div>
          ) : adminItems.length === 0 ? (
            <div className="p-12 text-center space-y-2">
              <p className="text-sm font-semibold text-text">
                Tidak ada jadwal delegasi yang cocok
              </p>
              <p className="text-xs text-text-muted">
                Ubah filter status/pegawai atau buat jadwal delegasi baru untuk pegawai.
              </p>
            </div>
          ) : (
            <>
              <div className="overflow-x-auto">
                <table className="w-full text-sm text-left">
                  <thead className="bg-panel/60 text-[11px] uppercase tracking-wider text-text-muted border-b border-border">
                    <tr>
                      <th className="px-4 py-3 font-semibold">Pegawai</th>
                      <th className="px-4 py-3 font-semibold">Pengganti</th>
                      <th className="px-4 py-3 font-semibold">Periode</th>
                      <th className="px-4 py-3 font-semibold">Status</th>
                      <th className="px-4 py-3 font-semibold">Keterangan</th>
                      <th className="px-4 py-3 font-semibold">Diisi Oleh</th>
                      <th className="px-4 py-3 font-semibold text-right">
                        Tindakan
                      </th>
                    </tr>
                  </thead>
                  <tbody className="divide-y divide-border">
                    {adminItems.map((item) => {
                      const editable =
                        item.status === "ACTIVE" || item.status === "UPCOMING";
                      return (
                        <tr
                          key={item.id}
                          className="hover:bg-panel/30 transition-colors"
                        >
                          <td className="px-4 py-3">
                            <p className="font-semibold text-text">
                              {item.user.name}
                            </p>
                            <p className="text-xs text-text-muted">
                              {item.user.positionName}
                            </p>
                          </td>
                          <td className="px-4 py-3">
                            <div className="flex items-center gap-1.5">
                              <ArrowRight
                                size={13}
                                className="text-primary shrink-0 hidden sm:inline"
                              />
                              <div>
                                <p className="font-semibold text-text">
                                  {item.delegate.name}
                                </p>
                                <p className="text-xs text-text-muted">
                                  {item.delegate.positionName}
                                </p>
                              </div>
                            </div>
                          </td>
                          <td className="px-4 py-3 whitespace-nowrap text-xs">
                            <span className="font-medium text-text">
                              {formatDateID(item.from)}
                            </span>
                            <span className="text-text-muted mx-1.5">s.d.</span>
                            <span className="font-medium text-text">
                              {formatDateID(item.until)}
                            </span>
                          </td>
                          <td className="px-4 py-3 whitespace-nowrap">
                            <StatusBadge status={item.status} />
                          </td>
                          <td className="px-4 py-3 text-xs text-text-muted max-w-xs">
                            {item.note || (
                              <span className="italic opacity-60">—</span>
                            )}
                          </td>
                          <td className="px-4 py-3 text-xs text-text-muted">
                            {item.createdBy.name}
                          </td>
                          <td className="px-4 py-3 text-right whitespace-nowrap">
                            {editable ? (
                              <div className="inline-flex items-center gap-1.5">
                                <button
                                  type="button"
                                  onClick={() =>
                                    setModalState({ mode: "edit-admin", item })
                                  }
                                  className="inline-flex items-center gap-1 rounded-lg border border-border bg-surface px-2.5 py-1.5 text-xs font-semibold text-text hover:border-primary/40 hover:text-primary transition-colors"
                                >
                                  <Pencil size={12} />
                                  Ubah
                                </button>
                                <button
                                  type="button"
                                  onClick={() => {
                                    setDeleteError("");
                                    setDeleteTarget({ scope: "admin", item });
                                  }}
                                  className="inline-flex items-center gap-1 rounded-lg border border-border bg-surface px-2.5 py-1.5 text-xs font-semibold text-text-muted hover:border-status-danger-text/50 hover:bg-status-danger-bg/50 hover:text-status-danger-text transition-colors"
                                >
                                  <Trash2 size={12} />
                                  Cabut
                                </button>
                              </div>
                            ) : (
                              <span className="text-[11px] text-text-muted italic">
                                Selesai
                              </span>
                            )}
                          </td>
                        </tr>
                      );
                    })}
                  </tbody>
                </table>
              </div>

              {adminTotal > 15 && (
                <div className="px-4 py-3 border-t border-border bg-panel/30 flex items-center justify-between text-xs text-text-muted">
                  <span>
                    Total <strong className="text-text">{adminTotal}</strong>{" "}
                    jadwal
                  </span>
                  <div className="flex items-center gap-1.5">
                    <button
                      type="button"
                      disabled={adminPage <= 1}
                      onClick={() => setAdminPage((p) => Math.max(1, p - 1))}
                      className="px-2.5 py-1 rounded-md border border-border bg-surface text-text disabled:opacity-40"
                    >
                      Sebelumnya
                    </button>
                    <span className="px-2">
                      {adminPage} / {adminTotalPages}
                    </span>
                    <button
                      type="button"
                      disabled={adminPage >= adminTotalPages}
                      onClick={() =>
                        setAdminPage((p) => Math.min(adminTotalPages, p + 1))
                      }
                      className="px-2.5 py-1 rounded-md border border-border bg-surface text-text disabled:opacity-40"
                    >
                      Berikutnya
                    </button>
                  </div>
                </div>
              )}
            </>
          )}
        </section>
      )}

      {/* Modal Tambah / Ubah */}
      {modalState && (
        <DelegationFormModal
          mode={modalState.mode}
          initial={modalState.item}
          currentUserId={user.id}
          onClose={() => setModalState(null)}
          onSuccess={(msg) => {
            setNotice(msg);
            refreshMine();
            if (canManageAll && activeTab === "admin") {
              refreshAdmin();
            }
          }}
        />
      )}

      {/* Modal Konfirmasi Cabut / Hapus */}
      {deleteTarget && (
        <div className="fixed inset-0 z-50 bg-black/50 backdrop-blur-[1px] flex items-center justify-center p-4">
          <div className="w-full max-w-md rounded-2xl border border-border bg-surface p-5 shadow-2xl space-y-4">
            <div className="flex items-start gap-3">
              <div className="p-2.5 rounded-xl bg-status-danger-bg text-status-danger-text shrink-0">
                <Trash2 size={18} />
              </div>
              <div className="space-y-1">
                <h3 className="text-base font-bold text-text">
                  Cabut Jadwal Delegasi?
                </h3>
                <p className="text-xs text-text-muted leading-relaxed">
                  Delegasi tugas{" "}
                  <strong className="text-text">
                    {deleteTarget.item.user.name}
                  </strong>{" "}
                  kepada{" "}
                  <strong className="text-text">
                    {deleteTarget.item.delegate.name}
                  </strong>{" "}
                  ({formatDateID(deleteTarget.item.from)} –{" "}
                  {formatDateID(deleteTarget.item.until)}) akan dihentikan dan
                  tugas dikembalikan ke pemilik asal.
                </p>
              </div>
            </div>

            {deleteError && (
              <div className="rounded-lg bg-status-danger-bg px-3 py-2 text-xs text-status-danger-text">
                {deleteError}
              </div>
            )}

            <div className="flex items-center justify-end gap-2 pt-2 border-t border-border">
              <button
                type="button"
                disabled={deleting}
                onClick={() => setDeleteTarget(null)}
                className="px-3.5 py-2 rounded-lg border border-border bg-surface text-xs font-semibold text-text hover:bg-panel transition-colors"
              >
                Batal
              </button>
              <button
                type="button"
                disabled={deleting}
                onClick={handleConfirmDelete}
                className="inline-flex items-center gap-1.5 px-3.5 py-2 rounded-lg bg-status-danger-text text-white text-xs font-semibold hover:opacity-90 transition-opacity disabled:opacity-50"
              >
                {deleting ? (
                  <>
                    <Loader2 size={13} className="animate-spin" />
                    Mencabut...
                  </>
                ) : (
                  "Ya, Cabut Delegasi"
                )}
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}

