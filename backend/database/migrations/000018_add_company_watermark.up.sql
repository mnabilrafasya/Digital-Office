-- 000018_add_company_watermark.up.sql
-- Watermark naskah resmi per perusahaan (teks, diatur admin perusahaan). NULL = tanpa watermark.
-- PSP yang sudah ada diisi teks watermark yang selama ini dipakai, supaya hasilnya tidak berubah;
-- database baru mendapatkannya dari seed reference.
SET XACT_ABORT ON;
BEGIN TRANSACTION;

ALTER TABLE companies ADD watermark_text NVARCHAR(100) NULL;

-- Lewat sp_executesql karena kolom baru belum dikenal saat batch ini dikompilasi
EXEC sp_executesql N'UPDATE companies SET watermark_text = N''PUPUK SRIWIDJAJA'' WHERE code = ''PSP''';

COMMIT TRANSACTION;
