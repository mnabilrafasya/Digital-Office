"use client";
import { useEffect, useRef } from "react";
import { useQuery } from "@tanstack/react-query";
import { X } from "lucide-react";
import { getAuditLogDetail } from "@/lib/api/audit";
import { activityLabel, activitySubject, dateTime, entityLabel, readableChanges } from "./audit-presenter";
import { QueryState, buttonClass } from "./audit-shared";
export function ActivityDetail({id,onClose}:{id:string;onClose:()=>void}) {
 const dialog=useRef<HTMLDialogElement>(null);
 const query=useQuery({queryKey:["activity-detail",id],queryFn:()=>getAuditLogDetail(id)});
 useEffect(()=>{const element=dialog.current!;element.showModal();return ()=>element.close();},[]);
 const log=query.data;
 const changes=log ? readableChanges(log) : [];
 return <dialog ref={dialog} onCancel={event=>{event.preventDefault();onClose();}} aria-labelledby="activity-detail-title" className="m-auto w-[calc(100%_-_2rem)] max-w-lg max-h-[85vh] overflow-y-auto rounded-2xl border border-border bg-surface p-6 text-text shadow-xl backdrop:bg-black/40">
  <header className="flex items-start justify-between gap-4"><div><p className="text-[10px] font-semibold uppercase tracking-wider text-primary mb-1">Riwayat Aktivitas</p><h2 id="activity-detail-title" className="text-lg font-bold">Detail aktivitas</h2></div><button onClick={onClose} aria-label="Tutup detail aktivitas" className="rounded-lg p-2 text-text-muted hover:bg-panel"><X size={18}/></button></header>
  <div className="mt-5"><QueryState loading={query.isPending} error={query.error} empty={!log} onRetry={()=>query.refetch()} emptyTitle="Aktivitas tidak ditemukan"/>
   {log && !query.isError && <div className="space-y-5"><p className="rounded-lg bg-primary/5 p-3 text-sm font-semibold text-primary">{activityLabel(log.action)}</p>
    <dl className="grid grid-cols-2 gap-4 text-sm"><div><dt className="text-xs text-text-muted">Dilakukan oleh</dt><dd className="mt-1 font-medium">{log.actor.name || "Sistem"}</dd></div><div><dt className="text-xs text-text-muted">Waktu</dt><dd className="mt-1">{dateTime(log.at)} WIB</dd></div><div className="col-span-2"><dt className="text-xs text-text-muted">{entityLabel(log.entityType)}</dt><dd className="mt-1 font-medium break-words">{activitySubject(log)}</dd></div></dl>
    {changes.length>0 && <div className="space-y-3 border-t border-border pt-4"><h3 className="text-sm font-semibold">Ringkasan perubahan</h3>{changes.map(change=><div key={change.label} className="text-xs"><p className="font-medium mb-1">{change.label}</p><p className="text-text-muted break-words">Sebelumnya: {change.before}</p><p className="mt-1 break-words">Menjadi: {change.after}</p></div>)}</div>}
    {log.letter?.masked && <p className="text-xs text-text-muted">Informasi surat dibatasi sesuai akses akun Anda.</p>}
   </div>}
  </div><footer className="flex justify-end mt-5 border-t border-border pt-4"><button onClick={onClose} className={buttonClass}>Tutup</button></footer>
 </dialog>;
}
