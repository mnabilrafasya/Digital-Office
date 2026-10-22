// ─────────────────────────────────────────────
// Tipe domain — Digital Office
// Enum dan interface sesuai PROJECT_CONTEXT bagian 5 & 11
// Jangan mengubah nilai enum LetterStatus / InboxStatus
// ─────────────────────────────────────────────

export type LetterStatus =
  | "DRAFT"
  | "IN_REVIEW"
  | "RETURNED_TO_DRAFTER"
  | "IN_APPROVAL"
  | "RETURNED_TO_REVIEWER"
  | "SENT"
  | "CANCELLED";

export type InboxStatus = "UNREAD" | "READ" | "DISPOSED" | "REDIRECTED";

export type Sifat = "BIASA" | "TERBATAS" | "RAHASIA";
export type Kecepatan = "BIASA" | "SEGERA" | "SANGAT_SEGERA";

// ─── Workflow ───────────────────────────────

export interface StepAction {
  type: "APPROVED" | "RETURNED";
  note?: string; // wajib bila RETURNED
  at: string;    // ISO timestamp
}

export interface WorkflowStep {
  id: string;
  role: "REVIEWER" | "APPROVER";
  order: number; // tahap 1, 2, ...
  employeeId: string;
  status: "PENDING" | "ACTIVE" | "DONE";
  actions: StepAction[];
}

// ─── Attachment ─────────────────────────────

export interface LetterAttachment {
  id: string;
  fileName: string;
  mimeType: string;
  sizeBytes: number;
  uploadedAt: string; // ISO
}

// ─── Letter ─────────────────────────────────

export interface Letter {
  id: string;
  companyId: string;          // isolasi data per perusahaan
  status: LetterStatus;
  drafterId: string;
  number: string | null;      // null sampai SENT
  letterDate: string | null;  // = waktu approval akhir
  title: string;              // maks 250
  summary?: string;
  specialNote?: string;
  letterTypeId: string;
  templateId?: string;
  classificationId: string;
  sifat: Sifat;
  kecepatan: Kecepatan;
  toUnitIds: string[];
  ccUnitIds: string[];
  steps: WorkflowStep[];
  returnedByStepId?: string;
  attachments: LetterAttachment[];
  currentVersion: number;
  cancelReason?: string;
}

// ─── Master data (untuk dropdown / picker) ──

export interface Employee {
  id: string;
  nik: string;
  name: string;
  jabatan: string;
  unitId: string;
  unitName: string;
}

export interface OrgUnit {
  id: string;
  name: string;
  companyId: string;
}

export interface LetterType {
  id: string;
  name: string;
}

export interface LetterTemplate {
  id: string;
  name: string;
  letterTypeId: string;
}

export interface Classification {
  id: string;
  code: string;
  name: string;
}

// ─── Form types ─────────────────────────────

export type KodeAlias = "R" | "A" | "REVIEWER" | "APPROVER";

/** Satu entri reviewer atau approver di form */
export interface StepEntry {
  id: string;          // uuid sementara di frontend
  employeeId: string;
  employeeName: string;
  jabatan: string;
  unitName: string;
  kodeAlias?: "R" | "A" | string;
  kode_alias?: "R" | "A" | string;
}

// ─── API Lookup ──────────────────────────────

export interface LookupItem {
  id: string;
  code: string;
  name: string;
  note?: string;
}

// ─── Letter List (kontrak bagian 13) ─────────

export interface CurrentAssigneeSummary {
  id: string;
  name: string;
  role: "REVIEWER" | "APPROVER";
  order: number;
}

export interface RecipientsSummary {
  first: string | null;
  total: number;
}

export interface LetterListItem {
  id: string;
  status: LetterStatus;
  letterNumber: string | null;
  title: string | null;
  letterType: { id: string; code: string; name: string } | null;
  confidentiality: { id: string; code: string; name: string } | null;
  urgency: { id: string; code: string; name: string } | null;
  senderUnit: { id: string; name: string };
  recipientsSummary: RecipientsSummary;
  createdAt: string;
  currentAssignee: CurrentAssigneeSummary | null;
  lastReturnNote: string | null;
  allowedActions: string[];
}

export interface ListLettersResponse {
  data: LetterListItem[];
  meta: { page: number; pageSize: number; total: number };
  counts: {
    ALL: number;
    DRAFT: number;
    IN_REVIEW: number;
    RETURNED: number;
    IN_APPROVAL: number;
    SENT: number;
    CANCELLED: number;
  };
}

export interface ListLettersParams {
  scope?: "mine" | "review" | "approval";
  status?: LetterStatus | "";
  q?: string;
  letterTypeId?: string;
  confidentialityId?: string;
  from?: string;
  to?: string;
  page?: number;
  pageSize?: number;
}
