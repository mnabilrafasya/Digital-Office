import { redirect } from "next/navigation";
import { getCurrentUser } from "@/lib/auth";
import { AuditMonitoringView } from "@/components/audit/audit-monitoring-view";

export default async function RekapPage() {
  const user = await getCurrentUser();
  if (!user) redirect("/login");
  return <AuditMonitoringView viewerId={user.id} />;
}
