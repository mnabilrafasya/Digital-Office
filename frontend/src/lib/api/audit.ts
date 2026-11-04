// ─────────────────────────────────────────────
// lib/api/audit.ts
// Klien API untuk Lacak Surat (Tracking), Audit Log, & Pantauan Surat (Monitoring)
// Terhubung ke backend Go: internal/audit/handler.go
// ─────────────────────────────────────────────

const API = "/api/v1";

export interface AuditPersonRef {
  id: string;
  name: string;
  positionName: string;
}

// ─── Lacak Surat (GET /api/v1/letters/{letterId}/tracking) ───

export interface TrackingPosition {
  label: string;
  holder: AuditPersonRef | null;
  role: "DRAFTER" | "REVIEWER" | "APPROVER" | null;
  step: number | null;
  since: string;
  waitingHours: number | null;
  stuck: boolean;
}

export interface TrackingTimelineEvent {
  at: string;
  action: string;
  label: string;
  actor: AuditPersonRef;
  role: string | null;
  target: AuditPersonRef | null;
  note: string | null;
}

export interface DistributionNode {
  inboxItemId: string;
  recipient: AuditPersonRef;
  via: "TO" | "CC" | "DISPOSITION" | "REDIRECT";
  from: AuditPersonRef | null;
  note: string | null;
  status: "UNREAD" | "READ" | "DISPOSED" | "REDIRECTED";
  receivedAt: string;
  readAt: string | null;
  forwards: DistributionNode[];
}

export interface LetterTracking {
  letterId: string;
  letterNumber: string | null;
  title: string | null;
  status: string;
  position: TrackingPosition;
  timeline: TrackingTimelineEvent[];
  distribution: DistributionNode[];
  scope: "FULL" | "RECIPIENT";
}

// ─── Audit Log (GET /api/v1/audit-logs & /letters/{id}/audit) ───

export interface AuditLogActor {
  id: string | null;
  name: string;
}

export interface AuditLogLetter {
  number: string | null;
  title: string | null;
  masked: boolean;
}

export interface AuditLogEntry {
  id: string;
  at: string;
  action: string;
  entityType: "LETTER" | "TEMPLATE" | "AUDIT" | string;
  entityId: string | null;
  actor: AuditLogActor;
  letter: AuditLogLetter | null;
  ipAddress: string | null;
  requestId: string | null;
  snapshot?: Record<string, unknown> | null;
}

export interface ListAuditLogsParams {
  from?: string;
  to?: string;
  actorId?: string;
  action?: string;
  entityType?: string;
  q?: string;
  page?: number;
  pageSize?: number;
}

// ─── Pantauan Surat (GET /api/v1/monitoring/letters) ───

export interface MonitoredLetter {
  letterId: string;
  title: string | null;
  titleMasked: boolean;
  status: string;
  senderUnit: string;
  drafter: AuditPersonRef;
  holder: AuditPersonRef | null;
  step: number | null;
  since: string;
  waitingHours: number;
  stuck: boolean;
}

export interface ListMonitoredLettersParams {
  status?: string;
  minWaitingHours?: string;
  page?: number;
  pageSize?: number;
}

export interface PaginatedResult<T> {
  data: T[];
  meta: {
    page: number;
    pageSize: number;
    total: number;
  };
}

/**
 * Mengambil posisi surat saat ini, timeline alur, dan pohon distribusi penerima.
 * Endpoint: GET /api/v1/letters/{letterId}/tracking
 */
export async function getLetterTracking(
  letterId: string
): Promise<LetterTracking | null> {
  const res = await fetch(`${API}/letters/${letterId}/tracking`, {
    cache: "no-store",
  });
  if (res.status === 404 || res.status === 403) return null;
  if (!res.ok) {
    const err = await res.json().catch(() => ({}));
    throw new Error(
      err.error?.message ?? `Gagal memuat pelacakan surat (HTTP ${res.status})`
    );
  }
  const json = (await res.json()) as { data?: LetterTracking };
  return json.data ?? null;
}

/**
 * Mengambil seluruh audit log untuk satu surat (wajib permission AUDIT_VIEW).
 * Endpoint: GET /api/v1/letters/{letterId}/audit
 */
export async function getLetterAuditLogs(
  letterId: string
): Promise<AuditLogEntry[]> {
  const res = await fetch(`${API}/letters/${letterId}/audit`, {
    cache: "no-store",
  });
  if (res.status === 403 || res.status === 404) return [];
  if (!res.ok) {
    const err = await res.json().catch(() => ({}));
    throw new Error(
      err.error?.message ?? `Gagal memuat audit log surat (HTTP ${res.status})`
    );
  }
  const json = (await res.json()) as { data?: AuditLogEntry[] };
  return Array.isArray(json.data) ? json.data : [];
}

