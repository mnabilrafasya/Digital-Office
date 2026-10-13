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

export type InboxStatus = "UNREAD" | "READ" | "REDIRECTED";

export type Sifat = "BIASA" | "TERBATAS" | "RAHASIA";
export type Kecepatan = "SEGERA" | "SANGAT_SEGERA";

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

/** Satu entri reviewer atau approver di form */
export interface StepEntry {
  id: string;          // uuid sementara di frontend
  employeeId: string;
  employeeName: string;
  jabatan: string;
  unitName: string;
}
