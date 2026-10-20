-- 004_classifications.sql
-- DATA DUMMY DEVELOPMENT: klasifikasi arsip per perusahaan (lookup CLASSIFICATION).
SET NOCOUNT ON;
SET XACT_ABORT ON;

INSERT INTO lookups (company_code, category, code, name, sort_order) VALUES
    ('PSP', 'CLASSIFICATION', 'UM.01',  N'Persuratan dan Kearsipan',               1),
    ('PSP', 'CLASSIFICATION', 'UM.02',  N'Rapat dan Koordinasi',                   2),
    ('PSP', 'CLASSIFICATION', 'UM.03',  N'Fasilitas dan Rumah Tangga Kantor',      3),
    ('PSP', 'CLASSIFICATION', 'SDM.01', N'Rekrutmen dan Penempatan',               4),
    ('PSP', 'CLASSIFICATION', 'SDM.02', N'Pelatihan dan Pengembangan',             5),
    ('PSP', 'CLASSIFICATION', 'SDM.03', N'Kesejahteraan dan Penggajian',           6),
    ('PSP', 'CLASSIFICATION', 'KU.01',  N'Anggaran',                               7),
    ('PSP', 'CLASSIFICATION', 'KU.02',  N'Pembayaran dan Perbendaharaan',          8),
    ('PSP', 'CLASSIFICATION', 'KU.03',  N'Akuntansi dan Pelaporan',                9),
    ('PSP', 'CLASSIFICATION', 'PG.01',  N'Perencanaan Pengadaan',                 10),
    ('PSP', 'CLASSIFICATION', 'PG.02',  N'Pelaksanaan Pengadaan Barang dan Jasa', 11),
    ('PSP', 'CLASSIFICATION', 'PG.03',  N'Pengelolaan Vendor',                    12),
    ('PSP', 'CLASSIFICATION', 'HK.01',  N'Peraturan dan Kebijakan Internal',      13),
    ('PSP', 'CLASSIFICATION', 'HK.02',  N'Perjanjian dan Kontrak',                14),
    ('PSP', 'CLASSIFICATION', 'HK.03',  N'Kepatuhan dan Tata Kelola (GCG)',       15),
    ('PSP', 'CLASSIFICATION', 'TI.01',  N'Tata Kelola dan Kebijakan TI',          16),
    ('PSP', 'CLASSIFICATION', 'TI.02',  N'Pengembangan Sistem Aplikasi',          17),
    ('PSP', 'CLASSIFICATION', 'TI.03',  N'Infrastruktur dan Jaringan',            18),
    ('PSP', 'CLASSIFICATION', 'TI.04',  N'Keamanan Informasi',                    19),
    ('PSP', 'CLASSIFICATION', 'PW.01',  N'Audit Internal',                        20),
    ('PSP', 'CLASSIFICATION', 'PW.02',  N'Manajemen Risiko',                      21),
    ('PSP', 'CLASSIFICATION', 'PR.01',  N'Perencanaan Produksi',                  22),
    ('PSP', 'CLASSIFICATION', 'PR.02',  N'Pemeliharaan Pabrik',                   23),
    ('PSP', 'CLASSIFICATION', 'PR.03',  N'Pengendalian Mutu',                     24),
    ('PSP', 'CLASSIFICATION', 'PS.01',  N'Penjualan',                             25),
    ('PSP', 'CLASSIFICATION', 'PS.02',  N'Distribusi dan Logistik',               26),
    ('PSP', 'CLASSIFICATION', 'K3.01',  N'Keselamatan dan Kesehatan Kerja',       27),
    ('PSP', 'CLASSIFICATION', 'K3.02',  N'Pengelolaan Lingkungan',                28),
    ('PKG', 'CLASSIFICATION', 'UM.01',  N'Persuratan dan Kearsipan',               1),
    ('PKG', 'CLASSIFICATION', 'KU.01',  N'Anggaran',                               2),
    ('PKG', 'CLASSIFICATION', 'TI.01',  N'Tata Kelola dan Kebijakan TI',           3);

IF (SELECT COUNT(*) FROM lookups WHERE category = 'CLASSIFICATION') <> 31
    THROW 50104, N'Seed dev gagal: jumlah klasifikasi harus 31', 1;
