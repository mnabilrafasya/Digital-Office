"use client";

import { useState, type FormEvent } from "react";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import {
  Stamp,
  Hash,
  Eye,
  Save,
  RotateCcw,
  CheckCircle2,
  AlertCircle,
  Sparkles,
  FileCheck2,
  RefreshCw,
} from "lucide-react";
import { masterRequest, type CompanySettings } from "@/lib/api/master-data";
import { buttonClass, primaryButtonClass, inputClass } from "./master-form";

const FORMAT_TOKENS = [
  {
    token: "{SEQ}",
    label: "Nomor Urut",
    desc: "Urutan surat otomatis (001, 002, dst.)",
    required: true,
  },
  {
    token: "{COMP}",
    label: "Kode Perusahaan",
    desc: "Kode identitas entitas perusahaan",
    required: true,
  },
  {
    token: "{MM_ROMAN}",
    label: "Bulan Romawi",
    desc: "Bulan berjalan dalam angka Romawi (I–XII)",
    required: true,
  },
  {
    token: "{YYYY}",
    label: "Tahun",
    desc: "Tahun terbit surat 4 digit",
    required: true,
  },
] as const;

function SettingsForm({ settings }: { settings: CompanySettings }) {
  const cache = useQueryClient();
  const [format, setFormat] = useState(settings.letterNumberFormat);
  const [watermark, setWatermark] = useState(settings.watermarkText ?? "");
  const [preview, setPreview] = useState(settings.numberPreview);
  const [previewedWatermark, setPreviewedWatermark] = useState(
    settings.watermarkText ?? "",
  );
  const [error, setError] = useState("");
  const [notice, setNotice] = useState("");
  const [busy, setBusy] = useState(false);

  const imageUrl = previewedWatermark.trim()
    ? `/api/v1/company/watermark-preview?text=${encodeURIComponent(previewedWatermark.trim())}`
    : "";

  async function run(action: () => Promise<void>) {
    setBusy(true);
    setError("");
    setNotice("");
    try {
      await action();
    } catch (err) {
      setError(err instanceof Error ? err.message : "Permintaan gagal.");
    } finally {
      setBusy(false);
    }
  }

  async function save(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    await run(async () => {
      const trimmedWatermark = watermark.trim();
      const result = await masterRequest<{ data: CompanySettings }>(
        "/company",
        "PUT",
        {
          letterNumberFormat: format.trim(),
          watermarkText: trimmedWatermark || null,
        },
      );
      setPreview(result.data.numberPreview);
      setPreviewedWatermark(result.data.watermarkText ?? "");
      setNotice(
        "Pengaturan format nomor surat dan watermark PDF berhasil disimpan.",
      );
      await cache.invalidateQueries({ queryKey: ["master"] });
    });
  }

  function resetDraft() {
    setFormat(settings.letterNumberFormat);
    setWatermark(settings.watermarkText ?? "");
    setPreview(settings.numberPreview);
    setPreviewedWatermark(settings.watermarkText ?? "");
    setError("");
    setNotice("");
  }

  const watermarkPresets = [
    settings.code,
    `${settings.code} - NASKAH RESMI`,
    "SALINAN RESMI",
  ];

  const formatPresets = [
    "{SEQ}/{COMP}/{MM_ROMAN}/{YYYY}",
    "{COMP}-{SEQ}/{MM_ROMAN}/{YYYY}",
    "{SEQ}/{MM_ROMAN}/{COMP}/{YYYY}",
  ];

  return (
    <form onSubmit={save} className="space-y-6">
      {/* Top Summary Banner */}
      <div className="rounded-xl border border-border bg-panel/40 p-4 sm:p-5 flex flex-wrap items-center justify-between gap-4">
        <div className="space-y-1">
          <div className="flex flex-wrap items-center gap-2">
            <span className="rounded-md bg-primary/10 text-primary font-mono font-bold text-xs px-2.5 py-1 border border-primary/20">
              {settings.code}
            </span>
            <h2 className="text-base font-bold text-text">{settings.name}</h2>
          </div>
          <p className="text-xs text-text-muted">
            Atur pola penomoran surat keluar otomatis serta watermark diagonal
            pada dokumen PDF resmi yang diterbitkan sistem.
          </p>
        </div>

        <div className="flex flex-wrap items-center gap-2.5 text-xs">
          <div className="rounded-lg border border-border bg-surface px-3 py-2">
            <span className="text-text-muted block text-[10px] uppercase tracking-wider font-semibold">
              Contoh Nomor Aktif
            </span>
            <span className="font-mono font-bold text-text">
              {settings.numberPreview}
            </span>
          </div>
          <div className="rounded-lg border border-border bg-surface px-3 py-2">
            <span className="text-text-muted block text-[10px] uppercase tracking-wider font-semibold">
              Watermark PDF Aktif
            </span>
            <span className="font-semibold text-text">
              {settings.watermarkText || "Tanpa Watermark"}
            </span>
          </div>
        </div>
      </div>

      {/* Feedback Alerts */}
      {error && (
        <div
          role="alert"
          className="flex items-center gap-2.5 rounded-xl bg-status-danger-bg px-4 py-3 text-xs font-medium text-status-danger-text"
        >
          <AlertCircle size={16} className="shrink-0" />
          <span>{error}</span>
        </div>
      )}
      {notice && (
        <div
          role="status"
          className="flex items-center gap-2.5 rounded-xl bg-status-success-bg px-4 py-3 text-xs font-medium text-status-success-text"
        >
          <CheckCircle2 size={16} className="shrink-0" />
          <span>{notice}</span>
        </div>
      )}

      {/* Two-Column Workspace */}
      <div className="grid gap-5 lg:grid-cols-2">
        {/* Card 1: Letter Numbering Format */}
        <div className="rounded-xl border border-border bg-surface p-5 space-y-5 flex flex-col justify-between">
          <div className="space-y-4">
            <div className="flex items-start gap-3">
              <div className="p-2.5 rounded-xl bg-primary/10 text-primary shrink-0">
                <Hash size={19} />
              </div>
              <div>
                <h3 className="font-bold text-sm text-text">
                  1. Format Penomoran Surat Otomatis
                </h3>
                <p className="text-xs text-text-muted mt-0.5 leading-relaxed">
                  Digunakan saat surat disetujui (Approved) untuk menerbitkan
                  nomor surat resmi secara berurutan.
                </p>
              </div>
            </div>

            <label className="block space-y-1.5 text-xs font-semibold text-text">
              <span>Pola Format Nomor Surat *</span>
              <input
                required
                value={format}
                onChange={(event) => setFormat(event.target.value)}
                placeholder="{SEQ}/{COMP}/{MM_ROMAN}/{YYYY}"
                className={`${inputClass} font-mono`}
              />
            </label>

            {/* Quick Presets */}
            <div className="space-y-1.5">
              <span className="text-[11px] font-medium text-text-muted">
                Contoh pola standar (klik untuk menerapkan):
              </span>
              <div className="flex flex-wrap gap-1.5">
                {formatPresets.map((preset) => (
                  <button
                    key={preset}
                    type="button"
                    onClick={() => setFormat(preset)}
                    className={`rounded-lg border px-2.5 py-1 text-xs font-mono transition-colors ${
                      format === preset
                        ? "border-primary bg-primary/10 text-primary font-semibold"
                        : "border-border bg-panel/60 text-text-muted hover:text-text"
                    }`}
                  >
                    {preset}
                  </button>
                ))}
              </div>
            </div>

            {/* Token Guide */}
            <div className="rounded-xl border border-border bg-panel/40 p-3.5 space-y-2.5">
              <p className="text-[11px] font-semibold text-text">
                Token Variabel yang Tersedia (klik untuk menyisipkan):
              </p>
              <div className="grid gap-2 sm:grid-cols-2">
                {FORMAT_TOKENS.map((item) => {
                  const included = format.includes(item.token);
                  return (
                    <button
                      key={item.token}
                      type="button"
                      onClick={() => {
                        if (!format.includes(item.token)) {
                          setFormat((prev) =>
                            prev ? `${prev}/${item.token}` : item.token,
                          );
                        }
                      }}
                      className="text-left rounded-lg border border-border bg-surface p-2.5 hover:border-primary/40 transition-colors"
                    >
                      <div className="flex items-center justify-between gap-1">
                        <code className="font-mono text-xs font-bold text-primary">
                          {item.token}
                        </code>
                        <span
                          className={`text-[10px] px-1.5 py-0.5 rounded font-semibold ${
                            included
                              ? "bg-status-success-bg text-status-success-text"
                              : item.required
                                ? "bg-status-warn-bg text-status-warn-text"
                                : "bg-panel text-text-muted"
                          }`}
                        >
                          {included
                            ? "Terpakai"
                            : item.required
                              ? "Wajib"
                              : "Opsional"}
                        </span>
                      </div>
                      <p className="text-[11px] text-text-muted mt-1 leading-snug">
                        {item.desc}
                      </p>
                    </button>
                  );
                })}
              </div>
            </div>
          </div>

          {/* Number Preview Box */}
          <div className="rounded-xl border border-primary/20 bg-primary/5 p-4 flex flex-wrap items-center justify-between gap-3">
            <div>
              <span className="text-[10px] font-semibold uppercase tracking-wider text-primary block">
                Hasil Pratinjau Nomor Surat
              </span>
              <output className="text-base font-mono font-bold text-text mt-0.5 block">
                {preview}
              </output>
            </div>
            <button
              type="button"
              className={`${buttonClass} bg-surface`}
              disabled={busy}
              onClick={() =>
                run(async () => {
                  const result = await masterRequest<{
                    data: { numberPreview: string };
                  }>(
                    `/company/number-preview?format=${encodeURIComponent(format)}`,
                  );
                  setPreview(result.data.numberPreview);
                })
              }
            >
              <Eye size={14} />
              Uji Pratinjau Nomor
            </button>
          </div>
        </div>

        {/* Card 2: PDF Watermark Configuration */}
        <div className="rounded-xl border border-border bg-surface p-5 space-y-5 flex flex-col justify-between">
          <div className="space-y-4">
            <div className="flex items-start gap-3">
              <div className="p-2.5 rounded-xl bg-primary/10 text-primary shrink-0">
                <Stamp size={19} />
              </div>
              <div>
                <div className="flex items-center gap-2">
                  <h3 className="font-bold text-sm text-text">
                    2. Watermark PDF Naskah Resmi
                  </h3>
                  <span className="inline-flex items-center gap-1 rounded-full bg-status-info-bg text-status-info-text px-2 py-0.5 text-[10px] font-semibold">
                    <Sparkles size={10} />
                    Otomatis di PDF
                  </span>
                </div>
                <p className="text-xs text-text-muted mt-0.5 leading-relaxed">
                  Teks watermark dicetak miring (diagonal) secara transparan
                  pada setiap halaman PDF surat resmi yang telah disetujui.
                </p>
              </div>
            </div>

            <label className="block space-y-1.5 text-xs font-semibold text-text">
              <span>Teks Watermark (maks. 100 karakter)</span>
              <input
                maxLength={100}
                value={watermark}
                placeholder={`Contoh: ${settings.code} atau kosongkan untuk tanpa watermark`}
                onChange={(event) => setWatermark(event.target.value)}
                className={inputClass}
              />
            </label>

            {/* Quick Watermark Presets */}
            <div className="space-y-1.5">
              <span className="text-[11px] font-medium text-text-muted">
                Pilihan cepat watermark:
              </span>
              <div className="flex flex-wrap gap-1.5">
                {watermarkPresets.map((preset) => (
                  <button
                    key={preset}
                    type="button"
                    onClick={() => {
                      setWatermark(preset);
                      setPreviewedWatermark(preset);
                    }}
                    className={`rounded-lg border px-2.5 py-1 text-xs transition-colors ${
                      watermark === preset
                        ? "border-primary bg-primary/10 text-primary font-semibold"
                        : "border-border bg-panel/60 text-text-muted hover:text-text"
                    }`}
                  >
                    {preset}
                  </button>
                ))}
                <button
                  type="button"
                  onClick={() => {
                    setWatermark("");
                    setPreviewedWatermark("");
                  }}
                  className={`rounded-lg border px-2.5 py-1 text-xs transition-colors ${
                    !watermark.trim()
                      ? "border-primary bg-primary/10 text-primary font-semibold"
                      : "border-border bg-panel/60 text-text-muted hover:text-text"
                  }`}
                >
                  Tanpa Watermark
                </button>
              </div>
            </div>

            {/* Live / On-demand A4 Watermark Preview */}
            <div className="rounded-xl border border-border bg-panel/50 p-4 space-y-3">
              <div className="flex flex-wrap items-center justify-between gap-2">
                <span className="text-xs font-semibold text-text flex items-center gap-1.5">
                  <FileCheck2 size={14} className="text-primary" />
                  Simulasi Tampilan Watermark pada Dokumen PDF
                </span>
                <button
                  type="button"
                  className={buttonClass}
                  disabled={busy || !watermark.trim()}
                  onClick={() => setPreviewedWatermark(watermark.trim())}
                >
                  <RefreshCw size={13} />
                  Pratinjau Watermark
                </button>
              </div>

              {imageUrl ? (
                <div className="rounded-lg border border-border bg-white p-4 shadow-inner flex flex-col items-center justify-center min-h-48">
                  {/* eslint-disable-next-line @next/next/no-img-element */}
                  <img
                    src={imageUrl}
                    alt={`Pratinjau watermark ${previewedWatermark}`}
                    className="max-h-44 mx-auto object-contain"
                    onError={() => {
                      setPreviewedWatermark("");
                      setError(
                        "Pratinjau gambar watermark gagal dimuat. Silakan coba lagi.",
                      );
                    }}
                  />
                  <span className="text-[11px] text-neutral-500 mt-2 font-mono">
                    Watermark aktif: &ldquo;{previewedWatermark}&rdquo;
                  </span>
                </div>
              ) : (
                <div className="rounded-lg border border-dashed border-border bg-surface/80 p-6 text-center min-h-44 flex flex-col items-center justify-center">
                  <Stamp size={26} className="text-text-muted mb-2 opacity-60" />
                  <p className="text-xs font-semibold text-text">
                    {watermark.trim()
                      ? "Klik tombol Pratinjau Watermark untuk melihat hasil render"
                      : "Mode Tanpa Watermark"}
                  </p>
                  <p className="text-[11px] text-text-muted mt-1 max-w-xs">
                    {watermark.trim()
                      ? `Teks "${watermark.trim()}" siap dirender sebagai watermark PDF.`
                      : "Dokumen PDF surat resmi akan diterbitkan bersih tanpa latar belakang watermark."}
                  </p>
                </div>
              )}
            </div>
          </div>

          <p className="text-[11px] text-text-muted leading-relaxed">
            Perubahan watermark dan format nomor berlaku untuk surat yang
            diterbitkan setelah pengaturan disimpan. PDF surat yang telah terbit
            sebelumnya tetap mempertahankan nomor dan watermark saat disahkan.
          </p>
        </div>
      </div>

      {/* Action Footer */}
      <div className="flex flex-wrap items-center justify-end gap-2.5 border-t border-border pt-4">
        <button
          type="button"
          disabled={busy}
          onClick={resetDraft}
          className={buttonClass}
        >
          <RotateCcw size={14} />
          Kembalikan
        </button>
        <button
          type="submit"
          disabled={busy}
          className={primaryButtonClass}
        >
          <Save size={14} />
          {busy ? "Menyimpan..." : "Simpan Pengaturan Perusahaan"}
        </button>
      </div>
    </form>
  );
}

export function CompanySettingsPanel() {
  const query = useQuery({
    queryKey: ["master", "settings"],
    queryFn: () => masterRequest<{ data: CompanySettings }>("/company"),
  });

  if (query.isPending) {
    return (
      <div role="status" className="space-y-4 py-4">
        <div className="h-20 rounded-xl bg-panel animate-pulse" />
        <div className="grid gap-5 lg:grid-cols-2">
          <div className="h-80 rounded-xl bg-panel animate-pulse" />
          <div className="h-80 rounded-xl bg-panel animate-pulse" />
        </div>
      </div>
    );
  }

  if (query.isError) {
    return (
      <div
        role="alert"
        className="rounded-xl bg-status-danger-bg p-4 text-sm text-status-danger-text flex items-center justify-between gap-3"
      >
        <span>{query.error.message}</span>
        <button
          type="button"
          onClick={() => query.refetch()}
          className="underline font-semibold"
        >
          Coba lagi
        </button>
      </div>
    );
  }

  return <SettingsForm settings={query.data.data} />;
}
