CREATE TABLE inbox_items (
    id                        BIGINT IDENTITY(1,1) NOT NULL,
    letter_id                 BIGINT           NOT NULL,
    user_id                   BIGINT           NOT NULL,
    source_type               VARCHAR(20)      NOT NULL,
    forward_id                BIGINT           NULL,
    status                    VARCHAR(20)      NOT NULL CONSTRAINT df_inbox_items_status DEFAULT 'UNREAD',
    read_at                   DATETIMEOFFSET   NULL,
    created_at                DATETIMEOFFSET NOT NULL CONSTRAINT df_inbox_items_created_at DEFAULT SYSDATETIMEOFFSET(),
    updated_at                DATETIMEOFFSET NOT NULL CONSTRAINT df_inbox_items_updated_at DEFAULT SYSDATETIMEOFFSET(),
    deleted_at                DATETIMEOFFSET NULL,
    CONSTRAINT pk_inbox_items PRIMARY KEY (id),
    CONSTRAINT fk_inbox_items_letter  FOREIGN KEY (letter_id)  REFERENCES letters (id),
    CONSTRAINT fk_inbox_items_user    FOREIGN KEY (user_id)    REFERENCES users (id),
    CONSTRAINT fk_inbox_items_forward FOREIGN KEY (forward_id) REFERENCES letter_forwards (id),
    CONSTRAINT uq_inbox_items_letter_user_forward UNIQUE (letter_id, user_id, forward_id),
    CONSTRAINT ck_inbox_items_source_type CHECK (source_type IN ('DIRECT','FORWARD')),
    CONSTRAINT ck_inbox_items_status      CHECK (status IN ('UNREAD','READ','DISPOSED','REDIRECTED')),
    -- DIRECT tidak punya forward_id, FORWARD wajib punya forward_id
    CONSTRAINT ck_inbox_items_source CHECK (
        (source_type = 'DIRECT' AND forward_id IS NULL)
        OR (source_type = 'FORWARD' AND forward_id IS NOT NULL)
    )
);

CREATE INDEX ix_inbox_items_user_status ON inbox_items (user_id, status);

ALTER TABLE letter_forwards ADD CONSTRAINT fk_letter_forwards_source_inbox
    FOREIGN KEY (source_inbox_item_id) REFERENCES inbox_items (id);
