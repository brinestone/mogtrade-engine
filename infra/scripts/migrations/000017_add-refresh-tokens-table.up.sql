CREATE TABLE
    refresh_tokens (
        id varchar(26) primary key not null,
        user_id varchar(26) not null,
        token_hash text not null,
        valid_window interval not null,
        revoked boolean not null default false,
        created_at timestamptz not null default now()
    );

CREATE INDEX "refresh_tokens_user_id_idx" ON "refresh_tokens" ("user_id");

CREATE INDEX "refresh_tokens_token_hash_idx" ON "refresh_tokens" ("token_hash");

ALTER TABLE "refresh_tokens"
ADD CONSTRAINT "refresh_tokens_user_id_user_id_fk" FOREIGN KEY (user_id) REFERENCES "user" ("id") ON DELETE CASCADE;

CREATE VIEW
    refresh_token_states AS
SELECT
    id,
    user_id,
    token_hash,
    valid_window,
    (created_at + valid_window)::TIMESTAMPTZ as expires_at,
    revoked,
    not revoked
    and (now() > (created_at + valid_window)) as usable,
    created_at
from
    refresh_tokens;