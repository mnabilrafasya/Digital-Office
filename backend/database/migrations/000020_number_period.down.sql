-- 000020_number_period.down.sql
-- Kembali ke nomor urut per tahun: letters.number_period (YYYYMM) menjadi letters.number_year (YYYY).
-- Ditolak bila sudah ada nomor urut kembar dalam satu tahun (hasil reset bulanan), karena aturan
-- unik per tahun tidak bisa dipasang lagi tanpa mengubah nomor surat yang sudah terbit.
SET XACT_ABORT ON;
BEGIN TRANSACTION;

EXEC sp_executesql N'
IF EXISTS (
    SELECT 1 FROM letters
    WHERE number_seq IS NOT NULL
    GROUP BY company_code, number_period / 100, number_seq
    HAVING COUNT(*) > 1
)
    THROW 50021, N''Tidak bisa kembali ke nomor urut per tahun: sudah ada nomor urut yang sama dalam satu tahun.'', 1';

DROP INDEX uq_letters_company_period_seq ON letters;
ALTER TABLE letters DROP CONSTRAINT ck_letters_number_period;
ALTER TABLE letters DROP CONSTRAINT ck_letters_number_all_or_none;

EXEC sp_executesql N'UPDATE letters SET number_period = number_period / 100 WHERE number_period IS NOT NULL';
EXEC sp_executesql N'ALTER TABLE letters ALTER COLUMN number_period SMALLINT NULL';
EXEC sp_rename 'letters.number_period', 'number_year', 'COLUMN';

EXEC sp_executesql N'
ALTER TABLE letters ADD CONSTRAINT ck_letters_number_all_or_none CHECK (
    (letter_number IS NULL AND number_seq IS NULL AND number_year IS NULL)
    OR (letter_number IS NOT NULL AND number_seq IS NOT NULL AND number_year IS NOT NULL)
)';

-- Nomor urut unik per perusahaan per tahun
EXEC sp_executesql N'
CREATE UNIQUE INDEX uq_letters_company_year_seq ON letters (company_code, number_year, number_seq)
    WHERE number_seq IS NOT NULL';

COMMIT TRANSACTION;
