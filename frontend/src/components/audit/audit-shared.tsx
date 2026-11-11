"use client";
import { Loader2, Inbox, RefreshCw, ChevronLeft, ChevronRight } from "lucide-react";
export const fieldClass="h-10 rounded-lg border border-border bg-surface px-3 text-sm text-text focus:outline-none focus:ring-2 focus:ring-primary/20";
export const buttonClass="inline-flex h-10 items-center justify-center gap-1.5 rounded-lg border border-border px-3 text-xs font-semibold text-text hover:bg-panel disabled:opacity-50 disabled:cursor-not-allowed";
export function QueryState({loading,error,empty,onRetry,emptyTitle}:{loading:boolean;error:Error|null;empty:boolean;onRetry:()=>void;emptyTitle:string}) {
 if(loading) return <div role="status" className="flex items-center justify-center gap-2 py-16 text-sm text-text-muted"><Loader2 size={17} className="animate-spin text-primary"/>Memuat data...</div>;
 if(error) return <div role="alert" className="rounded-xl border border-status-danger-text/20 bg-status-danger-bg p-5 text-sm text-status-danger-text"><p>{error.message}</p><button onClick={onRetry} className="mt-3 inline-flex items-center gap-1.5 font-semibold underline"><RefreshCw size={13}/>Coba lagi</button></div>;
 if(empty) return <div className="rounded-xl border border-border bg-surface py-14 text-center"><Inbox size={30} className="mx-auto mb-3 text-text-muted"/><p className="text-sm font-semibold text-text">{emptyTitle}</p><p className="mt-1 text-xs text-text-muted">Coba ubah filter atau muat ulang untuk melihat data terbaru.</p></div>;
 return null;
}
export function Paging({page,total,size,busy,onChange}:{page:number;total:number;size:number;busy:boolean;onChange:(page:number)=>void}) {
 const pages=Math.max(1,Math.ceil(total/size));
 return <div className="flex flex-wrap items-center justify-between gap-3 pt-3 text-xs text-text-muted"><span>{total} data · Halaman {page} dari {pages}</span><div className="flex gap-2"><button className={buttonClass} disabled={page<=1 || busy} onClick={()=>onChange(page-1)}><ChevronLeft size={14}/>Sebelumnya</button><button className={buttonClass} disabled={page>=pages || busy} onClick={()=>onChange(page+1)}>Berikutnya<ChevronRight size={14}/></button></div></div>;
}
