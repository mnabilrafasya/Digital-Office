-- 000016_create_audit_logs.up.sql
-- Jejak audit permanen. Sengaja TANPA FK supaya tetap utuh walau data aslinya berubah.
-- Hanya bisa ditambah: trigger menolak UPDATE/DELETE, termasuk dari aplikasi sendiri.
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
    request_id                VARCHAR(100)     NULL,
    created_at                DATETIMEOFFSET NOT NULL CONSTRAINT df_audit_logs_created_at DEFAULT SYSDATETIMEOFFSET(),
    CONSTRAINT pk_audit_logs PRIMARY KEY (id),
    CONSTRAINT ck_audit_logs_snapshot_json CHECK (snapshot IS NULL OR ISJSON(snapshot) = 1)
);

CREATE INDEX ix_audit_logs_entity                 ON audit_logs (entity_type, entity_id);
CREATE INDEX ix_audit_logs_company_created        ON audit_logs (company_code, created_at);
CREATE INDEX ix_audit_logs_company_actor_created  ON audit_logs (company_code, actor_id, created_at);
CREATE INDEX ix_audit_logs_company_action_created ON audit_logs (company_code, action, created_at);

-- CREATE TRIGGER wajib menjadi perintah pertama dalam satu batch, jadi dijalankan lewat EXEC
EXEC(N'
CREATE TRIGGER trg_audit_logs_immutable ON audit_logs
INSTEAD OF UPDATE, DELETE
AS
BEGIN
    THROW 51000, ''audit_logs hanya boleh ditambah; mengubah atau menghapus jejak audit tidak diizinkan.'', 1;
END');

COMMIT TRANSACTION;
