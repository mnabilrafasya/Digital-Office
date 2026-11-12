"use client";
import { useState } from "react";
import Link from "next/link";
import { usePathname, useRouter, useSearchParams } from "next/navigation";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { Activity, History, ChevronRight } from "lucide-react";
import { MonitoringPanel } from "./monitoring-panel";
import { ActivityPanel } from "./activity-panel";
function MonitoringContent({defaultTab,viewerId}:{defaultTab:"monitoring"|"audit";viewerId?:string}) {
 const search=useSearchParams(),router=useRouter(),pathname=usePathname();
 const activity=search.get("tab")==="activity" || (!search.has("tab") && defaultTab==="audit");
 function select(tab:string){const params=new URLSearchParams(search.toString());params.set("tab",tab);router.replace(`${pathname}${params.size ? `?${params}` : ""}`,{scroll:false});}
 return <div className="space-y-5 text-text">
  <header><nav aria-label="Breadcrumb" className="flex items-center gap-1 text-xs text-text-muted mb-2"><Link href="/dashboard" className="hover:text-text">Beranda</Link><ChevronRight size={12}/><span className="text-text font-medium">Monitoring Surat</span></nav><h1 className="text-2xl font-bold">Monitoring Surat</h1><p className="text-sm text-text-muted mt-1">Pantau proses surat dan telusuri aktivitas pengguna di perusahaan Anda.</p></header>
  <nav aria-label="Bagian monitoring" className="flex gap-1 p-1 w-fit max-w-full rounded-xl border border-border bg-panel"><button aria-current={!activity ? "page" : undefined} onClick={()=>select("monitoring")} className={`inline-flex items-center gap-2 rounded-lg px-4 py-2 text-xs font-semibold transition-colors ${!activity ? "bg-surface text-primary shadow-sm" : "text-text-muted hover:text-text"}`}><Activity size={15}/>Proses Surat</button><button aria-current={activity ? "page" : undefined} onClick={()=>select("activity")} className={`inline-flex items-center gap-2 rounded-lg px-4 py-2 text-xs font-semibold transition-colors ${activity ? "bg-surface text-primary shadow-sm" : "text-text-muted hover:text-text"}`}><History size={15}/>Riwayat Aktivitas</button></nav>
  {activity ? <ActivityPanel/> : <MonitoringPanel viewerId={viewerId}/>}
 </div>;
}
export function AuditMonitoringView({defaultTab="monitoring",viewerId}:{defaultTab?:"monitoring"|"audit";viewerId?:string}) {
 const [client]=useState(()=>new QueryClient({defaultOptions:{queries:{retry:false,refetchOnWindowFocus:false}}}));
 return <QueryClientProvider client={client}><MonitoringContent defaultTab={defaultTab} viewerId={viewerId}/></QueryClientProvider>;
}
