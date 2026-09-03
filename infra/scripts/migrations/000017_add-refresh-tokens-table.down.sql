DROP VIEW IF EXISTS refresh_token_states;

ALTER TABLE "refresh_tokens"
DROP CONSTRAINT refresh_tokens_user_id_user_id_fk;

drop index refresh_tokens_token_hash_idx;

drop index refresh_tokens_user_id_idx;

drop table refresh_tokens;