-- +goose Up
CREATE TABLE IF NOT EXISTS transactions (
    transaction_id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    wallet_id UUID NOT NULL REFERENCES wallets(wallet_id) ON DELETE CASCADE,
    type VARCHAR(20) NOT NULL, -- deposit, withdraw, exchange
    amount NUMERIC(20, 2) NOT NULL,
    from_currency CHAR(3),
    to_currency CHAR(3),
    created_at TIMESTAMP NOT NULL DEFAULT NOW()
);

-- +goose Down
DROP TABLE IF EXISTS transactions;
