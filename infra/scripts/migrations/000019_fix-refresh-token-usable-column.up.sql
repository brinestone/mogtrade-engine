drop view if exists refresh_token_states;
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
    created_at
from
    refresh_tokens;