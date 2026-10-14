CREATE TABLE letter_actions (
    id                        BIGINT IDENTITY(1,1) NOT NULL,
    letter_id                 BIGINT           NOT NULL,
    actor_id                  BIGINT           NOT NULL,
    target_user_id            BIGINT           NULL,
    actor_role                VARCHAR(20)      NOT NULL,
    action                    VARCHAR(30)      NOT NULL,
    from_status               VARCHAR(30)      NULL,
    to_status                 VARCHAR(30)      NOT NULL,
    note                      NVARCHAR(MAX)    NULL,
    version_number            INT              NULL,
    created_at                DATETIMEOFFSET NOT NULL CONSTRAINT df_letter_actions_created_at DEFAULT SYSDATETIMEOFFSET(),
    updated_at                DATETIMEOFFSET NOT NULL CONSTRAINT df_letter_actions_updated_at DEFAULT SYSDATETIMEOFFSET(),
    deleted_at                DATETIMEOFFSET NULL,
    CONSTRAINT pk_letter_actions PRIMARY KEY (id),
    CONSTRAINT fk_letter_actions_letter FOREIGN KEY (letter_id)      REFERENCES letters (id),
    CONSTRAINT fk_letter_actions_actor  FOREIGN KEY (actor_id)       REFERENCES users (id),
    CONSTRAINT fk_letter_actions_target FOREIGN KEY (target_user_id) REFERENCES users (id),
    CONSTRAINT ck_letter_actions_actor_role  CHECK (actor_role IN ('DRAFTER','REVIEWER','APPROVER')),
    CONSTRAINT ck_letter_actions_action      CHECK (action IN ('SAVE','SUBMIT','READ','APPROVE','RETURN','SIGN','CANCEL','REASSIGN')),
    CONSTRAINT ck_letter_actions_from_status CHECK (from_status IS NULL OR from_status IN ('DRAFT','IN_REVIEW','RETURNED_TO_DRAFTER','IN_APPROVAL','RETURNED_TO_REVIEWER','SENT','CANCELLED')),
    CONSTRAINT ck_letter_actions_to_status   CHECK (to_status IN ('DRAFT','IN_REVIEW','RETURNED_TO_DRAFTER','IN_APPROVAL','RETURNED_TO_REVIEWER','SENT','CANCELLED')),
    -- Mengembalikan, membatalkan, dan mengganti orang wajib disertai alasan
    CONSTRAINT ck_letter_actions_note_required CHECK (action NOT IN ('RETURN','CANCEL','REASSIGN') OR note IS NOT NULL)
);

CREATE INDEX ix_letter_actions_letter_created ON letter_actions (letter_id, created_at);
