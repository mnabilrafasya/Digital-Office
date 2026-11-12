// ─────────────────────────────────────────────
// components/review/audit-trail-utils.ts
// Utilitas pemrosesan riwayat aksi (audit trail) & konstruksi node diagram alur surat
// ─────────────────────────────────────────────

import type {
  LetterDetail,
  LetterHistoryEntry,
} from "@/lib/api/letter-detail";
import type { TrackingTimelineEvent } from "@/lib/api/audit";

export type ActionTone = "success" | "warn" | "danger" | "info" | "neutral";
export type ActionCategory = "milestone" | "version" | "activity";

export interface ActionMeta {
  label: string;
  shortLabel: string;
  tone: ActionTone;
  category: ActionCategory;
}

export const ACTION_META: Record<string, ActionMeta> = {
  CREATE: {
    label: "Draf Surat Dibuat",
    shortLabel: "Draf Dibuat",
    tone: "info",
    category: "milestone",
  },
  SAVE: {
    label: "Versi Naskah Disimpan",
    shortLabel: "Simpan Versi",
    tone: "neutral",
    category: "version",
  },
  SUBMIT: {
    label: "Dikirim untuk Diperiksa",
    shortLabel: "Kirim Review",
    tone: "info",
    category: "milestone",
  },
  APPROVE: {
    label: "Disetujui",
    shortLabel: "Disetujui",
    tone: "success",
    category: "milestone",
  },
  RETURN: {
    label: "Dikembalikan untuk Revisi",
    shortLabel: "Dikembalikan",
    tone: "warn",
    category: "milestone",
  },
  SIGN: {
    label: "Disahkan & Diterbitkan",
    shortLabel: "Terbit & Sah",
    tone: "success",
    category: "milestone",
  },
  UPLOAD_SIGNED: {
    label: "Scan Bertanda Tangan Basah Diunggah",
    shortLabel: "Unggah Scan TTD",
    tone: "success",
    category: "milestone",
  },
  RECEIVE: {
    label: "Surat Masuk Eksternal Dicatat",
    shortLabel: "Surat Masuk",
    tone: "info",
    category: "milestone",
  },
  INCOMING_RECEIVE: {
    label: "Surat Masuk Eksternal Dicatat",
    shortLabel: "Surat Masuk",
    tone: "info",
    category: "milestone",
  },
  CANCEL: {
    label: "Surat Dibatalkan",
    shortLabel: "Dibatalkan",
    tone: "danger",
    category: "milestone",
  },
  READ: {
    label: "Naskah Dibuka / Dibaca",
    shortLabel: "Dibaca",
    tone: "neutral",
    category: "activity",
  },
};

export function getActionMeta(action: string): ActionMeta {
  return (
    ACTION_META[action] ?? {
      label: action,
      shortLabel: action,
      tone: "neutral",
      category: "activity",
    }
  );
}

export function formatCompactTime(iso?: string | null): string {
  if (!iso) return "—";
  const d = new Date(iso);
  if (!Number.isFinite(d.getTime())) return "—";
  return new Intl.DateTimeFormat("id-ID", {
    day: "numeric",
    month: "short",
    hour: "2-digit",
    minute: "2-digit",
  }).format(d);
}

export function formatFullDateTime(iso?: string | null): string {
  if (!iso) return "—";
  const d = new Date(iso);
  if (!Number.isFinite(d.getTime())) return "—";
  return new Intl.DateTimeFormat("id-ID", {
    day: "numeric",
    month: "short",
    year: "numeric",
    hour: "2-digit",
    minute: "2-digit",
  }).format(d);
}

/** Item riwayat yang sudah dikelompokkan agar aksi berulang (READ/SAVE) tidak memenuhi layar */
export interface GroupedHistoryItem {
  id: string;
  entry: LetterHistoryEntry;
  meta: ActionMeta;
  repeatCount: number;
  groupedEntries: LetterHistoryEntry[];
  versionNumbers: number[];
}

export type HistoryFilterMode = "milestone" | "all" | "notes";

/**
 * Mengelompokkan aksi minor berturut-turut (seperti READ atau SAVE berulang oleh orang yang sama)
 * dan menyaring sesuai mode tampilan.
 */
