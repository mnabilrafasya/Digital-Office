-- 000001_create_companies.up.sql
-- Perusahaan dalam grup. Semua data dipisah per perusahaan lewat company_code.
SET XACT_ABORT ON;
BEGIN TRANSACTION;

CREATE TABLE companies (
    code                  VARCHAR(20)    NOT NULL,
    name                  NVARCHAR(150)  NOT NULL,
    city                  NVARCHAR(50)   NULL,
    letter_number_format  VARCHAR(100)   NULL,
    is_active             BIT            NOT NULL CONSTRAINT df_companies_is_active  DEFAULT 1,
    created_at            DATETIMEOFFSET NOT NULL CONSTRAINT df_companies_created_at DEFAULT SYSDATETIMEOFFSET(),
    updated_at            DATETIMEOFFSET NOT NULL CONSTRAINT df_companies_updated_at DEFAULT SYSDATETIMEOFFSET(),
    deleted_at            DATETIMEOFFSET NULL,
    CONSTRAINT pk_companies PRIMARY KEY (code)
);

COMMIT TRANSACTION;
