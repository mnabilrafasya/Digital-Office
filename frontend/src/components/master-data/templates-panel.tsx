"use client";

import { useState } from "react";
import dynamic from "next/dynamic";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import {
  FileText,
  Plus,
  Search,
  Pencil,
  Download,
  Power,
  Trash2,
  RefreshCw,
  Info,
  CheckCircle2,
  X,
  Tags,
  Loader2,
} from "lucide-react";
import {
  masterRequest,
  type MasterList,
  type MasterRow,
} from "@/lib/api/master-data";
import { invalidateTemplateCache } from "@/lib/templates/letter-templates";
import {
  MasterForm,
  inputClass,
  buttonClass,
  primaryButtonClass,
  rowActionButtonClass,
  rowDangerButtonClass,
} from "./master-form";

const TemplateDocxEditorModal = dynamic(
  () =>
    import("./template-docx-editor").then((m) => ({
      default: m.TemplateDocxEditorModal,
    })),
  {
    ssr: false,
    loading: () => (
      <div className="fixed inset-0 z-50 bg-black/50 flex items-center justify-center p-4">
        <div className="bg-surface border border-border rounded-2xl p-8 flex flex-col items-center gap-3 shadow-xl">
          <Loader2 size={24} className="animate-spin text-primary" />
          <p className="text-sm font-semibold text-text">
            Memuat Editor Template DevExpress Word...
          </p>
        </div>
      </div>
    ),
  },
);

const STATUS_TABS = [
  { key: "", label: "Semua Status" },
  { key: "ACTIVE", label: "Aktif" },
  { key: "INACTIVE", label: "Nonaktif" },
] as const;

