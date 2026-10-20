-- 000009_create_letters.up.sql
-- Surat. Isian boleh kosong selama DRAFT (simpan otomatis), dikunci CHECK setelah keluar dari DRAFT.
-- row_ver (ROWVERSION) dipakai sebagai ETag untuk optimistic locking.
SET XACT_ABORT ON;
BEGIN TRANSACTION;

CREATE TABLE letters (
    id                        BIGINT IDENTITY(1,1) NOT NULL,
    uuid                      UNIQUEIDENTIFIER NOT NULL CONSTRAINT df_letters_uuid DEFAULT NEWID(),
    company_code              VARCHAR(20)      NOT NULL,
    letter_number             VARCHAR(100)     NULL,
    number_seq                INT              NULL,
    number_year               SMALLINT         NULL,
    template_id               BIGINT           NULL,
    letter_type_id            BIGINT           NULL,
    classification_id         BIGINT           NULL,
    confidentiality_level_id  BIGINT           NULL,
    urgency_level_id          BIGINT           NULL,
    title                     NVARCHAR(255)    NULL,
    summary                   NVARCHAR(MAX)    NULL,
    special_note              NVARCHAR(MAX)    NULL,
    attachments               NVARCHAR(MAX)    NULL,
    status                    VARCHAR(30)      NOT NULL CONSTRAINT df_letters_status DEFAULT 'DRAFT',
    sender_unit_id            BIGINT           NOT NULL,
    drafter_id                BIGINT           NOT NULL,
    current_version_number    INT              NOT NULL CONSTRAINT df_letters_current_version_number DEFAULT 0,
    current_assignee_id       BIGINT           NULL,
    forwarded_by_id           BIGINT           NULL,
    returned_by_reviewer_id   BIGINT           NULL,
    returned_by_approver_id   BIGINT           NULL,
    signed_by_id              BIGINT           NULL,
    sender_unit_name          NVARCHAR(150)    NULL,
    approved_at               DATETIMEOFFSET   NULL,
    cancelled_by_id           BIGINT           NULL,
    cancelled_at              DATETIMEOFFSET   NULL,
    row_ver                   ROWVERSION,
    created_at                DATETIMEOFFSET NOT NULL CONSTRAINT df_letters_created_at DEFAULT SYSDATETIMEOFFSET(),
    updated_at                DATETIMEOFFSET NOT NULL CONSTRAINT df_letters_updated_at DEFAULT SYSDATETIMEOFFSET(),
    deleted_at                DATETIMEOFFSET NULL,
    CONSTRAINT pk_letters PRIMARY KEY (id),
    CONSTRAINT uq_letters_uuid UNIQUE (uuid),
    CONSTRAINT fk_letters_company          FOREIGN KEY (company_code)             REFERENCES companies (code),
    CONSTRAINT fk_letters_template         FOREIGN KEY (template_id)              REFERENCES letter_templates (id),
    CONSTRAINT fk_letters_type             FOREIGN KEY (letter_type_id)           REFERENCES lookups (id),
    CONSTRAINT fk_letters_classification   FOREIGN KEY (classification_id)        REFERENCES lookups (id),
    CONSTRAINT fk_letters_confidentiality  FOREIGN KEY (confidentiality_level_id) REFERENCES lookups (id),
    CONSTRAINT fk_letters_urgency          FOREIGN KEY (urgency_level_id)         REFERENCES lookups (id),
    CONSTRAINT fk_letters_sender_unit      FOREIGN KEY (sender_unit_id)           REFERENCES units (id),
    CONSTRAINT fk_letters_drafter          FOREIGN KEY (drafter_id)               REFERENCES users (id),
    CONSTRAINT fk_letters_assignee         FOREIGN KEY (current_assignee_id)      REFERENCES users (id),
    CONSTRAINT fk_letters_forwarded_by     FOREIGN KEY (forwarded_by_id)          REFERENCES users (id),
    CONSTRAINT fk_letters_returned_by_rev  FOREIGN KEY (returned_by_reviewer_id)  REFERENCES users (id),
    CONSTRAINT fk_letters_returned_by_appr FOREIGN KEY (returned_by_approver_id)  REFERENCES users (id),
    CONSTRAINT fk_letters_signed_by        FOREIGN KEY (signed_by_id)             REFERENCES users (id),
    CONSTRAINT fk_letters_cancelled_by     FOREIGN KEY (cancelled_by_id)          REFERENCES users (id),
    CONSTRAINT ck_letters_status CHECK (status IN ('DRAFT','IN_REVIEW','RETURNED_TO_DRAFTER','IN_APPROVAL','RETURNED_TO_REVIEWER','SENT','CANCELLED')),
    -- Lampiran: daftar file dalam format JSON (hanya drafter yang boleh mengubah, dijaga di Go)
    CONSTRAINT ck_letters_attachments_json CHECK (attachments IS NULL OR ISJSON(attachments) = 1),
    -- Draf boleh belum lengkap (simpan otomatis). Setelah keluar dari DRAFT, field utama wajib terisi.
    CONSTRAINT ck_letters_complete_after_draft CHECK (
        status = 'DRAFT'
        OR (letter_type_id IS NOT NULL AND classification_id IS NOT NULL
            AND confidentiality_level_id IS NOT NULL AND urgency_level_id IS NOT NULL
            AND title IS NOT NULL)
    ),
    -- Nomor surat: ketiga kolom terisi bersamaan atau kosong bersamaan
    CONSTRAINT ck_letters_number_all_or_none CHECK (
        (letter_number IS NULL AND number_seq IS NULL AND number_year IS NULL)
        OR (letter_number IS NOT NULL AND number_seq IS NOT NULL AND number_year IS NOT NULL)
    ),
    -- Surat terkirim pasti bernomor, bertanggal, dan ada penandatangannya
    CONSTRAINT ck_letters_sent_complete CHECK (
        status <> 'SENT'
        OR (letter_number IS NOT NULL AND approved_at IS NOT NULL AND signed_by_id IS NOT NULL)
    ),
    -- Surat batal pasti tercatat siapa dan kapan membatalkan
    CONSTRAINT ck_letters_cancel_complete CHECK (
        status <> 'CANCELLED'
        OR (cancelled_by_id IS NOT NULL AND cancelled_at IS NOT NULL)
    )
);

-- Nomor surat unik selamanya (tidak difilter deleted_at)
CREATE UNIQUE INDEX uq_letters_letter_number ON letters (letter_number) WHERE letter_number IS NOT NULL;
-- Nomor urut unik per unit per tahun
CREATE UNIQUE INDEX uq_letters_unit_year_seq ON letters (sender_unit_id, number_year, number_seq) WHERE number_seq IS NOT NULL;
-- Daftar tugas (reviewer/approver), daftar surat drafter, dan daftar per perusahaan
CREATE INDEX ix_letters_assignee_status ON letters (current_assignee_id, status) WHERE deleted_at IS NULL;
CREATE INDEX ix_letters_drafter_status  ON letters (drafter_id, status)          WHERE deleted_at IS NULL;
CREATE INDEX ix_letters_company_status  ON letters (company_code, status)        WHERE deleted_at IS NULL;

COMMIT TRANSACTION;
