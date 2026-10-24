-- +goose Up
CREATE TYPE payment_direction AS ENUM ('deposit', 'payout');
CREATE TYPE payment_status AS ENUM ('pending', 'succeeded', 'failed', 'cancelled');

CREATE TABLE payments (
    id VARCHAR(26) PRIMARY KEY,
    user_id VARCHAR(26) NOT NULL REFERENCES "user"("id") ON DELETE SET NULL,
    provider TEXT NOT NULL,
    direction payment_direction NOT NULL,
    provider_ref TEXT NOT NULL,
    idempotency_token TEXT NOT NULL UNIQUE,
    amount NUMERIC(18,4) NOT NULL,
    currency TEXT NOT NULL,
    status payment_status NOT NULL DEFAULT 'pending',
    wallet_transaction_id VARCHAR(26) REFERENCES wallet_transactions(id),
    metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (provider, provider_ref)
);
CREATE INDEX idx_payments_user ON payments(user_id);
CREATE INDEX idx_payments_status ON payments(status);

-- +goose Down
DROP TABLE IF EXISTS payments;
DROP TYPE IF EXISTS payment_status;
DROP TYPE IF EXISTS payment_direction;