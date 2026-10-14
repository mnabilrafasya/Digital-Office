ALTER TABLE letter_forwards DROP CONSTRAINT IF EXISTS fk_letter_forwards_source_inbox;

DROP TABLE IF EXISTS inbox_items;
