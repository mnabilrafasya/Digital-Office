-- 000015_create_inbox_items.down.sql
-- FK lintas tabel dilepas dulu sebelum tabel dihapus.
SET XACT_ABORT ON;
BEGIN TRANSACTION;

ALTER TABLE letter_forwards DROP CONSTRAINT IF EXISTS fk_letter_forwards_source_inbox;

DROP TABLE IF EXISTS inbox_items;

COMMIT TRANSACTION;
