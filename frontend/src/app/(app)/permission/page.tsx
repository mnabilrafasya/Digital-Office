import { redirect } from "next/navigation";

// Tautan lama tetap menuju riwayat pada halaman monitoring yang sama.
export default function PermissionAuditPage() {
  redirect("/rekap?tab=activity");
}
