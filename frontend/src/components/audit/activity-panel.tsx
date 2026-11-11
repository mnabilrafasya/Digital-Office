"use client";
import { useState, type FormEvent } from "react";
import { useQuery } from "@tanstack/react-query";
import { Search, Download, RefreshCw, Eye } from "lucide-react";
import { listAuditLogs, downloadAuditLogsCsv } from "@/lib/api/audit";
import { activityLabel, activitySubject, dateTime, ENTITY_LABELS, entityLabel, initialDates } from "./audit-presenter";
import { ActivityDetail } from "./activity-detail";
import { QueryState, Paging, fieldClass, buttonClass } from "./audit-shared";
const size=15;
export function ActivityPanel() {
 const [filters,setFilters]=useState(()=>({...initialDates(),q:"",entityType:""}));
 const [applied,setApplied]=useState(filters);
 const [page,setPage]=useState(1);
 const [detail,setDetail]=useState<string|null>(null);
 const [error,setError]=useState("");
 const [exporting,setExporting]=useState(false);
 const query=useQuery({queryKey:["activities",applied,page],queryFn:()=>listAuditLogs({...applied,page,pageSize:size})});
 const rows=query.data?.data ?? [];
 function apply(event:FormEvent<HTMLFormElement>) {
  event.preventDefault();
  if(filters.to<filters.from){setError("Tanggal akhir harus sama atau sesudah tanggal awal.");return;}
  setApplied({...filters,q:filters.q.trim()});setPage(1);setError("");
 }
 function reset(){const next={...initialDates(),q:"",entityType:""};setFilters(next);setApplied(next);setPage(1);setError("");}
 async function download(){
  const days=(Date.parse(applied.to)-Date.parse(applied.from))/86400000+1;
  if(!Number.isFinite(days) || days<1 || days>31){setError("Untuk mengunduh CSV, pilih rentang tanggal maksimal 31 hari lalu klik Terapkan.");return;}
  setExporting(true);setError("");
  try{await downloadAuditLogsCsv(applied);}catch(err){setError(err instanceof Error ? err.message : "Riwayat gagal diunduh. Coba lagi.");}finally{setExporting(false);}
 }
 return <section className="space-y-4">
  <form onSubmit={apply} className="rounded-xl border border-border bg-surface p-4 space-y-3">
   <div className="flex flex-wrap gap-3 items-end"><label className="text-xs font-medium text-text-muted space-y-1.5"><span className="block">Dari tanggal</span><input aria-label="Dari tanggal" type="date" required value={filters.from} onChange={event=>setFilters({...filters,from:event.target.value})} className={fieldClass}/></label><label className="text-xs font-medium text-text-muted space-y-1.5"><span className="block">Sampai tanggal</span><input aria-label="Sampai tanggal" type="date" required min={filters.from} value={filters.to} onChange={event=>setFilters({...filters,to:event.target.value})} className={fieldClass}/></label>
    <label className="text-xs font-medium text-text-muted space-y-1.5"><span className="block">Jenis data</span><select value={filters.entityType} onChange={event=>setFilters({...filters,entityType:event.target.value})} className={fieldClass}><option value="">Semua data</option>{Object.entries(ENTITY_LABELS).map(([value,label])=><option key={value} value={value}>{label}</option>)}</select></label>
    <label className="text-xs font-medium text-text-muted space-y-1.5 flex-1 min-w-48"><span className="block">Pencarian</span><div className="relative"><Search size={14} className="absolute left-3 top-3 text-text-muted"/><input aria-label="Cari aktivitas berdasarkan pengguna atau surat" maxLength={100} placeholder="Nama pengguna, nomor, atau perihal surat..." value={filters.q} onChange={event=>setFilters({...filters,q:event.target.value})} className={`${fieldClass} w-full pl-9`}/></div></label>
   </div><div className="flex gap-2"><button type="submit" className="inline-flex h-9 items-center rounded-lg bg-primary px-4 text-xs font-semibold text-white hover:bg-primary/90">Terapkan</button><button type="button" onClick={reset} className={`${buttonClass} h-9`}>Reset filter</button><span className="self-center ml-auto text-[11px] text-text-muted hidden sm:block">Awalnya menampilkan 7 hari terakhir</span></div>
  </form>
  {error && <p role="alert" className="rounded-lg bg-status-danger-bg p-3 text-xs text-status-danger-text">{error}</p>}
  <div className="flex flex-wrap items-center justify-between gap-3"><div><h2 className="text-sm font-semibold text-text">Riwayat aktivitas pengguna</h2><p className="text-xs text-text-muted mt-1">Aktivitas terbaru ditampilkan paling atas.</p></div><div className="flex gap-2"><button onClick={()=>query.refetch()} disabled={query.isFetching} className={buttonClass} aria-label="Muat ulang aktivitas"><RefreshCw size={14} className={query.isFetching ? "animate-spin" : ""}/></button><button onClick={download} disabled={exporting || query.isPending || query.isError || !query.data?.meta.total} className={buttonClass}><Download size={14}/>{exporting ? "Mengunduh..." : "Unduh CSV"}</button></div></div>
  <QueryState loading={query.isPending} error={query.error} empty={!rows.length} onRetry={()=>query.refetch()} emptyTitle="Belum ada aktivitas pada rentang ini"/>
  {!query.isPending && !query.isError && rows.length>0 && <div className="rounded-xl border border-border overflow-x-auto bg-surface"><table className="w-full text-left text-sm"><thead className="bg-panel/50 text-xs text-text-muted"><tr><th className="px-4 py-3 font-medium">Waktu (WIB)</th><th className="px-4 py-3 font-medium">Pengguna</th><th className="px-4 py-3 font-medium">Aktivitas</th><th className="px-4 py-3 font-medium">Data terkait</th><th className="px-4 py-3"><span className="sr-only">Detail</span></th></tr></thead><tbody className="divide-y divide-border">{rows.map(log=><tr key={log.id} className="hover:bg-panel/25"><td className="px-4 py-4 whitespace-nowrap text-xs text-text-muted">{dateTime(log.at)}</td><td className="px-4 py-4 font-medium text-text">{log.actor.name || "Sistem"}</td><td className="px-4 py-4 text-xs text-text">{activityLabel(log.action)}</td><td className="px-4 py-4 max-w-xs"><p className="text-xs font-medium text-text">{activitySubject(log)}</p>{log.letter?.number && <p className="text-[11px] text-text-muted mt-1">{log.letter.number}</p>}{!log.letter && activitySubject(log)!==entityLabel(log.entityType) && <p className="text-[11px] text-text-muted mt-1">{entityLabel(log.entityType)}</p>}</td><td className="px-4 py-4 text-right"><button onClick={()=>setDetail(log.id)} className="inline-flex items-center gap-1.5 text-xs font-semibold text-primary hover:underline"><Eye size={13}/>Detail</button></td></tr>)}</tbody></table></div>}
  {query.data && !query.isError && <Paging page={page} total={query.data.meta.total} size={size} busy={query.isFetching} onChange={setPage}/>}
  {detail && <ActivityDetail id={detail} onClose={()=>setDetail(null)}/>}
 </section>;
}
