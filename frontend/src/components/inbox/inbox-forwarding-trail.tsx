"use client";

// ─────────────────────────────────────────────
// components/inbox/inbox-forwarding-trail.tsx
// Visualisasi Diagram & Timeline Ringkas Alur Penerusan (Disposisi & Pengalihan) Surat Masuk
// Menggabungkan (stitching) seluruh rantai perjalanan surat lintas putaran (multi-hop & multi-turn)
// dari Penerbit -> Disposisi/Pengalihan Hulu -> Kotak Masuk Anda -> Penerusan Hilir tanpa mengulang.
// ─────────────────────────────────────────────

import { useState, useRef, useEffect, useCallback, useMemo } from "react";
import {
  Share2,
  ArrowRight,
  Building2,
  MailOpen,
  Send,
  CheckCircle2,
  ChevronLeft,
  ChevronRight,
  ChevronDown,
  ChevronUp,
  MessageSquareQuote,
} from "lucide-react";
import {
  inboxSourceLabel,
  inboxStatusLabel,
  type InboxItemDetail,
} from "@/lib/api/inbox";
import type { LetterDetail, PersonRef } from "@/lib/api/letter-detail";
import type { DistributionNode } from "@/lib/api/audit";
import { formatCompactTime } from "@/components/review/audit-trail-utils";
import { cn } from "@/lib/utils";

interface GraphNodeRecord {
  inboxItemId: string;
  parentInboxItemId: string | null;
  recipient: PersonRef;
  via: "TO" | "CC" | "DISPOSITION" | "REDIRECT";
  from: PersonRef | null;
  targetUnitName: string | null;
  note: string | null;
  status: "UNREAD" | "READ" | "DISPOSED" | "REDIRECTED";
  receivedAt: string;
}

const GRAPH_STORAGE_PREFIX = "digital-office:dist-graph-v3:";

function loadGraph(letterId: string): Record<string, GraphNodeRecord> {
  if (typeof window === "undefined") return {};
  try {
    const raw = window.localStorage.getItem(`${GRAPH_STORAGE_PREFIX}${letterId}`);
    if (!raw) return {};
    const parsed = JSON.parse(raw) as unknown;
    return parsed && typeof parsed === "object" && !Array.isArray(parsed)
      ? (parsed as Record<string, GraphNodeRecord>)
      : {};
  } catch {
    return {};
  }
}

function saveGraph(letterId: string, graph: Record<string, GraphNodeRecord>) {
  if (typeof window === "undefined") return;
  try {
    window.localStorage.setItem(
      `${GRAPH_STORAGE_PREFIX}${letterId}`,
      JSON.stringify(graph)
    );
  } catch {
    // Abaikan bila storage penuh
  }
}

/** Mencari Pimpinan Unit penerima langsung (DIRECT TO/CC) dari detail surat & copyFor */
function resolveInitialDirectRecipient(
  letter?: LetterDetail | null,
  copyFor?: string | null
): { person: PersonRef; unitName: string; recipientType: "TO" | "CC" } | null {
  if (!letter) return null;
  const allRecipients: Array<{
    r: LetterDetail["recipients"]["to"][number];
    recipientType: "TO" | "CC";
  }> = [
    ...letter.recipients.to.map((r) => ({ r, recipientType: "TO" as const })),
    ...letter.recipients.cc.map((r) => ({ r, recipientType: "CC" as const })),
  ];

  if (copyFor) {
    const matched = allRecipients.find(
      ({ r }) =>
        r.type === "UNIT" &&
        r.resolvedHead &&
        r.resolvedHead.positionName.toLowerCase() === copyFor.toLowerCase()
    );
    if (matched && matched.r.type === "UNIT" && matched.r.resolvedHead) {
      return {
        person: matched.r.resolvedHead,
        unitName: matched.r.unit.name,
        recipientType: matched.recipientType,
      };
    }
  }

  const firstUnit = allRecipients.find(
    ({ r }) => r.type === "UNIT" && r.resolvedHead
  );
  if (firstUnit && firstUnit.r.type === "UNIT" && firstUnit.r.resolvedHead) {
    return {
      person: firstUnit.r.resolvedHead,
      unitName: firstUnit.r.unit.name,
      recipientType: firstUnit.recipientType,
    };
  }
  return null;
}

