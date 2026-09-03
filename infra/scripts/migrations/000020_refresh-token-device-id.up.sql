alter table refresh_tokens
add column device_id text not null default ''::text;

create index "refresh_tokens_device_id_idx" on refresh_tokens (device_id);

create or replace view
    refresh_token_states as
select
    id,
    user_id,
    token_hash,
    valid_window,
    (created_at + valid_window)::timestamptz as expires_at,
    revoked,
    not revoked
    and (now() <= (created_at + valid_window)) as usable,
    created_at,
    device_id
from
    refresh_tokens;