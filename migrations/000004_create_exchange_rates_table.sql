-- +goose Up
CREATE TABLE IF NOT EXISTS exchange_rates (
    rate_id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    currency CHAR(3) NOT NULL UNIQUE,
    rate_to_usd NUMERIC(20, 6) NOT NULL,
    updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);

-- +goose Down
DROP TABLE IF EXISTS exchange_rates;