/**
 * Mengambil daftar audit log perusahaan dengan filter (wajib permission AUDIT_VIEW).
 * Endpoint: GET /api/v1/audit-logs
 */
export async function listAuditLogs(
  params: ListAuditLogsParams = {}
): Promise<PaginatedResult<AuditLogEntry>> {
  const sp = new URLSearchParams();
  if (params.from) sp.set("from", params.from);
  if (params.to) sp.set("to", params.to);
  if (params.actorId) sp.set("actorId", params.actorId);
  if (params.action) sp.set("action", params.action);
  if (params.entityType) sp.set("entityType", params.entityType);
  if (params.q?.trim()) sp.set("q", params.q.trim());
  sp.set("page", String(params.page ?? 1));
  sp.set("pageSize", String(params.pageSize ?? 20));

  const res = await fetch(`${API}/audit-logs?${sp.toString()}`, {
    cache: "no-store",
  });
  if (!res.ok) {
    const err = await res.json().catch(() => ({}));
    throw new Error(
      err.error?.message ?? `Gagal memuat daftar audit log (HTTP ${res.status})`
    );
  }
  const json = (await res.json()) as PaginatedResult<AuditLogEntry>;
  return {
    data: Array.isArray(json.data) ? json.data : [],
    meta: json.meta ?? { page: 1, pageSize: 20, total: 0 },
  };
}

/**
 * Mengambil detail satu baris audit log beserta snapshot JSON (wajib permission AUDIT_VIEW).
 * Endpoint: GET /api/v1/audit-logs/{logId}
 */
export async function getAuditLogDetail(
  logId: string
): Promise<AuditLogEntry | null> {
  const res = await fetch(`${API}/audit-logs/${logId}`, {
    cache: "no-store",
  });
  if (res.status === 404) return null;
  if (!res.ok) {
    const err = await res.json().catch(() => ({}));
    throw new Error(
      err.error?.message ?? `Gagal memuat detail audit log (HTTP ${res.status})`
    );
  }
  const json = (await res.json()) as { data?: AuditLogEntry };
  return json.data ?? null;
}

/**
 * Mengunduh CSV audit log (rentang wajib diisi, maksimal 31 hari).
 * Endpoint: GET /api/v1/audit-logs/export
 */
export async function downloadAuditLogsCsv(
  params: Required<Pick<ListAuditLogsParams, "from" | "to">> &
    Omit<ListAuditLogsParams, "from" | "to" | "page" | "pageSize">
): Promise<void> {
  const sp = new URLSearchParams({
    from: params.from,
    to: params.to,
  });
  if (params.actorId) sp.set("actorId", params.actorId);
  if (params.action) sp.set("action", params.action);
  if (params.entityType) sp.set("entityType", params.entityType);
  if (params.q?.trim()) sp.set("q", params.q.trim());

  const res = await fetch(`${API}/audit-logs/export?${sp.toString()}`, {
    cache: "no-store",
  });
  if (!res.ok) {
    const err = await res.json().catch(() => ({}));
    throw new Error(
      err.error?.message ?? `Gagal mengekspor CSV audit log (HTTP ${res.status})`
    );
  }
  const blob = await res.blob();
  const url = URL.createObjectURL(blob);
  const a = document.createElement("a");
  a.href = url;
  a.download = `audit-log_${params.from}_${params.to}.csv`;
  document.body.appendChild(a);
  a.click();
  document.body.removeChild(a);
  URL.revokeObjectURL(url);
}

/**
 * Mengambil daftar pantauan surat yang sedang berjalan (wajib permission AUDIT_VIEW).
 * Endpoint: GET /api/v1/monitoring/letters
 */
export async function listMonitoredLetters(
  params: ListMonitoredLettersParams = {}
): Promise<PaginatedResult<MonitoredLetter>> {
  const sp = new URLSearchParams();
  if (params.status) sp.set("status", params.status);
  if (params.minWaitingHours) sp.set("minWaitingHours", params.minWaitingHours);
  sp.set("page", String(params.page ?? 1));
  sp.set("pageSize", String(params.pageSize ?? 20));

  const res = await fetch(`${API}/monitoring/letters?${sp.toString()}`, {
    cache: "no-store",
  });
  if (!res.ok) {
    const err = await res.json().catch(() => ({}));
    throw new Error(
      err.error?.message ?? `Gagal memuat pantauan surat (HTTP ${res.status})`
    );
  }
  const json = (await res.json()) as PaginatedResult<MonitoredLetter>;
  return {
    data: Array.isArray(json.data) ? json.data : [],
    meta: json.meta ?? { page: 1, pageSize: 20, total: 0 },
  };
}

