import { redirect } from "next/navigation";
import { getCurrentUser } from "@/lib/auth";
import { DelegasiView } from "@/components/delegasi/delegasi-view";

export default async function DelegasiPage() {
  const user = await getCurrentUser();
  if (!user) {
    redirect("/login");
  }
  return <DelegasiView key={user.id} user={user} />;
}

