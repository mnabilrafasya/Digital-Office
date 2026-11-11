"use client";

import { useState } from "react";
import { useQuery } from "@tanstack/react-query";
import {
  Search,
  RefreshCw,
  BookOpen,
  Info,
  Tags,
  FolderArchive,
  ShieldCheck,
  Gauge,
  X,
  Database,
} from "lucide-react";
import { masterRequest, type MasterList } from "@/lib/api/master-data";
import { ResourcePanel } from "./resource-panel";
import { lookupResource } from "./config";
import { inputClass, buttonClass } from "./master-form";

function codeBadgeTone(category: string, code: string): string {
  if (category === "CONFIDENTIALITY") {
    if (code.includes("SANGAT_RAHASIA")) {
      return "bg-status-danger-bg text-status-danger-text border-status-danger-text/20";
    }
    if (code.includes("RAHASIA")) {
      return "bg-status-warn-bg text-status-warn-text border-status-warn-text/20";
    }
    if (code.includes("TERBATAS")) {
      return "bg-status-info-bg text-status-info-text border-status-info-text/20";
    }
  }
  if (category === "URGENCY") {
    if (code.includes("SANGAT_SEGERA") || code.includes("KILAT")) {
      return "bg-status-danger-bg text-status-danger-text border-status-danger-text/20";
    }
    if (code.includes("SEGERA") || code.includes("PENTING")) {
      return "bg-status-warn-bg text-status-warn-text border-status-warn-text/20";
    }
  }
  return "bg-primary/10 text-primary border-primary/15";
}

export function LookupPanel({
  category,
  label,
  permissions,
  userId,
  companyCode,
}: {
  category: string;
  label: string;
  permissions: string[];
  userId: string;
  companyCode: string;
}) {
  if (
    (category === "LETTER_TYPE" || category === "CLASSIFICATION") &&
    permissions.includes("ORG_MANAGE")
  ) {
    return (
      <ResourcePanel
        key={category}
        resource={lookupResource(category, label)}
        userId={userId}
        companyCode={companyCode}
      />
    );
  }
  return <LookupReference key={category} category={category} label={label} />;
}