export function TemplatesPanel({ permissions }: { permissions: string[] }) {
  const cache = useQueryClient();
  const manage = permissions.includes("TEMPLATE_MANAGE");
  const create = permissions.includes("TEMPLATE_CREATE");
  const [q, setQ] = useState("");
  const [status, setStatus] = useState("");
  const [action, setAction] = useState<{
    kind: "create" | "edit" | "status" | "delete";
    row?: MasterRow;
  } | null>(null);
  const [notice, setNotice] = useState("");

  const query = useQuery({
    queryKey: ["master", "template-table", manage, status],
    queryFn: () =>
      masterRequest<MasterList>(
        manage ? `/admin/templates?status=${status}` : "/templates",
      ),
  });

  const all = query.data?.data ?? [];
  const rows = all.filter((row) =>
    `${row.name} ${row.description ?? ""} ${(row.letterType as MasterRow | null)?.name ?? ""}`
      .toLowerCase()
      .includes(q.toLowerCase()),
  );

  async function saveStatus() {
    if (!action || action.kind !== "status") return;
    await masterRequest(
      `/admin/templates/${encodeURIComponent(String(action.row?.id))}/status`,
      "PATCH",
      { active: !action.row?.isActive },
    );
    invalidateTemplateCache();
    setNotice("Status template surat berhasil diperbarui.");
    await cache.invalidateQueries({ queryKey: ["master"] });
  }

  async function deleteTemplate() {
    if (!action || action.kind !== "delete") return;
    await masterRequest(
      `/admin/templates/${encodeURIComponent(String(action.row?.id))}`,
      "DELETE",
    );
    invalidateTemplateCache();
    setNotice("Template surat berhasil dihapus.");
    await cache.invalidateQueries({ queryKey: ["master"] });
  }

  return (
    <section className="space-y-5">
      {/* Header */}
      <div className="flex flex-wrap justify-between items-start gap-3">
        <div>
          <div className="flex items-center gap-2">
            <h2 className="font-bold text-base text-text">
              Daftar Template Surat (.docx)
            </h2>
            <span className="rounded-full bg-panel border border-border px-2.5 py-0.5 text-xs font-semibold text-text-muted">
              {rows.length} template
            </span>
          </div>
          <p className="text-xs text-text-muted mt-0.5">
            Unggah berkas Microsoft Word (.docx) atau desain langsung menggunakan
            editor tata naskah DevExpress Word.
          </p>
        </div>
        {create && (
          <button
            type="button"
            onClick={() => {
              setNotice("");
              setAction({ kind: "create" });
            }}
            className={primaryButtonClass}
          >
            <Plus size={15} />
            Tambah & Desain Template
          </button>
        )}
      </div>

      {/* Status Tabs for Admin */}
      {manage && (
        <div className="flex items-center justify-between border-b border-border gap-3">
          <div
            role="tablist"
            aria-label="Filter status template"
            className="flex items-center gap-1 overflow-x-auto no-scrollbar"
          >
            {STATUS_TABS.map((tab) => {
              const isActive = status === tab.key;
              return (
                <button
                  key={tab.key || "ALL"}
                  type="button"
                  role="tab"
                  aria-selected={isActive}
                  onClick={() => setStatus(tab.key)}
                  className={`relative px-3.5 py-2.5 text-xs font-semibold transition-colors whitespace-nowrap ${
                    isActive
                      ? "text-primary"
                      : "text-text-muted hover:text-text"
                  }`}
                >
                  {tab.label}
                  {isActive && (
                    <span className="absolute inset-x-0 bottom-0 h-0.5 bg-primary rounded-full" />
                  )}
                </button>
              );
            })}
          </div>

          <button
            type="button"
            className="inline-flex items-center gap-1.5 text-xs font-medium text-text-muted hover:text-primary px-2 py-1.5 transition-colors shrink-0"
            onClick={() => query.refetch()}
            disabled={query.isFetching}
          >
            <RefreshCw
              size={13}
              className={query.isFetching ? "animate-spin text-primary" : ""}
            />
            <span className="hidden sm:inline">Muat ulang</span>
          </button>
        </div>
      )}

      {/* Search Bar */}
      <div className="flex flex-wrap gap-2 rounded-xl bg-panel/40 p-3 border border-border">
        <div className="relative flex-1 min-w-52">
          <Search
            size={15}
            className="absolute top-1/2 -translate-y-1/2 left-3 text-text-muted pointer-events-none"
          />
          <input
            aria-label="Cari template"
            placeholder="Cari nama, jenis surat, atau deskripsi template..."
            value={q}
            onChange={(event) => setQ(event.target.value)}
            className={`${inputClass} pl-9 pr-8`}
          />
          {q && (
            <button
              type="button"
              onClick={() => setQ("")}
              aria-label="Hapus pencarian"
              className="absolute right-2.5 top-1/2 -translate-y-1/2 text-text-muted hover:text-text"
            >
              <X size={14} />
            </button>
          )}
        </div>
        {!manage && (
          <button
            type="button"
            className={buttonClass}
            onClick={() => query.refetch()}
            disabled={query.isFetching}
            aria-label="Muat ulang template"
          >
            <RefreshCw
              size={14}
              className={query.isFetching ? "animate-spin" : ""}
            />
            Muat ulang
          </button>
        )}
      </div>

      {notice && (
        <p
          role="status"
          className="flex items-center justify-between gap-2 bg-status-success-bg text-status-success-text rounded-xl px-4 py-3 text-xs font-medium"
        >
          <span className="flex items-center gap-2">
            <CheckCircle2 size={15} />
            {notice}
          </span>
          <button
            type="button"
            onClick={() => setNotice("")}
            aria-label="Tutup notifikasi"
          >
            <X size={14} />
          </button>
        </p>
      )}

      {query.isPending ? (
        <div
          role="status"
          className="rounded-xl border border-border overflow-hidden divide-y divide-border"
        >
          {[1, 2, 3].map((item) => (
            <div
              key={item}
              className="p-4 flex items-center justify-between gap-4"
            >
              <div className="space-y-2 flex-1">
                <div className="h-4 w-52 rounded bg-panel animate-pulse" />
                <div className="h-3 w-72 rounded bg-panel/70 animate-pulse" />
              </div>
              <div className="h-7 w-24 rounded-full bg-panel animate-pulse" />
            </div>
          ))}
        </div>
      ) : query.isError ? (
        <div
          role="alert"
          className="p-4 bg-status-danger-bg text-status-danger-text rounded-xl text-sm flex items-center justify-between gap-3"
        >
          <span>{query.error.message}</span>
          <button
            type="button"
            className="underline font-semibold"
            onClick={() => query.refetch()}
          >
            Coba lagi
          </button>
        </div>
      ) : (
        <div className="border border-border rounded-xl overflow-x-auto bg-surface">
          <table className="w-full text-sm text-left">
            <thead className="bg-panel/70 text-[11px] uppercase tracking-wider text-text-muted border-b border-border">
              <tr>
                <th className="px-4 py-3 font-semibold">
                  Template & Deskripsi
                </th>
                <th className="px-4 py-3 font-semibold">Jenis Surat</th>
                {manage && (
                  <th className="px-4 py-3 font-semibold">Pemakaian</th>
                )}
                <th className="px-4 py-3 font-semibold">Status</th>
                <th className="px-4 py-3 font-semibold text-right">Tindakan</th>
              </tr>
            </thead>
            <tbody className="divide-y divide-border">
              {rows.map((row) => {
                const active = !manage || Boolean(row.isActive);
                const letterType = row.letterType as MasterRow | null;
                return (
                  <tr
                    key={String(row.id)}
                    className={`hover:bg-panel/35 transition-colors ${
                      !active ? "opacity-75" : ""
                    }`}
                  >
                    <td className="px-3.5 py-3 align-top">
                      <div className="flex gap-2.5 items-start min-w-0">
                        <span className="p-2 bg-primary/10 rounded-xl text-primary shrink-0 mt-0.5">
                          <FileText size={17} />
                        </span>
                        <div className="space-y-1 min-w-0">
                          <p className="font-semibold text-text leading-snug">
                            {String(row.name)}
                          </p>
                          <p className="text-xs text-text-muted max-w-md leading-relaxed">
                            {String(
                              row.description ||
                                "Tidak ada catatan penggunaan khusus.",
                            )}
                          </p>
                        </div>
                      </div>
                    </td>
                    <td className="px-3.5 py-3 align-top">
                      <span className="inline-flex items-center gap-1.5 rounded-md bg-panel border border-border px-2.5 py-1 text-xs font-medium text-text whitespace-nowrap">
                        <Tags size={12} className="text-primary" />
                        {String(letterType?.name ?? "Semua Jenis Surat")}
                      </span>
                    </td>
                    {manage && (
                      <td className="px-3.5 py-3 align-top text-xs whitespace-nowrap">
                        <span className="rounded-md bg-panel px-2 py-1 font-medium text-text-muted border border-border">
                          <strong className="text-text">
                            {Number(row.usedBy ?? 0)}
                          </strong>{" "}
                          surat
                        </span>
                      </td>
                    )}
                    <td className="px-3.5 py-3 align-top whitespace-nowrap">
                      <span
                        className={`inline-flex items-center gap-1.5 rounded-full text-xs font-semibold px-2.5 py-1 ${
                          active
                            ? "bg-status-success-bg text-status-success-text"
                            : "bg-panel text-text-muted border border-border"
                        }`}
                      >
                        <span
                          className={`h-1.5 w-1.5 rounded-full ${
                            active ? "bg-status-success-text" : "bg-text-muted"
                          }`}
                        />
                        {active ? "Aktif" : "Nonaktif"}
                      </span>
                    </td>
                    <td className="px-3.5 py-3 align-top">
                      <div className="flex flex-wrap justify-end gap-1.5">
                        <a
                          aria-label={`Unduh ${row.name}`}
                          title="Unduh berkas .docx"
                          href={`/api/v1/templates/${encodeURIComponent(String(row.id))}/file`}
                          className={rowActionButtonClass}
                        >
                          <Download size={13} />
                          <span>.docx</span>
                        </a>
                        {create && active && (
                          <button
                            type="button"
                            className={rowActionButtonClass}
                            onClick={() => setAction({ kind: "edit", row })}
                          >
                            <Pencil size={13} />
                            Edit di Editor
                          </button>
                        )}
                        {manage && (
                          <>
                            <button
                              type="button"
                              className={rowActionButtonClass}
                              title={
                                active
                                  ? "Nonaktifkan template"
                                  : "Aktifkan template"
                              }
                              onClick={() => setAction({ kind: "status", row })}
                            >
                              <Power size={13} />
                              <span>{active ? "Nonaktifkan" : "Aktifkan"}</span>
                            </button>
                            <button
                              type="button"
                              className={rowDangerButtonClass}
                              title={`Hapus template ${String(row.name)}`}
                              onClick={() => setAction({ kind: "delete", row })}
                            >
                              <Trash2 size={13} />
                              <span>Hapus</span>
                            </button>
                          </>
                        )}
                      </div>
                    </td>
                  </tr>
                );
              })}

              {!rows.length && (
                <tr>
                  <td
                    colSpan={manage ? 5 : 4}
                    className="py-12 text-center"
                  >
                    <FileText
                      size={30}
                      className="mx-auto text-text-muted mb-3 opacity-70"
                    />
                    <p className="font-semibold text-sm text-text">
                      {q
                        ? "Template tidak ditemukan"
                        : "Belum ada template surat"}
                    </p>
                    <p className="text-xs text-text-muted mt-1">
                      {q
                        ? "Gunakan kata kunci pencarian lain."
                        : "Klik Tambah & Desain Template untuk membuat atau mengunggah template .docx pertama."}
                    </p>
                  </td>
                </tr>
              )}
            </tbody>
          </table>
        </div>
      )}

      <div className="flex items-start gap-2.5 rounded-xl bg-panel/40 border border-border p-3.5 text-xs text-text-muted">
        <Info size={15} className="shrink-0 text-primary mt-0.5" />
        <p>
          {manage
            ? "Template nonaktif tidak muncul pada pilihan pembuatan surat baru. Aktifkan kembali template terlebih dahulu bila ingin mengedit naskahnya di editor DevExpress."
            : "Pengaktifan dan penonaktifan template dikelola oleh administrator perusahaan."}
        </p>
      </div>

      {(action?.kind === "create" || action?.kind === "edit") && (
        <TemplateDocxEditorModal
          mode={action.kind}
          row={action.kind === "edit" ? action.row : undefined}
          onClose={() => setAction(null)}
          onSaved={async (msg) => {
            setNotice(msg);
            await cache.invalidateQueries({ queryKey: ["master"] });
          }}
        />
      )}

      {action?.kind === "status" && (
        <MasterForm
          title={`${action.row?.isActive ? "Nonaktifkan" : "Aktifkan"} Template`}
          fields={[]}
          description={`Ubah status template "${action.row?.name}"? Surat lama yang sudah memakai template ini tetap tersimpan dengan aman.`}
          onClose={() => setAction(null)}
          onSubmit={saveStatus}
        />
      )}

      {action?.kind === "delete" && (
        <MasterForm
          title="Hapus Template Surat"
          variant="danger"
          fields={[]}
          description={`Hapus template "${action.row?.name}" dari daftar master data? Surat lama yang pernah memakai template ini tetap tersimpan dengan aman di database.`}
          onClose={() => setAction(null)}
          onSubmit={deleteTemplate}
        />
      )}
    </section>
  );
}
