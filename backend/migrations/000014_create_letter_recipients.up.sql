CREATE TABLE letter_recipients (
    id                        BIGINT IDENTITY(1,1) NOT NULL,
    letter_id                 BIGINT           NOT NULL,
    recipient_unit_id         BIGINT           NOT NULL,
    resolved_head_id          BIGINT           NULL,
    sort_order                INT              NOT NULL CONSTRAINT df_letter_recipients_sort_order DEFAULT 0,
    created_at                DATETIMEOFFSET NOT NULL CONSTRAINT df_letter_recipients_created_at DEFAULT SYSDATETIMEOFFSET(),
    updated_at                DATETIMEOFFSET NOT NULL CONSTRAINT df_letter_recipients_updated_at DEFAULT SYSDATETIMEOFFSET(),
    deleted_at                DATETIMEOFFSET NULL,
    CONSTRAINT pk_letter_recipients PRIMARY KEY (id),
    CONSTRAINT fk_letter_recipients_letter FOREIGN KEY (letter_id)         REFERENCES letters (id),
    CONSTRAINT fk_letter_recipients_unit   FOREIGN KEY (recipient_unit_id) REFERENCES units (id),
    CONSTRAINT fk_letter_recipients_head   FOREIGN KEY (resolved_head_id)  REFERENCES users (id)
);

CREATE UNIQUE INDEX uq_letter_recipients_letter_unit ON letter_recipients (letter_id, recipient_unit_id) WHERE deleted_at IS NULL;
CREATE INDEX ix_letter_recipients_unit ON letter_recipients (recipient_unit_id) WHERE deleted_at IS NULL;
