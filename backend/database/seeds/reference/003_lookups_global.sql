-- 003_lookups_global.sql
-- Lookup yang berlaku untuk semua perusahaan (company_code NULL). Idempoten (MERGE).
-- Kode status surat dan inbox terkunci di Go; di sini hanya label, warna, dan urutan.
SET NOCOUNT ON;
SET XACT_ABORT ON;

MERGE lookups WITH (HOLDLOCK) AS t
USING (VALUES
    -- Sifat
    ('CONFIDENTIALITY', 'BIASA',    N'Biasa',    '#6B7280', 1, N'Boleh diketahui umum di internal perusahaan'),
    ('CONFIDENTIALITY', 'TERBATAS', N'Terbatas', '#F59E0B', 2, N'Hanya untuk unit atau pejabat terkait'),
    ('CONFIDENTIALITY', 'RAHASIA',  N'Rahasia',  '#DC2626', 3, N'Hanya untuk penerima yang dituju'),
    -- Kecepatan
    ('URGENCY', 'BIASA',         N'Biasa',         '#6B7280', 1, N'Ditanggapi sesuai antrean, maksimal 5 hari kerja'),
    ('URGENCY', 'SEGERA',        N'Segera',        '#F59E0B', 2, N'Ditanggapi maksimal 2x24 jam'),
    ('URGENCY', 'SANGAT_SEGERA', N'Sangat Segera', '#DC2626', 3, N'Ditanggapi maksimal 1x24 jam'),
    -- Jenis surat
    ('LETTER_TYPE', 'NOTA_DINAS',       N'Nota Dinas',       NULL, 1, N'Komunikasi resmi antar unit atau pejabat'),
    ('LETTER_TYPE', 'MEMO',             N'Memo Internal',    NULL, 2, N'Pesan singkat, misalnya dari atasan ke bawahan'),
    ('LETTER_TYPE', 'SURAT_UNDANGAN',   N'Surat Undangan',   NULL, 3, N'Mengundang rapat atau acara'),
    ('LETTER_TYPE', 'SURAT_EDARAN',     N'Surat Edaran',     NULL, 4, N'Pemberitahuan atau kebijakan untuk banyak unit'),
    ('LETTER_TYPE', 'SURAT_PERMOHONAN', N'Surat Permohonan', NULL, 5, N'Meminta dana, akses, atau dukungan'),
    ('LETTER_TYPE', 'SURAT_TUGAS',      N'Surat Tugas',      NULL, 6, N'Menugaskan pegawai untuk kegiatan tertentu'),
    ('LETTER_TYPE', 'SURAT_KEPUTUSAN',  N'Surat Keputusan',  NULL, 7, N'Menetapkan keputusan resmi'),
    ('LETTER_TYPE', 'BERITA_ACARA',     N'Berita Acara',     NULL, 8, N'Mencatat kejadian, serah terima, atau hasil pemeriksaan'),
    -- Status surat
    ('LETTER_STATUS', 'DRAFT',                N'Draf',                     '#6B7280', 1, NULL),
    ('LETTER_STATUS', 'IN_REVIEW',            N'Sedang Direview',          '#2563EB', 2, NULL),
    ('LETTER_STATUS', 'RETURNED_TO_DRAFTER',  N'Dikembalikan ke Drafter',  '#D97706', 3, NULL),
    ('LETTER_STATUS', 'IN_APPROVAL',          N'Menunggu Persetujuan',     '#7C3AED', 4, NULL),
    ('LETTER_STATUS', 'RETURNED_TO_REVIEWER', N'Dikembalikan ke Reviewer', '#EA580C', 5, NULL),
    ('LETTER_STATUS', 'SENT',                 N'Terkirim',                 '#16A34A', 6, NULL),
    ('LETTER_STATUS', 'CANCELLED',            N'Dibatalkan',               '#DC2626', 7, NULL),
    -- Status inbox
    ('INBOX_STATUS', 'UNREAD',     N'Belum Dibaca',      '#2563EB', 1, NULL),
    ('INBOX_STATUS', 'READ',       N'Sudah Dibaca',      '#6B7280', 2, NULL),
    ('INBOX_STATUS', 'DISPOSED',   N'Sudah Didisposisi', '#16A34A', 3, NULL),
    ('INBOX_STATUS', 'REDIRECTED', N'Dialihkan',         '#7C3AED', 4, NULL)
) AS s (category, code, name, color, sort_order, note)
    ON t.company_code IS NULL AND t.category = s.category AND t.code = s.code AND t.deleted_at IS NULL
WHEN MATCHED AND (t.name <> s.name
                  OR ISNULL(t.color, '') <> ISNULL(s.color, '')
                  OR t.sort_order <> s.sort_order
                  OR ISNULL(t.note, N'') <> ISNULL(s.note, N'')) THEN
    UPDATE SET name = s.name, color = s.color, sort_order = s.sort_order, note = s.note,
               updated_at = SYSDATETIMEOFFSET()
WHEN NOT MATCHED BY TARGET THEN
    INSERT (company_code, category, code, name, color, sort_order, note)
    VALUES (NULL, s.category, s.code, s.name, s.color, s.sort_order, s.note);
