"use client";

// ─────────────────────────────────────────────
// components/surat/incoming-external-form.tsx
// Form khusus untuk Jenis Surat "Surat Masuk Eksternal" (SURAT_MASUK_EKSTERNAL)
// Menghilangkan bagian Lampiran dan Editor DevExpress, digantikan dengan
// form metadata surat asal + Tujuan Disposisi + Unggah & Pratinjau PDF Scan.
// Terhubung langsung ke POST /api/v1/letters/incoming
// ─────────────────────────────────────────────

import { useState, useEffect, useCallback, useRef } from "react";
import { useRouter } from "next/navigation";
import { useFormContext } from "react-hook-form";
import {
  Building2,
  Calendar,
  CheckCircle2,
  Eye,
  FileText,
  Hash,
  Loader2,
  Plus,
  Search,
  Send,
  Trash2,
  UploadCloud,
  UserCheck,
  Users,
  X,
  AlertCircle,
} from "lucide-react";
import {
  fetchCandidates,
  fetchCurrentUserId,
  fetchRecipientAliases,
  type CandidateItem,
  type RecipientAliasItem,
} from "@/lib/api/form";
import type { BuatSuratFormValues } from "@/lib/schemas";
import { createIncomingLetter } from "@/lib/api/letter-draft";
import { formatBytes } from "@/lib/api/letter-detail";
import { BasicFields } from "./basic-fields";
import {
  FieldError,
  FieldLabel,
  SectionCard,
  inputClass,
} from "./form-primitives";
import { cn } from "@/lib/utils";

const DISPOSITION_PRESETS = [
  "Tindak lanjuti segera",
  "Pelajari dan laporkan hasilnya",
  "Koordinasikan dengan unit terkait",
  "Untuk diketahui & arsip",
  "Wakili / hadiri kegiatan",
];

const MAX_PDF_BYTES = 25 * 1024 * 1024; // 25 MB sesuai backend MaxFinalPDFBytes

function todayIsoDate(): string {
  const d = new Date();
  const yyyy = d.getFullYear();
  const mm = String(d.getMonth() + 1).padStart(2, "0");
  const dd = String(d.getDate()).padStart(2, "0");
  return `${yyyy}-${mm}-${dd}`;
}

