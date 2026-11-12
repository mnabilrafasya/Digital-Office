"use client";

import { useEffect, useState } from "react";
import Link from "next/link";
import {
  FilePlus,
  FileText,
  ClipboardCheck,
  Inbox,
  UserCheck,
  BarChart3,
  Database,
  ArrowRight,
  Building2,
  ShieldCheck,
  AlertCircle,
  CheckCircle2,
  Sparkles,
  ChevronRight,
  Layers,
  Calendar,
  Loader2,
} from "lucide-react";
import { format } from "date-fns";
import { id as idLocale } from "date-fns/locale";
import type { AuthUser } from "@/lib/auth";
import { getPendingTaskCounts, listLetters } from "@/lib/api/letters";
import { getInboxList } from "@/lib/api/inbox";
import { getMineDelegations, type DelegationItem } from "@/lib/api/delegasi";
import { StatusBadge } from "@/components/daftar-surat/status-badge";
import type { LetterListItem } from "@/lib/types";
import type { InboxItem } from "@/lib/api/inbox";

function getGreeting(date = new Date()): string {
  const hour = date.getHours();
  if (hour >= 4 && hour < 11) return "Selamat Pagi";
  if (hour >= 11 && hour < 15) return "Selamat Siang";
  if (hour >= 15 && hour < 18) return "Selamat Sore";
  return "Selamat Malam";
}

