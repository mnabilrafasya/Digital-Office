-- 000017_harden_audit_logs.up.sql
-- Audit log untuk pelacakan: request_id (menghubungkan log dengan log server), indeks pencarian
-- per pelaku dan per aksi, serta trigger yang menolak UPDATE/DELETE supaya jejak tidak bisa diubah
-- atau dihapus, termasuk oleh aplikasi sendiri.
SET XACT_ABORT ON;
BEGIN TRANSACTION;

ALTER TABLE audit_logs ADD request_id VARCHAR(100) NULL;

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
