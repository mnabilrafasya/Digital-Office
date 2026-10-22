import { redirect } from "next/navigation";

// ─────────────────────────────────────────────
// app/(app)/surat/review/page.tsx
// Dialihkan ke Daftar Surat terpadu pada tab Tugas Saya
// ─────────────────────────────────────────────

export default function MenungguReviewPage() {
  redirect("/surat/daftar?scope=tasks&status=IN_REVIEW");
}