function LookupReference({
  category,
  label,
}: {
  category: string;
  label: string;
}) {
  const [search, setSearch] = useState("");
  const query = useQuery({
    queryKey: ["master", "lookup", category],
    queryFn: () =>
      masterRequest<MasterList>(
        `/lookups?category=${encodeURIComponent(category)}`,
      ),
  });

  const rows = (query.data?.data ?? []).filter((row) =>
    `${row.code} ${row.name} ${row.note ?? ""}`
      .toLowerCase()
      .includes(search.toLowerCase()),
  );

  const Icon =
    category === "CLASSIFICATION"
      ? FolderArchive
      : category === "CONFIDENTIALITY"
        ? ShieldCheck
        : category === "URGENCY"
          ? Gauge
          : Tags;

  return (
    <section className="space-y-5">
      <div className="flex flex-wrap items-start justify-between gap-3">
        <div>
          <div className="flex items-center gap-2">
            <h2 className="font-bold text-base text-text">Daftar {label}</h2>
            <span className="rounded-full bg-panel border border-border px-2.5 py-0.5 text-xs font-semibold text-text-muted">
              {rows.length} data
            </span>
          </div>
          <p className="text-xs text-text-muted mt-0.5">
            Referensi aktif yang ditampilkan pada formulir pembuatan dan
            pengelolaan surat.
          </p>
        </div>
        <span className="inline-flex items-center gap-1.5 bg-status-info-bg text-status-info-text px-3 py-1 rounded-full text-xs font-semibold">
          <BookOpen size={13} />
          Referensi Sistem
        </span>
      </div>

      <div className="flex gap-3 items-start p-3.5 rounded-xl bg-primary/5 border border-primary/15 text-xs text-text-muted leading-relaxed">
        <Info size={16} className="text-primary shrink-0 mt-0.5" />
        <p>
          {["LETTER_TYPE", "CLASSIFICATION"].includes(category)
            ? "Anda sedang melihat daftar referensi aktif. Hubungi Administrator Perusahaan bila diperlukan penambahan atau penyesuaian kode pada kategori ini."
            : "Referensi sifat kerahasiaan dan tingkat kecepatan surat ini ditetapkan secara baku oleh sistem untuk menjaga keseragaman alur persetujuan dan distribusi surat."}
        </p>
      </div>

      <div className="flex gap-2 rounded-xl bg-panel/40 p-3 border border-border">
        <div className="relative flex-1">
          <Search
            size={15}
            className="absolute left-3 top-1/2 -translate-y-1/2 text-text-muted pointer-events-none"
          />
          <input
            aria-label={`Cari ${label}`}
            placeholder={`Cari kode atau nama ${label.toLowerCase()}...`}
            value={search}
            onChange={(event) => setSearch(event.target.value)}
            className={`${inputClass} pl-9 pr-8`}
          />
          {search && (
            <button
              type="button"
              onClick={() => setSearch("")}
              aria-label="Hapus pencarian"
              className="absolute right-2.5 top-1/2 -translate-y-1/2 text-text-muted hover:text-text"
            >
              <X size={14} />
            </button>
          )}
        </div>
        <button
          type="button"
          onClick={() => query.refetch()}
          disabled={query.isFetching}
          className={buttonClass}
          aria-label="Muat ulang"
        >
          <RefreshCw
            size={14}
            className={query.isFetching ? "animate-spin" : ""}
          />
          <span className="hidden sm:inline">Muat ulang</span>
        </button>
      </div>

      {query.isPending ? (
        <div role="status" className="space-y-3 py-2">
          {[1, 2, 3].map((key) => (
            <div key={key} className="h-14 rounded-xl bg-panel animate-pulse" />
          ))}
        </div>
      ) : query.isError ? (
        <div
          role="alert"
          className="rounded-xl bg-status-danger-bg p-4 text-sm text-status-danger-text flex items-center justify-between gap-3"
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
        <div className="overflow-x-auto border border-border rounded-xl bg-surface">
          <table className="text-left text-sm w-full">
            <thead className="bg-panel/70 text-[11px] uppercase tracking-wider text-text-muted border-b border-border">
              <tr>
                <th className="px-4 py-3 w-14 font-semibold">No.</th>
                <th className="px-4 py-3 font-semibold">Kode</th>
                <th className="px-4 py-3 font-semibold">Nama {label}</th>
                <th className="px-4 py-3 font-semibold">Keterangan</th>
                <th className="px-4 py-3 font-semibold">Status</th>
              </tr>
            </thead>
            <tbody className="divide-y divide-border">
              {rows.map((row, index) => {
                const code = String(row.code ?? "");
                return (
                  <tr
                    key={String(row.id)}
                    className="hover:bg-panel/35 transition-colors"
                  >
                    <td className="px-4 py-3.5 text-xs text-text-muted font-mono">
                      {index + 1}
                    </td>
                    <td className="px-4 py-3.5">
                      <span
                        className={`inline-flex font-mono text-xs font-bold rounded-md border px-2.5 py-1 ${codeBadgeTone(category, code)}`}
                      >
                        {code}
                      </span>
                    </td>
                    <td className="px-4 py-3.5 font-semibold text-text">
                      {String(row.name)}
                    </td>
                    <td className="px-4 py-3.5 text-xs text-text-muted max-w-md leading-relaxed">
                      {String(row.note || "—")}
                    </td>
                    <td className="px-4 py-3.5">
                      <span className="inline-flex items-center gap-1.5 rounded-full bg-status-success-bg text-status-success-text px-2.5 py-1 text-xs font-semibold">
                        <span className="h-1.5 w-1.5 rounded-full bg-status-success-text" />
                        Aktif
                      </span>
                    </td>
                  </tr>
                );
              })}
              {!rows.length && (
                <tr>
                  <td colSpan={5} className="p-12 text-center">
                    <Icon
                      size={30}
                      className="mx-auto text-text-muted mb-3 opacity-70"
                    />
                    <p className="font-semibold text-sm text-text">
                      {search
                        ? "Data tidak ditemukan"
                        : `Belum ada ${label.toLowerCase()}`}
                    </p>
                    <p className="mt-1 text-xs text-text-muted">
                      {search
                        ? "Gunakan kata kunci lain atau kosongkan pencarian."
                        : "Data yang tersedia akan ditampilkan di sini."}
                    </p>
                  </td>
                </tr>
              )}
            </tbody>
          </table>
        </div>
      )}

      <p className="inline-flex items-center gap-1.5 text-xs text-text-muted">
        <Database size={13} className="text-primary" />
        Menampilkan <strong className="text-text">{rows.length}</strong> dari{" "}
        <strong className="text-text">{query.data?.data?.length ?? 0}</strong>{" "}
        data
      </p>
    </section>
  );
}
