drop view if exists refresh_token_states;

alter table refresh_tokens
drop column revoked;

alter table refresh_tokens
add column revoked_at timestamptz;

create or replace view
    refresh_token_states as
select
    id,
    user_id,
    token_hash,
    valid_window,
    (created_at + valid_window)::timestamptz as expires_at,
    revoked_at,
    revoked_at is not null as revoked,
    revoked_at is null
    and (now() <= (created_at + valid_window)) as usable,
    created_at,
    device_id
from
    refresh_tokens;