export function DashboardView({ user }: { user: AuthUser }) {
  const [greeting] = useState(() => getGreeting());
  const [todayFormatted] = useState(() =>
    format(new Date(), "EEEE, d MMMM yyyy", { locale: idLocale })
  );

  // Status & KPI
  const [taskCounts, setTaskCounts] = useState({
    review: 0,
    approval: 0,
    tasksTotal: 0,
  });
  const [inboxUnread, setInboxUnread] = useState(0);
  const [inboxTotal, setInboxTotal] = useState(0);
  const [mineLettersCount, setMineLettersCount] = useState(0);

  // Delegasi
  const [mineDelegations, setMineDelegations] = useState<DelegationItem[]>([]);
  const [representing, setRepresenting] = useState<DelegationItem[]>([]);

  // Daftar ringkas
  const [recentTasks, setRecentTasks] = useState<LetterListItem[]>([]);
  const [recentInbox, setRecentInbox] = useState<InboxItem[]>([]);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    let cancelled = false;

    async function loadDashboardData() {
      try {
        const [
          countsRes,
          inboxRes,
          mineRes,
          tasksRes,
          delegationsRes,
        ] = await Promise.all([
          getPendingTaskCounts(),
          getInboxList({ page: 1, pageSize: 5 }),
          listLetters({ scope: "mine", page: 1, pageSize: 1 }),
          listLetters({ scope: "tasks", page: 1, pageSize: 5 }),
          getMineDelegations().catch(() => ({ delegations: [], representing: [] })),
        ]);

        if (cancelled) return;

        setTaskCounts(countsRes);
        setInboxUnread(inboxRes.counts.UNREAD);
        setInboxTotal(inboxRes.meta.total);
        setMineLettersCount(mineRes.meta.total);
        setRecentTasks(tasksRes.data ?? []);
        setRecentInbox(inboxRes.data ?? []);
        setMineDelegations(delegationsRes.delegations ?? []);
        setRepresenting(delegationsRes.representing ?? []);
      } catch (err) {
        console.error("Gagal memuat data dashboard:", err);
      } finally {
        if (!cancelled) setLoading(false);
      }
    }

    void loadDashboardData();

    return () => {
      cancelled = true;
    };
  }, []);

  const activeSelfDelegation = mineDelegations.find((d) => d.status === "ACTIVE");
  const activeRepresenting = representing.filter((d) => d.status === "ACTIVE");

  const canCreate = user.permissions.includes("LETTER_CREATE");
  const canAudit = user.permissions.includes("AUDIT_VIEW");
  const canManageMaster = user.permissions.some((p) =>
    [
      "TEMPLATE_CREATE",
      "TEMPLATE_MANAGE",
      "COMPANY_MANAGE",
      "COMPANY_SETTINGS",
      "ORG_MANAGE",
      "USER_MANAGE",
    ].includes(p)
  );

  return (
    <div className="space-y-6 text-text">
      {/* ── Banner Sapaan & Profil Pengguna ── */}
      <section className="relative overflow-hidden rounded-2xl border border-border bg-surface p-5 sm:p-6 shadow-xs">
        <div className="absolute right-0 top-0 -mt-6 -mr-6 w-56 h-56 rounded-full bg-primary/5 pointer-events-none blur-2xl" />
        <div className="relative flex flex-wrap items-start justify-between gap-4">
          <div className="space-y-1.5">
            <div className="flex flex-wrap items-center gap-2">
              <span className="text-xs font-semibold text-primary uppercase tracking-wide flex items-center gap-1.5">
                <Sparkles size={13} />
                Portal Tata Naskah Dinas Digital
              </span>
              <span className="text-border">·</span>
              <span className="text-xs text-text-muted flex items-center gap-1">
                <Calendar size={12} />
                {todayFormatted}
              </span>
            </div>
            <h1 className="text-2xl sm:text-3xl font-extrabold tracking-tight">
              {greeting}, {user.name}!
            </h1>
            <p className="text-xs sm:text-sm text-text-muted max-w-2xl leading-relaxed">
              Selamat datang di sistem persuratan resmi{" "}
              <strong className="text-text">{user.company.name}</strong>. Anda
              dapat membuat, menelaah, menandatangani surat dinas, serta mengelola
              disposisi dan delegasi tugas.
            </p>
          </div>

          <div className="flex flex-wrap items-center gap-2 shrink-0">
            <div className="inline-flex items-center gap-2 rounded-xl border border-border bg-panel/40 px-3 py-2 text-xs text-text-muted shadow-2xs">
              <Building2 size={14} className="text-primary shrink-0" />
              <div className="text-left">
                <p className="font-bold text-text leading-tight">{user.company.code}</p>
                <p className="text-[10px] text-text-muted truncate max-w-40">
                  {user.unit?.name ?? "Unit Kerja"}
                </p>
              </div>
              <span className="h-6 w-px bg-border mx-1" />
              <ShieldCheck size={14} className="text-status-success-text shrink-0" />
              <div className="text-left">
                <p className="font-bold text-text leading-tight">{user.role?.name ?? "Pegawai"}</p>
                <p className="text-[10px] text-text-muted truncate max-w-44">
                  {user.position?.name ?? "Jabatan"}
                </p>
              </div>
            </div>
          </div>
        </div>

        {/* ── Banner Status Delegasi Aktif ── */}
        {(activeSelfDelegation || activeRepresenting.length > 0) && (
          <div className="mt-4 pt-4 border-t border-border/80 flex flex-col sm:flex-row sm:items-center justify-between gap-3 text-xs">
            {activeSelfDelegation && (
              <div className="flex items-center gap-2 text-status-warn-text">
                <AlertCircle size={15} className="shrink-0" />
                <span>
                  Tugas Anda saat ini sedang didelegasikan ke{" "}
                  <strong>{activeSelfDelegation.delegate.name}</strong> (s.d.{" "}
                  {activeSelfDelegation.until}).
                </span>
              </div>
            )}
            {activeRepresenting.length > 0 && (
              <div className="flex items-center gap-2 text-primary font-medium">
                <UserCheck size={15} className="shrink-0" />
                <span>
                  Anda sedang mewakili {activeRepresenting.length} rekan kerja:{" "}
                  <strong>
                    {activeRepresenting.map((r) => r.user.name).join(", ")}
                  </strong>
                  .
                </span>
              </div>
            )}
            <Link
              href="/delegasi"
              className="inline-flex items-center gap-1 font-semibold text-primary hover:underline ml-auto"
            >
              Lihat Delegasi <ChevronRight size={13} />
            </Link>
          </div>
        )}
      </section>

      {/* ── 4 Kotak KPI Utama ── */}
      <section className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-3.5">
        {/* KPI 1: Tugas Menunggu */}
        <Link
          href={taskCounts.approval > 0 ? "/surat/approval" : "/surat/review"}
          className="group rounded-xl border border-border bg-surface p-4 hover:border-primary/40 hover:shadow-sm transition-all flex flex-col justify-between"
        >
          <div className="flex items-start justify-between gap-2">
            <span className="p-2.5 rounded-xl bg-accent/10 text-accent group-hover:bg-accent group-hover:text-white transition-colors">
              <ClipboardCheck size={20} />
            </span>
            {taskCounts.tasksTotal > 0 && (
              <span className="px-2 py-0.5 rounded-full text-[11px] font-bold bg-accent text-white">
                {taskCounts.tasksTotal} Butuh Tindakan
              </span>
            )}
          </div>
          <div className="mt-3">
            <p className="text-xs font-semibold text-text-muted">Tugas Surat Saya</p>
            <p className="text-2xl font-black text-text mt-0.5">
              {loading ? "…" : taskCounts.tasksTotal}
            </p>
            <p className="text-[11px] text-text-muted mt-1">
              {taskCounts.review} review · {taskCounts.approval} approval
            </p>
          </div>
        </Link>

        {/* KPI 2: Kotak Masuk Inbox */}
        <Link
          href="/inbox"
          className="group rounded-xl border border-border bg-surface p-4 hover:border-primary/40 hover:shadow-sm transition-all flex flex-col justify-between"
        >
          <div className="flex items-start justify-between gap-2">
            <span className="p-2.5 rounded-xl bg-primary/10 text-primary group-hover:bg-primary group-hover:text-white transition-colors">
              <Inbox size={20} />
            </span>
            {inboxUnread > 0 && (
              <span className="px-2 py-0.5 rounded-full text-[11px] font-bold bg-primary text-white">
                {inboxUnread} Baru
              </span>
            )}
          </div>
          <div className="mt-3">
            <p className="text-xs font-semibold text-text-muted">Kotak Masuk (Inbox)</p>
            <p className="text-2xl font-black text-text mt-0.5">
              {loading ? "…" : inboxTotal}
            </p>
            <p className="text-[11px] text-text-muted mt-1">
              {inboxUnread} surat belum dibaca
            </p>
          </div>
        </Link>

        {/* KPI 3: Surat Saya */}
        <Link
          href="/surat/daftar"
          className="group rounded-xl border border-border bg-surface p-4 hover:border-primary/40 hover:shadow-sm transition-all flex flex-col justify-between"
        >
          <div className="flex items-start justify-between gap-2">
            <span className="p-2.5 rounded-xl bg-status-info-bg text-status-info-text group-hover:bg-status-info-text group-hover:text-white transition-colors">
              <FileText size={20} />
            </span>
            <span className="text-xs text-text-muted flex items-center gap-1 group-hover:text-primary transition-colors">
              Lihat Semua <ArrowRight size={12} />
            </span>
          </div>
          <div className="mt-3">
            <p className="text-xs font-semibold text-text-muted">Surat yang Saya Buat</p>
            <p className="text-2xl font-black text-text mt-0.5">
              {loading ? "…" : mineLettersCount}
            </p>
            <p className="text-[11px] text-text-muted mt-1">
              Draf, diajukan, &amp; diterbitkan
            </p>
          </div>
        </Link>

        {/* KPI 4: Delegasi */}
        <Link
          href="/delegasi"
          className="group rounded-xl border border-border bg-surface p-4 hover:border-primary/40 hover:shadow-sm transition-all flex flex-col justify-between"
        >
          <div className="flex items-start justify-between gap-2">
            <span className="p-2.5 rounded-xl bg-status-success-bg text-status-success-text group-hover:bg-status-success-text group-hover:text-white transition-colors">
              <UserCheck size={20} />
            </span>
            <span className="text-[11px] font-semibold text-primary">
              Kelola Jadwal
            </span>
          </div>
          <div className="mt-3">
            <p className="text-xs font-semibold text-text-muted">Status Delegasi</p>
            <p className="text-sm font-bold text-text mt-1 truncate">
              {activeSelfDelegation
                ? `Ke ${activeSelfDelegation.delegate.name}`
                : activeRepresenting.length > 0
                  ? `Mewakili ${activeRepresenting.length} Pejabat`
                  : "Tidak Ada Delegasi Aktif"}
            </p>
            <p className="text-[11px] text-text-muted mt-1">
              {activeSelfDelegation
                ? `Berlaku s.d. ${activeSelfDelegation.until}`
                : "Atur pengganti saat dinas/cuti"}
            </p>
          </div>
        </Link>
      </section>

      {/* ── Panduan Alur Persuratan Digital (Mulai dari Mana?) ── */}
      <section className="rounded-2xl border border-border bg-surface p-5 sm:p-6 shadow-xs space-y-5">
        <div className="flex flex-wrap items-center justify-between gap-2 border-b border-border/80 pb-3">
          <div>
            <div className="flex items-center gap-2">
              <Layers size={17} className="text-primary" />
              <h2 className="text-base font-bold text-text">
                Panduan Alur Persuratan Digital — Mulai dari Mana?
              </h2>
            </div>
            <p className="text-xs text-text-muted mt-0.5">
              Tata naskah dinas digital terbagi menjadi 5 tahapan utama yang
              saling terhubung secara otomatis.
            </p>
          </div>
        </div>

        <div className="grid grid-cols-1 md:grid-cols-3 lg:grid-cols-5 gap-3">
          {/* Step 1 */}
          <div className="rounded-xl border border-border bg-panel/30 p-4 space-y-3 flex flex-col justify-between">
            <div className="space-y-2">
              <div className="flex items-center justify-between">
                <span className="w-6 h-6 rounded-full bg-primary text-white text-xs font-bold flex items-center justify-center">
                  1
                </span>
                <span className="text-[11px] font-semibold text-primary">
                  Drafter
                </span>
              </div>
              <h3 className="text-sm font-bold text-text">Penciptaan Surat</h3>
              <p className="text-xs text-text-muted leading-relaxed">
                Buat naskah dinas internal menggunakan template .docx resmi, atau catat
                surat masuk yang diterima dari pihak eksternal.
              </p>
            </div>
            <div className="space-y-1.5 pt-2 border-t border-border/60">
              {canCreate ? (
                <>
                  <Link
                    href="/surat/baru"
                    className="flex items-center justify-between w-full px-2.5 py-1.5 rounded-lg bg-surface hover:bg-panel border border-border text-[11px] font-semibold text-text hover:text-primary transition-colors"
                  >
                    <span>+ Buat Surat Internal</span>
                    <ArrowRight size={12} />
                  </Link>
                  <Link
                    href="/surat/baru?type=SURAT_MASUK_EKSTERNAL"
                    className="flex items-center justify-between w-full px-2.5 py-1.5 rounded-lg bg-surface hover:bg-panel border border-border text-[11px] font-semibold text-text hover:text-primary transition-colors"
                  >
                    <span>+ Catat Surat Masuk</span>
                    <ArrowRight size={12} />
                  </Link>
                </>
              ) : (
                <span className="text-[11px] text-text-muted italic block">
                  Khusus role pembuat surat
                </span>
              )}
            </div>
          </div>

          {/* Step 2 */}
          <div className="rounded-xl border border-border bg-panel/30 p-4 space-y-3 flex flex-col justify-between">
            <div className="space-y-2">
              <div className="flex items-center justify-between">
                <span className="w-6 h-6 rounded-full bg-primary text-white text-xs font-bold flex items-center justify-center">
                  2
                </span>
                <span className="text-[11px] font-semibold text-accent">
                  Reviewer
                </span>
              </div>
              <h3 className="text-sm font-bold text-text">Pemeriksaan &amp; Koreksi</h3>
              <p className="text-xs text-text-muted leading-relaxed">
                Reviewer memeriksa redaksi di editor Word terintegrasi, memberikan
                catatan revisi ke drafter, atau menyetujui ke tahap approver.
              </p>
            </div>
            <div className="pt-2 border-t border-border/60">
              <Link
                href="/surat/review"
                className="flex items-center justify-between w-full px-2.5 py-1.5 rounded-lg bg-surface hover:bg-panel border border-border text-[11px] font-semibold text-text hover:text-primary transition-colors"
              >
                <span>Tugas Menunggu Review</span>
                <span className="font-bold text-accent">
                  {taskCounts.review}
                </span>
              </Link>
            </div>
          </div>

          {/* Step 3 */}
          <div className="rounded-xl border border-border bg-panel/30 p-4 space-y-3 flex flex-col justify-between">
            <div className="space-y-2">
              <div className="flex items-center justify-between">
                <span className="w-6 h-6 rounded-full bg-primary text-white text-xs font-bold flex items-center justify-center">
                  3
                </span>
                <span className="text-[11px] font-semibold text-status-success-text">
                  Pejabat / VP
                </span>
              </div>
              <h3 className="text-sm font-bold text-text">Persetujuan &amp; Tanda Tangan</h3>
              <p className="text-xs text-text-muted leading-relaxed">
                Pejabat menyahkan naskah dengan QR Code resmi sistem, atau menyetujui
                cetak PDF untuk verifikasi tanda tangan basah.
              </p>
            </div>
            <div className="pt-2 border-t border-border/60">
              <Link
                href="/surat/approval"
                className="flex items-center justify-between w-full px-2.5 py-1.5 rounded-lg bg-surface hover:bg-panel border border-border text-[11px] font-semibold text-text hover:text-primary transition-colors"
              >
                <span>Menunggu Approval</span>
                <span className="font-bold text-status-success-text">
                  {taskCounts.approval}
                </span>
              </Link>
            </div>
          </div>

          {/* Step 4 */}
          <div className="rounded-xl border border-border bg-panel/30 p-4 space-y-3 flex flex-col justify-between">
            <div className="space-y-2">
              <div className="flex items-center justify-between">
                <span className="w-6 h-6 rounded-full bg-primary text-white text-xs font-bold flex items-center justify-center">
                  4
                </span>
                <span className="text-[11px] font-semibold text-text-muted">
                  Otomatis Sistem
                </span>
              </div>
              <h3 className="text-sm font-bold text-text">Penomoran &amp; Distribusi</h3>
              <p className="text-xs text-text-muted leading-relaxed">
                Nomor surat resmi diterbitkan otomatis ber-watermark PDF, lalu masuk ke
                Inbox seluruh unit penerima dan tembusan (CC).
              </p>
            </div>
            <div className="pt-2 border-t border-border/60">
              <Link
                href="/inbox"
                className="flex items-center justify-between w-full px-2.5 py-1.5 rounded-lg bg-surface hover:bg-panel border border-border text-[11px] font-semibold text-text hover:text-primary transition-colors"
              >
                <span>Buka Kotak Masuk</span>
                <ArrowRight size={12} />
              </Link>
            </div>
          </div>

          {/* Step 5 */}
          <div className="rounded-xl border border-border bg-panel/30 p-4 space-y-3 flex flex-col justify-between">
            <div className="space-y-2">
              <div className="flex items-center justify-between">
                <span className="w-6 h-6 rounded-full bg-primary text-white text-xs font-bold flex items-center justify-center">
                  5
                </span>
                <span className="text-[11px] font-semibold text-primary">
                  Tindak Lanjut
                </span>
              </div>
              <h3 className="text-sm font-bold text-text">Disposisi &amp; Delegasi</h3>
              <p className="text-xs text-text-muted leading-relaxed">
                Pimpinan mendisposisikan surat ke bawahan di Inbox, atau membuat jadwal
                delegasi bila sedang dinas luar kantor.
              </p>
            </div>
            <div className="space-y-1.5 pt-2 border-t border-border/60">
              <Link
                href="/inbox"
                className="flex items-center justify-between w-full px-2.5 py-1.5 rounded-lg bg-surface hover:bg-panel border border-border text-[11px] font-semibold text-text hover:text-primary transition-colors"
              >
                <span>Disposisi di Inbox</span>
                <ArrowRight size={12} />
              </Link>
              <Link
                href="/delegasi"
                className="flex items-center justify-between w-full px-2.5 py-1.5 rounded-lg bg-surface hover:bg-panel border border-border text-[11px] font-semibold text-text hover:text-primary transition-colors"
              >
                <span>Atur Delegasi</span>
                <ArrowRight size={12} />
              </Link>
            </div>
          </div>
        </div>
      </section>

      {/* ── 2 Kolom Real-time: Tugas Segera vs Kotak Masuk Terkini ── */}
      <section className="grid grid-cols-1 lg:grid-cols-2 gap-5">
        {/* Kolom Kiri: Tugas Menunggu Tindak Lanjut */}
        <div className="rounded-2xl border border-border bg-surface p-5 shadow-xs flex flex-col justify-between space-y-4">
          <div>
            <div className="flex items-center justify-between gap-2 border-b border-border/70 pb-3">
              <div className="flex items-center gap-2">
                <ClipboardCheck size={17} className="text-accent" />
                <h2 className="text-base font-bold text-text">
                  Tugas Perlu Tindak Lanjut Segera
                </h2>
              </div>
              <Link
                href="/surat/daftar?scope=tasks"
                className="text-xs font-semibold text-primary hover:underline flex items-center gap-1"
              >
                Semua Tugas <ChevronRight size={13} />
              </Link>
            </div>

            {loading ? (
              <div className="py-12 flex items-center justify-center gap-2 text-xs text-text-muted">
                <Loader2 size={16} className="animate-spin text-primary" />
                Memuat daftar tugas...
              </div>
            ) : recentTasks.length === 0 ? (
              <div className="py-12 text-center space-y-2">
                <CheckCircle2 size={28} className="mx-auto text-status-success-text opacity-80" />
                <p className="text-sm font-semibold text-text">
                  Semua tugas telah selesai
                </p>
                <p className="text-xs text-text-muted max-w-sm mx-auto">
                  Saat ini tidak ada surat dinas yang sedang menunggu review atau
                  approval dari Anda.
                </p>
              </div>
            ) : (
              <ul className="divide-y divide-border/60 mt-1">
                {recentTasks.map((item) => (
                  <li
                    key={item.id}
                    className="py-3 flex items-start justify-between gap-3 group hover:bg-panel/30 rounded-lg px-2 -mx-2 transition-colors"
                  >
                    <div className="min-w-0 space-y-1">
                      <div className="flex items-center gap-2 flex-wrap">
                        <StatusBadge status={item.status} />
                        <span className="text-[11px] font-medium text-text-muted">
                          {item.letterType?.name ?? "Surat Dinas"}
                        </span>
                        {item.onBehalfOf && (
                          <span className="text-[10px] font-semibold text-primary bg-primary/10 px-1.5 py-0.2 rounded">
                            Mewakili {item.onBehalfOf.name}
                          </span>
                        )}
                      </div>
                      <Link
                        href={`/surat/${item.id}`}
                        className="text-sm font-semibold text-text hover:text-primary transition-colors line-clamp-1"
                      >
                        {item.title || "(Tanpa perihal)"}
                      </Link>
                      <p className="text-xs text-text-muted truncate">
                        Pengirim: {item.senderUnit?.name ?? "—"}
                      </p>
                    </div>

                    <Link
                      href={`/surat/${item.id}`}
                      className="inline-flex items-center gap-1 px-3 py-1.5 rounded-lg bg-surface border border-border text-xs font-semibold text-text hover:border-primary/40 hover:text-primary transition-colors shrink-0 shadow-2xs mt-1"
                    >
                      Buka <ArrowRight size={12} />
                    </Link>
                  </li>
                ))}
              </ul>
            )}
          </div>

          <div className="pt-2 border-t border-border/70 flex items-center justify-between text-xs text-text-muted">
            <span>
              Total <strong className="text-text">{taskCounts.tasksTotal}</strong>{" "}
              tugas aktif
            </span>
            <div className="flex gap-2">
              <Link
                href="/surat/review"
                className="hover:text-primary transition-colors"
              >
                Review ({taskCounts.review})
              </Link>
              <span>·</span>
              <Link
                href="/surat/approval"
                className="hover:text-primary transition-colors"
              >
                Approval ({taskCounts.approval})
              </Link>
            </div>
          </div>
        </div>

        {/* Kolom Kanan: Surat Masuk Terkini di Inbox */}
        <div className="rounded-2xl border border-border bg-surface p-5 shadow-xs flex flex-col justify-between space-y-4">
          <div>
            <div className="flex items-center justify-between gap-2 border-b border-border/70 pb-3">
              <div className="flex items-center gap-2">
                <Inbox size={17} className="text-primary" />
                <h2 className="text-base font-bold text-text">
                  Surat Masuk Terkini (Kotak Masuk)
                </h2>
              </div>
              <Link
                href="/inbox"
                className="text-xs font-semibold text-primary hover:underline flex items-center gap-1"
              >
                Buka Inbox <ChevronRight size={13} />
              </Link>
            </div>

            {loading ? (
              <div className="py-12 flex items-center justify-center gap-2 text-xs text-text-muted">
                <Loader2 size={16} className="animate-spin text-primary" />
                Memuat surat masuk...
              </div>
            ) : recentInbox.length === 0 ? (
              <div className="py-12 text-center space-y-2">
                <Inbox size={28} className="mx-auto text-text-muted opacity-60" />
                <p className="text-sm font-semibold text-text">
                  Kotak Masuk Anda Kosong
                </p>
                <p className="text-xs text-text-muted max-w-sm mx-auto">
                  Belum ada surat masuk resmi atau disposisi baru yang ditujukan
                  kepada Anda.
                </p>
              </div>
            ) : (
              <ul className="divide-y divide-border/60 mt-1">
                {recentInbox.map((item) => (
                  <li
                    key={item.id}
                    className="py-3 flex items-start justify-between gap-3 group hover:bg-panel/30 rounded-lg px-2 -mx-2 transition-colors"
                  >
                    <div className="min-w-0 space-y-1">
                      <div className="flex items-center gap-2 flex-wrap">
                        {item.status === "UNREAD" ? (
                          <span className="px-2 py-0.5 rounded-full text-[10px] font-bold bg-primary text-white">
                            Belum Dibaca
                          </span>
                        ) : (
                          <span className="px-2 py-0.5 rounded-full text-[10px] font-medium bg-panel text-text-muted border border-border">
                            Sudah Dibaca
                          </span>
                        )}
                        <span className="text-[11px] font-mono text-text-muted">
                          {item.letter.letterNumber || "Surat"}
                        </span>
                        {item.source.recipientType === "CC" && (
                          <span className="text-[10px] px-1.5 py-0.2 rounded bg-panel border border-border text-text-muted">
                            Tembusan
                          </span>
                        )}
                      </div>
                      <Link
                        href={`/inbox/${item.id}`}
                        className="text-sm font-semibold text-text hover:text-primary transition-colors line-clamp-1"
                      >
                        {item.letter.title || "(Tanpa perihal)"}
                      </Link>
                      <p className="text-xs text-text-muted truncate">
                        Dari: {item.letter.senderUnit?.name || item.letter.external?.sender || "—"}
                      </p>
                    </div>

                    <Link
                      href={`/inbox/${item.id}`}
                      className="inline-flex items-center gap-1 px-3 py-1.5 rounded-lg bg-surface border border-border text-xs font-semibold text-text hover:border-primary/40 hover:text-primary transition-colors shrink-0 shadow-2xs mt-1"
                    >
                      Baca <ArrowRight size={12} />
                    </Link>
                  </li>
                ))}
              </ul>
            )}
          </div>

          <div className="pt-2 border-t border-border/70 flex items-center justify-between text-xs text-text-muted">
            <span>
              Total <strong className="text-text">{inboxTotal}</strong> surat di
              kotak masuk
            </span>
            <Link href="/inbox" className="hover:text-primary transition-colors">
              Buka Kotak Masuk Lengkap →
            </Link>
          </div>
        </div>
      </section>

      {/* ── Akses Cepat Fitur Lainnya ── */}
      <section className="space-y-3">
        <h2 className="text-sm font-bold text-text-muted uppercase tracking-wider">
          Akses Cepat Modul Aplikasi
        </h2>
        <div className="grid grid-cols-2 sm:grid-cols-3 lg:grid-cols-6 gap-3">
          <Link
            href="/surat/baru"
            className="p-3.5 rounded-xl border border-border bg-surface hover:border-primary/40 hover:shadow-2xs transition-all flex flex-col items-center text-center gap-2 group"
          >
            <span className="p-2 rounded-lg bg-primary/10 text-primary group-hover:bg-primary group-hover:text-white transition-colors">
              <FilePlus size={18} />
            </span>
            <span className="text-xs font-bold text-text group-hover:text-primary transition-colors">
              Buat Surat
            </span>
          </Link>

          <Link
            href="/inbox"
            className="p-3.5 rounded-xl border border-border bg-surface hover:border-primary/40 hover:shadow-2xs transition-all flex flex-col items-center text-center gap-2 group"
          >
            <span className="p-2 rounded-lg bg-primary/10 text-primary group-hover:bg-primary group-hover:text-white transition-colors">
              <Inbox size={18} />
            </span>
            <span className="text-xs font-bold text-text group-hover:text-primary transition-colors">
              Inbox
            </span>
          </Link>

          <Link
            href="/delegasi"
            className="p-3.5 rounded-xl border border-border bg-surface hover:border-primary/40 hover:shadow-2xs transition-all flex flex-col items-center text-center gap-2 group"
          >
            <span className="p-2 rounded-lg bg-primary/10 text-primary group-hover:bg-primary group-hover:text-white transition-colors">
              <UserCheck size={18} />
            </span>
            <span className="text-xs font-bold text-text group-hover:text-primary transition-colors">
              Delegasi
            </span>
          </Link>

          <Link
            href="/surat/daftar"
            className="p-3.5 rounded-xl border border-border bg-surface hover:border-primary/40 hover:shadow-2xs transition-all flex flex-col items-center text-center gap-2 group"
          >
            <span className="p-2 rounded-lg bg-primary/10 text-primary group-hover:bg-primary group-hover:text-white transition-colors">
              <FileText size={18} />
            </span>
            <span className="text-xs font-bold text-text group-hover:text-primary transition-colors">
              Daftar Surat
            </span>
          </Link>

          {canAudit && (
            <Link
              href="/rekap"
              className="p-3.5 rounded-xl border border-border bg-surface hover:border-primary/40 hover:shadow-2xs transition-all flex flex-col items-center text-center gap-2 group"
            >
              <span className="p-2 rounded-lg bg-primary/10 text-primary group-hover:bg-primary group-hover:text-white transition-colors">
                <BarChart3 size={18} />
              </span>
              <span className="text-xs font-bold text-text group-hover:text-primary transition-colors">
                Monitoring
              </span>
            </Link>
          )}

          {canManageMaster && (
            <Link
              href="/master-data"
              className="p-3.5 rounded-xl border border-border bg-surface hover:border-primary/40 hover:shadow-2xs transition-all flex flex-col items-center text-center gap-2 group"
            >
              <span className="p-2 rounded-lg bg-primary/10 text-primary group-hover:bg-primary group-hover:text-white transition-colors">
                <Database size={18} />
              </span>
              <span className="text-xs font-bold text-text group-hover:text-primary transition-colors">
                Master Data
              </span>
            </Link>
          )}
        </div>
      </section>
    </div>
  );
}

