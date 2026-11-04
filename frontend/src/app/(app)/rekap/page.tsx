import { AuditMonitoringView } from "@/components/audit/audit-monitoring-view";

// ─────────────────────────────────────────────
// app/(app)/rekap/page.tsx
// Halaman Rekap & Pantauan Persuratan (Monitoring + Audit Log)
// ─────────────────────────────────────────────

export default function RekapPage() {
  return <AuditMonitoringView defaultTab="monitoring" />;
}

