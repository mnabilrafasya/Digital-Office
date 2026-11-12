"use client";

import { X, Save, Trash2, Eye, EyeOff, AlertCircle } from "lucide-react";
import { useEffect, useRef, useState, type FormEvent } from "react";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import { valueAt, type MasterRow, type Choice } from "@/lib/api/master-data";
import { loadChoices, type Field } from "./config";

export const inputClass =
  "w-full rounded-lg border border-border bg-surface px-3 py-2 text-sm text-text placeholder:text-text-muted/70 focus:outline-none focus:ring-2 focus:ring-primary/30 focus:border-primary/50 transition-colors";

const baseButtonClass =
  "inline-flex items-center justify-center gap-1.5 rounded-lg border px-3 py-2 text-xs font-semibold transition-colors focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary/40 disabled:opacity-50 disabled:cursor-not-allowed";

export const buttonClass = `${baseButtonClass} border-border bg-surface text-text hover:bg-panel/60 hover:border-primary/40 hover:text-primary`;

export const primaryButtonClass = `${baseButtonClass} border-primary bg-primary text-white hover:bg-primary-dark shadow-2xs`;

export const softPrimaryButtonClass = `${baseButtonClass} border-primary/25 bg-primary/10 text-primary hover:bg-primary hover:text-white`;

export const dangerButtonClass = `${baseButtonClass} border-status-danger-text bg-status-danger-text text-white hover:opacity-90 shadow-2xs`;

export const rowActionButtonClass =
  "inline-flex items-center justify-center gap-1 rounded-lg border border-border bg-surface px-2.5 py-1.5 text-xs font-semibold text-text transition-colors hover:bg-panel/60 hover:border-primary/40 hover:text-primary focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary/40 disabled:opacity-50 disabled:cursor-not-allowed";

export const rowDangerButtonClass =
  "inline-flex items-center justify-center gap-1 rounded-lg border border-border bg-surface px-2.5 py-1.5 text-xs font-semibold text-text-muted transition-colors hover:border-status-danger-text/50 hover:bg-status-danger-bg/50 hover:text-status-danger-text focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-status-danger-text/40 disabled:opacity-50 disabled:cursor-not-allowed";

function SelectField({
  field,
  row,
  company,
}: {
  field: Field;
  row?: MasterRow;
  company?: string;
}) {
  const query = useQuery({
    queryKey: ["master", "choices", field.choices, company],
    queryFn: () => loadChoices(field.choices!, company),
  });

  const selected = String(valueAt(row ?? {}, field.source ?? field.key) ?? "");
  const choices: Choice[] = (query.data ?? []).filter(
    (choice) => !(field.key === "parentId" && choice.value === row?.id),
  );

  // Nilai nonaktif lama tetap dapat dilihat/dipertahankan saat mengedit.
  if (selected && !choices.some((choice) => choice.value === selected)) {
    choices.push({ value: selected, label: `${selected} (nilai saat ini)` });
  }

  return (
    <>
      <select
        key={`${field.key}-${selected}-${query.isSuccess}`}
        name={field.key}
        defaultValue={selected}
        required={field.required || !query.isSuccess}
        className={inputClass}
      >
        <option value="">
          {query.isPending
            ? "Memuat pilihan..."
            : field.required
              ? "Pilih..."
              : field.key === "letterTypeId"
                ? "Semua jenis surat"
                : "Tidak dipilih (Tanpa unit induk)"}
        </option>
        {choices.map((choice) => (
          <option key={choice.value} value={choice.value}>
            {choice.label}
          </option>
        ))}
      </select>
      {query.isError && (
        <div role="alert" className="text-status-danger-text text-xs mt-1">
          {query.error.message}{" "}
          <button
            type="button"
            onClick={() => query.refetch()}
            className="underline font-semibold"
          >
            Coba lagi
          </button>
        </div>
      )}
    </>
  );
}

