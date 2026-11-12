-- 000020_number_period.up.sql
-- Nomor urut surat mulai lagi setiap bulan (sebelumnya setiap tahun).
-- Kolom letters.number_year (tahun, mis. 2026) diganti menjadi letters.number_period (tahun dan
-- bulan surat disetujui, mis. 202611). Nomor urut unik per perusahaan per periode.
-- Surat lama dikonversi dari bulan approved_at-nya; nomor suratnya tidak berubah.
-- Perintah yang memakai nama kolom baru dijalankan lewat sp_executesql (satu file = satu batch).
SET XACT_ABORT ON;
BEGIN TRANSACTION;

IF EXISTS (SELECT 1 FROM letters WHERE number_year IS NOT NULL AND approved_at IS NULL)
    THROW 50020, N'Ada surat bernomor tanpa tanggal persetujuan; periode nomornya tidak bisa ditentukan.', 1;

DROP INDEX uq_letters_company_year_seq ON letters;
ALTER TABLE letters DROP CONSTRAINT ck_letters_number_all_or_none;

EXEC sp_rename 'letters.number_year', 'number_period', 'COLUMN';
EXEC sp_executesql N'ALTER TABLE letters ALTER COLUMN number_period INT NULL';
EXEC sp_executesql N'
UPDATE letters
SET number_period = number_period * 100 + DATEPART(MONTH, approved_at)
WHERE number_period IS NOT NULL';

EXEC sp_executesql N'
ALTER TABLE letters ADD
    -- Nomor surat: nomor, urutan, dan periode terisi bersamaan atau kosong bersamaan
    CONSTRAINT ck_letters_number_all_or_none CHECK (
        (letter_number IS NULL AND number_seq IS NULL AND number_period IS NULL)
        OR (letter_number IS NOT NULL AND number_seq IS NOT NULL AND number_period IS NOT NULL)
    ),
    -- Periode berbentuk YYYYMM dengan bulan 01-12
    CONSTRAINT ck_letters_number_period CHECK (
        number_period IS NULL OR (number_period BETWEEN 190001 AND 999912 AND number_period % 100 BETWEEN 1 AND 12)
    )';

-- Nomor urut unik per perusahaan per bulan
EXEC sp_executesql N'
CREATE UNIQUE INDEX uq_letters_company_period_seq ON letters (company_code, number_period, number_seq)
    WHERE number_seq IS NOT NULL';

COMMIT TRANSACTION;
