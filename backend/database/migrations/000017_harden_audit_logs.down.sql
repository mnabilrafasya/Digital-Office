-- 000017_harden_audit_logs.down.sql
SET XACT_ABORT ON;
BEGIN TRANSACTION;

DROP TRIGGER IF EXISTS trg_audit_logs_immutable;
DROP INDEX IF EXISTS ix_audit_logs_company_action_created ON audit_logs;
DROP INDEX IF EXISTS ix_audit_logs_company_actor_created ON audit_logs;
ALTER TABLE audit_logs DROP COLUMN IF EXISTS request_id;

COMMIT TRANSACTION;
