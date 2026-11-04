"use client";

// ─────────────────────────────────────────────
// app/(app)/master-data/page.tsx
// Halaman Master Data Persuratan:
// - Manajemen Template Naskah (.docx) di MinIO (GET/POST/PUT /api/v1/templates)
// - Referensi Jenis Surat, Klasifikasi Arsip, & Alias Penerima
// ─────────────────────────────────────────────

import { useEffect, useState } from "react";
import {
  Database,
  FileText,
  Plus,
  Download,
  Pencil,
  Loader2,
  Users,
  FolderArchive,
} from "lucide-react";
import {
  fetchTemplates,
  type LetterTemplateDefinition,
} from "@/lib/templates/letter-templates";
import {
  fetchLookups,
  fetchRecipientAliases,
  type LookupItem,
  type RecipientAliasItem,
} from "@/lib/api/form";
import { TemplateManagerModal } from "@/components/surat/template-manager-modal";

export default function MasterDataPage() {
  const [templates, setTemplates] = useState<LetterTemplateDefinition[]>([]);
  const [letterTypes, setLetterTypes] = useState<LookupItem[]>([]);
  const [classifications, setClassifications] = useState<LookupItem[]>([]);
  const [aliases, setAliases] = useState<RecipientAliasItem[]>([]);
  const [loading, setLoading] = useState(true);
  const [modalOpen, setModalOpen] = useState(false);

  useEffect(() => {
    let cancelled = false;
    Promise.all([
      fetchTemplates(),
      fetchLookups("LETTER_TYPE").catch(() => []),
      fetchLookups("CLASSIFICATION").catch(() => []),
      fetchRecipientAliases().catch(() => []),
    ])
      .then(([tpls, lts, cls, als]) => {
        if (cancelled) return;
        setTemplates(tpls);
        setLetterTypes(lts);
        setClassifications(cls);
        setAliases(als);
      })
      .finally(() => {
        if (!cancelled) setLoading(false);
      });
    return () => {
      cancelled = true;
    };
  }, []);

  if (loading) {
    return (
      <div className="flex items-center justify-center min-h-[50vh] gap-2 text-text-muted">
        <Loader2 size={18} className="animate-spin text-primary" />
        <span className="text-sm">Memuat master data persuratan...</span>
      </div>
    );
  }

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex items-center justify-between gap-4 flex-wrap">
        <div>
          <div className="flex items-center gap-2">
            <Database size={20} className="text-primary" />
            <h1 className="text-xl font-bold text-text">
              Master Data & Template Naskah
            </h1>
          </div>
          <p className="text-xs text-text-muted mt-0.5">
            Kelola berkas template naskah dinas (.docx) di MinIO serta lihat referensi jenis surat, klasifikasi arsip, dan alias penerima.
          </p>
        </div>
        <button
          type="button"
          onClick={() => setModalOpen(true)}
          className="inline-flex items-center gap-1.5 px-4 py-2 rounded-lg bg-primary text-white text-xs font-semibold hover:bg-primary-dark transition-colors"
        >
          <Plus size={14} /> Kelola / Tambah Template (.docx)
        </button>
      </div>

      {/* 1. Daftar Template Naskah Dinas */}
      <div className="border border-border rounded-xl bg-surface overflow-hidden">
        <div className="px-4 py-3 border-b border-border bg-panel/30 flex items-center justify-between">
          <div className="flex items-center gap-2">
            <FileText size={15} className="text-primary" />
            <h2 className="text-sm font-semibold text-text">
              Template Naskah Dinas ({templates.length})
            </h2>
          </div>
        </div>
        <ul className="divide-y divide-border">
          {templates.map((tpl) => (
            <li
              key={tpl.id}
              className="px-4 py-3 flex items-center justify-between gap-4 hover:bg-panel/20 transition-colors"
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
                  <p className="text-xs text-text-muted">{tpl.description}</p>
                )}
              </div>
              <div className="flex items-center gap-2 shrink-0">
                <a
                  href={tpl.fileUrl}
                  download
                  className="inline-flex items-center gap-1 px-2.5 py-1.5 rounded-lg border border-border text-xs text-text-muted hover:text-text hover:bg-panel transition-colors"
                >
                  <Download size={12} /> Unduh .docx
                </a>
                <button
                  type="button"
                  onClick={() => setModalOpen(true)}
                  className="inline-flex items-center gap-1 px-2.5 py-1.5 rounded-lg border border-border text-xs text-primary font-medium hover:bg-primary/5 transition-colors"
                >
                  <Pencil size={12} /> Kelola
                </button>
              </div>
            </li>
          ))}
        </ul>
      </div>

      {/* 2. Grid Referensi Jenis Surat, Alias Penerima, & Klasifikasi */}
      <div className="grid grid-cols-1 lg:grid-cols-2 gap-5">
        {/* Jenis Surat & Alias Penerima */}
        <div className="border border-border rounded-xl bg-surface overflow-hidden">
          <div className="px-4 py-3 border-b border-border bg-panel/30 flex items-center gap-2">
            <Users size={15} className="text-primary" />
            <h2 className="text-sm font-semibold text-text">
              Jenis Surat ({letterTypes.length}) & Alias Penerima ({aliases.length})
            </h2>
          </div>
          <div className="p-4 space-y-4">
            <div>
              <p className="text-[11px] font-semibold uppercase tracking-wide text-text-muted mb-2">
                Jenis Surat Aktif
              </p>
              <div className="flex flex-wrap gap-2">
                {letterTypes.map((lt) => (
                  <span
                    key={lt.id}
                    className="text-xs px-2.5 py-1 rounded-lg bg-panel border border-border text-text"
                  >
                    <strong className="font-mono text-primary">{lt.code}</strong> · {lt.name}
                  </span>
                ))}
              </div>
            </div>
            <div>
              <p className="text-[11px] font-semibold uppercase tracking-wide text-text-muted mb-2">
                Alias Level Penerima Surat (`/api/v1/recipients/aliases`)
              </p>
              <div className="flex flex-wrap gap-2">
                {aliases.map((al) => (
                  <span
                    key={al.levelId}
                    className="text-xs px-2.5 py-1 rounded-lg bg-primary/10 text-primary font-medium"
                  >
                    {al.label} ({al.code}) — {al.memberCount} Pejabat
                  </span>
                ))}
              </div>
            </div>
          </div>
        </div>

        {/* Klasifikasi Arsip */}
        <div className="border border-border rounded-xl bg-surface overflow-hidden">
          <div className="px-4 py-3 border-b border-border bg-panel/30 flex items-center gap-2">
            <FolderArchive size={15} className="text-primary" />
            <h2 className="text-sm font-semibold text-text">
              Klasifikasi Arsip ({classifications.length})
            </h2>
          </div>
          <ul className="divide-y divide-border max-h-64 overflow-y-auto">
            {classifications.map((c) => (
              <li key={c.id} className="px-4 py-2.5 text-xs flex items-start gap-2">
                <span className="font-mono font-bold text-primary shrink-0">
                  {c.code}
                </span>
                <div className="min-w-0">
                  <p className="font-medium text-text">{c.name}</p>
                  {c.note && <p className="text-text-muted">{c.note}</p>}
                </div>
              </li>
            ))}
          </ul>
        </div>
      </div>

      {modalOpen && (
        <TemplateManagerModal
          templates={templates}
          letterTypes={letterTypes}
          onClose={() => setModalOpen(false)}
          onUpdated={(fresh) => setTemplates(fresh)}
        />
      )}
    </div>
  );
}
