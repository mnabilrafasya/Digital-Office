-- 000016_create_audit_logs.up.sql
-- Jejak permanen. Sengaja TANPA FK supaya tetap utuh walau data aslinya berubah.
-- Tidak boleh di-soft delete (dijaga di Go).
SET XACT_ABORT ON;
BEGIN TRANSACTION;

CREATE TABLE audit_logs (
    id                        BIGINT IDENTITY(1,1) NOT NULL,
    company_code              VARCHAR(20)      NULL,
    entity_type               VARCHAR(30)      NOT NULL,
    entity_id                 BIGINT           NOT NULL,
    entity_uuid               UNIQUEIDENTIFIER NULL,
    action                    VARCHAR(30)      NOT NULL,
    actor_id                  BIGINT           NULL,
    actor_name                NVARCHAR(150)    NOT NULL,
    snapshot                  NVARCHAR(MAX)    NULL,
    ip_address                VARCHAR(45)      NULL,
    created_at                DATETIMEOFFSET NOT NULL CONSTRAINT df_audit_logs_created_at DEFAULT SYSDATETIMEOFFSET(),
    updated_at                DATETIMEOFFSET NOT NULL CONSTRAINT df_audit_logs_updated_at DEFAULT SYSDATETIMEOFFSET(),
    deleted_at                DATETIMEOFFSET NULL,
    CONSTRAINT pk_audit_logs PRIMARY KEY (id),
    CONSTRAINT ck_audit_logs_snapshot_json CHECK (snapshot IS NULL OR ISJSON(snapshot) = 1)
);

CREATE INDEX ix_audit_logs_entity          ON audit_logs (entity_type, entity_id);
CREATE INDEX ix_audit_logs_company_created ON audit_logs (company_code, created_at);

COMMIT TRANSACTION;
