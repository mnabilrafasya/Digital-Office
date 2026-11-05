"use client";

// ─────────────────────────────────────────────
// components/review/document-viewer/index.tsx
// Dynamic import wrapper untuk ReviewDocumentViewer di /surat/[id].
// ssr: false WAJIB untuk library DevExpress browser.
// ─────────────────────────────────────────────

import dynamic from "next/dynamic";
import { Loader2 } from "lucide-react";
import type { ReviewDocumentInnerProps } from "./review-document-inner";

function ViewerLoading() {
  return (
    <div
      style={{ minHeight: "520px" }}
      className="border border-border rounded-xl bg-panel/30 flex flex-col items-center justify-center gap-3 text-text-muted p-8"
    >
      <div className="w-12 h-12 rounded-2xl bg-surface border border-border flex items-center justify-center shadow-sm">
        <Loader2 size={24} className="animate-spin text-primary" />
      </div>
      <div className="text-center">
        <p className="text-sm font-semibold text-text">Memuat Pratinjau Naskah Dinas...</p>
        <p className="text-xs text-text-muted mt-1">
          Menyiapkan dokumen resmi dan tata naskah digital
        </p>
      </div>
    </div>
  );
}

const ReviewDocumentInner = dynamic(
  () => import("./review-document-inner").then((m) => ({ default: m.default })),
  { ssr: false, loading: ViewerLoading }
);

const OfficialDocumentInner = dynamic(
  () => import("./official-document-inner").then((m) => ({ default: m.default })),
  { ssr: false, loading: ViewerLoading }
);

// Surat yang sudah terbit ditampilkan sebagai naskah resmi (sudah terisi nomor, tanggal, dan QR)
// dalam mode baca; PDF resminya diekspor dari tampilan yang sama. Editor biasa hanya untuk naskah
// yang masih diproses.
export function ReviewDocumentViewer(props: ReviewDocumentInnerProps) {
  if (props.letter.status === "SENT") {
    return <OfficialDocumentInner letter={props.letter} />;
  }
  return <ReviewDocumentInner {...props} />;
}
