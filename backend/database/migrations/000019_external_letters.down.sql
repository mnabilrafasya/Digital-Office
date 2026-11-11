-- 000019_external_letters.down.sql
-- Hanya bisa dibongkar bila belum ada surat eksternal, status APPROVED/RECEIVED, atau surat dengan
-- lebih dari satu approver.
SET XACT_ABORT ON;
BEGIN TRANSACTION;

ALTER TABLE letter_actions DROP CONSTRAINT ck_letter_actions_to_status;
ALTER TABLE letter_actions ADD CONSTRAINT ck_letter_actions_to_status CHECK (to_status IN
    ('DRAFT','IN_REVIEW','RETURNED_TO_DRAFTER','IN_APPROVAL','RETURNED_TO_REVIEWER','SENT','CANCELLED'));
ALTER TABLE letter_actions DROP CONSTRAINT ck_letter_actions_from_status;
ALTER TABLE letter_actions ADD CONSTRAINT ck_letter_actions_from_status CHECK (from_status IS NULL OR from_status IN
    ('DRAFT','IN_REVIEW','RETURNED_TO_DRAFTER','IN_APPROVAL','RETURNED_TO_REVIEWER','SENT','CANCELLED'));
ALTER TABLE letter_actions DROP CONSTRAINT ck_letter_actions_action;
ALTER TABLE letter_actions ADD CONSTRAINT ck_letter_actions_action CHECK (action IN
    ('SAVE','SUBMIT','READ','APPROVE','RETURN','SIGN','CANCEL','REASSIGN'));

ALTER TABLE letters DROP CONSTRAINT ck_letters_sent_complete;
ALTER TABLE letters ADD CONSTRAINT ck_letters_sent_complete CHECK (
    status <> 'SENT'
    OR (letter_number IS NOT NULL AND approved_at IS NOT NULL AND signed_by_id IS NOT NULL)
);
ALTER TABLE letters DROP CONSTRAINT ck_letters_complete_after_draft;
ALTER TABLE letters ADD CONSTRAINT ck_letters_complete_after_draft CHECK (
    status = 'DRAFT'
    OR (letter_type_id IS NOT NULL AND classification_id IS NOT NULL
        AND confidentiality_level_id IS NOT NULL AND urgency_level_id IS NOT NULL
        AND title IS NOT NULL)
);
ALTER TABLE letters DROP CONSTRAINT ck_letters_status;
ALTER TABLE letters ADD CONSTRAINT ck_letters_status CHECK (status IN
    ('DRAFT','IN_REVIEW','RETURNED_TO_DRAFTER','IN_APPROVAL','RETURNED_TO_REVIEWER','SENT','CANCELLED'));

ALTER TABLE letters DROP CONSTRAINT ck_letters_external_info_json;
ALTER TABLE letters DROP COLUMN external_info;

CREATE UNIQUE INDEX uq_letter_participants_one_approver
    ON letter_participants (letter_id) WHERE role = 'APPROVER' AND deleted_at IS NULL;
ALTER TABLE letter_participants ADD CONSTRAINT ck_letter_participants_approver_step CHECK (role <> 'APPROVER' OR step_order = 1);

COMMIT TRANSACTION;
