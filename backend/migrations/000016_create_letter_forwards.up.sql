CREATE TABLE letter_forwards (
    id                        BIGINT IDENTITY(1,1) NOT NULL,
    letter_id                 BIGINT           NOT NULL,
    type                      VARCHAR(20)      NOT NULL,
    source_inbox_item_id      BIGINT           NOT NULL,
    from_user_id              BIGINT           NOT NULL,
    from_user_name            NVARCHAR(150)    NOT NULL,
    from_position             NVARCHAR(150)    NOT NULL,
    target_type               VARCHAR(20)      NOT NULL,
    target_user_id            BIGINT           NULL,
    target_unit_id            BIGINT           NULL,
    note                      NVARCHAR(MAX)    NULL,
    created_at                DATETIMEOFFSET NOT NULL CONSTRAINT df_letter_forwards_created_at DEFAULT SYSDATETIMEOFFSET(),
    updated_at                DATETIMEOFFSET NOT NULL CONSTRAINT df_letter_forwards_updated_at DEFAULT SYSDATETIMEOFFSET(),
    deleted_at                DATETIMEOFFSET NULL,
    CONSTRAINT pk_letter_forwards PRIMARY KEY (id),
    CONSTRAINT fk_letter_forwards_letter      FOREIGN KEY (letter_id)      REFERENCES letters (id),
    CONSTRAINT fk_letter_forwards_from_user   FOREIGN KEY (from_user_id)   REFERENCES users (id),
    CONSTRAINT fk_letter_forwards_target_user FOREIGN KEY (target_user_id) REFERENCES users (id),
    CONSTRAINT fk_letter_forwards_target_unit FOREIGN KEY (target_unit_id) REFERENCES units (id),
    CONSTRAINT ck_letter_forwards_type        CHECK (type IN ('DISPOSITION','REDIRECT')),
    CONSTRAINT ck_letter_forwards_target_type CHECK (target_type IN ('USER','UNIT')),
    -- Tujuan USER wajib isi target_user_id saja, tujuan UNIT wajib isi target_unit_id saja
    CONSTRAINT ck_letter_forwards_target CHECK (
        (target_type = 'USER' AND target_user_id IS NOT NULL AND target_unit_id IS NULL)
        OR (target_type = 'UNIT' AND target_unit_id IS NOT NULL AND target_user_id IS NULL)
    ),
    -- Pengalihan (REDIRECT) hanya ke orang, bukan ke unit
    CONSTRAINT ck_letter_forwards_redirect_user CHECK (type <> 'REDIRECT' OR target_type = 'USER')
);

CREATE INDEX ix_letter_forwards_letter       ON letter_forwards (letter_id);
CREATE INDEX ix_letter_forwards_source_inbox ON letter_forwards (source_inbox_item_id);
