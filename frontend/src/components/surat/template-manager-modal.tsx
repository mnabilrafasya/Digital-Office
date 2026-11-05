"use client";

// ─────────────────────────────────────────────
// components/surat/template-manager-modal.tsx
// Modal Kelola Template Naskah (.docx) — Tambah & Perbarui Template di MinIO
// Terhubung ke POST /api/v1/templates & PUT /api/v1/templates/{templateId}
// ─────────────────────────────────────────────

import { useEffect, useState } from "react";
import { createPortal } from "react-dom";
import {
  X,
  Plus,
  Pencil,
  Upload,
  Download,
  Loader2,
  FileText,
  CheckCircle2,
} from "lucide-react";
import {
  createTemplate,
  updateTemplate,
  fetchTemplates,
  type LetterTemplateDefinition,
} from "@/lib/templates/letter-templates";
import type { LookupItem } from "@/lib/api/form";
import { inputClass } from "./form-primitives";
import { cn } from "@/lib/utils";

interface TemplateManagerModalProps {
  templates: LetterTemplateDefinition[];
  letterTypes: LookupItem[];
  onClose: () => void;
  onUpdated: (items: LetterTemplateDefinition[], selectedId?: string) => void;
}

export function TemplateManagerModal({
  templates,
  letterTypes,
  onClose,
  onUpdated,
}: TemplateManagerModalProps) {
  const [mounted, setMounted] = useState(false);

  useEffect(() => {
    setMounted(true);
  }, []);
  const [mode, setMode] = useState<"list" | "create" | "edit">("list");
  const [editingId, setEditingId] = useState<string | null>(null);
  const [name, setName] = useState("");
  const [description, setDescription] = useState("");
  const [letterTypeId, setLetterTypeId] = useState("");
  const [file, setFile] = useState<File | null>(null);
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [successMsg, setSuccessMsg] = useState<string | null>(null);

  function openCreate() {
    setMode("create");
    setEditingId(null);
    setName("");
    setDescription("");
    setLetterTypeId("");
    setFile(null);
    setError(null);
    setSuccessMsg(null);
  }

  function openEdit(tpl: LetterTemplateDefinition) {
    setMode("edit");
    setEditingId(tpl.id);
    setName(tpl.name);
    setDescription(tpl.description ?? "");
    setLetterTypeId(tpl.letterType?.id ?? "");
    setFile(null);
    setError(null);
    setSuccessMsg(null);
  }

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault();
    if (!name.trim()) {
      setError("Nama template wajib diisi.");
      return;
    }
    if (mode === "create" && !file) {
      setError("Berkas .docx wajib dipilih untuk membuat template baru.");
      return;
    }

    setSaving(true);
    setError(null);
    try {
      let saved: LetterTemplateDefinition;
      if (mode === "create") {
        saved = await createTemplate({
          name: name.trim(),
          description: description.trim() || undefined,
          letterTypeId: letterTypeId || undefined,
          file: file!,
        });
      } else {
        saved = await updateTemplate(editingId!, {
          name: name.trim(),
          description: description.trim() || undefined,
          letterTypeId: letterTypeId || undefined,
          file: file ?? undefined,
        });
      }
      const freshList = await fetchTemplates();
      onUpdated(freshList, saved.id);
      setSuccessMsg(
        mode === "create"
          ? `Template "${saved.name}" berhasil ditambahkan.`
          : `Template "${saved.name}" berhasil diperbarui.`
      );
      setMode("list");
    } catch (err) {
      setError(
        err instanceof Error ? err.message : "Gagal menyimpan template."
      );
    } finally {
      setSaving(false);
    }
  }

  if (!mounted) return null;

  return createPortal(
    <div className="fixed inset-0 z-50 bg-black/50 flex items-center justify-center p-4">
      <div className="bg-surface border border-border rounded-xl shadow-xl w-full max-w-2xl overflow-hidden flex flex-col max-h-[85vh]">
        {/* Header */}
        <div className="px-5 py-4 border-b border-border flex items-center justify-between bg-panel/40">
          <div className="flex items-center gap-2">
            <FileText size={17} className="text-primary" />
            <h3 className="text-sm font-bold text-text">
              {mode === "create"
                ? "Tambah Template Naskah (.docx)"
                : mode === "edit"
                  ? "Perbarui Template Naskah (.docx)"
                  : "Kelola Template Naskah Dinas"}
            </h3>
          </div>
          <button
            type="button"
            onClick={onClose}
            className="text-text-muted hover:text-text p-1 rounded-md"
          >
            <X size={16} />
          </button>
        </div>

        {/* Body */}
        <div className="p-5 overflow-y-auto space-y-4 flex-1">
          {successMsg && (
            <div className="flex items-center gap-2 px-3 py-2.5 rounded-lg bg-status-success-bg text-status-success-text text-xs font-medium">
              <CheckCircle2 size={14} className="shrink-0" />
              <span>{successMsg}</span>
            </div>
          )}

          {error && (
            <div className="px-3 py-2.5 rounded-lg bg-status-danger-bg text-status-danger-text text-xs font-medium">
              {error}
            </div>
          )}

          {mode === "list" ? (
            <>
              <div className="flex items-center justify-between gap-2">
                <p className="text-xs text-text-muted">
                  Daftar template `.docx` yang tersimpan di server MinIO perusahaan.
                </p>
                <button
                  type="button"
                  onClick={openCreate}
                  className="inline-flex items-center gap-1.5 px-3 py-1.5 rounded-lg bg-primary text-white text-xs font-semibold hover:bg-primary-dark transition-colors shrink-0"
                >
                  <Plus size={13} /> Tambah Template
                </button>
              </div>

              <ul className="divide-y divide-border border border-border rounded-xl overflow-hidden">
                {templates.map((tpl) => {
                  const isNumericId = /^\d+$/.test(tpl.id);
                  return (
                    <li
                      key={tpl.id}
                      className="px-4 py-3 flex items-start justify-between gap-3 bg-surface hover:bg-panel/30 transition-colors"
                    >
                      <div className="min-w-0 space-y-0.5">
                        <div className="flex items-center gap-2 flex-wrap">
                          <span className="text-sm font-semibold text-text">
                            {tpl.name}
                          </span>
                          {tpl.letterType && (
                            <span className="text-[10px] px-2 py-0.5 rounded-full bg-primary/10 text-primary font-medium">
                              {tpl.letterType.name}
                            </span>
                          )}
                        </div>
                        {tpl.description && (
                          <p className="text-xs text-text-muted">
                            {tpl.description}
                          </p>
                        )}
                      </div>
                      <div className="flex items-center gap-2 shrink-0">
                        <a
                          href={tpl.fileUrl}
                          download
                          className="inline-flex items-center gap-1 px-2.5 py-1 rounded-md border border-border text-xs text-text-muted hover:text-text hover:bg-panel transition-colors"
                        >
                          <Download size={12} /> Unduh
                        </a>
                        {isNumericId && (
                          <button
                            type="button"
                            onClick={() => openEdit(tpl)}
                            className="inline-flex items-center gap-1 px-2.5 py-1 rounded-md border border-border text-xs text-primary font-medium hover:bg-primary/5 transition-colors"
                          >
                            <Pencil size={12} /> Edit
                          </button>
                        )}
                      </div>
                    </li>
                  );
                })}
              </ul>
            </>
          ) : (
            <form onSubmit={handleSubmit} className="space-y-4">
              <div>
                <label className="block text-xs font-semibold text-text mb-1">
                  Nama Template <span className="text-status-danger-text">*</span>
                </label>
                <input
                  type="text"
                  maxLength={150}
                  value={name}
                  onChange={(e) => setName(e.target.value)}
                  placeholder="Contoh: Nota Dinas Internal"
                  className={inputClass()}
                />
              </div>

              <div>
                <label className="block text-xs font-semibold text-text mb-1">
                  Jenis Surat Terkait (Opsional)
                </label>
                <select
                  value={letterTypeId}
                  onChange={(e) => setLetterTypeId(e.target.value)}
                  className={inputClass()}
                >
                  <option value="">— Semua / Umum —</option>
                  {letterTypes.map((lt) => (
                    <option key={lt.id} value={lt.id}>
                      {lt.name} ({lt.code})
                    </option>
                  ))}
                </select>
              </div>

              <div>
                <label className="block text-xs font-semibold text-text mb-1">
                  Deskripsi Singkat (Opsional)
                </label>
                <textarea
                  rows={2}
                  maxLength={500}
                  value={description}
                  onChange={(e) => setDescription(e.target.value)}
                  placeholder="Keterangan penggunaan template dan placeholder seperti ###NoSurat###..."
                  className={cn(inputClass(), "resize-none")}
                />
              </div>

              <div>
                <label className="block text-xs font-semibold text-text mb-1">
                  Berkas Dokumen Word (.docx){" "}
                  {mode === "create" ? (
                    <span className="text-status-danger-text">*</span>
                  ) : (
                    <span className="text-text-muted font-normal">
                      (Kosongkan bila tidak mengganti berkas .docx)
                    </span>
                  )}
                </label>
                <label className="flex items-center justify-center gap-2 px-4 py-4 border-2 border-dashed border-border rounded-xl cursor-pointer hover:border-primary/50 hover:bg-panel/30 transition-colors text-xs text-text-muted">
                  <Upload size={15} className="text-primary shrink-0" />
                  <span>
                    {file
                      ? `${file.name} (${Math.round(file.size / 1024)} KB)`
                      : "Klik untuk memilih berkas .docx (maks. 25 MB)"}
                  </span>
                  <input
                    type="file"
                    accept=".docx,application/vnd.openxmlformats-officedocument.wordprocessingml.document"
                    onChange={(e) => setFile(e.target.files?.[0] ?? null)}
                    className="hidden"
                  />
                </label>
              </div>

              <div className="flex items-center justify-end gap-2 pt-2 border-t border-border">
                <button
                  type="button"
                  onClick={() => {
                    setMode("list");
                    setError(null);
                  }}
                  disabled={saving}
                  className="px-3.5 py-2 rounded-lg border border-border text-xs font-medium text-text hover:bg-panel transition-colors"
                >
                  Batal
                </button>
                <button
                  type="submit"
                  disabled={saving}
                  className="inline-flex items-center gap-1.5 px-4 py-2 rounded-lg bg-primary text-white text-xs font-semibold hover:bg-primary-dark transition-colors disabled:opacity-60"
                >
                  {saving && <Loader2 size={13} className="animate-spin" />}
                  {mode === "create" ? "Unggah & Simpan" : "Simpan Perubahan"}
                </button>
              </div>
            </form>
          )}
        </div>
      </div>
    </div>,
    document.body
  );
}

