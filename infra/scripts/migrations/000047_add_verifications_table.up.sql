CREATE TABLE verifications (
    id varchar(26) PRIMARY KEY not null,
    user_id VARCHAR(26) NOT NULL REFERENCES "user"(id) ON DELETE CASCADE,
    token VARCHAR(255) NOT NULL UNIQUE,
    type VARCHAR(50) NOT NULL CHECK (type IN ('email_reset', 'email_verify')),
    expires_at TIMESTAMP WITH TIME ZONE NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    used BOOLEAN DEFAULT FALSE
);