/** Menggabungkan pohon tracking.distribution, siblingItems, dan item aktif ke dalam graf kesatuan */
function buildMergedGraph(
  letterId: string,
  item: InboxItemDetail,
  distribution?: DistributionNode[],
  siblingItems?: InboxItemDetail[]
): Record<string, GraphNodeRecord> {
  const graph: Record<string, GraphNodeRecord> = { ...loadGraph(letterId) };

  // 1. Masukkan seluruh node dari pohon tracking.distribution (jika tersedia)
  if (distribution && distribution.length > 0) {
    const walk = (nodes: DistributionNode[], parentId: string | null) => {
      for (const n of nodes) {
        const prev = graph[n.inboxItemId];
        graph[n.inboxItemId] = {
          inboxItemId: n.inboxItemId,
          parentInboxItemId: parentId ?? prev?.parentInboxItemId ?? null,
          recipient: n.recipient,
          via: n.via,
          from: n.from ?? prev?.from ?? null,
          targetUnitName: prev?.targetUnitName ?? null,
          note: n.note ?? prev?.note ?? null,
          status: n.status,
          receivedAt: n.receivedAt,
        };
        if (n.forwards && n.forwards.length > 0) {
          walk(n.forwards, n.inboxItemId);
        }
      }
    };
    walk(distribution, null);
  }

  // 2. Masukkan informasi dari semua item milik user (siblingItems + item aktif)
  const allUserItems = [...(siblingItems ?? []), item];
  for (const it of allUserItems) {
    const prev = graph[it.id];
    const via: GraphNodeRecord["via"] =
      it.source.type === "DIRECT"
        ? it.source.recipientType === "CC"
          ? "CC"
          : "TO"
        : it.source.forwardType === "REDIRECT"
          ? "REDIRECT"
          : "DISPOSITION";

    if (prev) {
      graph[it.id] = {
        ...prev,
        via,
        from: it.source.forwardedBy ?? prev.from,
        note: it.source.note ?? prev.note,
        status: it.status,
        receivedAt: it.receivedAt,
      };
    }

    // Hubungkan penerusan keluar (it.forwards) ke node anak di graf
    for (const fw of it.forwards) {
      // Cari apakah ada node di graf yang cocok dengan penerima forward ini
      const matchingChild = Object.values(graph).find(
        (cand) =>
          cand.inboxItemId !== it.id &&
          cand.recipient.id === fw.recipient.id &&
          (cand.parentInboxItemId === it.id ||
            (!cand.parentInboxItemId && cand.receivedAt >= it.receivedAt))
      );
      if (matchingChild) {
        graph[matchingChild.inboxItemId] = {
          ...matchingChild,
          parentInboxItemId: it.id,
          via: fw.type,
          from: matchingChild.from ?? null,
          targetUnitName: fw.targetUnit?.name ?? matchingChild.targetUnitName,
          note: fw.note ?? matchingChild.note,
        };
      } else {
        // Buat node representasi untuk forward keluar bila belum ada inboxItemId-nya di graf
        const synthId = `fw-${fw.id}`;
        graph[synthId] = {
          inboxItemId: synthId,
          parentInboxItemId: it.id,
          recipient: fw.recipient,
          via: fw.type,
          from: prev?.recipient ?? null,
          targetUnitName: fw.targetUnit?.name ?? null,
          note: fw.note,
          status: "UNREAD",
          receivedAt: fw.createdAt,
        };
      }
    }
  }

  // 3. Rekonsiliasi parentInboxItemId yang masih null untuk node FORWARD berdasarkan from.id & waktu
  const sortedNodes = Object.values(graph).sort(
    (a, b) => new Date(a.receivedAt).getTime() - new Date(b.receivedAt).getTime()
  );

  for (const node of sortedNodes) {
    if (
      !node.parentInboxItemId &&
      (node.via === "DISPOSITION" || node.via === "REDIRECT") &&
      node.from
    ) {
      // Cari node milik pengirim (node.from.id) yang diterima paling dekat sebelum node.receivedAt
      const nodeTime = new Date(node.receivedAt).getTime();
      const candidates = sortedNodes.filter(
        (cand) =>
          cand.inboxItemId !== node.inboxItemId &&
          !cand.inboxItemId.startsWith("fw-") &&
          cand.recipient.id === node.from?.id &&
          new Date(cand.receivedAt).getTime() <= nodeTime + 2000
      );
      const bestParent = candidates[candidates.length - 1];
      if (bestParent) {
        node.parentInboxItemId = bestParent.inboxItemId;
      }
    }
  }

  return graph;
}

interface HopDisplayNode {
  id: string;
  stepNumber: number;
  badgeLabel: string;
  tone: "info" | "warn";
  actorName: string;
  actorPosition: string;
  fromName?: string | null;
  targetUnitName?: string | null;
  timeLabel: string;
  rawTime: string;
  note: string | null;
}

export interface InboxForwardingTrailProps {
  item: InboxItemDetail;
  letter?: LetterDetail | null;
  copyFor?: string | null;
  distribution?: DistributionNode[];
  siblingItems?: InboxItemDetail[];
  compact?: boolean;
}

