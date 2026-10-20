CREATE TYPE verification_status AS ENUM ('primed', 'pending');
CREATE TYPE verification_type AS ENUM ('email_reset', 'email_verify', 'phone_verify');

CREATE TABLE verifications (
    id varchar(26) PRIMARY KEY not null,
    user_id VARCHAR(26) NOT NULL REFERENCES "user"(id) ON DELETE CASCADE,
    token VARCHAR(255) NOT NULL UNIQUE,
    valid_window interval NOT NULL DEFAULT '24 hours',
    type verification_type NOT NULL,
    ip varchar(45) NOT NULL DEFAULT '',
    callback_url text,
    status verification_status NOT NULL DEFAULT 'pending',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    used_at TIMESTAMP WITH TIME ZONE,
    used BOOLEAN DEFAULT FALSE
);
