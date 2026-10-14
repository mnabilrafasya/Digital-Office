CREATE TABLE letter_templates (
    id                        BIGINT IDENTITY(1,1) NOT NULL,
    company_code              VARCHAR(20)      NOT NULL,
    letter_type_id            BIGINT           NULL,
    name                      NVARCHAR(150)    NOT NULL,
    object_key                UNIQUEIDENTIFIER NOT NULL,
    created_by                BIGINT           NOT NULL,
    is_active                 BIT              NOT NULL CONSTRAINT df_letter_templates_is_active DEFAULT 1,
    created_at                DATETIMEOFFSET NOT NULL CONSTRAINT df_letter_templates_created_at DEFAULT SYSDATETIMEOFFSET(),
    updated_at                DATETIMEOFFSET NOT NULL CONSTRAINT df_letter_templates_updated_at DEFAULT SYSDATETIMEOFFSET(),
    deleted_at                DATETIMEOFFSET NULL,
    CONSTRAINT pk_letter_templates PRIMARY KEY (id),
    CONSTRAINT fk_letter_templates_company    FOREIGN KEY (company_code)   REFERENCES companies (code),
    CONSTRAINT fk_letter_templates_type       FOREIGN KEY (letter_type_id) REFERENCES lookups (id),
    CONSTRAINT fk_letter_templates_created_by FOREIGN KEY (created_by)     REFERENCES users (id),
    CONSTRAINT uq_letter_templates_object_key UNIQUE (object_key)
);

CREATE INDEX ix_letter_templates_company_type ON letter_templates (company_code, letter_type_id) WHERE deleted_at IS NULL;
