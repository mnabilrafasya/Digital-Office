import { redirect } from "next/navigation";
import { getCurrentUser } from "@/lib/auth";
import { DashboardView } from "@/components/dashboard/dashboard-view";

export default async function DashboardPage() {
  const user = await getCurrentUser();
  if (!user) {
    redirect("/login");
  }
  return <DashboardView key={user.id} user={user} />;
}

