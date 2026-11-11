-- 000019_external_letters.up.sql
-- Approver bertingkat dan surat eksternal:
--   - Surat boleh punya lebih dari satu approver (berurutan, step 1..n).
--   - letters.external_info (JSON): tujuan surat eksternal keluar, atau data asal surat masuk eksternal.
--   - Status baru: APPROVED (surat eksternal keluar disetujui, menunggu scan bertanda tangan basah)
--     dan RECEIVED (surat masuk eksternal yang diunggah dan didisposisikan).
--   - Aksi baru di letter_actions: UPLOAD_SIGNED (scan diunggah) dan RECEIVE (surat masuk dicatat).
SET XACT_ABORT ON;
BEGIN TRANSACTION;

-- Approver bertingkat
ALTER TABLE letter_participants DROP CONSTRAINT ck_letter_participants_approver_step;
DROP INDEX uq_letter_participants_one_approver ON letter_participants;

-- Data eksternal
ALTER TABLE letters ADD external_info NVARCHAR(MAX) NULL
    CONSTRAINT ck_letters_external_info_json CHECK (external_info IS NULL OR ISJSON(external_info) = 1);

-- Status surat
ALTER TABLE letters DROP CONSTRAINT ck_letters_status;
ALTER TABLE letters ADD CONSTRAINT ck_letters_status CHECK (status IN
    ('DRAFT','IN_REVIEW','RETURNED_TO_DRAFTER','IN_APPROVAL','RETURNED_TO_REVIEWER','APPROVED','SENT','RECEIVED','CANCELLED'));

-- Surat masuk eksternal hanya berisi jenis dan perihal (data asalnya di external_info)
ALTER TABLE letters DROP CONSTRAINT ck_letters_complete_after_draft;
ALTER TABLE letters ADD CONSTRAINT ck_letters_complete_after_draft CHECK (
    status = 'DRAFT'
    OR (status = 'RECEIVED' AND letter_type_id IS NOT NULL AND title IS NOT NULL)
    OR (letter_type_id IS NOT NULL AND classification_id IS NOT NULL
        AND confidentiality_level_id IS NOT NULL AND urgency_level_id IS NOT NULL
        AND title IS NOT NULL)
);

-- Surat yang sudah disetujui approver terakhir (APPROVED/SENT) pasti bernomor, bertanggal, dan
-- ada penandatangannya
ALTER TABLE letters DROP CONSTRAINT ck_letters_sent_complete;
ALTER TABLE letters ADD CONSTRAINT ck_letters_sent_complete CHECK (
    status NOT IN ('APPROVED','SENT')
    OR (letter_number IS NOT NULL AND approved_at IS NOT NULL AND signed_by_id IS NOT NULL)
);

-- Riwayat aksi
ALTER TABLE letter_actions DROP CONSTRAINT ck_letter_actions_action;
ALTER TABLE letter_actions ADD CONSTRAINT ck_letter_actions_action CHECK (action IN
    ('SAVE','SUBMIT','READ','APPROVE','RETURN','SIGN','CANCEL','REASSIGN','UPLOAD_SIGNED','RECEIVE'));
ALTER TABLE letter_actions DROP CONSTRAINT ck_letter_actions_from_status;
ALTER TABLE letter_actions ADD CONSTRAINT ck_letter_actions_from_status CHECK (from_status IS NULL OR from_status IN
    ('DRAFT','IN_REVIEW','RETURNED_TO_DRAFTER','IN_APPROVAL','RETURNED_TO_REVIEWER','APPROVED','SENT','RECEIVED','CANCELLED'));
ALTER TABLE letter_actions DROP CONSTRAINT ck_letter_actions_to_status;
ALTER TABLE letter_actions ADD CONSTRAINT ck_letter_actions_to_status CHECK (to_status IN
    ('DRAFT','IN_REVIEW','RETURNED_TO_DRAFTER','IN_APPROVAL','RETURNED_TO_REVIEWER','APPROVED','SENT','RECEIVED','CANCELLED'));

COMMIT TRANSACTION;