export function InboxForwardingTrail({
  item,
  letter,
  copyFor,
  distribution,
  siblingItems,
  compact = false,
}: InboxForwardingTrailProps) {
  const [showAllForwards, setShowAllForwards] = useState(false);
  const [selectedNote, setSelectedNote] = useState<{
    title: string;
    note: string;
  } | null>(
    item.source.note
      ? {
          title: inboxSourceLabel(item.source),
          note: item.source.note,
        }
      : null
  );

  // Bangun graf gabungan (memory + localStorage)
  const mergedGraph = useMemo(
    () => buildMergedGraph(item.letter.id, item, distribution, siblingItems),
    [item, distribution, siblingItems]
  );

  // Simpan graf terbaru ke localStorage agar lintas akun di browser tetap berkesinambungan
  useEffect(() => {
    saveGraph(item.letter.id, mergedGraph);
  }, [item.letter.id, mergedGraph]);

  // Rekonstruksi rantai Hulu (Upstream) & Hilir (Downstream) secara utuh tanpa mengulang
  const { upstreamHops, downstreamHops, currentTurnNumber } = useMemo(() => {
    const directRoot = resolveInitialDirectRecipient(letter, copyFor);
    const up: HopDisplayNode[] = [];
    const down: HopDisplayNode[] = [];

    // ── A. Susun Rantai Hulu (Semua tahap sebelum item.id) ──
    if (item.source.type === "FORWARD" && item.source.forwardedBy) {
      // Telusuri parentInboxItemId ke atas dari item.id
      const ancestorChain: GraphNodeRecord[] = [];
      const visited = new Set<string>([item.id]);

      const currentInGraph = mergedGraph[item.id];
      let parentId = currentInGraph?.parentInboxItemId ?? null;

      // Jika belum punya parentId langsung, cari node pengirim (item.source.forwardedBy.id) sebelum item.receivedAt
      if (!parentId) {
        const itemTime = new Date(item.receivedAt).getTime();
        const senderNodes = Object.values(mergedGraph)
          .filter(
            (n) =>
              n.inboxItemId !== item.id &&
              !n.inboxItemId.startsWith("fw-") &&
              n.recipient.id === item.source.forwardedBy?.id &&
              new Date(n.receivedAt).getTime() <= itemTime + 2000
          )
          .sort(
            (a, b) =>
              new Date(a.receivedAt).getTime() - new Date(b.receivedAt).getTime()
          );
        if (senderNodes.length > 0) {
          parentId = senderNodes[senderNodes.length - 1].inboxItemId;
        }
      }

      while (parentId && mergedGraph[parentId] && !visited.has(parentId)) {
        visited.add(parentId);
        const pNode = mergedGraph[parentId];
        ancestorChain.unshift(pNode);
        parentId = pNode.parentInboxItemId;
      }

      // Periksa apakah simpul paling hulu masih punya pengirim di atasnya (misal saat ScopeRecipient memotong root)
      const earliestNode = ancestorChain[0];
      const earliestSender = earliestNode
        ? earliestNode.from
        : item.source.forwardedBy;
      const earliestVia = earliestNode
        ? earliestNode.via
        : item.source.forwardType ?? "DISPOSITION";
      const earliestTime = earliestNode
        ? earliestNode.receivedAt
        : item.receivedAt;
      const earliestNote = earliestNode ? earliestNode.note : item.source.note;

      // 1. Jika ada Pimpinan Unit Penerima Langsung (directRoot) yang belum ada di paling awal rantai
      const firstPersonId = earliestNode
        ? earliestNode.recipient.id
        : item.source.forwardedBy.id;

      if (
        directRoot &&
        directRoot.person.id !== firstPersonId &&
        (!earliestSender || directRoot.person.id !== earliestSender.id)
      ) {
        up.push({
          id: `up-root-${directRoot.person.id}`,
          stepNumber: up.length + 1,
          badgeLabel: "Disposisi Awal",
          tone: "info",
          actorName: directRoot.person.name,
          actorPosition: directRoot.person.positionName,
          fromName: item.letter.senderUnit.name,
          targetUnitName: directRoot.unitName,
          timeLabel: formatCompactTime(item.letter.letterDate),
          rawTime: item.letter.letterDate ?? item.receivedAt,
          note: null,
        });
      }

      // 2. Jika earliestNode adalah FORWARD dan punya pengirim (earliestSender) di atasnya
      if (earliestNode && earliestSender) {
        const isDisp = earliestVia !== "REDIRECT";
        up.push({
          id: `up-pre-${earliestSender.id}-${earliestNode.inboxItemId}`,
          stepNumber: up.length + 1,
          badgeLabel:
            up.length === 0
              ? isDisp
                ? "Disposisi Awal"
                : "Pengalihan Awal"
              : isDisp
                ? `Disposisi #${up.length + 1}`
                : `Pengalihan #${up.length + 1}`,
          tone: isDisp ? "info" : "warn",
          actorName: earliestSender.name,
          actorPosition: earliestSender.positionName,
          timeLabel: formatCompactTime(earliestTime),
          rawTime: earliestTime,
          note: earliestNote,
        });
      }

      // 3. Masukkan seluruh simpul di dalam ancestorChain secara berurutan
      if (ancestorChain.length > 0) {
        for (let i = 0; i < ancestorChain.length; i++) {
          const anc = ancestorChain[i];
          const nextHop = ancestorChain[i + 1];
          const outgoingVia = nextHop
            ? nextHop.via
            : (item.source.forwardType ?? "DISPOSITION");
          const outgoingTime = nextHop ? nextHop.receivedAt : item.receivedAt;
          const outgoingNote = nextHop ? nextHop.note : item.source.note;
          const isDisp = outgoingVia !== "REDIRECT";

          up.push({
            id: `up-anc-${anc.inboxItemId}`,
            stepNumber: up.length + 1,
            badgeLabel:
              up.length === 0
                ? isDisp
                  ? "Disposisi Awal"
                  : "Pengalihan Awal"
                : isDisp
                  ? `Disposisi #${up.length + 1}`
                  : `Pengalihan #${up.length + 1}`,
            tone: isDisp ? "info" : "warn",
            actorName: anc.recipient.name,
            actorPosition: anc.recipient.positionName,
            fromName: anc.from?.name ?? null,
            targetUnitName: anc.targetUnitName,
            timeLabel: formatCompactTime(outgoingTime),
            rawTime: outgoingTime,
            note: outgoingNote,
          });
        }
      } else {
        // Fallback jika ancestorChain kosong: tambahkan pengirim langsung (item.source.forwardedBy)
        const isDisp = item.source.forwardType === "DISPOSITION";
        up.push({
          id: `up-direct-${item.source.forwardedBy.id}`,
          stepNumber: up.length + 1,
          badgeLabel:
            up.length === 0
              ? isDisp
                ? "Disposisi Masuk"
                : "Pengalihan Masuk"
              : isDisp
                ? `Disposisi #${up.length + 1}`
                : `Pengalihan #${up.length + 1}`,
          tone: isDisp ? "info" : "warn",
          actorName: item.source.forwardedBy.name,
          actorPosition: item.source.forwardedBy.positionName,
          timeLabel: formatCompactTime(item.receivedAt),
          rawTime: item.receivedAt,
          note: item.source.note,
        });
      }
    }

    // ── B. Susun Rantai Hilir (Penerusan keluar dari item.id & kelanjutannya) ──
    const visitedDown = new Set<string>([item.id]);

    // Mulai dari item.forwards (penerusan langsung dari item ini)
    for (const f of item.forwards) {
      const isDisp = f.type === "DISPOSITION";
      down.push({
        id: `down-fw-${f.id}`,
        stepNumber: up.length + 1 + down.length + 1,
        badgeLabel: isDisp ? "Disposisi" : "Dialihkan",
        tone: isDisp ? "info" : "warn",
        actorName: f.recipient.name,
        actorPosition: f.recipient.positionName,
        fromName: "Anda",
        targetUnitName:
          f.targetType === "UNIT" && f.targetUnit ? f.targetUnit.name : null,
        timeLabel: formatCompactTime(f.createdAt),
        rawTime: f.createdAt,
        note: f.note,
      });

      // Cek di mergedGraph apakah penerima f.recipient ini meneruskannya lagi ke tahap berikutnya
      const matchingChildNode = Object.values(mergedGraph).find(
        (n) =>
          !n.inboxItemId.startsWith("fw-") &&
          n.parentInboxItemId === item.id &&
          n.recipient.id === f.recipient.id
      );

      if (matchingChildNode) {
        visitedDown.add(matchingChildNode.inboxItemId);
        const walkDescendants = (parentId: string, parentActorName: string) => {
          const children = Object.values(mergedGraph)
            .filter(
              (n) =>
                n.parentInboxItemId === parentId &&
                !visitedDown.has(n.inboxItemId)
            )
            .sort(
              (a, b) =>
                new Date(a.receivedAt).getTime() -
                new Date(b.receivedAt).getTime()
            );

          for (const child of children) {
            visitedDown.add(child.inboxItemId);
            const childDisp = child.via !== "REDIRECT";
            down.push({
              id: `down-node-${child.inboxItemId}`,
              stepNumber: up.length + 1 + down.length + 1,
              badgeLabel: childDisp ? "Disposisi Lanjut" : "Dialihkan Lanjut",
              tone: childDisp ? "info" : "warn",
              actorName: child.recipient.name,
              actorPosition: child.recipient.positionName,
              fromName: child.from?.name ?? parentActorName,
              targetUnitName: child.targetUnitName,
              timeLabel: formatCompactTime(child.receivedAt),
              rawTime: child.receivedAt,
              note: child.note,
            });
            walkDescendants(child.inboxItemId, child.recipient.name);
          }
        };
        walkDescendants(
          matchingChildNode.inboxItemId,
          matchingChildNode.recipient.name
        );
      }
    }

    // Hitung putaran ke berapa bagi user saat ini (jika surat pernah mampir >1 kali)
    const sortedSiblings = [...(siblingItems ?? [])].sort(
      (a, b) =>
        new Date(a.receivedAt).getTime() - new Date(b.receivedAt).getTime()
    );
    const turnIdx = sortedSiblings.findIndex((s) => s.id === item.id);
    const turnNum = turnIdx >= 0 ? turnIdx + 1 : 1;

    return {
      upstreamHops: up,
      downstreamHops: down,
      currentTurnNumber: turnNum,
    };
  }, [item, letter, copyFor, mergedGraph, siblingItems]);

  // Gabungan seluruh kejadian penerusan (hulu + hilir) untuk daftar ringkas di bagian bawah
  const allTrailEvents = useMemo(() => {
    return [...upstreamHops, ...downstreamHops];
  }, [upstreamHops, downstreamHops]);

  // State & ref slider horizontal diagram penerusan
  const scrollRef = useRef<HTMLDivElement>(null);
  const [hasOverflow, setHasOverflow] = useState(false);
  const [canScrollLeft, setCanScrollLeft] = useState(false);
  const [canScrollRight, setCanScrollRight] = useState(false);
  const [scrollPercent, setScrollPercent] = useState(0);

  const updateScrollState = useCallback(() => {
    const el = scrollRef.current;
    if (!el) return;
    const maxScroll = el.scrollWidth - el.clientWidth;
    const overflow = maxScroll > 4;
    setHasOverflow(overflow);
    setCanScrollLeft(el.scrollLeft > 4);
    setCanScrollRight(el.scrollLeft < maxScroll - 4);
    setScrollPercent(
      overflow ? Math.round((el.scrollLeft / maxScroll) * 100) : 0
    );
  }, []);

  useEffect(() => {
    const el = scrollRef.current;
    if (!el) return;
    updateScrollState();
    const ro = new ResizeObserver(() => updateScrollState());
    ro.observe(el);
    return () => ro.disconnect();
  }, [upstreamHops.length, downstreamHops.length, updateScrollState]);

  function scrollByAmount(delta: number) {
    scrollRef.current?.scrollBy({ left: delta, behavior: "smooth" });
  }

  function handleSliderChange(val: number) {
    const el = scrollRef.current;
    if (!el) return;
    const maxScroll = el.scrollWidth - el.clientWidth;
    el.scrollLeft = (val / 100) * maxScroll;
    setScrollPercent(val);
  }

  const totalNodes = 2 + upstreamHops.length + downstreamHops.length;
  const visibleEvents = showAllForwards
    ? allTrailEvents
    : allTrailEvents.slice(-3);
  const hiddenCount = Math.max(0, allTrailEvents.length - 3);

  return (
    <div className="w-full min-w-0 max-w-full border border-border rounded-xl bg-surface overflow-hidden">
      {/* Header Kartu */}
      <div className="px-4 py-2.5 border-b border-border bg-panel/30 flex items-center justify-between gap-2 flex-wrap">
        <div className="flex items-center gap-1.5 flex-wrap">
          <Share2 size={14} className="text-primary shrink-0" />
          <h3 className="text-xs sm:text-sm font-semibold text-text">
            Alur Disposisi & Pengalihan
          </h3>
          <span className="text-[10px] font-semibold px-2 py-0.5 rounded-full bg-surface border border-border text-text-muted">
            {totalNodes} Simpul Alur
          </span>
          {currentTurnNumber > 1 && (
            <span className="text-[10px] font-semibold px-2 py-0.5 rounded-full bg-primary/10 text-primary">
              Putaran ke-{currentTurnNumber}
            </span>
          )}
        </div>

        {hasOverflow && (
          <div className="inline-flex items-center gap-1 ml-auto">
            <button
              type="button"
              onClick={() => scrollByAmount(-180)}
              disabled={!canScrollLeft}
              className="w-6 h-6 rounded-md border border-border bg-surface flex items-center justify-center text-text-muted hover:text-text disabled:opacity-35 transition-colors"
              title="Geser ke kiri"
            >
              <ChevronLeft size={13} />
            </button>
            <button
              type="button"
              onClick={() => scrollByAmount(180)}
              disabled={!canScrollRight}
              className="w-6 h-6 rounded-md border border-border bg-surface flex items-center justify-center text-text-muted hover:text-text disabled:opacity-35 transition-colors"
              title="Geser ke kanan"
            >
              <ChevronRight size={13} />
            </button>
          </div>
        )}
      </div>

      <div className="p-3.5 space-y-3">
        {/* Diagram Visual Rantai Penerusan (Horizontal Scrollable + Slider) */}
        <div
          ref={scrollRef}
          onScroll={updateScrollState}
          className="w-full max-w-full overflow-x-auto pb-1.5 scroll-smooth"
        >
          <div className="inline-flex items-stretch gap-1.5 pr-2">
            {/* Simpul 1: Unit Pengirim & Penandatangan */}
            <div className="flex items-center shrink-0">
              <div
                className={cn(
                  "rounded-xl border border-status-success-text/30 bg-status-success-bg/30 flex flex-col justify-between shrink-0",
                  compact ? "w-38 p-2" : "w-42 p-2.5"
                )}
              >
                <div>
                  <div className="flex items-center gap-1.5 mb-1">
                    <span className="w-5 h-5 rounded-full bg-status-success-text text-white flex items-center justify-center shrink-0">
                      <Send size={10} />
                    </span>
                    <span className="text-[10px] font-bold uppercase tracking-wide text-status-success-text truncate">
                      {item.letter.direction === "INCOMING"
                        ? "Pengirim Eksternal"
                        : "Penerbit Surat"}
                    </span>
                  </div>
                  <p className="text-xs font-semibold text-text truncate">
                    {item.letter.direction === "INCOMING" &&
                    item.letter.external?.sender
                      ? item.letter.external.sender
                      : item.letter.senderUnit.name}
                  </p>
                  <p className="text-[11px] text-text-muted truncate">
                    {item.letter.approver?.name ??
                      item.letter.external?.referenceNumber ??
                      item.letter.senderUnit.name}
                  </p>
                </div>
                <div className="mt-1.5 pt-1 border-t border-border/50 text-[10px] text-text-muted">
                  {formatCompactTime(item.letter.letterDate)}
                </div>
              </div>

              <div className="flex items-center px-1 shrink-0">
                <div className="w-3 h-0.5 bg-status-success-text" />
                <ArrowRight
                  size={13}
                  className="-ml-1 text-status-success-text"
                />
              </div>
            </div>

            {/* Simpul Hulu (Upstream): Seluruh riwayat disposisi & pengalihan sebelum masuk ke item ini */}
            {upstreamHops.map((hop) => (
              <div key={hop.id} className="flex items-center shrink-0">
                <button
                  type="button"
                  onClick={() =>
                    hop.note &&
                    setSelectedNote({
                      title: `${hop.badgeLabel} (${hop.actorName})`,
                      note: hop.note,
                    })
                  }
                  className={cn(
                    "text-left rounded-xl border flex flex-col justify-between shrink-0 transition-all",
                    compact ? "w-38 p-2" : "w-42 p-2.5",
                    hop.tone === "info"
                      ? "border-status-info-text/30 bg-status-info-bg/30 hover:border-status-info-text"
                      : "border-status-warn-text/30 bg-status-warn-bg/30 hover:border-status-warn-text"
                  )}
                >
                  <div>
                    <div className="flex items-center gap-1.5 mb-1">
                      <span
                        className={cn(
                          "w-5 h-5 rounded-full text-white flex items-center justify-center shrink-0",
                          hop.tone === "info"
                            ? "bg-status-info-text"
                            : "bg-status-warn-text"
                        )}
                      >
                        {hop.tone === "info" ? (
                          <Share2 size={10} />
                        ) : (
                          <ArrowRight size={10} />
                        )}
                      </span>
                      <span
                        className={cn(
                          "text-[10px] font-bold uppercase tracking-wide truncate",
                          hop.tone === "info"
                            ? "text-status-info-text"
                            : "text-status-warn-text"
                        )}
                      >
                        {hop.badgeLabel}
                      </span>
                    </div>
                    <p className="text-xs font-semibold text-text truncate">
                      {hop.actorName}
                    </p>
                    <p className="text-[11px] text-text-muted truncate">
                      {hop.actorPosition}
                    </p>
                  </div>
                  <div className="mt-1.5 pt-1 border-t border-border/50 flex items-center justify-between text-[10px] text-text-muted">
                    <span>{hop.timeLabel}</span>
                    {hop.note && (
                      <span className="text-[9px] font-semibold px-1 rounded bg-surface border border-border text-primary">
                        Catatan
                      </span>
                    )}
                  </div>
                </button>

                <div className="flex items-center px-1 shrink-0">
                  <div className="w-3 h-0.5 bg-primary" />
                  <ArrowRight size={13} className="-ml-1 text-primary" />
                </div>
              </div>
            ))}

            {/* Simpul Aktif: Kotak Masuk Anda */}
            <div className="flex items-center shrink-0">
              <div
                className={cn(
                  "rounded-xl border border-primary bg-primary/5 ring-2 ring-primary/15 flex flex-col justify-between shrink-0",
                  compact ? "w-38 p-2" : "w-42 p-2.5"
                )}
              >
                <div>
                  <div className="flex items-center justify-between gap-1 mb-1">
                    <span className="w-5 h-5 rounded-full bg-primary text-white flex items-center justify-center shrink-0">
                      <MailOpen size={10} />
                    </span>
                    <span className="text-[10px] font-bold uppercase tracking-wide text-primary truncate">
                      Kotak Masuk Anda
                    </span>
                  </div>
                  <p className="text-xs font-semibold text-text truncate">
                    {item.source.type === "DIRECT"
                      ? item.source.recipientType === "CC"
                        ? "Penerima Tembusan"
                        : "Penerima Utama"
                      : currentTurnNumber > 1
                        ? `Penerima (Putaran ${currentTurnNumber})`
                        : "Penerima Terusan"}
                  </p>
                  <p className="text-[11px] text-text-muted truncate">
                    Status: {inboxStatusLabel(item.status)}
                  </p>
                </div>
                <div className="mt-1.5 pt-1 border-t border-border/50 text-[10px] text-text-muted">
                  {formatCompactTime(item.receivedAt)}
                </div>
              </div>

              {downstreamHops.length > 0 && (
                <div className="flex items-center px-1 shrink-0">
                  <div className="w-3 h-0.5 bg-primary" />
                  <ArrowRight size={13} className="-ml-1 text-primary" />
                </div>
              )}
            </div>

            {/* Simpul Hilir (Downstream): Daftar Penerusan Keluar & Kelanjutannya */}
            {downstreamHops.map((hop, idx) => (
              <div key={hop.id} className="flex items-center shrink-0">
                <button
                  type="button"
                  onClick={() =>
                    hop.note &&
                    setSelectedNote({
                      title: `${hop.badgeLabel} ke ${hop.actorName}`,
                      note: hop.note,
                    })
                  }
                  className={cn(
                    "text-left rounded-xl border transition-all flex flex-col justify-between shrink-0",
                    compact ? "w-40 p-2" : "w-44 p-2.5",
                    hop.tone === "info"
                      ? "bg-status-info-bg/35 border-status-info-text/30 hover:border-status-info-text"
                      : "bg-status-warn-bg/35 border-status-warn-text/30 hover:border-status-warn-text"
                  )}
                >
                  <div>
                    <div className="flex items-center justify-between gap-1 mb-1">
                      <span
                        className={cn(
                          "w-5 h-5 rounded-full text-white flex items-center justify-center shrink-0",
                          hop.tone === "info"
                            ? "bg-status-info-text"
                            : "bg-status-warn-text"
                        )}
                      >
                        {hop.tone === "info" ? (
                          <Share2 size={10} />
                        ) : (
                          <ArrowRight size={10} />
                        )}
                      </span>
                      <span
                        className={cn(
                          "text-[10px] font-bold uppercase tracking-wide truncate",
                          hop.tone === "info"
                            ? "text-status-info-text"
                            : "text-status-warn-text"
                        )}
                      >
                        {hop.badgeLabel}
                      </span>
                    </div>
                    <p className="text-xs font-semibold text-text truncate">
                      {hop.actorName}
                    </p>
                    <p className="text-[11px] text-text-muted truncate">
                      {hop.targetUnitName ?? hop.actorPosition}
                    </p>
                  </div>
                  <div className="mt-1.5 pt-1 border-t border-border/50 flex items-center justify-between gap-1 text-[10px] text-text-muted">
                    <span>{hop.timeLabel}</span>
                    {hop.note && (
                      <span className="text-[9px] font-semibold px-1 rounded bg-surface border border-border text-primary">
                        Instruksi
                      </span>
                    )}
                  </div>
                </button>

                {idx < downstreamHops.length - 1 && (
                  <div className="flex items-center px-1 shrink-0">
                    <div className="w-3 h-0.5 bg-border" />
                    <ArrowRight size={12} className="-ml-1 text-text-muted" />
                  </div>
                )}
              </div>
            ))}
          </div>
        </div>

        {/* Slider Geser Horizontal bila simpul penerusan panjang */}
        {hasOverflow && (
          <div className="flex items-center gap-2 px-2.5 py-1 rounded-lg bg-panel/40 border border-border/70">
            <button
              type="button"
              onClick={() => scrollByAmount(-180)}
              disabled={!canScrollLeft}
              className="text-text-muted hover:text-text disabled:opacity-30"
              title="Geser kiri"
            >
              <ChevronLeft size={13} className="shrink-0" />
            </button>
            <input
              type="range"
              min={0}
              max={100}
              value={scrollPercent}
              onChange={(e) => handleSliderChange(Number(e.target.value))}
              aria-label="Geser alur disposisi"
              className="w-full h-1.5 bg-border rounded-lg appearance-none cursor-pointer accent-primary"
            />
            <button
              type="button"
              onClick={() => scrollByAmount(180)}
              disabled={!canScrollRight}
              className="text-text-muted hover:text-text disabled:opacity-30"
              title="Geser kanan"
            >
              <ChevronRight size={13} className="shrink-0" />
            </button>
            <span className="text-[10px] text-text-muted shrink-0">Geser</span>
          </div>
        )}

        {/* Kotak Instruksi / Catatan Terpilih */}
        {selectedNote && (
          <div className="rounded-lg bg-panel/40 border border-border px-3 py-2 text-xs flex items-start gap-2">
            <MessageSquareQuote
              size={14}
              className="text-primary shrink-0 mt-0.5"
            />
            <div className="min-w-0 flex-1">
              <span className="font-semibold text-text">
                Instruksi ({selectedNote.title}):{" "}
              </span>
              <span className="text-text-muted italic">
                &ldquo;{selectedNote.note}&rdquo;
              </span>
            </div>
          </div>
        )}

        {/* Daftar Ringkas Riwayat Penerusan Gabungan (Hulu + Hilir) */}
        {allTrailEvents.length > 0 && (
          <div className="pt-2 border-t border-border/70 space-y-2">
            <div className="flex items-center justify-between gap-2">
              <p className="text-[11px] font-semibold uppercase tracking-wide text-text-muted">
                Rekam Jejak Disposisi & Pengalihan ({allTrailEvents.length} Tahap)
              </p>
              {(hiddenCount > 0 || showAllForwards) && (
                <button
                  type="button"
                  onClick={() => setShowAllForwards((v) => !v)}
                  className="inline-flex items-center gap-1 text-[11px] font-medium text-primary hover:underline"
                >
                  {showAllForwards ? (
                    <>
                      <ChevronUp size={12} /> Ringkas (3 terbaru)
                    </>
                  ) : (
                    <>
                      <ChevronDown size={12} /> Lihat semua ({allTrailEvents.length})
                    </>
                  )}
                </button>
              )}
            </div>
            <ul className="space-y-1.5">
              {visibleEvents.map((ev) => (
                <li
                  key={ev.id}
                  className="px-2.5 py-1.5 rounded-lg bg-panel/25 border border-border/60 flex items-start justify-between gap-2 text-xs"
                >
                  <div className="min-w-0 flex-1">
                    <div className="flex items-center gap-1.5 flex-wrap">
                      <CheckCircle2
                        size={11}
                        className={
                          ev.tone === "info"
                            ? "text-status-info-text shrink-0"
                            : "text-status-warn-text shrink-0"
                        }
                      />
                      <span className="text-[10px] font-bold px-1.5 py-0.2 rounded bg-surface border border-border text-text-muted">
                        Tahap {ev.stepNumber}
                      </span>
                      <span className="font-semibold text-text">
                        {ev.badgeLabel}: {ev.actorName}
                      </span>
                      <span className="text-[11px] text-text-muted">
                        ({ev.actorPosition})
                      </span>
                      {ev.targetUnitName && (
                        <span className="inline-flex items-center gap-0.5 text-[10px] px-1.5 py-0.2 rounded bg-surface border border-border text-text-muted">
                          <Building2 size={9} /> {ev.targetUnitName}
                        </span>
                      )}
                    </div>
                    {ev.note && (
                      <p className="text-[11px] text-text-muted italic mt-0.5 line-clamp-1">
                        &ldquo;{ev.note}&rdquo;
                      </p>
                    )}
                  </div>
                  <span className="text-[10px] text-text-muted shrink-0">
                    {ev.timeLabel}
                  </span>
                </li>
              ))}
            </ul>
          </div>
        )}
      </div>
    </div>
  );
}
