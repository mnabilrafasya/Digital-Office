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

function mapBackendLetter(b: BackendLetterItem): LetterListItem {
  const isDrafterEditable =
    b.status === "DRAFT" || b.status === "RETURNED_TO_DRAFTER";
  const isReviewable = b.status === "IN_REVIEW";
  const isApprovable = b.status === "IN_APPROVAL";

  const actions: string[] = [];
  if (isDrafterEditable) actions.push("EDIT");
  if (isReviewable) actions.push("REVIEW");
  if (isApprovable) actions.push("APPROVE");

  return {
    id: b.id,
    status: b.status as LetterStatus,
    letterNumber: b.letterNumber ?? null,
    title: b.title ?? null,
    letterType: b.letterType ?? { id: "", code: "", name: "-" },
    confidentiality: b.confidentiality ?? { id: "", code: "BIASA", name: "Biasa" },
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

export async function listLetters(
  params: ListLettersParams = {}
): Promise<ListLettersResponse> {
  const {
    scope = "mine",
    status,
    q,
    letterTypeId,
    confidentialityId,
    from,
    to,
    page = 1,
    pageSize = 10,
  } = params;

  const isTaskScope = scope === "approval" || scope === "review" || scope === "tasks";
  const backendScope = isTaskScope ? "tasks" : "mine";

  // 1. Fetch ringkasan count dari semua surat user (pageSize=100)
  const countParams = new URLSearchParams({
    scope: backendScope,
    page: "1",
    pageSize: "100",
  });

  // 2. Fetch data halaman yang sedang aktif
  const queryParams = new URLSearchParams({
    scope: backendScope,
    page: String(page),
    pageSize: String(pageSize),
  });

  // Normalisasi status untuk tab 'RETURNED' -> 'RETURNED_TO_DRAFTER,RETURNED_TO_REVIEWER'
  if (status) {
    if (status === "RETURNED_TO_DRAFTER") {
      queryParams.set("status", "RETURNED_TO_DRAFTER,RETURNED_TO_REVIEWER");
    } else {
      queryParams.set("status", status);
    }
  } else if (scope === "review") {
    queryParams.set("status", "IN_REVIEW");
  } else if (scope === "approval") {
    queryParams.set("status", "IN_APPROVAL");
  }

  if (q && q.trim()) {
    queryParams.set("q", q.trim());
  }

  const [resCurrent, resAll] = await Promise.all([
    fetch(`${API}/letters?${queryParams.toString()}`, { cache: "no-store" }),
    fetch(`${API}/letters?${countParams.toString()}`, { cache: "no-store" }).catch(() => null),
  ]);

  if (!resCurrent.ok) {
    const err = await resCurrent.json().catch(() => ({}));
    throw new Error(err.error?.message ?? `Gagal memuat daftar surat (HTTP ${resCurrent.status})`);
  }

  const jsonCurrent = (await resCurrent.json()) as BackendListResponse;
  const items = jsonCurrent.data.map(mapBackendLetter);

  // Filter client-side tambahan untuk filter yang belum ada parameternya di backend
  let filteredItems = items;
  if (letterTypeId) {
    filteredItems = filteredItems.filter((it) => it.letterType?.id === letterTypeId);
  }
  if (confidentialityId) {
    filteredItems = filteredItems.filter((it) => it.confidentiality?.id === confidentialityId);
  }
  if (from) {
    filteredItems = filteredItems.filter((it) => it.createdAt >= from);
  }
  if (to) {
    filteredItems = filteredItems.filter((it) => it.createdAt <= to + "T23:59:59+07:00");
  }

  // Agregasi counts
  let allItems: BackendLetterItem[] = [];
  if (resAll && resAll.ok) {
    const jsonAll = (await resAll.json().catch(() => null)) as BackendListResponse | null;
    if (jsonAll?.data) {
      allItems = jsonAll.data;
    }
  } else {
    allItems = jsonCurrent.data;
  }

  const counts = {
    ALL: allItems.length,
    DRAFT: allItems.filter((l) => l.status === "DRAFT").length,
    IN_REVIEW: allItems.filter((l) => l.status === "IN_REVIEW").length,
    RETURNED: allItems.filter(
      (l) => l.status === "RETURNED_TO_DRAFTER" || l.status === "RETURNED_TO_REVIEWER"
    ).length,
    IN_APPROVAL: allItems.filter((l) => l.status === "IN_APPROVAL").length,
    SENT: allItems.filter((l) => l.status === "SENT").length,
    CANCELLED: allItems.filter((l) => l.status === "CANCELLED").length,
  };

  return {
    data: filteredItems,
    meta: {
      page: jsonCurrent.meta.page,
      pageSize: jsonCurrent.meta.pageSize,
      total: jsonCurrent.meta.total,
    },
    counts,
  };
}