export function buildGroupedHistory(
  history: LetterHistoryEntry[],
  filter: HistoryFilterMode = "milestone"
): GroupedHistoryItem[] {
  // Urutkan dari terbaru ke terlama untuk tampilan timeline ringkas
  const sorted = [...history].sort(
    (a, b) => new Date(b.createdAt).getTime() - new Date(a.createdAt).getTime()
  );

  const filtered = sorted.filter((item) => {
    const meta = getActionMeta(item.action);
    if (filter === "milestone") {
      return meta.category === "milestone" || Boolean(item.note);
    }
    if (filter === "notes") {
      return Boolean(item.note) || item.action === "RETURN";
    }
    return true;
  });

  const groups: GroupedHistoryItem[] = [];

  for (const item of filtered) {
    const meta = getActionMeta(item.action);
    const prev = groups[groups.length - 1];

    // Gabungkan aksi READ atau SAVE berturut-turut tanpa catatan
    const canMerge =
      prev &&
      !item.note &&
      !prev.entry.note &&
      item.action === prev.entry.action &&
      (item.action === "READ" || item.action === "SAVE") &&
      item.actor.id === prev.entry.actor.id;

    if (canMerge) {
      prev.repeatCount += 1;
      prev.groupedEntries.push(item);
      if (
        item.versionNumber != null &&
        !prev.versionNumbers.includes(item.versionNumber)
      ) {
        prev.versionNumbers.push(item.versionNumber);
      }
    } else {
      groups.push({
        id: `${item.createdAt}-${item.action}-${groups.length}`,
        entry: item,
        meta,
        repeatCount: 1,
        groupedEntries: [item],
        versionNumbers:
          item.versionNumber != null ? [item.versionNumber] : [],
      });
    }
  }

  return groups;
}

/** Konversi fallback dari TrackingTimelineEvent ke LetterHistoryEntry */
export function trackingTimelineToHistory(
  timeline: TrackingTimelineEvent[]
): LetterHistoryEntry[] {
  return timeline.map((ev) => ({
    action: ev.action,
    actorRole: ev.role ?? "",
    actor: ev.actor,
    target: ev.target,
    fromStatus: null,
    toStatus: "",
    note: ev.note,
    versionNumber: null,
    createdAt: ev.at,
  }));
}

export interface FlowDiagramNode {
  id: string;
  roleLabel: string;
  roleCode: "DRAFTER" | "REVIEWER" | "APPROVER" | "FINAL";
  actorName: string;
  actorPosition: string;
  state: "DONE" | "ACTIVE" | "RETURNED" | "PENDING" | "CANCELLED";
  lastActionAt: string | null;
  returnCount: number;
  latestReturnNote: string | null;
  savedVersions: number[];
}

export interface FlowDiagramSummary {
  nodes: FlowDiagramNode[];
  totalMilestones: number;
  revisionCount: number;
  readCount: number;
  saveCount: number;
}

/**
 * Membangun node visualisasi diagram perjalanan surat dari hulu ke hilir:
 * Pembuat (Drafter) -> Reviewer 1..N -> Approver -> Terbit/Selesai
 */