export function MasterForm({
  title,
  fields,
  row,
  company,
  onClose,
  onSubmit,
  description,
  variant = "default",
}: {
  title: string;
  fields: Field[];
  row?: MasterRow;
  company?: string;
  description?: string;
  variant?: "default" | "danger";
  onClose: () => void;
  onSubmit: (values: MasterRow) => Promise<void>;
}) {
  const cache = useQueryClient();
  const dialog = useRef<HTMLDialogElement>(null);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");
  const [showPassword, setShowPassword] = useState(false);

  useEffect(() => {
    const element = dialog.current!;
    element.showModal();
    return () => element.close();
  }, []);

  async function submit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (
      fields.some(
        (field) =>
          field.choices &&
          cache.getQueryState(["master", "choices", field.choices, company])
            ?.status !== "success",
      )
    ) {
      setError("Pilihan form belum tersedia. Tunggu pemuatan atau coba lagi.");
      return;
    }

    const data = new FormData(event.currentTarget);
    const values: MasterRow = {};
    for (const field of fields) {
      const value =
        field.type === "file"
          ? data.get(field.key)
          : field.type === "checkbox"
            ? data.has(field.key)
            : String(data.get(field.key) ?? "");

      values[field.key] =
        field.type === "number"
          ? Number(value)
          : field.type === "password"
            ? value
            : typeof value === "string"
              ? value.trim()
              : value;

      if (
        field.type === "password" &&
        (Array.from(String(value)).length < 8 ||
          new TextEncoder().encode(String(value)).length > 72)
      ) {
        setError("Password harus minimal 8 karakter dan maksimal 72 byte.");
        return;
      }
    }

    setBusy(true);
    setError("");
    try {
      await onSubmit(values);
      onClose();
    } catch (err) {
      setError(err instanceof Error ? err.message : "Gagal menyimpan data.");
    } finally {
      setBusy(false);
    }
  }

  return (
    <dialog
      ref={dialog}
      onCancel={(event) => {
        event.preventDefault();
        if (!busy) onClose();
      }}
      aria-labelledby="master-form-title"
      className="m-auto w-[calc(100%-2rem)] max-w-xl max-h-[90vh] overflow-y-auto rounded-2xl border border-border bg-surface p-0 text-text shadow-2xl backdrop:bg-black/45"
    >
      {/* Modal Header */}
      <div className="flex items-start justify-between gap-4 border-b border-border px-6 py-4 bg-panel/30">
        <div>
          <p className="text-[10px] font-bold uppercase tracking-wider text-primary mb-0.5">
            Master Data
          </p>
          <h2 id="master-form-title" className="text-lg font-bold text-text">
            {title}
          </h2>
        </div>
        <button
          type="button"
          disabled={busy}
          onClick={onClose}
          aria-label="Tutup form"
          className="p-1.5 rounded-lg hover:bg-panel text-text-muted hover:text-text transition-colors"
        >
          <X size={18} />
        </button>
      </div>

      <form onSubmit={submit} className="p-6 space-y-5">
        {description && (
          <div className="rounded-xl bg-panel/60 border border-border p-3.5 text-xs text-text-muted leading-relaxed">
            {description}
          </div>
        )}

        {fields.length > 0 && (
          <p className="text-[11px] text-text-muted">
            Kolom bertanda{" "}
            <span className="text-status-danger-text font-bold">*</span> wajib
            diisi.
          </p>
        )}

        <fieldset disabled={busy} className="space-y-4">
          {fields.map((field) =>
            field.type === "checkbox" ? (
              <label
                key={field.key}
                className="flex items-start gap-3 rounded-xl border border-border bg-panel/30 p-3.5 cursor-pointer hover:border-primary/40 transition-colors"
              >
                <input
                  name={field.key}
                  type="checkbox"
                  defaultChecked={Boolean(row?.[field.key])}
                  className="mt-0.5 h-4 w-4 rounded accent-primary shrink-0"
                />
                <div className="space-y-0.5">
                  <span className="text-sm font-semibold text-text block">
                    {field.label}
                  </span>
                  {field.hint && (
                    <span className="block text-xs text-text-muted leading-relaxed">
                      {field.hint}
                    </span>
                  )}
                </div>
              </label>
            ) : (
              <label
                key={field.key}
                className="block space-y-1.5 text-xs font-semibold text-text"
              >
                <span>
                  {field.label}
                  {field.required ? (
                    <span className="text-status-danger-text ml-0.5">*</span>
                  ) : (
                    ""
                  )}
                </span>
                {field.type === "textarea" ? (
                  <textarea
                    name={field.key}
                    required={field.required}
                    maxLength={field.max}
                    placeholder={field.placeholder}
                    rows={3}
                    defaultValue={String(row?.[field.key] ?? "")}
                    className={inputClass}
                  />
                ) : field.type === "file" ? (
                  <input
                    type="file"
                    name={field.key}
                    accept={field.accept}
                    required={field.required}
                    className={`${inputClass} file:mr-3 file:rounded-md file:border-0 file:bg-primary/10 file:text-primary file:font-semibold file:px-3 file:py-1 file:text-xs`}
                  />
                ) : field.type === "select" ? (
                  <SelectField field={field} row={row} company={company} />
                ) : field.type === "password" ? (
                  <div className="relative">
                    <input
                      name={field.key}
                      type={showPassword ? "text" : "password"}
                      required={field.required}
                      placeholder={field.placeholder}
                      autoComplete="new-password"
                      className={`${inputClass} pr-10`}
                    />
                    <button
                      type="button"
                      onClick={() => setShowPassword((prev) => !prev)}
                      aria-label={
                        showPassword
                          ? "Sembunyikan password"
                          : "Tampilkan password"
                      }
                      className="absolute right-2.5 top-1/2 -translate-y-1/2 text-text-muted hover:text-text p-1"
                    >
                      {showPassword ? <EyeOff size={15} /> : <Eye size={15} />}
                    </button>
                  </div>
                ) : (
                  <input
                    name={field.key}
                    type={field.type ?? "text"}
                    required={field.required}
                    maxLength={field.max}
                    pattern={field.pattern}
                    placeholder={field.placeholder}
                    min={field.type === "number" ? 0 : undefined}
                    max={field.type === "number" ? 9999 : undefined}
                    step={field.type === "number" ? 1 : undefined}
                    defaultValue={String(
                      valueAt(row ?? {}, field.source ?? field.key) ??
                        field.initial ??
                        "",
                    )}
                    autoComplete="off"
                    className={inputClass}
                  />
                )}
                {field.hint && (
                  <span className="block text-[11px] font-normal text-text-muted leading-relaxed">
                    {field.hint}
                  </span>
                )}
              </label>
            ),
          )}
        </fieldset>

        {error && (
          <div
            role="alert"
            className="flex items-center gap-2 rounded-xl bg-status-danger-bg px-3.5 py-2.5 text-xs font-medium text-status-danger-text"
          >
            <AlertCircle size={15} className="shrink-0" />
            <span>{error}</span>
          </div>
        )}

        <div className="flex justify-end gap-2 border-t border-border pt-4">
          <button
            type="button"
            disabled={busy}
            onClick={onClose}
            className={buttonClass}
          >
            Batal
          </button>
          <button
            type="submit"
            disabled={busy}
            className={
              variant === "danger" ? dangerButtonClass : primaryButtonClass
            }
          >
            {variant === "danger" ? <Trash2 size={14} /> : <Save size={14} />}
            {busy
              ? variant === "danger"
                ? "Menghapus..."
                : "Menyimpan..."
              : variant === "danger"
                ? "Ya, Hapus Data"
                : fields.length === 0
                  ? "Konfirmasi"
                  : "Simpan Perubahan"}
          </button>
        </div>
      </form>
    </dialog>
  );
}
