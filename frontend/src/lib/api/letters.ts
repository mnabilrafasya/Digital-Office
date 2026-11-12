// ─────────────────────────────────────────────
// lib/api/letters.ts
// Client API untuk Daftar Surat — terhubung ke Backend Go
// GET /api/v1/letters?scope=mine&status=...&q=...&page=...&pageSize=...
// ─────────────────────────────────────────────

import type {
  LetterListItem,
  ListLettersParams,
  ListLettersResponse,
  LetterStatus,
} from "@/lib/types";

const API = "/api/v1";

interface BackendLetterItem {
  id: string;
  letterNumber: string | null;
  title: string | null;
  status: string;
  letterType: { id: string; code: string; name: string } | null;
  confidentiality: { id: string; code: string; name: string } | null;
  urgency: { id: string; code: string; name: string } | null;
  senderUnit: { id: string; code: string; name: string };
  currentAssignee: { id: string; name: string; positionName: string } | null;
  currentStep: { role: string; order: number } | null;
  createdAt: string;
  updatedAt: string;
}

interface BackendListResponse {
  data: BackendLetterItem[];
  meta: {
    page: number;
    pageSize: number;
    total: number;
  };
}

function mapBackendLetter(
  b: BackendLetterItem,
  backendScope: "mine" | "tasks",
): LetterListItem {
  const isDrafterEditable =
    b.status === "DRAFT" || b.status === "RETURNED_TO_DRAFTER";
  const isReviewable =
    b.status === "IN_REVIEW" || b.status === "RETURNED_TO_REVIEWER";
  const isApprovable = b.status === "IN_APPROVAL" || b.status === "APPROVED";

  const actions: string[] = [];
  if (isDrafterEditable) actions.push("EDIT");
  if (backendScope === "tasks") {
    if (isReviewable) actions.push("REVIEW");
    if (isApprovable) actions.push("APPROVE");
    if (b.status === "APPROVED") actions.push("UPLOAD_SIGNED");
  }

  return {
    id: b.id,
    status: b.status as LetterStatus,
    letterNumber: b.letterNumber ?? null,
    title: b.title ?? null,
    letterType: b.letterType ?? { id: "", code: "", name: "-" },
    confidentiality: b.confidentiality ?? {
      id: "",
      code: "BIASA",
      name: "Biasa",
    },
    urgency: b.urgency ?? { id: "", code: "BIASA", name: "Biasa" },
    senderUnit: {
      id: b.senderUnit?.id ?? "",
      name: b.senderUnit?.name ?? "-",
    },
    recipientsSummary: {
      first: null,
      total: 0,
    },
    createdAt: b.createdAt,
    currentAssignee: b.currentAssignee
      ? {
          id: String(b.currentAssignee.id),
          name: b.currentAssignee.name,
          role: (b.currentStep?.role as "REVIEWER" | "APPROVER") ?? "REVIEWER",
          order: b.currentStep?.order ?? 1,
        }
      : null,
    lastReturnNote: null,
    allowedActions: actions,
  };
}

const COUNT_GROUPS = {
  DRAFT: "DRAFT",
  IN_REVIEW: "IN_REVIEW,RETURNED_TO_REVIEWER",
  RETURNED: "RETURNED_TO_DRAFTER,RETURNED_TO_REVIEWER",
  IN_APPROVAL: "IN_APPROVAL,APPROVED",
  SENT: "SENT,RECEIVED",
  CANCELLED: "CANCELLED",
} as const;

/** Jumlah per status = meta.total dari request pageSize=1 (akurat untuk berapa pun surat). */
async function fetchTotal(scope: string, status: string): Promise<number> {
  const sp = new URLSearchParams({ scope, page: "1", pageSize: "1" });
  if (status) {
    sp.set("status", status);
  }
  const res = await fetch(`${API}/letters?${sp.toString()}`, {
    cache: "no-store",
  });
  if (!res.ok) return 0;
  const json = (await res.json().catch(() => null)) as {
    meta?: { total?: number };
  } | null;
  return json?.meta?.total ?? 0;
}

export interface PendingTaskCounts {
  review: number;
  approval: number;
  tasksTotal: number;
}

/**
 * Mengambil jumlah tugas review dan approval yang sedang menunggu tindak lanjut user aktif.
 * Digunakan untuk menampilkan badge notifikasi oranye pada Sidebar & tab Tugas Saya.
 */
export async function getPendingTaskCounts(): Promise<PendingTaskCounts> {
  try {
    const [review, approval, tasksTotal] = await Promise.all([
      fetchTotal("tasks", "IN_REVIEW,RETURNED_TO_REVIEWER"),
      fetchTotal("tasks", "IN_APPROVAL,APPROVED"),
      fetchTotal("tasks", ""),
    ]);
    return { review, approval, tasksTotal };
  } catch {
    return { review: 0, approval: 0, tasksTotal: 0 };
  }
}

export async function listLetters(
  params: ListLettersParams = {},
): Promise<ListLettersResponse> {
  // Filter jenis/sifat/tanggal belum didukung backend: sengaja diabaikan.
  const { scope = "mine", status, q, page = 1, pageSize = 10 } = params;

  const isTaskScope =
    scope === "approval" || scope === "review" || scope === "tasks";
  const backendScope = isTaskScope ? "tasks" : "mine";

  const queryParams = new URLSearchParams({
    scope: backendScope,
    page: String(page),
    pageSize: String(pageSize),
  });

  if (status) {
    queryParams.set(
      "status",
      status === "RETURNED_TO_DRAFTER"
        ? COUNT_GROUPS.RETURNED
        : status === "IN_REVIEW"
          ? COUNT_GROUPS.IN_REVIEW
          : status === "IN_APPROVAL"
            ? COUNT_GROUPS.IN_APPROVAL
            : status === "SENT"
              ? COUNT_GROUPS.SENT
              : status,
    );
  } else if (scope === "review") {
    queryParams.set("status", "IN_REVIEW,RETURNED_TO_REVIEWER");
  } else if (scope === "approval") {
    queryParams.set("status", "IN_APPROVAL,APPROVED");
  }

  if (q && q.trim()) {
    queryParams.set("q", q.trim());
  }

  const [resCurrent, allTotal, ...totals] = await Promise.all([
    fetch(`${API}/letters?${queryParams.toString()}`, { cache: "no-store" }),
    fetchTotal(backendScope, ""),
    ...Object.values(COUNT_GROUPS).map((g) => fetchTotal(backendScope, g)),
  ]);

  if (!resCurrent.ok) {
    const err = await resCurrent.json().catch(() => ({}));
    throw new Error(
      err.error?.message ??
        `Gagal memuat daftar surat (HTTP ${resCurrent.status})`,
    );
  }

  const jsonCurrent = (await resCurrent.json()) as BackendListResponse;
  const [DRAFT, IN_REVIEW, RETURNED, IN_APPROVAL, SENT, CANCELLED] = totals;

  return {
    data: jsonCurrent.data.map((item) => mapBackendLetter(item, backendScope)),
    meta: {
      page: jsonCurrent.meta.page,
      pageSize: jsonCurrent.meta.pageSize,
      total: jsonCurrent.meta.total,
    },
    counts: {
      ALL: allTotal,
      DRAFT,
      IN_REVIEW,
      RETURNED,
      IN_APPROVAL,
      SENT,
      CANCELLED,
    },
  };
}
