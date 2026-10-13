"use client";

// ─────────────────────────────────────────────
// components/surat/lampiran-section.tsx
// Section 2: Lampiran
// Validasi: PDF, DOCX, XLSX, PPTX; max 25 MB per file
// ─────────────────────────────────────────────

import { useRef, useState } from "react";
import {
  Upload,
  FileText,
  FileSpreadsheet,
  Presentation,
  File,
  Eye,
  Trash2,
  RefreshCw,
  Search,
  CloudUpload,
  Paperclip,
} from "lucide-react";
import { formatBytes, formatDateTimeWIB, generateId, mimeToLabel } from "@/lib/utils";
import { cn } from "@/lib/utils";
import type { LetterAttachment } from "@/lib/types";

const ALLOWED_MIME = [
  "application/pdf",
  "application/vnd.openxmlformats-officedocument.wordprocessingml.document",
  "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet",
  "application/vnd.openxmlformats-officedocument.presentationml.presentation",
  "application/msword",
];
const MAX_BYTES = 25 * 1024 * 1024; // 25 MB

interface UploadItem {
  id: string;
  file: File;
  state: "uploading" | "done" | "error";
  progress: number;
  errorMsg?: string;
  attachment?: LetterAttachment;
}

interface LampiranSectionProps {
  value: LetterAttachment[];
  onChange: (attachments: LetterAttachment[]) => void;
}

function FileIcon({ mimeType }: { mimeType: string }) {
  if (mimeType === "application/pdf") return <FileText size={18} className="text-red-500" />;
  if (mimeType.includes("spreadsheet") || mimeType.includes("excel"))
    return <FileSpreadsheet size={18} className="text-green-600" />;
  if (mimeType.includes("presentation"))
    return <Presentation size={18} className="text-orange-500" />;
  return <File size={18} className="text-blue-500" />;
}