export function IncomingExternalForm({
  onCancel,
  onResetNew,
}: {
  onCancel: () => void;
  onResetNew: () => void;
}) {
  const router = useRouter();
  const { getValues } = useFormContext<BuatSuratFormValues>();
  const fileInputRef = useRef<HTMLInputElement>(null);

  // Field metadata surat masuk eksternal
  const [sender, setSender] = useState("");
  const [referenceNumber, setReferenceNumber] = useState("");
  const [letterDate, setLetterDate] = useState(() => todayIsoDate());
  const [title, setTitle] = useState(() => getValues("title") ?? "");
  const [note, setNote] = useState("");

  // Tujuan disposisi (Orang & Alias Jabatan) — dibuka saat diklik agar ringkas
  const [pickerOpen, setPickerOpen] = useState(false);
  const [pickerTab, setPickerTab] = useState<"user" | "alias">("user");
  const [searchQuery, setSearchQuery] = useState("");
  const [currentUserId, setCurrentUserId] = useState<string | null>(null);
  const [candidateUsers, setCandidateUsers] = useState<CandidateItem[]>([]);
  const [aliases, setAliases] = useState<RecipientAliasItem[]>([]);
  const [searchingUsers, setSearchingUsers] = useState(false);
  const [selectedUsers, setSelectedUsers] = useState<CandidateItem[]>([]);
  const [selectedAliases, setSelectedAliases] = useState<RecipientAliasItem[]>(
    []
  );

  // File PDF scan & URL pratinjau
  const [pdfFile, setPdfFile] = useState<File | null>(null);
  const [pdfPreviewUrl, setPdfPreviewUrl] = useState<string | null>(null);
  const pdfPreviewUrlRef = useRef<string | null>(null);
  const [isDragging, setIsDragging] = useState(false);

  // Status pengiriman & validasi
  const [submitting, setSubmitting] = useState(false);
  const [errorMessage, setErrorMessage] = useState<string | null>(null);
  const [fieldErrors, setFieldErrors] = useState<Record<string, string>>({});
  const [createdLetterId, setCreatedLetterId] = useState<string | null>(null);

  const updatePdfFile = useCallback((nextFile: File | null) => {
    if (pdfPreviewUrlRef.current) {
      URL.revokeObjectURL(pdfPreviewUrlRef.current);
      pdfPreviewUrlRef.current = null;
    }
    if (!nextFile) {
      setPdfFile(null);
      setPdfPreviewUrl(null);
      return;
    }
    const url = URL.createObjectURL(nextFile);
    pdfPreviewUrlRef.current = url;
    setPdfFile(nextFile);
    setPdfPreviewUrl(url);
  }, []);

  // Bersihkan Object URL ketika form dilepas dari DOM
  useEffect(() => {
    return () => {
      if (pdfPreviewUrlRef.current) {
        URL.revokeObjectURL(pdfPreviewUrlRef.current);
        pdfPreviewUrlRef.current = null;
      }
    };
  }, []);

  // Ambil daftar alias jabatan & user saat ini sekali
  useEffect(() => {
    fetchRecipientAliases()
      .then((data) => setAliases(data))
      .catch(() => {});
    fetchCurrentUserId()
      .then((uid) => setCurrentUserId(uid))
      .catch(() => {});
  }, []);

  // Cari kandidat pegawai untuk disposisi hanya saat panel dibuka
  const searchUsers = useCallback(async (q: string) => {
    setSearchingUsers(true);
    try {
      const data = await fetchCandidates({
        type: "REVIEWER",
        q: q.trim() || undefined,
        limit: 20,
      });
      setCandidateUsers(data);
    } catch {
      // Abaikan gagal pencarian sementara
    } finally {
      setSearchingUsers(false);
    }
  }, []);

  useEffect(() => {
    if (!pickerOpen || pickerTab !== "user") return;
    const t = setTimeout(() => {
      void searchUsers(searchQuery);
    }, 250);
    return () => clearTimeout(t);
  }, [pickerOpen, pickerTab, searchQuery, searchUsers]);

  const visibleCandidates = currentUserId
    ? candidateUsers.filter((u) => String(u.id) !== currentUserId)
    : candidateUsers;

  const totalRecipients = selectedUsers.length + selectedAliases.length;

  function handleSelectPdf(file: File | null | undefined) {
    if (!file) return;
    const isPdf =
      file.type === "application/pdf" ||
      file.name.toLowerCase().endsWith(".pdf");
    if (!isPdf) {
      setFieldErrors((prev) => ({
        ...prev,
        file: "Dokumen surat masuk eksternal wajib berformat PDF (.pdf).",
      }));
      return;
    }
    if (file.size > MAX_PDF_BYTES) {
      setFieldErrors((prev) => ({
        ...prev,
        file: "Ukuran file PDF maksimal 25 MB.",
      }));
      return;
    }
    setFieldErrors((prev) => {
      const next = { ...prev };
      delete next.file;
      return next;
    });
    updatePdfFile(file);
  }

  function validate(): boolean {
    const errs: Record<string, string> = {};
    if (!sender.trim()) {
      errs.sender = "Nama perusahaan / instansi pengirim wajib diisi.";
    } else if (sender.trim().length > 200) {
      errs.sender = "Nama pengirim maksimal 200 karakter.";
    }

    if (!referenceNumber.trim()) {
      errs.referenceNumber = "Nomor surat asal wajib diisi.";
    } else if (referenceNumber.trim().length > 100) {
      errs.referenceNumber = "Nomor surat asal maksimal 100 karakter.";
    }

    if (!letterDate.trim()) {
      errs.letterDate = "Tanggal surat asal wajib diisi.";
    }

    if (!title.trim()) {
      errs.title = "Perihal surat wajib diisi.";
    } else if (title.trim().length > 250) {
      errs.title = "Perihal maksimal 250 karakter.";
    }

    if (totalRecipients < 1) {
      errs.recipients =
        "Pilih minimal 1 tujuan disposisi (orang atau alias jabatan).";
    } else if (totalRecipients > 50) {
      errs.recipients = "Tujuan disposisi maksimal 50 penerima.";
    }

    if (note.trim().length > 1000) {
      errs.note = "Catatan disposisi maksimal 1000 karakter.";
    }

    if (!pdfFile) {
      errs.file = "File scan PDF surat masuk wajib diunggah.";
    }

    setFieldErrors(errs);
    return Object.keys(errs).length === 0;
  }

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault();
    setErrorMessage(null);
    if (!validate() || !pdfFile) return;

    setSubmitting(true);
    try {
      const detail = await createIncomingLetter({
        file: pdfFile,
        title: title.trim(),
        sender: sender.trim(),
        referenceNumber: referenceNumber.trim(),
        letterDate: letterDate.trim(),
        userIds: selectedUsers.map((u) => String(u.id)),
        aliasLevelIds: selectedAliases.map((a) => String(a.levelId)),
        note: note.trim() || undefined,
      });
      setCreatedLetterId(detail.id);
    } catch (err) {
      setErrorMessage(
        err instanceof Error
          ? err.message
          : "Gagal menyimpan surat masuk eksternal."
      );
    } finally {
      setSubmitting(false);
    }
  }

  const normalizedAliasQuery = searchQuery.trim().toLowerCase();
  const filteredAliases = aliases.filter((a) => {
    if (!normalizedAliasQuery) return true;
    return (
      a.label.toLowerCase().includes(normalizedAliasQuery) ||
      a.code.toLowerCase().includes(normalizedAliasQuery)
    );
  });

  return (
    <form onSubmit={handleSubmit} className="space-y-5">
      {/* ── Section 1: Form Detail & Disposisi Surat Masuk Eksternal ── */}
      <SectionCard number={1} title="Detail & Disposisi Surat Masuk Eksternal">
        <div className="space-y-5">
          {/* Banner penjelasan */}

          <div className="grid grid-cols-1 md:grid-cols-2 gap-5">
            {/* Dropdown Jenis Surat (agar pengguna tetap bisa berganti ke jenis surat lain) */}
            <BasicFields onlyLetterType />

            {/* Pengirim Eksternal */}
            <div>
              <FieldLabel
                label="Pengirim Eksternal (Perusahaan / Instansi)"
                required
                htmlFor="incomingSender"
              />
              <div className="relative">
                <Building2
                  size={15}
                  className="absolute left-3 top-1/2 -translate-y-1/2 text-text-muted pointer-events-none"
                />
                <input
                  id="incomingSender"
                  type="text"
                  maxLength={200}
                  placeholder="Contoh: PT Pupuk Indonesia (Persero) / Kementerian BUMN..."
                  value={sender}
                  onChange={(e) => {
                    setSender(e.target.value);
                    if (fieldErrors.sender) {
                      setFieldErrors((p) => ({ ...p, sender: "" }));
                    }
                  }}
                  className={cn(inputClass(!!fieldErrors.sender), "pl-9")}
                />
              </div>
              <FieldError message={fieldErrors.sender} />
            </div>

            {/* Nomor Surat Asal */}
            <div>
              <FieldLabel
                label="Nomor Surat Asal"
                required
                htmlFor="incomingRefNumber"
              />
              <div className="relative">
                <Hash
                  size={15}
                  className="absolute left-3 top-1/2 -translate-y-1/2 text-text-muted pointer-events-none"
                />
                <input
                  id="incomingRefNumber"
                  type="text"
                  maxLength={100}
                  placeholder="Contoh: 042/EKS/PI/XI/2026..."
                  value={referenceNumber}
                  onChange={(e) => {
                    setReferenceNumber(e.target.value);
                    if (fieldErrors.referenceNumber) {
                      setFieldErrors((p) => ({ ...p, referenceNumber: "" }));
                    }
                  }}
                  className={cn(
                    inputClass(!!fieldErrors.referenceNumber),
                    "pl-9 font-mono"
                  )}
                />
              </div>
              <FieldError message={fieldErrors.referenceNumber} />
            </div>

            {/* Tanggal Surat Asal */}
            <div>
              <FieldLabel
                label="Tanggal Surat Asal"
                required
                htmlFor="incomingLetterDate"
              />
              <div className="relative">
                <Calendar
                  size={15}
                  className="absolute left-3 top-1/2 -translate-y-1/2 text-text-muted pointer-events-none"
                />
                <input
                  id="incomingLetterDate"
                  type="date"
                  value={letterDate}
                  onChange={(e) => {
                    setLetterDate(e.target.value);
                    if (fieldErrors.letterDate) {
                      setFieldErrors((p) => ({ ...p, letterDate: "" }));
                    }
                  }}
                  className={cn(inputClass(!!fieldErrors.letterDate), "pl-9")}
                />
              </div>
              <FieldError message={fieldErrors.letterDate} />
            </div>

            {/* Perihal / Judul Surat */}
            <div className="md:col-span-2">
              <FieldLabel
                label="Perihal / Judul Surat"
                required
                htmlFor="incomingTitle"
              />
              <div className="relative">
                <input
                  id="incomingTitle"
                  type="text"
                  maxLength={250}
                  placeholder="Tuliskan perihal surat masuk sesuai dokumen asli..."
                  value={title}
                  onChange={(e) => {
                    setTitle(e.target.value);
                    if (fieldErrors.title) {
                      setFieldErrors((p) => ({ ...p, title: "" }));
                    }
                  }}
                  className={cn(inputClass(!!fieldErrors.title), "pr-20")}
                />
                <span className="absolute right-3 top-1/2 -translate-y-1/2 text-xs font-mono text-text-muted pointer-events-none">
                  {title.length}/250
                </span>
              </div>
              <FieldError message={fieldErrors.title} />
            </div>

            {/* Tujuan Disposisi Internal (Orang & Alias Jabatan) */}
            <div className="md:col-span-2 space-y-2.5">
              <div className="flex items-center justify-between">
                <FieldLabel
                  label="Tujuan Disposisi Internal (Penerima Surat Masuk)"
                  required
                />
                <span className="text-xs text-text-muted">
                  Terpilih: <strong>{totalRecipients}</strong> / 50
                </span>
              </div>
              <p className="text-xs text-text-muted">
                Pilih minimal 1 pejabat/pegawai lain atau alias jabatan yang akan menerima disposisi surat masuk eksternal ini di Kotak Masuk (tujuan disposisi tidak boleh hanya diri sendiri).
              </p>

              {/* Chip penerima terpilih */}
              {totalRecipients > 0 && (
                <div className="flex flex-wrap gap-1.5 p-3 rounded-xl bg-panel/50 border border-border">
                  {selectedUsers.map((u) => (
                    <span
                      key={`user-${u.id}`}
                      className="inline-flex items-center gap-1.5 text-xs bg-primary/10 text-primary font-medium px-2.5 py-1 rounded-full"
                    >
                      <UserCheck size={12} />
                      <span>
                        {u.name}{" "}
                        <span className="opacity-75 font-normal">
                          ({u.positionName})
                        </span>
                      </span>
                      <button
                        type="button"
                        onClick={() =>
                          setSelectedUsers((prev) =>
                            prev.filter((item) => item.id !== u.id)
                          )
                        }
                        className="hover:text-status-danger-text ml-0.5"
                        aria-label={`Hapus ${u.name}`}
                      >
                        <X size={12} />
                      </button>
                    </span>
                  ))}
                  {selectedAliases.map((a) => (
                    <span
                      key={`alias-${a.levelId}`}
                      className="inline-flex items-center gap-1.5 text-xs bg-status-info-bg text-status-info-text font-medium px-2.5 py-1 rounded-full"
                    >
                      <Users size={12} />
                      <span>
                        {a.label}{" "}
                        <span className="opacity-75 font-normal">
                          ({a.memberCount} pejabat)
                        </span>
                      </span>
                      <button
                        type="button"
                        onClick={() =>
                          setSelectedAliases((prev) =>
                            prev.filter((item) => item.levelId !== a.levelId)
                          )
                        }
                        className="hover:text-status-danger-text ml-0.5"
                        aria-label={`Hapus ${a.label}`}
                      >
                        <X size={12} />
                      </button>
                    </span>
                  ))}
                </div>
              )}

              {/* Tombol pemicu pemilih Orang / Alias Jabatan */}
              <button
                type="button"
                onClick={() => setPickerOpen((v) => !v)}
                className="flex items-center gap-1.5 text-sm text-primary hover:text-primary-dark border border-dashed border-primary/40 hover:border-primary px-3 py-2 rounded-lg w-full justify-center transition-colors"
              >
                <Plus size={14} />
                {pickerOpen
                  ? "Tutup Pilihan Tujuan Disposisi"
                  : "+ Pilih Tujuan Disposisi (Orang / Alias Jabatan)"}
              </button>

              {/* Kotak pemilih Orang / Alias Jabatan (muncul saat diklik) */}
              {pickerOpen && (
              <div className="border border-border rounded-xl overflow-hidden bg-surface shadow-md">
                <div className="flex items-center justify-between border-b border-border bg-panel/40 px-3">
                  <div className="flex gap-2">
                    <button
                      type="button"
                      onClick={() => {
                        setPickerTab("user");
                        setSearchQuery("");
                      }}
                      className={cn(
                        "px-3 py-2 text-xs font-semibold border-b-2 -mb-px transition-colors flex items-center gap-1.5",
                        pickerTab === "user"
                          ? "border-primary text-primary"
                          : "border-transparent text-text-muted hover:text-text"
                      )}
                    >
                      <UserCheck size={13} /> Orang / Departemen
                    </button>
                    <button
                      type="button"
                      onClick={() => {
                        setPickerTab("alias");
                        setSearchQuery("");
                      }}
                      className={cn(
                        "px-3 py-2 text-xs font-semibold border-b-2 -mb-px transition-colors flex items-center gap-1.5",
                        pickerTab === "alias"
                          ? "border-primary text-primary"
                          : "border-transparent text-text-muted hover:text-text"
                      )}
                    >
                      <Users size={13} /> Alias Jabatan ({aliases.length})
                    </button>
                  </div>
                  <button
                    type="button"
                    onClick={() => setPickerOpen(false)}
                    className="text-xs text-text-muted hover:text-text py-1.5"
                  >
                    Tutup
                  </button>
                </div>

                {/* Bar pencarian */}
                <div className="p-2.5 border-b border-border">
                  <div className="relative">
                    <Search
                      size={14}
                      className="absolute left-3 top-1/2 -translate-y-1/2 text-text-muted"
                    />
                    <input
                      autoFocus
                      type="text"
                      value={searchQuery}
                      onChange={(e) => setSearchQuery(e.target.value)}
                      placeholder={
                        pickerTab === "user"
                          ? "Cari nama pegawai, jabatan, atau unit..."
                          : "Cari alias jabatan (contoh: Seluruh VP)..."
                      }
                      className="w-full pl-8 pr-8 py-1.5 text-sm border border-border rounded-lg bg-surface outline-none focus:border-primary"
                    />
                    {searchQuery && (
                      <button
                        type="button"
                        onClick={() => setSearchQuery("")}
                        className="absolute right-2.5 top-1/2 -translate-y-1/2 text-text-muted hover:text-text"
                      >
                        <X size={13} />
                      </button>
                    )}
                  </div>
                </div>

                {/* Daftar hasil */}
                <div className="max-h-48 overflow-y-auto divide-y divide-border">
                  {pickerTab === "user" ? (
                    searchingUsers ? (
                      <div className="py-6 flex items-center justify-center gap-2 text-xs text-text-muted">
                        <Loader2 size={14} className="animate-spin" />
                        Mencari pegawai...
                      </div>
                    ) : visibleCandidates.length === 0 ? (
                      <div className="py-6 text-center text-xs text-text-muted">
                        Pegawai tidak ditemukan.
                      </div>
                    ) : (
                      visibleCandidates.map((u) => {
                        const isSelected = selectedUsers.some(
                          (s) => s.id === u.id
                        );
                        return (
                          <button
                            key={u.id}
                            type="button"
                            onClick={() => {
                              setSelectedUsers((prev) =>
                                isSelected
                                  ? prev.filter((s) => s.id !== u.id)
                                  : [...prev, u]
                              );
                              if (fieldErrors.recipients) {
                                setFieldErrors((p) => ({
                                  ...p,
                                  recipients: "",
                                }));
                              }
                            }}
                            className={cn(
                              "w-full px-3.5 py-2 text-left flex items-center justify-between gap-2 text-xs transition-colors",
                              isSelected
                                ? "bg-primary/10 text-primary font-semibold"
                                : "hover:bg-panel text-text"
                            )}
                          >
                            <div className="min-w-0">
                              <p className="truncate font-medium">{u.name}</p>
                              <p className="text-[11px] text-text-muted truncate">
                                {u.positionName} &bull; {u.unitName}
                              </p>
                            </div>
                            {isSelected && (
                              <CheckCircle2
                                size={14}
                                className="text-primary shrink-0"
                              />
                            )}
                          </button>
                        );
                      })
                    )
                  ) : filteredAliases.length === 0 ? (
                    <div className="py-6 text-center text-xs text-text-muted">
                      Alias jabatan tidak ditemukan.
                    </div>
                  ) : (
                    filteredAliases.map((a) => {
                      const isSelected = selectedAliases.some(
                        (s) => s.levelId === a.levelId
                      );
                      return (
                        <button
                          key={a.levelId}
                          type="button"
                          onClick={() => {
                            setSelectedAliases((prev) =>
                              isSelected
                                ? prev.filter((s) => s.levelId !== a.levelId)
                                : [...prev, a]
                            );
                            if (fieldErrors.recipients) {
                              setFieldErrors((p) => ({ ...p, recipients: "" }));
                            }
                          }}
                          className={cn(
                            "w-full px-3.5 py-2 text-left flex items-center justify-between gap-2 text-xs transition-colors",
                            isSelected
                              ? "bg-primary/10 text-primary font-semibold"
                              : "hover:bg-panel text-text"
                          )}
                        >
                          <div>
                            <p className="font-medium">{a.label}</p>
                            <p className="text-[11px] text-text-muted">
                              {a.memberCount} pejabat aktif
                            </p>
                          </div>
                          {isSelected && (
                            <CheckCircle2
                              size={14}
                              className="text-primary shrink-0"
                            />
                          )}
                        </button>
                      );
                    })
                  )}
                </div>
              </div>
              )}
              <FieldError message={fieldErrors.recipients} />
            </div>

            {/* Catatan / Keterangan Disposisi */}
            <div className="md:col-span-2 space-y-2">
              <FieldLabel
                label="Catatan / Instruksi Disposisi (Keterangan)"
                htmlFor="incomingNote"
              />
              <div className="flex flex-wrap gap-1.5">
                {DISPOSITION_PRESETS.map((preset) => (
                  <button
                    key={preset}
                    type="button"
                    onClick={() => setNote(preset)}
                    className={cn(
                      "px-2.5 py-1 rounded-lg text-xs border transition-colors",
                      note === preset
                        ? "bg-primary/10 border-primary text-primary font-medium"
                        : "border-border text-text-muted hover:text-text hover:bg-panel"
                    )}
                  >
                    {preset}
                  </button>
                ))}
              </div>
              <textarea
                id="incomingNote"
                rows={3}
                maxLength={1000}
                value={note}
                onChange={(e) => setNote(e.target.value)}
                placeholder="Tuliskan instruksi disposisi atau keterangan tambahan untuk penerima (opsional)..."
                className={`${inputClass(!!fieldErrors.note)} resize-none`}
              />
              <div className="flex justify-between text-[11px] text-text-muted">
                <FieldError message={fieldErrors.note} />
                <span>{note.length} / 1000</span>
              </div>
            </div>
          </div>
        </div>
      </SectionCard>

      {/* ── Section 2: Unggah & Pratinjau Dokumen PDF Scan (Menggantikan Lampiran & DevExpress) ── */}
      <SectionCard
        number={2}
        title="Unggah & Pratinjau Dokumen PDF Surat Masuk"
      >
        <div className="space-y-4">
          <input
            ref={fileInputRef}
            type="file"
            accept="application/pdf,.pdf"
            className="hidden"
            onChange={(e) => {
              handleSelectPdf(e.target.files?.[0]);
              e.target.value = "";
            }}
          />

          {!pdfFile ? (
            <div
              onDragOver={(e) => {
                e.preventDefault();
                setIsDragging(true);
              }}
              onDragLeave={() => setIsDragging(false)}
              onDrop={(e) => {
                e.preventDefault();
                setIsDragging(false);
                handleSelectPdf(e.dataTransfer.files?.[0]);
              }}
              onClick={() => fileInputRef.current?.click()}
              className={cn(
                "border-2 border-dashed rounded-xl p-8 text-center cursor-pointer transition-colors flex flex-col items-center justify-center gap-2.5",
                isDragging
                  ? "border-primary bg-primary/5"
                  : fieldErrors.file
                    ? "border-status-danger-text bg-status-danger-bg/20"
                    : "border-border hover:border-primary/50 hover:bg-panel/40"
              )}
            >
              <div className="w-12 h-12 rounded-2xl bg-primary/10 text-primary flex items-center justify-center">
                <UploadCloud size={24} />
              </div>
              <div>
                <p className="text-sm font-semibold text-text">
                  Klik untuk memilih file PDF atau seret &amp; lepas di sini
                </p>
                <p className="text-xs text-text-muted mt-0.5">
                  Wajib format PDF (.pdf) hasil scan surat masuk &bull; Maksimal
                  25 MB
                </p>
              </div>
            </div>
          ) : (
            <div className="space-y-4">
              {/* Info bar file PDF terpilih */}
              <div className="flex items-center justify-between gap-3 flex-wrap p-3.5 rounded-xl bg-panel/50 border border-border">
                <div className="flex items-center gap-3 min-w-0">
                  <div className="w-10 h-10 rounded-lg bg-red-500/10 text-red-600 flex items-center justify-center shrink-0">
                    <FileText size={20} />
                  </div>
                  <div className="min-w-0">
                    <p className="text-sm font-semibold text-text truncate">
                      {pdfFile.name}
                    </p>
                    <p className="text-xs text-text-muted">
                      {formatBytes(pdfFile.size)} &bull; Dokumen PDF Siap
                      Diunggah
                    </p>
                  </div>
                </div>
                <div className="flex items-center gap-2 shrink-0">
                  {pdfPreviewUrl && (
                    <a
                      href={pdfPreviewUrl}
                      target="_blank"
                      rel="noopener noreferrer"
                      className="inline-flex items-center gap-1.5 px-3 py-1.5 rounded-lg border border-border bg-surface text-xs font-medium text-text hover:bg-panel transition-colors"
                    >
                      <Eye size={13} /> Buka Tab Baru
                    </a>
                  )}
                  <button
                    type="button"
                    onClick={() => fileInputRef.current?.click()}
                    className="inline-flex items-center gap-1.5 px-3 py-1.5 rounded-lg border border-border bg-surface text-xs font-medium text-primary hover:bg-panel transition-colors"
                  >
                    <UploadCloud size={13} /> Ganti PDF
                  </button>
                  <button
                    type="button"
                    onClick={() => updatePdfFile(null)}
                    className="inline-flex items-center gap-1.5 px-3 py-1.5 rounded-lg border border-status-danger-text/30 text-xs font-medium text-status-danger-text hover:bg-status-danger-bg transition-colors"
                  >
                    <Trash2 size={13} /> Hapus
                  </button>
                </div>
              </div>

              {/* Tampilan Pratinjau PDF Langsung */}
              {pdfPreviewUrl && (
                <div className="border border-border rounded-xl overflow-hidden bg-neutral-900/5">
                  <div className="px-4 py-2 bg-panel border-b border-border flex items-center justify-between">
                    <span className="text-xs font-semibold text-text flex items-center gap-1.5">
                      <FileText size={13} className="text-primary" />
                      Pratinjau Dokumen PDF Surat Masuk
                    </span>
                    <span className="text-[11px] text-status-success-text font-medium flex items-center gap-1">
                      <CheckCircle2 size={12} /> Mode Tampilan PDF
                    </span>
                  </div>
                  <iframe
                    src={pdfPreviewUrl}
                    title="Pratinjau PDF Surat Masuk Eksternal"
                    className="w-full h-195 border-0 bg-white"
                  />
                </div>
              )}
            </div>
          )}

          <FieldError message={fieldErrors.file} />
        </div>
      </SectionCard>

      {errorMessage && (
        <div className="flex items-center gap-2.5 bg-status-danger-bg border border-status-danger-text/30 text-status-danger-text rounded-xl px-4 py-3 text-sm">
          <AlertCircle size={16} className="shrink-0" />
          <span>{errorMessage}</span>
        </div>
      )}

      {/* ── Sticky Action Bar ── */}
      <div className="fixed bottom-0 left-0 md:left-56 right-0 z-30 bg-surface/95 backdrop-blur border-t border-border px-6 py-3.5 flex items-center justify-between gap-3 shadow-lg">
        <button
          type="button"
          onClick={onCancel}
          disabled={submitting}
          className="px-4 py-2 rounded-lg border border-border text-sm font-medium text-text hover:bg-panel transition-colors"
        >
          Batal
        </button>

        <button
          type="submit"
          disabled={submitting}
          className="inline-flex items-center gap-2 px-5 py-2.5 rounded-lg bg-primary text-white text-sm font-semibold hover:bg-primary-dark disabled:opacity-60 transition-colors shadow-xs"
        >
          {submitting ? (
            <>
              <Loader2 size={16} className="animate-spin" />
              Menyimpan &amp; Mendisposisikan...
            </>
          ) : (
            <>
              <Send size={15} />
              Simpan &amp; Disposisikan Surat Masuk
            </>
          )}
        </button>
      </div>

      {/* ── Modal Sukses Pencatatan Surat Masuk Eksternal ── */}
      {createdLetterId && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50 backdrop-blur-sm p-4">
          <div className="bg-surface border border-border rounded-2xl shadow-2xl max-w-md w-full p-6 space-y-4 text-center">
            <div className="w-12 h-12 rounded-full bg-status-success-bg text-status-success-text flex items-center justify-center mx-auto">
              <CheckCircle2 size={26} />
            </div>
            <div className="space-y-1">
              <h3 className="text-base font-bold text-text">
                Surat Masuk Eksternal Berhasil Dicatat
              </h3>
              <p className="text-xs text-text-muted leading-relaxed">
                Dokumen scan PDF telah disimpan dan didisposisikan ke{" "}
                <strong className="text-text">{totalRecipients} tujuan</strong>{" "}
                pada halaman Kotak Masuk (Kategori Eksternal).
              </p>
            </div>
            <div className="flex flex-col sm:flex-row gap-2 pt-2">
              <button
                type="button"
                onClick={() => router.push(`/surat/${createdLetterId}`)}
                className="flex-1 px-4 py-2.5 rounded-lg bg-primary text-white text-xs font-semibold hover:bg-primary-dark transition-colors"
              >
                Lihat Detail &amp; PDF Surat
              </button>
              <button
                type="button"
                onClick={() => router.push("/inbox?category=EXTERNAL")}
                className="flex-1 px-4 py-2.5 rounded-lg border border-border text-xs font-semibold text-text hover:bg-panel transition-colors"
              >
                Buka Surat Eksternal
              </button>
            </div>
            <button
              type="button"
              onClick={onResetNew}
              className="text-xs text-primary hover:underline font-medium"
            >
              + Catat Surat Masuk Eksternal Lainnya
            </button>
          </div>
        </div>
      )}
    </form>
  );
}

