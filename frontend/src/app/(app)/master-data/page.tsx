import { redirect } from "next/navigation";
import { getCurrentUser } from "@/lib/auth";
import { MasterDataView } from "@/components/master-data/master-data-view";

export default async function MasterDataPage() {
  const user = await getCurrentUser();
  if (!user) {
    redirect("/login");
  }
  return <MasterDataView key={user.id} user={user} />;
}