export function LampiranSection({ value, onChange }: LampiranSectionProps) {
  const inputRef = useRef<HTMLInputElement>(null);
  const [dragging, setDragging] = useState(false);
  const [uploadItems, setUploadItems] = useState<UploadItem[]>([]);
  const [filterQuery, setFilterQuery] = useState("");

  function validateFile(file: File): string | null {
    if (!ALLOWED_MIME.includes(file.type)) {
      return `Tipe file tidak didukung. Gunakan PDF, DOCX, XLSX, atau PPTX.`;
    }
    if (file.size > MAX_BYTES) {
      return `Ukuran file melebihi batas 25 MB.`;
    }
    return null;
  }

  function processFiles(files: FileList | File[]) {
    const arr = Array.from(files);
    arr.forEach((file) => {
      const errMsg = validateFile(file);
      const id = generateId();

      if (errMsg) {
        setUploadItems((prev) => [
          ...prev,
          { id, file, state: "error", progress: 0, errorMsg: errMsg },
        ]);
        return;
      }

      // Simulasi upload (tidak ada backend nyata)
      setUploadItems((prev) => [
        ...prev,
        { id, file, state: "uploading", progress: 0 },
      ]);

      // Fake progress
      let prog = 0;
      const interval = setInterval(() => {
        prog += Math.floor(Math.random() * 25) + 15;
        if (prog >= 100) {
          prog = 100;
          clearInterval(interval);
          const attachment: LetterAttachment = {
            id,
            fileName: file.name,
            mimeType: file.type,
            sizeBytes: file.size,
            uploadedAt: new Date().toISOString(),
          };
          setUploadItems((prev) =>
            prev.map((item) =>
              item.id === id ? { ...item, state: "done", progress: 100, attachment } : item,
            ),
          );
          onChange([...value, attachment]);
        } else {
          setUploadItems((prev) =>
            prev.map((item) =>
              item.id === id ? { ...item, progress: prog } : item,
            ),
          );
        }
      }, 200);
    });
  }

  function retryUpload(item: UploadItem) {
    setUploadItems((prev) => prev.filter((i) => i.id !== item.id));
    processFiles([item.file]);
  }

  function removeAttachment(id: string) {
    onChange(value.filter((a) => a.id !== id));
    setUploadItems((prev) => prev.filter((i) => i.id !== id));
  }

  // Gabungkan: attachment yang sudah selesai + yang error/sedang upload
  const displayAttachments = value;
  const filtered = displayAttachments.filter((a) =>
    filterQuery === "" || a.fileName.toLowerCase().includes(filterQuery.toLowerCase()),
  );

  const pendingItems = uploadItems.filter((i) => i.state === "uploading" || i.state === "error");

  return (
    <div className="space-y-4">
      {/* Panel Upload */}
      <div
        className={cn(
          "border-2 border-dashed rounded-xl p-6 text-center transition-colors cursor-pointer",
          dragging
            ? "border-primary bg-primary/5"
            : "border-border hover:border-primary/50 hover:bg-panel/60",
        )}
        onDragEnter={(e) => { e.preventDefault(); setDragging(true); }}
        onDragLeave={(e) => { e.preventDefault(); setDragging(false); }}
        onDragOver={(e) => { e.preventDefault(); }}
        onDrop={(e) => {
          e.preventDefault();
          setDragging(false);
          if (e.dataTransfer.files.length > 0) processFiles(e.dataTransfer.files);
        }}
        onClick={() => inputRef.current?.click()}
      >
        <input
          ref={inputRef}
          type="file"
          multiple
          accept=".pdf,.docx,.xlsx,.pptx,.doc"
          className="hidden"
          onChange={(e) => {
            if (e.target.files) processFiles(e.target.files);
            e.target.value = "";
          }}
        />
        <CloudUpload
          size={36}
          className={cn("mx-auto mb-3", dragging ? "text-primary" : "text-text-muted")}
        />
        <p className="text-sm font-medium text-text">
          Seret & lepas berkas di sini, atau{" "}
          <span className="text-primary underline underline-offset-2">pilih dari komputer</span>
        </p>
        <p className="text-xs text-text-muted mt-1">
          Format: PDF, DOCX, XLSX, PPTX &nbsp;&bull;&nbsp; Maks. 25 MB per berkas
        </p>

        <button
          type="button"
          onClick={(e) => { e.stopPropagation(); inputRef.current?.click(); }}
          className="mt-4 inline-flex items-center gap-2 px-4 py-2 bg-primary text-white text-sm rounded-lg hover:bg-primary-dark transition-colors"
        >
          <Upload size={14} />
          Pilih File dari Komputer
        </button>
      </div>

      {/* Progress items */}
      {pendingItems.length > 0 && (
        <div className="space-y-2">
          {pendingItems.map((item) => (
            <div
              key={item.id}
              className={cn(
                "flex items-center gap-3 px-4 py-3 rounded-lg border text-sm",
                item.state === "error"
                  ? "border-status-danger-bg bg-status-danger-bg/40"
                  : "border-border bg-panel",
              )}
            >
              <Paperclip size={14} className="text-text-muted shrink-0" />
              <div className="flex-1 min-w-0">
                <div className="font-medium text-text truncate">{item.file.name}</div>
                {item.state === "uploading" ? (
                  <div className="mt-1.5 h-1.5 w-full bg-border rounded-full overflow-hidden">
                    <div
                      className="h-full bg-primary rounded-full transition-all duration-300"
                      style={{ width: `${item.progress}%` }}
                    />
                  </div>
                ) : (
                  <div className="text-status-danger-text text-xs mt-0.5">{item.errorMsg}</div>
                )}
              </div>
              {item.state === "error" && (
                <button
                  type="button"
                  onClick={() => retryUpload(item)}
                  className="text-text-muted hover:text-primary transition-colors shrink-0"
                  title="Coba lagi"
                >
                  <RefreshCw size={14} />
                </button>
              )}
            </div>
          ))}
        </div>
      )}

      {/* Tabel lampiran */}
      {displayAttachments.length > 0 && (
        <div className="space-y-3">
          {/* Filter + counter */}
          <div className="flex items-center gap-3">
            <div className="relative flex-1 max-w-xs">
              <Search size={13} className="absolute left-3 top-1/2 -translate-y-1/2 text-text-muted" />
              <input
                type="text"
                placeholder="Saring nama berkas lampiran..."
                value={filterQuery}
                onChange={(e) => setFilterQuery(e.target.value)}
                className="w-full pl-8 pr-3 py-2 text-sm border border-border rounded-lg bg-surface outline-none focus:border-primary focus:ring-1 focus:ring-primary/20"
              />
            </div>
            <span className="text-xs text-text-muted whitespace-nowrap">
              Menampilkan {filtered.length} dari {displayAttachments.length} berkas
            </span>
          </div>

          {/* Tabel */}
          <div className="overflow-x-auto rounded-lg border border-border">
            <table className="w-full text-sm">
              <thead>
                <tr className="bg-panel text-text-muted text-xs">
                  <th className="px-4 py-2.5 text-left font-semibold w-8">#</th>
                  <th className="px-4 py-2.5 text-left font-semibold w-20">Aksi</th>
                  <th className="px-4 py-2.5 text-left font-semibold">Nama Berkas & Format</th>
                  <th className="px-4 py-2.5 text-left font-semibold w-24">Ukuran</th>
                  <th className="px-4 py-2.5 text-left font-semibold w-40">Waktu Unggah</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-border">
                {filtered.length === 0 ? (
                  <tr>
                    <td colSpan={5} className="px-4 py-8 text-center text-text-muted">
                      Tidak ada berkas yang cocok dengan filter.
                    </td>
                  </tr>
                ) : (
                  filtered.map((att, idx) => (
                    <tr key={att.id} className="hover:bg-panel/50 transition-colors">
                      <td className="px-4 py-3 text-text-muted">{idx + 1}</td>
                      <td className="px-4 py-3">
                        <div className="flex items-center gap-1.5">
                          <button
                            type="button"
                            title="Lihat"
                            className="p-1.5 rounded hover:bg-panel text-text-muted hover:text-primary transition-colors"
                          >
                            <Eye size={13} />
                          </button>
                          <button
                            type="button"
                            title="Hapus"
                            onClick={() => removeAttachment(att.id)}
                            className="p-1.5 rounded hover:bg-status-danger-bg text-text-muted hover:text-status-danger-text transition-colors"
                          >
                            <Trash2 size={13} />
                          </button>
                        </div>
                      </td>
                      <td className="px-4 py-3">
                        <div className="flex items-center gap-2">
                          <FileIcon mimeType={att.mimeType} />
                          <div>
                            <div className="font-medium text-text">{att.fileName}</div>
                            <div className="text-xs text-text-muted">{mimeToLabel(att.mimeType)}</div>
                          </div>
                        </div>
                      </td>
                      <td className="px-4 py-3 text-text-muted">{formatBytes(att.sizeBytes)}</td>
                      <td className="px-4 py-3 text-text-muted text-xs">
                        {formatDateTimeWIB(att.uploadedAt)}
                      </td>
                    </tr>
                  ))
                )}
              </tbody>
            </table>
          </div>
        </div>
      )}

      {/* State kosong */}
      {displayAttachments.length === 0 && pendingItems.length === 0 && (
        <div className="flex flex-col items-center justify-center py-8 text-text-muted">
          <Paperclip size={28} className="mb-2 opacity-40" />
          <p className="text-sm">Belum ada lampiran yang diunggah.</p>
        </div>
      )}
    </div>
  );
}
