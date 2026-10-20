-- 004_position_levels.sql
-- Level jabatan (lookup POSITION_LEVEL) per perusahaan, dipakai sebagai alias penerima
-- (contoh: alias VP = "Seluruh VP"). Idempoten (MERGE).
SET NOCOUNT ON;
SET XACT_ABORT ON;

MERGE lookups WITH (HOLDLOCK) AS t
USING (
    SELECT c.code AS company_code, v.code, v.name, v.note, v.sort_order
    FROM companies c
    CROSS JOIN (VALUES
        ('DIREKTUR', N'Direktur', N'Direksi',                  1),
        ('SVP',      N'SVP',      N'Senior Vice President',    2),
        ('VP',       N'VP',       N'Vice President',           3),
        ('AVP',      N'AVP',      N'Assistant Vice President', 4),
        ('STAF',     N'Staf',     N'Pelaksana / staf',         5)
    ) AS v (code, name, note, sort_order)
    WHERE c.code IN ('PSP', 'PKG')
) AS s
    ON t.company_code = s.company_code AND t.category = 'POSITION_LEVEL'
       AND t.code = s.code AND t.deleted_at IS NULL
WHEN MATCHED AND (t.name <> s.name OR t.sort_order <> s.sort_order OR ISNULL(t.note, N'') <> s.note) THEN
    UPDATE SET name = s.name, note = s.note, sort_order = s.sort_order, updated_at = SYSDATETIMEOFFSET()
WHEN NOT MATCHED BY TARGET THEN
    INSERT (company_code, category, code, name, sort_order, note)
    VALUES (s.company_code, 'POSITION_LEVEL', s.code, s.name, s.sort_order, s.note);
