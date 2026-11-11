"use client";
import { useState } from "react";
import { useQuery } from "@tanstack/react-query";
import Link from "next/link";
import { RefreshCw, Clock3, ArrowUpRight } from "lucide-react";
import { listMonitoredLetters } from "@/lib/api/audit";
import { getStatusMeta } from "@/lib/status";
import type { LetterStatus } from "@/lib/types";
import { StatusBadge } from "@/components/daftar-surat/status-badge";
import { dateTime, waitingTime } from "./audit-presenter";
import { QueryState, Paging, fieldClass, buttonClass } from "./audit-shared";
const size=15;
const statuses=["IN_REVIEW","IN_APPROVAL","RETURNED_TO_DRAFTER","RETURNED_TO_REVIEWER","APPROVED"];
export function MonitoringPanel({viewerId}:{viewerId?:string}) {
 const [status,setStatus]=useState("");
 const [hours,setHours]=useState("");
 const [page,setPage]=useState(1);
 const query=useQuery({queryKey:["monitoring",status,hours,page],queryFn:()=>listMonitoredLetters({status:status || undefined,minWaitingHours:hours || undefined,page,pageSize:size})});
 const items=query.data?.data ?? [];
 return <section className="space-y-4">
  <div className="rounded-xl border border-border bg-surface p-4 flex flex-wrap items-end gap-3">
   <label className="space-y-1.5 text-xs font-medium text-text-muted"><span className="block">Status surat</span><select value={status} onChange={event=>{setStatus(event.target.value);setPage(1);}} className={`${fieldClass} min-w-48`}><option value="">Semua proses berjalan</option>{statuses.map(value=><option key={value} value={value}>{getStatusMeta(value).label}{value==="RETURNED_TO_DRAFTER" ? " ke pembuat" : value==="RETURNED_TO_REVIEWER" ? " ke pemeriksa" : ""}</option>)}</select></label>
   <label className="space-y-1.5 text-xs font-medium text-text-muted"><span className="block">Lama menunggu</span><select value={hours} onChange={event=>{setHours(event.target.value);setPage(1);}} className={fieldClass}><option value="">Semua durasi</option><option value="24">Minimal 1 hari</option><option value="48">Minimal 2 hari</option></select></label>
   {(status || hours) && <button className={buttonClass} onClick={()=>{setStatus("");setHours("");setPage(1);}}>Reset filter</button>}
   <button className={`${buttonClass} ml-auto`} disabled={query.isFetching} onClick={()=>query.refetch()}><RefreshCw size={14} className={query.isFetching ? "animate-spin" : ""}/>Muat ulang</button>
  </div>
  <div className="flex items-center justify-between gap-3"><div><h2 className="text-sm font-semibold text-text">Surat yang sedang diproses</h2><p className="mt-1 text-xs text-text-muted">Urutan dimulai dari surat yang paling lama menunggu.</p></div>{query.data && !query.isError && <span className="rounded-full bg-panel px-3 py-1 text-xs text-text-muted">{query.data.meta.total} surat</span>}</div>
  <QueryState loading={query.isPending} error={query.error} empty={!items.length} onRetry={()=>query.refetch()} emptyTitle="Tidak ada surat yang sesuai filter"/>
  {!query.isPending && !query.isError && items.length>0 && <div className="overflow-x-auto rounded-xl border border-border bg-surface"><table className="w-full text-left text-sm"><thead className="bg-panel/50 text-xs text-text-muted"><tr><th className="px-4 py-3 font-medium">Surat & unit pengirim</th><th className="px-4 py-3 font-medium">Sedang ditangani oleh</th><th className="px-4 py-3 font-medium">Status</th><th className="px-4 py-3 font-medium">Lama menunggu</th><th className="px-4 py-3"><span className="sr-only">Tindakan</span></th></tr></thead><tbody className="divide-y divide-border">{items.map(item=><tr key={item.letterId} className="hover:bg-panel/25"><td className="px-4 py-4 max-w-xs"><p className="font-semibold text-text">{item.titleMasked ? "Perihal dibatasi" : item.title || "Surat tanpa perihal"}</p><p className="mt-1 text-xs text-text-muted">{item.senderUnit}</p></td><td className="px-4 py-4"><p className="font-medium text-text">{item.holder?.name ?? "Belum ditugaskan"}</p><p className="mt-1 text-xs text-text-muted">{item.holder?.positionName}{item.step ? ` · Tahap ${item.step}` : ""}</p></td><td className="px-4 py-4"><StatusBadge status={item.status as LetterStatus}/></td><td className="px-4 py-4"><span className={`inline-flex items-center gap-1.5 text-xs font-semibold ${item.stuck ? "text-status-warn-text" : "text-text"}`}><Clock3 size={13}/>{waitingTime(item.waitingHours)}</span><p className="mt-1 text-[11px] text-text-muted">Sejak {dateTime(item.since)}</p></td><td className="px-4 py-4 text-right">{!item.titleMasked && viewerId && (item.drafter.id===viewerId || item.holder?.id===viewerId) && <Link href={`/surat/${item.letterId}`} className="inline-flex items-center gap-1 text-xs font-semibold text-primary hover:underline">Lihat surat<ArrowUpRight size={13}/></Link>}</td></tr>)}</tbody></table></div>}
  {query.data && !query.isError && <Paging page={page} total={query.data.meta.total} size={size} busy={query.isFetching} onChange={setPage}/>}
 </section>;
}
