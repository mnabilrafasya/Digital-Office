import { redirect } from "next/navigation";

// ─────────────────────────────────────────────
// app/(app)/surat/approval/page.tsx
// Dialihkan ke Daftar Surat terpadu pada tab Tugas Saya
// ─────────────────────────────────────────────

export default function MenungguApprovalPage() {
  redirect("/surat/daftar?scope=tasks&status=IN_APPROVAL");
}