export function buildFlowDiagram(
  letter: LetterDetail,
  history: LetterHistoryEntry[]
): FlowDiagramSummary {
  const isReturnedToDrafter = letter.status === "RETURNED_TO_DRAFTER";
  const isDraft = letter.status === "DRAFT";
  const isSent = letter.status === "SENT";
  const isCancelled = letter.status === "CANCELLED";

  const drafterActions = history.filter(
    (h) => h.actor.id === letter.drafter.id || h.actorRole === "DRAFTER"
  );
  const drafterVersions = drafterActions
    .map((h) => h.versionNumber)
    .filter((v): v is number => v != null);
  const lastDrafterAction =
    drafterActions[drafterActions.length - 1]?.createdAt ?? letter.createdAt;

  const nodes: FlowDiagramNode[] = [
    {
      id: "node-drafter",
      roleLabel:
        letter.direction === "INCOMING"
          ? "Pencatat Surat Masuk"
          : "Pembuat Surat",
      roleCode: "DRAFTER",
      actorName: letter.drafter.name,
      actorPosition: letter.drafter.positionName,
      state: isReturnedToDrafter
        ? "RETURNED"
        : isDraft
          ? "ACTIVE"
          : "DONE",
      lastActionAt: lastDrafterAction,
      returnCount: 0,
      latestReturnNote: isReturnedToDrafter ? letter.lastReturnNote : null,
      savedVersions: Array.from(new Set(drafterVersions)),
    },
  ];

  const totalApprovers = letter.steps.filter(
    (s) => s.role === "APPROVER"
  ).length;

  for (const step of letter.steps) {
    const stepActions = history.filter(
      (h) =>
        h.actor.id === step.user.id &&
        (!h.actorRole || h.actorRole === step.role)
    );
    const returnsByStep = stepActions.filter((h) => h.action === "RETURN");
    const latestReturn = returnsByStep[returnsByStep.length - 1];
    const stepVersions = stepActions
      .map((h) => h.versionNumber)
      .filter((v): v is number => v != null);
    const lastStepAction = stepActions[stepActions.length - 1];

    const isReturnedReviewer =
      letter.status === "RETURNED_TO_REVIEWER" &&
      step.state === "ACTIVE" &&
      step.role === "REVIEWER";

    nodes.push({
      id: `node-${step.role}-${step.order}`,
      roleLabel:
        step.role === "APPROVER"
          ? totalApprovers > 1
            ? `Approver ${step.order}`
            : "Approver"
          : `Reviewer ${step.order}`,
      roleCode: step.role,
      actorName: step.user.name,
      actorPosition: step.user.positionName,
      state: isReturnedReviewer
        ? "RETURNED"
        : step.state === "DONE"
          ? "DONE"
          : step.state === "ACTIVE"
            ? "ACTIVE"
            : "PENDING",
      lastActionAt: lastStepAction?.createdAt ?? null,
      returnCount: returnsByStep.length,
      latestReturnNote: step.returnNote ?? latestReturn?.note ?? null,
      savedVersions: Array.from(new Set(stepVersions)),
    });
  }

  // Node hilir: Terbit & Distribusi (atau Dibatalkan)
  const recipientCount =
    letter.recipients.to.length + letter.recipients.cc.length;
  const isWaitingSignedScan = letter.status === "APPROVED";
  const isReceivedIncoming = letter.status === "RECEIVED";
  const externalTargetLabel =
    letter.external?.recipient?.name ??
    letter.external?.sender ??
    `${recipientCount} Unit/Alias Penerima`;

  nodes.push({
    id: "node-final",
    roleLabel: isCancelled
      ? "Dibatalkan"
      : isWaitingSignedScan
        ? "Menunggu Scan TTD"
        : isReceivedIncoming
          ? "Surat Masuk Eksternal"
          : "Terbit & Kirim",
    roleCode: "FINAL",
    actorName: isCancelled
      ? "Naskah Dibatalkan"
      : letter.letterNumber ?? "Nomor Otomatis",
    actorPosition: isCancelled
      ? "Alur dihentikan"
      : letter.direction === "OUTGOING" || letter.direction === "INCOMING"
        ? externalTargetLabel
        : `${recipientCount} Unit/Alias Penerima`,
    state: isCancelled
      ? "CANCELLED"
      : isSent || isReceivedIncoming
        ? "DONE"
        : isWaitingSignedScan
          ? "ACTIVE"
          : "PENDING",
    lastActionAt: letter.approval?.approvedAt ?? null,
    returnCount: 0,
    latestReturnNote: null,
    savedVersions: [],
  });

  const revisionCount = history.filter((h) => h.action === "RETURN").length;
  const readCount = history.filter((h) => h.action === "READ").length;
  const saveCount = history.filter((h) => h.action === "SAVE").length;
  const totalMilestones = history.filter(
    (h) => getActionMeta(h.action).category === "milestone"
  ).length;

  return {
    nodes,
    totalMilestones,
    revisionCount,
    readCount,
    saveCount,
  };
}

