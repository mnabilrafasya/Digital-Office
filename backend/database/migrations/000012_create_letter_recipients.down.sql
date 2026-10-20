-- 000012_create_letter_recipients.down.sql
SET XACT_ABORT ON;
BEGIN TRANSACTION;

DROP TABLE IF EXISTS letter_recipients;

COMMIT TRANSACTION;
