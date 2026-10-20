-- 002_roles_permissions.sql
-- Role (1 user = 1 role) dan permission per role. Idempoten (MERGE).
SET NOCOUNT ON;
SET XACT_ABORT ON;

MERGE roles WITH (HOLDLOCK) AS t
USING (VALUES
    ('ADMIN',    N'Administrator Sistem'),
    ('PIMPINAN', N'Pimpinan / Pejabat Struktural'),
    ('PEGAWAI',  N'Pegawai')
) AS s (code, name)
    ON t.code = s.code AND t.deleted_at IS NULL
WHEN MATCHED AND t.name <> s.name THEN
    UPDATE SET name = s.name, updated_at = SYSDATETIMEOFFSET()
WHEN NOT MATCHED BY TARGET THEN
    INSERT (code, name) VALUES (s.code, s.name);

MERGE permissions WITH (HOLDLOCK) AS t
USING (
    SELECT r.id AS role_id, v.code, v.name
    FROM (VALUES
        ('ADMIN',    'LETTER_CREATE',   N'Membuat dan mengajukan surat'),
        ('ADMIN',    'LETTER_REVIEW',   N'Dapat dipilih sebagai reviewer'),
        ('ADMIN',    'TEMPLATE_CREATE', N'Membuat template surat'),
        ('ADMIN',    'AUDIT_VIEW',      N'Melihat log sistem'),
        ('PIMPINAN', 'LETTER_CREATE',   N'Membuat dan mengajukan surat'),
        ('PIMPINAN', 'LETTER_REVIEW',   N'Dapat dipilih sebagai reviewer'),
        ('PIMPINAN', 'LETTER_APPROVE',  N'Dapat dipilih sebagai approver (penandatangan)'),
        ('PIMPINAN', 'INBOX_DISPOSE',   N'Mendisposisikan surat masuk'),
        ('PIMPINAN', 'TEMPLATE_CREATE', N'Membuat template surat'),
        ('PEGAWAI',  'LETTER_CREATE',   N'Membuat dan mengajukan surat'),
        ('PEGAWAI',  'LETTER_REVIEW',   N'Dapat dipilih sebagai reviewer'),
        ('PEGAWAI',  'TEMPLATE_CREATE', N'Membuat template surat')
    ) AS v (role_code, code, name)
    JOIN roles r ON r.code = v.role_code AND r.deleted_at IS NULL
) AS s
    ON t.role_id = s.role_id AND t.code = s.code AND t.deleted_at IS NULL
WHEN MATCHED AND t.name <> s.name THEN
    UPDATE SET name = s.name, updated_at = SYSDATETIMEOFFSET()
WHEN NOT MATCHED BY TARGET THEN
    INSERT (role_id, code, name) VALUES (s.role_id, s.code, s.name);
