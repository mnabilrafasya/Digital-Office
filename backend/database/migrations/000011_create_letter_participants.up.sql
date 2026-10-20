-- 000011_create_letter_participants.up.sql
-- Reviewer (minimal 1, berurutan step_order) dan approver (tepat 1, step_order 1) surat.
SET XACT_ABORT ON;
BEGIN TRANSACTION;

CREATE TABLE letter_participants (
    id                        BIGINT IDENTITY(1,1) NOT NULL,
    letter_id                 BIGINT           NOT NULL,
    user_id                   BIGINT           NOT NULL,
    role                      VARCHAR(20)      NOT NULL,
    step_order                INT              NOT NULL,
    created_at                DATETIMEOFFSET NOT NULL CONSTRAINT df_letter_participants_created_at DEFAULT SYSDATETIMEOFFSET(),
    updated_at                DATETIMEOFFSET NOT NULL CONSTRAINT df_letter_participants_updated_at DEFAULT SYSDATETIMEOFFSET(),
    deleted_at                DATETIMEOFFSET NULL,
    CONSTRAINT pk_letter_participants PRIMARY KEY (id),
    CONSTRAINT fk_letter_participants_letter FOREIGN KEY (letter_id) REFERENCES letters (id),
    CONSTRAINT fk_letter_participants_user   FOREIGN KEY (user_id)   REFERENCES users (id),
    CONSTRAINT ck_letter_participants_role CHECK (role IN ('REVIEWER','APPROVER')),
    CONSTRAINT ck_letter_participants_step CHECK (step_order >= 1),
    -- Approver selalu tahap 1 (hanya ada satu approver per surat)
    CONSTRAINT ck_letter_participants_approver_step CHECK (role <> 'APPROVER' OR step_order = 1)
);

-- Satu orang hanya sekali per peran dalam satu surat
CREATE UNIQUE INDEX uq_letter_participants_letter_user_role ON letter_participants (letter_id, user_id, role) WHERE deleted_at IS NULL;
-- Satu urutan hanya diisi satu orang per peran
CREATE UNIQUE INDEX uq_letter_participants_letter_role_step ON letter_participants (letter_id, role, step_order) WHERE deleted_at IS NULL;
CREATE INDEX ix_letter_participants_user ON letter_participants (user_id) WHERE deleted_at IS NULL;
-- Tepat satu approver aktif per surat
CREATE UNIQUE INDEX uq_letter_participants_one_approver
    ON letter_participants (letter_id) WHERE role = 'APPROVER' AND deleted_at IS NULL;

COMMIT TRANSACTION;
