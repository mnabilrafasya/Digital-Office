import { AuditMonitoringView } from "@/components/audit/audit-monitoring-view";

// ─────────────────────────────────────────────
// app/(app)/permission/page.tsx
// Halaman Permission & Jejak Audit Dokumen
// ─────────────────────────────────────────────

export default function PermissionAuditPage() {
  return <AuditMonitoringView defaultTab="audit" />;
}

