CREATE TABLE lookups (
    id            BIGINT IDENTITY(1,1) NOT NULL,
    company_code  VARCHAR(20)    NULL,
    category      VARCHAR(30)    NOT NULL,
    code          VARCHAR(30)    NOT NULL,
    name          NVARCHAR(100)  NOT NULL,
    color         VARCHAR(7)     NULL,
    sort_order    INT            NOT NULL CONSTRAINT df_lookups_sort_order DEFAULT 0,
    is_active     BIT            NOT NULL CONSTRAINT df_lookups_is_active  DEFAULT 1,
    note          NVARCHAR(MAX)  NULL,
    created_at    DATETIMEOFFSET NOT NULL CONSTRAINT df_lookups_created_at DEFAULT SYSDATETIMEOFFSET(),
    updated_at    DATETIMEOFFSET NOT NULL CONSTRAINT df_lookups_updated_at DEFAULT SYSDATETIMEOFFSET(),
    deleted_at    DATETIMEOFFSET NULL,
    CONSTRAINT pk_lookups PRIMARY KEY (id),
    CONSTRAINT fk_lookups_company FOREIGN KEY (company_code) REFERENCES companies (code),
    CONSTRAINT ck_lookups_category CHECK (category IN ('LETTER_TYPE','CONFIDENTIALITY','URGENCY','LETTER_STATUS','INBOX_STATUS'))
);
CREATE UNIQUE INDEX uq_lookups_company_category_code ON lookups (company_code, category, code) WHERE deleted_at IS NULL;